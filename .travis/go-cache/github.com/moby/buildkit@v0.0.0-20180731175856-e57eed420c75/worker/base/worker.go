package base

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/diff"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/rootfs"
	cdsnapshot "github.com/containerd/containerd/snapshots"
	"github.com/moby/buildkit/cache"
	"github.com/moby/buildkit/cache/blobs"
	"github.com/moby/buildkit/cache/metadata"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/executor"
	"github.com/moby/buildkit/exporter"
	imageexporter "github.com/moby/buildkit/exporter/containerimage"
	localexporter "github.com/moby/buildkit/exporter/local"
	ociexporter "github.com/moby/buildkit/exporter/oci"
	"github.com/moby/buildkit/frontend"
	gw "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/snapshot"
	"github.com/moby/buildkit/snapshot/imagerefchecker"
	"github.com/moby/buildkit/solver"
	"github.com/moby/buildkit/solver/llbsolver/ops"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/source"
	"github.com/moby/buildkit/source/containerimage"
	"github.com/moby/buildkit/source/git"
	"github.com/moby/buildkit/source/http"
	"github.com/moby/buildkit/source/local"
	"github.com/moby/buildkit/util/contentutil"
	"github.com/moby/buildkit/util/progress"
	"github.com/moby/buildkit/worker"
	digest "github.com/opencontainers/go-digest"
	ociidentity "github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const labelCreatedAt = "buildkit/createdat"

// TODO: this file should be removed. containerd defines ContainerdWorker, oci defines OCIWorker. There is no base worker.

// WorkerOpt is specific to a worker.
// See also CommonOpt.
type WorkerOpt struct {
	ID             string
	Labels         map[string]string
	Platforms      []specs.Platform
	SessionManager *session.Manager
	MetadataStore  *metadata.Store
	Executor       executor.Executor
	Snapshotter    snapshot.Snapshotter
	ContentStore   content.Store
	Applier        diff.Applier
	Differ         diff.Comparer
	ImageStore     images.Store // optional
}

// Worker is a local worker instance with dedicated snapshotter, cache, and so on.
// TODO: s/Worker/OpWorker/g ?
type Worker struct {
	WorkerOpt
	CacheManager  cache.Manager
	SourceManager *source.Manager
	Exporters     map[string]exporter.Exporter
	ImageSource   source.Source
}

// NewWorker instantiates a local worker
func NewWorker(opt WorkerOpt) (*Worker, error) {
	imageRefChecker := imagerefchecker.New(imagerefchecker.Opt{
		ImageStore:   opt.ImageStore,
		Snapshotter:  opt.Snapshotter,
		ContentStore: opt.ContentStore,
	})

	cm, err := cache.NewManager(cache.ManagerOpt{
		Snapshotter:     opt.Snapshotter,
		MetadataStore:   opt.MetadataStore,
		PruneRefChecker: imageRefChecker,
	})
	if err != nil {
		return nil, err
	}

	sm, err := source.NewManager()
	if err != nil {
		return nil, err
	}

	is, err := containerimage.NewSource(containerimage.SourceOpt{
		Snapshotter:    opt.Snapshotter,
		ContentStore:   opt.ContentStore,
		SessionManager: opt.SessionManager,
		Applier:        opt.Applier,
		ImageStore:     opt.ImageStore,
		CacheAccessor:  cm,
	})
	if err != nil {
		return nil, err
	}

	sm.Register(is)

	gs, err := git.NewSource(git.Opt{
		CacheAccessor: cm,
		MetadataStore: opt.MetadataStore,
	})
	if err != nil {
		return nil, err
	}

	sm.Register(gs)

	hs, err := http.NewSource(http.Opt{
		CacheAccessor: cm,
		MetadataStore: opt.MetadataStore,
	})
	if err != nil {
		return nil, err
	}

	sm.Register(hs)

	ss, err := local.NewSource(local.Opt{
		SessionManager: opt.SessionManager,
		CacheAccessor:  cm,
		MetadataStore:  opt.MetadataStore,
	})
	if err != nil {
		return nil, err
	}
	sm.Register(ss)

	exporters := map[string]exporter.Exporter{}

	iw, err := imageexporter.NewImageWriter(imageexporter.WriterOpt{
		Snapshotter:  opt.Snapshotter,
		ContentStore: opt.ContentStore,
		Differ:       opt.Differ,
	})
	if err != nil {
		return nil, err
	}

	imageExporter, err := imageexporter.New(imageexporter.Opt{
		Images:         opt.ImageStore,
		SessionManager: opt.SessionManager,
		ImageWriter:    iw,
	})
	if err != nil {
		return nil, err
	}
	exporters[client.ExporterImage] = imageExporter

	localExporter, err := localexporter.New(localexporter.Opt{
		SessionManager: opt.SessionManager,
	})
	if err != nil {
		return nil, err
	}
	exporters[client.ExporterLocal] = localExporter

	ociExporter, err := ociexporter.New(ociexporter.Opt{
		SessionManager: opt.SessionManager,
		ImageWriter:    iw,
		Variant:        ociexporter.VariantOCI,
	})
	if err != nil {
		return nil, err
	}
	exporters[client.ExporterOCI] = ociExporter

	dockerExporter, err := ociexporter.New(ociexporter.Opt{
		SessionManager: opt.SessionManager,
		ImageWriter:    iw,
		Variant:        ociexporter.VariantDocker,
	})
	if err != nil {
		return nil, err
	}
	exporters[client.ExporterDocker] = dockerExporter

	return &Worker{
		WorkerOpt:     opt,
		CacheManager:  cm,
		SourceManager: sm,
		Exporters:     exporters,
		ImageSource:   is,
	}, nil
}

func (w *Worker) ID() string {
	return w.WorkerOpt.ID
}

func (w *Worker) Labels() map[string]string {
	return w.WorkerOpt.Labels
}

func (w *Worker) Platforms() []specs.Platform {
	return w.WorkerOpt.Platforms
}

func (w *Worker) LoadRef(id string) (cache.ImmutableRef, error) {
	return w.CacheManager.Get(context.TODO(), id)
}

func (w *Worker) ResolveOp(v solver.Vertex, s frontend.FrontendLLBBridge) (solver.Op, error) {
	if baseOp, ok := v.Sys().(*pb.Op); ok {
		switch op := baseOp.Op.(type) {
		case *pb.Op_Source:
			return ops.NewSourceOp(v, op, baseOp.Platform, w.SourceManager, w)
		case *pb.Op_Exec:
			return ops.NewExecOp(v, op, w.CacheManager, w.SessionManager, w.MetadataStore, w.Executor, w)
		case *pb.Op_Build:
			return ops.NewBuildOp(v, op, s, w)
		}
	}
	return nil, errors.Errorf("could not resolve %v", v)
}

func (w *Worker) ResolveImageConfig(ctx context.Context, ref string, opt gw.ResolveImageConfigOpt) (digest.Digest, []byte, error) {
	// ImageSource is typically source/containerimage
	resolveImageConfig, ok := w.ImageSource.(resolveImageConfig)
	if !ok {
		return "", nil, errors.Errorf("worker %q does not implement ResolveImageConfig", w.ID())
	}
	return resolveImageConfig.ResolveImageConfig(ctx, ref, opt)
}

type resolveImageConfig interface {
	ResolveImageConfig(ctx context.Context, ref string, opt gw.ResolveImageConfigOpt) (digest.Digest, []byte, error)
}

func (w *Worker) Exec(ctx context.Context, meta executor.Meta, rootFS cache.ImmutableRef, stdin io.ReadCloser, stdout, stderr io.WriteCloser) error {
	active, err := w.CacheManager.New(ctx, rootFS)
	if err != nil {
		return err
	}
	defer active.Release(context.TODO())
	return w.Executor.Exec(ctx, meta, active, nil, stdin, stdout, stderr)
}

func (w *Worker) DiskUsage(ctx context.Context, opt client.DiskUsageInfo) ([]*client.UsageInfo, error) {
	return w.CacheManager.DiskUsage(ctx, opt)
}

func (w *Worker) Prune(ctx context.Context, ch chan client.UsageInfo, opt client.PruneInfo) error {
	return w.CacheManager.Prune(ctx, ch, opt)
}

func (w *Worker) Exporter(name string) (exporter.Exporter, error) {
	exp, ok := w.Exporters[name]
	if !ok {
		return nil, errors.Errorf("exporter %q could not be found", name)
	}
	return exp, nil
}

func (w *Worker) GetRemote(ctx context.Context, ref cache.ImmutableRef, createIfNeeded bool) (*solver.Remote, error) {
	diffPairs, err := blobs.GetDiffPairs(ctx, w.ContentStore, w.Snapshotter, w.Differ, ref, createIfNeeded)
	if err != nil {
		return nil, errors.Wrap(err, "failed calculaing diff pairs for exported snapshot")
	}
	if len(diffPairs) == 0 {
		return nil, nil
	}

	createdTimes := getCreatedTimes(ref)
	if len(createdTimes) != len(diffPairs) {
		return nil, errors.Errorf("invalid createdtimes/diffpairs")
	}

	descs := make([]ocispec.Descriptor, len(diffPairs))

	for i, dp := range diffPairs {
		info, err := w.ContentStore.Info(ctx, dp.Blobsum)
		if err != nil {
			return nil, err
		}

		tm, err := createdTimes[i].MarshalText()
		if err != nil {
			return nil, err
		}

		descs[i] = ocispec.Descriptor{
			Digest:    dp.Blobsum,
			Size:      info.Size,
			MediaType: images.MediaTypeDockerSchema2LayerGzip,
			Annotations: map[string]string{
				"containerd.io/uncompressed": dp.DiffID.String(),
				labelCreatedAt:               string(tm),
			},
		}
	}

	return &solver.Remote{
		Descriptors: descs,
		Provider:    w.ContentStore,
	}, nil
}

func getCreatedTimes(ref cache.ImmutableRef) (out []time.Time) {
	parent := ref.Parent()
	if parent != nil {
		defer parent.Release(context.TODO())
		out = getCreatedTimes(parent)
	}
	return append(out, cache.GetCreatedAt(ref.Metadata()))
}

func (w *Worker) FromRemote(ctx context.Context, remote *solver.Remote) (cache.ImmutableRef, error) {
	eg, gctx := errgroup.WithContext(ctx)
	for _, desc := range remote.Descriptors {
		func(desc ocispec.Descriptor) {
			eg.Go(func() error {
				done := oneOffProgress(ctx, fmt.Sprintf("pulling %s", desc.Digest))
				return done(contentutil.Copy(gctx, w.ContentStore, remote.Provider, desc))
			})
		}(desc)
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	cs, release := snapshot.NewContainerdSnapshotter(w.Snapshotter)
	defer release()

	unpackProgressDone := oneOffProgress(ctx, "unpacking")
	chainIDs, err := w.unpack(ctx, remote.Descriptors, cs)
	if err != nil {
		return nil, unpackProgressDone(err)
	}
	unpackProgressDone(nil)

	for i, chainID := range chainIDs {
		tm := time.Now()
		if tmstr, ok := remote.Descriptors[i].Annotations[labelCreatedAt]; ok {
			if err := (&tm).UnmarshalText([]byte(tmstr)); err != nil {
				return nil, err
			}
		}
		ref, err := w.CacheManager.Get(ctx, chainID, cache.WithDescription(fmt.Sprintf("imported %s", remote.Descriptors[i].Digest)), cache.WithCreationTime(tm))
		if err != nil {
			return nil, err
		}
		if i == len(remote.Descriptors)-1 {
			return ref, nil
		}
		ref.Release(context.TODO())
	}
	return nil, errors.Errorf("unreachable")
}

func (w *Worker) unpack(ctx context.Context, descs []ocispec.Descriptor, s cdsnapshot.Snapshotter) ([]string, error) {
	layers, err := getLayers(ctx, descs)
	if err != nil {
		return nil, err
	}

	var chain []digest.Digest
	for _, layer := range layers {
		labels := map[string]string{
			"containerd.io/uncompressed": layer.Diff.Digest.String(),
		}
		if _, err := rootfs.ApplyLayer(ctx, layer, chain, s, w.Applier, cdsnapshot.WithLabels(labels)); err != nil {
			return nil, err
		}
		chain = append(chain, layer.Diff.Digest)

		chainID := ociidentity.ChainID(chain)
		if err := w.Snapshotter.SetBlob(ctx, string(chainID), layer.Diff.Digest, layer.Blob.Digest); err != nil {
			return nil, err
		}
	}

	ids := make([]string, len(chain))
	for i := range chain {
		ids[i] = string(ociidentity.ChainID(chain[:i+1]))
	}

	return ids, nil
}

// Labels returns default labels
// utility function. could be moved to the constructor logic?
func Labels(executor, snapshotter string) map[string]string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	labels := map[string]string{
		worker.LabelExecutor:    executor,
		worker.LabelSnapshotter: snapshotter,
		worker.LabelHostname:    hostname,
	}
	return labels
}

// ID reads the worker id from the `workerid` file.
// If not exist, it creates a random one,
func ID(root string) (string, error) {
	f := filepath.Join(root, "workerid")
	b, err := ioutil.ReadFile(f)
	if err != nil {
		if os.IsNotExist(err) {
			id := identity.NewID()
			err := ioutil.WriteFile(f, []byte(id), 0400)
			return id, err
		} else {
			return "", err
		}
	}
	return string(b), nil
}

func getLayers(ctx context.Context, descs []ocispec.Descriptor) ([]rootfs.Layer, error) {
	layers := make([]rootfs.Layer, len(descs))
	for i, desc := range descs {
		diffIDStr := desc.Annotations["containerd.io/uncompressed"]
		if diffIDStr == "" {
			return nil, errors.Errorf("%s missing uncompressed digest", desc.Digest)
		}
		diffID, err := digest.Parse(diffIDStr)
		if err != nil {
			return nil, err
		}
		layers[i].Diff = ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageLayer,
			Digest:    diffID,
		}
		layers[i].Blob = ocispec.Descriptor{
			MediaType: desc.MediaType,
			Digest:    desc.Digest,
			Size:      desc.Size,
		}
	}
	return layers, nil
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
