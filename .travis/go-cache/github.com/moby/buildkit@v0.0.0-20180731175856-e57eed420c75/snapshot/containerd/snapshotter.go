package containerd

import (
	"context"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	ctdsnapshot "github.com/containerd/containerd/snapshots"
	"github.com/moby/buildkit/cache/metadata"
	"github.com/moby/buildkit/snapshot"
	"github.com/moby/buildkit/snapshot/blobmapping"
)

func NewSnapshotter(snapshotter ctdsnapshot.Snapshotter, store content.Store, mdstore *metadata.Store, ns string, gc func(context.Context) error) snapshot.Snapshotter {
	return blobmapping.NewSnapshotter(blobmapping.Opt{
		Content:       store,
		Snapshotter:   snapshot.FromContainerdSnapshotter(&nsSnapshotter{ns, snapshotter, gc}),
		MetadataStore: mdstore,
	})
}

type nsSnapshotter struct {
	ns string
	ctdsnapshot.Snapshotter
	gc garbageCollectFn
}

func (s *nsSnapshotter) Stat(ctx context.Context, key string) (ctdsnapshot.Info, error) {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	info, err := s.Snapshotter.Stat(ctx, key)
	if err == nil {
		if _, ok := info.Labels["labels.containerd.io/gc.root"]; !ok {
			if err := addRootLabel()(&info); err != nil {
				return info, err
			}
			return s.Update(ctx, info, "labels.containerd.io/gc.root")
		}
	}
	return info, err
}

func (s *nsSnapshotter) Update(ctx context.Context, info ctdsnapshot.Info, fieldpaths ...string) (ctdsnapshot.Info, error) {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	return s.Snapshotter.Update(ctx, info, fieldpaths...)
}

func (s *nsSnapshotter) Usage(ctx context.Context, key string) (ctdsnapshot.Usage, error) {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	return s.Snapshotter.Usage(ctx, key)
}
func (s *nsSnapshotter) Mounts(ctx context.Context, key string) ([]mount.Mount, error) {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	return s.Snapshotter.Mounts(ctx, key)
}
func (s *nsSnapshotter) Prepare(ctx context.Context, key, parent string, opts ...ctdsnapshot.Opt) ([]mount.Mount, error) {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	return s.Snapshotter.Prepare(ctx, key, parent, addRootLabel(opts...))
}
func (s *nsSnapshotter) View(ctx context.Context, key, parent string, opts ...ctdsnapshot.Opt) ([]mount.Mount, error) {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	return s.Snapshotter.View(ctx, key, parent, addRootLabel(opts...))
}
func (s *nsSnapshotter) Commit(ctx context.Context, name, key string, opts ...ctdsnapshot.Opt) error {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	return s.Snapshotter.Commit(ctx, name, key, addRootLabel(opts...))
}
func (s *nsSnapshotter) Remove(ctx context.Context, key string) error {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	if _, err := s.Update(ctx, ctdsnapshot.Info{
		Name: key,
	}, "labels.containerd.io/gc.root"); err != nil {
		return err
	} // calling snapshotter.Remove here causes a race in containerd
	if s.gc == nil {
		return nil
	}
	return s.gc(ctx)
}
func (s *nsSnapshotter) Walk(ctx context.Context, fn func(context.Context, ctdsnapshot.Info) error) error {
	ctx = namespaces.WithNamespace(ctx, s.ns)
	return s.Snapshotter.Walk(ctx, fn)
}

func addRootLabel(opts ...ctdsnapshot.Opt) ctdsnapshot.Opt {
	return func(info *ctdsnapshot.Info) error {
		for _, opt := range opts {
			if err := opt(info); err != nil {
				return err
			}
		}
		if info.Labels == nil {
			info.Labels = map[string]string{}
		}
		info.Labels["containerd.io/gc.root"] = time.Now().UTC().Format(time.RFC3339Nano)
		return nil
	}
}
