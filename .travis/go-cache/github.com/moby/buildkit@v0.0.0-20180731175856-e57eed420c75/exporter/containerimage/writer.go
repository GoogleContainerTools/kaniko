package containerimage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/diff"
	"github.com/containerd/containerd/images"
	"github.com/moby/buildkit/cache"
	"github.com/moby/buildkit/cache/blobs"
	"github.com/moby/buildkit/exporter"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/snapshot"
	"github.com/moby/buildkit/util/progress"
	"github.com/moby/buildkit/util/system"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	emptyGZLayer = digest.Digest("sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1")
)

type WriterOpt struct {
	Snapshotter  snapshot.Snapshotter
	ContentStore content.Store
	Differ       diff.Comparer
}

func NewImageWriter(opt WriterOpt) (*ImageWriter, error) {
	return &ImageWriter{opt: opt}, nil
}

type ImageWriter struct {
	opt WriterOpt
}

func (ic *ImageWriter) Commit(ctx context.Context, inp exporter.Source, oci bool) (*ocispec.Descriptor, error) {
	platformsBytes, ok := inp.Metadata[exptypes.ExporterPlatformsKey]

	if len(inp.Refs) > 0 && !ok {
		return nil, errors.Errorf("unable to export multiple refs, missing platforms mapping")
	}

	if len(inp.Refs) == 0 {
		layers, err := ic.exportLayers(ctx, inp.Ref)
		if err != nil {
			return nil, err
		}
		return ic.commitDistributionManifest(ctx, inp.Ref, inp.Metadata[exptypes.ExporterImageConfigKey], layers[0], oci)
	}

	var p exptypes.Platforms
	if err := json.Unmarshal(platformsBytes, &p); err != nil {
		return nil, errors.Wrapf(err, "failed to parse platforms passed to exporter")
	}

	if len(p.Platforms) != len(inp.Refs) {
		return nil, errors.Errorf("number of platforms does not match references %d %d", len(p.Platforms), len(inp.Refs))
	}

	refs := make([]cache.ImmutableRef, 0, len(inp.Refs))
	layersMap := make(map[string]int, len(inp.Refs))
	for id, r := range inp.Refs {
		layersMap[id] = len(refs)
		refs = append(refs, r)
	}

	layers, err := ic.exportLayers(ctx, refs...)
	if err != nil {
		return nil, err
	}

	idx := struct {
		// MediaType is reserved in the OCI spec but
		// excluded from go types.
		MediaType string `json:"mediaType,omitempty"`

		ocispec.Index
	}{
		MediaType: ocispec.MediaTypeImageIndex,
		Index: ocispec.Index{
			Versioned: specs.Versioned{
				SchemaVersion: 2,
			},
		},
	}

	if !oci {
		idx.MediaType = images.MediaTypeDockerSchema2ManifestList
	}

	labels := map[string]string{}

	for i, p := range p.Platforms {
		r, ok := inp.Refs[p.ID]
		if !ok {
			return nil, errors.Errorf("failed to find ref for ID %s", p.ID)
		}
		config := inp.Metadata[fmt.Sprintf("%s/%s", exptypes.ExporterImageConfigKey, p.ID)]

		desc, err := ic.commitDistributionManifest(ctx, r, config, layers[layersMap[p.ID]], oci)
		if err != nil {
			return nil, err
		}
		dp := p.Platform
		desc.Platform = &dp
		idx.Manifests = append(idx.Manifests, *desc)

		labels[fmt.Sprintf("containerd.io/gc.ref.content.%d", i)] = desc.Digest.String()
	}

	idxBytes, err := json.MarshalIndent(idx, "", "   ")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal index")
	}

	idxDigest := digest.FromBytes(idxBytes)
	idxDesc := ocispec.Descriptor{
		Digest:    idxDigest,
		Size:      int64(len(idxBytes)),
		MediaType: idx.MediaType,
	}
	idxDone := oneOffProgress(ctx, "exporting manifest list "+idxDigest.String())

	if err := content.WriteBlob(ctx, ic.opt.ContentStore, idxDigest.String(), bytes.NewReader(idxBytes), idxDesc, content.WithLabels(labels)); err != nil {
		return nil, idxDone(errors.Wrapf(err, "error writing manifest list blob %s", idxDigest))
	}
	idxDone(nil)

	for _, desc := range idx.Manifests {
		// delete manifest root. manifest will remain linked to the index
		if err := ic.opt.ContentStore.Delete(context.TODO(), desc.Digest); err != nil {
			return nil, errors.Wrap(err, "error removing manifest root")
		}
	}

	return &idxDesc, nil
}

func (ic *ImageWriter) exportLayers(ctx context.Context, refs ...cache.ImmutableRef) ([][]blobs.DiffPair, error) {
	eg, ctx := errgroup.WithContext(ctx)
	layersDone := oneOffProgress(ctx, "exporting layers")

	out := make([][]blobs.DiffPair, len(refs))

	for i, ref := range refs {
		func(i int, ref cache.ImmutableRef) {
			eg.Go(func() error {
				diffPairs, err := blobs.GetDiffPairs(ctx, ic.opt.ContentStore, ic.opt.Snapshotter, ic.opt.Differ, ref, true)
				if err != nil {
					return errors.Wrap(err, "failed calculaing diff pairs for exported snapshot")
				}
				out[i] = diffPairs
				return nil
			})
		}(i, ref)
	}

	if err := layersDone(eg.Wait()); err != nil {
		return nil, err
	}

	return out, nil
}

func (ic *ImageWriter) commitDistributionManifest(ctx context.Context, ref cache.ImmutableRef, config []byte, layers []blobs.DiffPair, oci bool) (*ocispec.Descriptor, error) {
	if len(config) == 0 {
		var err error
		config, err = emptyImageConfig()
		if err != nil {
			return nil, err
		}
	}

	history, err := parseHistoryFromConfig(config)
	if err != nil {
		return nil, err
	}

	diffPairs, history := normalizeLayersAndHistory(layers, history, ref)

	config, err = patchImageConfig(config, diffPairs, history)
	if err != nil {
		return nil, err
	}

	var (
		configDigest = digest.FromBytes(config)
		manifestType = ocispec.MediaTypeImageManifest
		configType   = ocispec.MediaTypeImageConfig
		layerType    = ocispec.MediaTypeImageLayerGzip
	)

	// Use docker media types for older Docker versions and registries
	if !oci {
		manifestType = images.MediaTypeDockerSchema2Manifest
		configType = images.MediaTypeDockerSchema2Config
		layerType = images.MediaTypeDockerSchema2LayerGzip
	}

	mfst := struct {
		// MediaType is reserved in the OCI spec but
		// excluded from go types.
		MediaType string `json:"mediaType,omitempty"`

		ocispec.Manifest
	}{
		MediaType: manifestType,
		Manifest: ocispec.Manifest{
			Versioned: specs.Versioned{
				SchemaVersion: 2,
			},
			Config: ocispec.Descriptor{
				Digest:    configDigest,
				Size:      int64(len(config)),
				MediaType: configType,
			},
		},
	}

	labels := map[string]string{
		"containerd.io/gc.ref.content.0": configDigest.String(),
	}

	for i, dp := range diffPairs {
		info, err := ic.opt.ContentStore.Info(ctx, dp.Blobsum)
		if err != nil {
			return nil, errors.Wrapf(err, "could not find blob %s from contentstore", dp.Blobsum)
		}
		mfst.Layers = append(mfst.Layers, ocispec.Descriptor{
			Digest:    dp.Blobsum,
			Size:      info.Size,
			MediaType: layerType,
		})
		labels[fmt.Sprintf("containerd.io/gc.ref.content.%d", i+1)] = dp.Blobsum.String()
	}

	mfstJSON, err := json.MarshalIndent(mfst, "", "   ")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal manifest")
	}

	mfstDigest := digest.FromBytes(mfstJSON)
	mfstDesc := ocispec.Descriptor{
		Digest: mfstDigest,
		Size:   int64(len(mfstJSON)),
	}
	mfstDone := oneOffProgress(ctx, "exporting manifest "+mfstDigest.String())

	if err := content.WriteBlob(ctx, ic.opt.ContentStore, mfstDigest.String(), bytes.NewReader(mfstJSON), mfstDesc, content.WithLabels(labels)); err != nil {
		return nil, mfstDone(errors.Wrapf(err, "error writing manifest blob %s", mfstDigest))
	}
	mfstDone(nil)

	configDesc := ocispec.Descriptor{
		Digest:    configDigest,
		Size:      int64(len(config)),
		MediaType: configType,
	}
	configDone := oneOffProgress(ctx, "exporting config "+configDigest.String())

	if err := content.WriteBlob(ctx, ic.opt.ContentStore, configDigest.String(), bytes.NewReader(config), configDesc); err != nil {
		return nil, configDone(errors.Wrap(err, "error writing config blob"))
	}
	configDone(nil)

	// delete config root. config will remain linked to the manifest
	if err := ic.opt.ContentStore.Delete(context.TODO(), configDigest); err != nil {
		return nil, errors.Wrap(err, "error removing config root")
	}

	return &ocispec.Descriptor{
		Digest:    mfstDigest,
		Size:      int64(len(mfstJSON)),
		MediaType: manifestType,
	}, nil
}

func (ic *ImageWriter) ContentStore() content.Store {
	return ic.opt.ContentStore
}

func emptyImageConfig() ([]byte, error) {
	img := ocispec.Image{
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}
	img.RootFS.Type = "layers"
	img.Config.WorkingDir = "/"
	img.Config.Env = []string{"PATH=" + system.DefaultPathEnv}
	dt, err := json.Marshal(img)
	return dt, errors.Wrap(err, "failed to create empty image config")
}

func parseHistoryFromConfig(dt []byte) ([]ocispec.History, error) {
	var config struct {
		History []ocispec.History
	}
	if err := json.Unmarshal(dt, &config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal history from config")
	}
	return config.History, nil
}

func patchImageConfig(dt []byte, dps []blobs.DiffPair, history []ocispec.History) ([]byte, error) {
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(dt, &m); err != nil {
		return nil, errors.Wrap(err, "failed to parse image config for patch")
	}

	var rootFS ocispec.RootFS
	rootFS.Type = "layers"
	for _, dp := range dps {
		rootFS.DiffIDs = append(rootFS.DiffIDs, dp.DiffID)
	}
	dt, err := json.Marshal(rootFS)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal rootfs")
	}
	m["rootfs"] = dt

	dt, err = json.Marshal(history)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal history")
	}
	m["history"] = dt

	if _, ok := m["created"]; !ok {
		var tm *time.Time
		for _, h := range history {
			if h.Created != nil {
				tm = h.Created
			}
		}
		dt, err = json.Marshal(&tm)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal creation time")
		}
		m["created"] = dt
	}

	dt, err = json.Marshal(m)
	return dt, errors.Wrap(err, "failed to marshal config after patch")
}

func normalizeLayersAndHistory(diffs []blobs.DiffPair, history []ocispec.History, ref cache.ImmutableRef) ([]blobs.DiffPair, []ocispec.History) {

	refMeta := getRefMetadata(ref, len(diffs))

	var historyLayers int
	for _, h := range history {
		if !h.EmptyLayer {
			historyLayers += 1
		}
	}

	if historyLayers > len(diffs) {
		// this case shouldn't happen but if it does force set history layers empty
		// from the bottom
		logrus.Warn("invalid image config with unaccounted layers")
		historyCopy := make([]ocispec.History, 0, len(history))
		var l int
		for _, h := range history {
			if l >= len(diffs) {
				h.EmptyLayer = true
			}
			if !h.EmptyLayer {
				l++
			}
			historyCopy = append(historyCopy, h)
		}
		history = historyCopy
	}

	if len(diffs) > historyLayers {
		// some history items are missing. add them based on the ref metadata
		for _, md := range refMeta[historyLayers:] {
			history = append(history, ocispec.History{
				Created:   &md.createdAt,
				CreatedBy: md.description,
				Comment:   "buildkit.exporter.image.v0",
			})
		}
	}

	var layerIndex int
	for i, h := range history {
		if !h.EmptyLayer {
			if h.Created == nil {
				h.Created = &refMeta[layerIndex].createdAt
			}
			if diffs[layerIndex].Blobsum == emptyGZLayer {
				h.EmptyLayer = true
				diffs = append(diffs[:layerIndex], diffs[layerIndex+1:]...)
			} else {
				layerIndex++
			}
		}
		history[i] = h
	}

	return diffs, history
}

type refMetadata struct {
	description string
	createdAt   time.Time
}

func getRefMetadata(ref cache.ImmutableRef, limit int) []refMetadata {
	if limit <= 0 {
		return nil
	}
	meta := refMetadata{
		description: "created by buildkit", // shouldn't be shown but don't fail build
		createdAt:   time.Now(),
	}
	if ref == nil {
		return append(getRefMetadata(nil, limit-1), meta)
	}
	if descr := cache.GetDescription(ref.Metadata()); descr != "" {
		meta.description = descr
	}
	meta.createdAt = cache.GetCreatedAt(ref.Metadata())
	p := ref.Parent()
	if p != nil {
		defer p.Release(context.TODO())
	}
	return append(getRefMetadata(p, limit-1), meta)
}

func oneOffProgress(ctx context.Context, id string) func(err error) error {
	pw, _, _ := progress.FromContext(ctx)
	now := time.Now()
	st := progress.Status{
		Started: &now,
	}
	pw.Write(id, st)
	return func(err error) error {
		// TODO: set error on status
		now := time.Now()
		st.Completed = &now
		pw.Write(id, st)
		pw.Close()
		return err
	}
}
