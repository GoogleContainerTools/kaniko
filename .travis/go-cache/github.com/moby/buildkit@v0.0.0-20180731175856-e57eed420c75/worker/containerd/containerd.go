package containerd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd"
	introspection "github.com/containerd/containerd/api/services/introspection/v1"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/snapshots"
	"github.com/moby/buildkit/cache/metadata"
	"github.com/moby/buildkit/executor/containerdexecutor"
	"github.com/moby/buildkit/identity"
	containerdsnapshot "github.com/moby/buildkit/snapshot/containerd"
	"github.com/moby/buildkit/util/throttle"
	"github.com/moby/buildkit/util/winlayers"
	"github.com/moby/buildkit/worker/base"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewWorkerOpt creates a WorkerOpt.
// But it does not set the following fields:
//  - SessionManager
func NewWorkerOpt(root string, address, snapshotterName string, labels map[string]string, opts ...containerd.ClientOpt) (base.WorkerOpt, error) {
	// TODO: take lock to make sure there are no duplicates
	opts = append([]containerd.ClientOpt{containerd.WithDefaultNamespace("buildkit")}, opts...)
	client, err := containerd.New(address, opts...)
	if err != nil {
		return base.WorkerOpt{}, errors.Wrapf(err, "failed to connect client to %q . make sure containerd is running", address)
	}
	return newContainerd(root, client, snapshotterName, labels)
}

func newContainerd(root string, client *containerd.Client, snapshotterName string, labels map[string]string) (base.WorkerOpt, error) {
	if strings.Contains(snapshotterName, "/") {
		return base.WorkerOpt{}, errors.Errorf("bad snapshotter name: %q", snapshotterName)
	}
	name := "containerd-" + snapshotterName
	root = filepath.Join(root, name)
	if err := os.MkdirAll(root, 0700); err != nil {
		return base.WorkerOpt{}, errors.Wrapf(err, "failed to create %s", root)
	}

	md, err := metadata.NewStore(filepath.Join(root, "metadata.db"))
	if err != nil {
		return base.WorkerOpt{}, err
	}
	df := client.DiffService()
	// TODO: should use containerd daemon instance ID (containerd/containerd#1862)?
	id, err := base.ID(root)
	if err != nil {
		return base.WorkerOpt{}, err
	}
	xlabels := base.Labels("containerd", snapshotterName)
	for k, v := range labels {
		xlabels[k] = v
	}

	throttledGC := throttle.Throttle(time.Second, func() {
		// TODO: how to avoid this?
		ctx := context.TODO()
		snapshotter := client.SnapshotService(snapshotterName)
		ctx = namespaces.WithNamespace(ctx, "buildkit")
		key := identity.NewID()
		if _, err := snapshotter.Prepare(ctx, key, "", snapshots.WithLabels(map[string]string{
			"containerd.io/gc.root": time.Now().UTC().Format(time.RFC3339Nano),
		})); err != nil {
			logrus.Errorf("GC error: %+v", err)
		}
		if err := snapshotter.Remove(ctx, key); err != nil {
			logrus.Errorf("GC error: %+v", err)
		}
	})

	gc := func(ctx context.Context) error {
		throttledGC()
		return nil
	}

	cs := containerdsnapshot.NewContentStore(client.ContentStore(), "buildkit", gc)

	resp, err := client.IntrospectionService().Plugins(context.TODO(), &introspection.PluginsRequest{Filters: []string{"type==io.containerd.runtime.v1"}})
	if err != nil {
		return base.WorkerOpt{}, errors.Wrap(err, "failed to list runtime plugin")
	}
	if len(resp.Plugins) == 0 {
		return base.WorkerOpt{}, errors.Wrap(err, "failed to get runtime plugin")
	}

	var platforms []specs.Platform
	for _, plugin := range resp.Plugins {
		for _, p := range plugin.Platforms {
			platforms = append(platforms, specs.Platform{
				OS:           p.OS,
				Architecture: p.Architecture,
				Variant:      p.Variant,
			})
		}
	}

	opt := base.WorkerOpt{
		ID:            id,
		Labels:        xlabels,
		MetadataStore: md,
		Executor:      containerdexecutor.New(client, root),
		Snapshotter:   containerdsnapshot.NewSnapshotter(client.SnapshotService(snapshotterName), cs, md, "buildkit", gc),
		ContentStore:  cs,
		Applier:       winlayers.NewFileSystemApplierWithWindows(cs, df),
		Differ:        winlayers.NewWalkingDiffWithWindows(cs, df),
		ImageStore:    client.ImageService(),
		Platforms:     platforms,
	}
	return opt, nil
}
