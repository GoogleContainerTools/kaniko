package blobs

import (
	"context"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/diff"
	"github.com/containerd/containerd/mount"
	"github.com/moby/buildkit/cache"
	"github.com/moby/buildkit/snapshot"
	"github.com/moby/buildkit/util/flightcontrol"
	"github.com/moby/buildkit/util/winlayers"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var g flightcontrol.Group

const containerdUncompressed = "containerd.io/uncompressed"

type DiffPair struct {
	DiffID  digest.Digest
	Blobsum digest.Digest
}

var ErrNoBlobs = errors.Errorf("no blobs for snapshot")

func GetDiffPairs(ctx context.Context, contentStore content.Store, snapshotter snapshot.Snapshotter, differ diff.Comparer, ref cache.ImmutableRef, createBlobs bool) ([]DiffPair, error) {
	if ref == nil {
		return nil, nil
	}

	if err := ref.Finalize(ctx, true); err != nil {
		return nil, err
	}

	if isTypeWindows(ref) {
		ctx = winlayers.UseWindowsLayerMode(ctx)
	}

	return getDiffPairs(ctx, contentStore, snapshotter, differ, ref, createBlobs)
}

func getDiffPairs(ctx context.Context, contentStore content.Store, snapshotter snapshot.Snapshotter, differ diff.Comparer, ref cache.ImmutableRef, createBlobs bool) ([]DiffPair, error) {
	if ref == nil {
		return nil, nil
	}

	eg, ctx := errgroup.WithContext(ctx)
	var diffPairs []DiffPair
	var currentPair DiffPair
	parent := ref.Parent()
	if parent != nil {
		defer parent.Release(context.TODO())
		eg.Go(func() error {
			dp, err := getDiffPairs(ctx, contentStore, snapshotter, differ, parent, createBlobs)
			if err != nil {
				return err
			}
			diffPairs = dp
			return nil
		})
	}
	eg.Go(func() error {
		dp, err := g.Do(ctx, ref.ID(), func(ctx context.Context) (interface{}, error) {
			diffID, blob, err := snapshotter.GetBlob(ctx, ref.ID())
			if err != nil {
				return nil, err
			}
			if blob != "" {
				return DiffPair{DiffID: diffID, Blobsum: blob}, nil
			} else if !createBlobs {
				return nil, errors.WithStack(ErrNoBlobs)
			}
			// reference needs to be committed
			parent := ref.Parent()
			var lower []mount.Mount
			if parent != nil {
				defer parent.Release(context.TODO())
				m, err := parent.Mount(ctx, true)
				if err != nil {
					return nil, err
				}
				lower, err = m.Mount()
				if err != nil {
					return nil, err
				}
				defer m.Release()
			}
			m, err := ref.Mount(ctx, true)
			if err != nil {
				return nil, err
			}
			upper, err := m.Mount()
			if err != nil {
				return nil, err
			}
			defer m.Release()
			descr, err := differ.Compare(ctx, lower, upper,
				diff.WithMediaType(ocispec.MediaTypeImageLayerGzip),
				diff.WithReference(ref.ID()),
				diff.WithLabels(map[string]string{
					"containerd.io/gc.root": time.Now().UTC().Format(time.RFC3339Nano),
				}),
			)
			if err != nil {
				return nil, err
			}
			info, err := contentStore.Info(ctx, descr.Digest)
			if err != nil {
				return nil, err
			}
			diffIDStr, ok := info.Labels[containerdUncompressed]
			if !ok {
				return nil, errors.Errorf("invalid differ response with no diffID")
			}
			diffIDDigest, err := digest.Parse(diffIDStr)
			if err != nil {
				return nil, err
			}
			if err := snapshotter.SetBlob(ctx, ref.ID(), diffIDDigest, descr.Digest); err != nil {
				return nil, err
			}
			return DiffPair{DiffID: diffIDDigest, Blobsum: descr.Digest}, nil
		})
		if err != nil {
			return err
		}
		currentPair = dp.(DiffPair)
		return nil
	})
	err := eg.Wait()
	if err != nil {
		return nil, err
	}
	return append(diffPairs, currentPair), nil
}

func isTypeWindows(ref cache.ImmutableRef) bool {
	if cache.GetLayerType(ref) == "windows" {
		return true
	}
	if parent := ref.Parent(); parent != nil {
		defer parent.Release(context.TODO())
		return isTypeWindows(parent)
	}
	return false
}
