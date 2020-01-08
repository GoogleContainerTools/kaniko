package cache

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/containerd/snapshots/native"
	"github.com/moby/buildkit/cache/metadata"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/snapshot"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	t.Parallel()
	ctx := namespaces.WithNamespace(context.Background(), "buildkit-test")

	tmpdir, err := ioutil.TempDir("", "cachemanager")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	snapshotter, err := native.NewSnapshotter(filepath.Join(tmpdir, "snapshots"))
	require.NoError(t, err)
	cm := getCacheManager(t, tmpdir, snapshotter)

	_, err = cm.Get(ctx, "foobar")
	require.Error(t, err)

	checkDiskUsage(ctx, t, cm, 0, 0)

	active, err := cm.New(ctx, nil, CachePolicyRetain)
	require.NoError(t, err)

	m, err := active.Mount(ctx, false)
	require.NoError(t, err)

	lm := snapshot.LocalMounter(m)
	target, err := lm.Mount()
	require.NoError(t, err)

	fi, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, fi.IsDir(), true)

	err = lm.Unmount()
	require.NoError(t, err)

	_, err = cm.GetMutable(ctx, active.ID())
	require.Error(t, err)
	require.Equal(t, ErrLocked, errors.Cause(err))

	checkDiskUsage(ctx, t, cm, 1, 0)

	snap, err := active.Commit(ctx)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 1, 0)

	_, err = cm.GetMutable(ctx, active.ID())
	require.Error(t, err)
	require.Equal(t, ErrLocked, errors.Cause(err))

	err = snap.Release(ctx)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 0, 1)

	active, err = cm.GetMutable(ctx, active.ID())
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 1, 0)

	snap, err = active.Commit(ctx)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 1, 0)

	err = snap.Finalize(ctx, true)
	require.NoError(t, err)

	err = snap.Release(ctx)
	require.NoError(t, err)

	_, err = cm.GetMutable(ctx, active.ID())
	require.Error(t, err)
	require.Equal(t, errNotFound, errors.Cause(err))

	_, err = cm.GetMutable(ctx, snap.ID())
	require.Error(t, err)
	require.Equal(t, errInvalid, errors.Cause(err))

	snap, err = cm.Get(ctx, snap.ID())
	require.NoError(t, err)

	snap2, err := cm.Get(ctx, snap.ID())
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 1, 0)

	err = snap.Release(ctx)
	require.NoError(t, err)

	active2, err := cm.New(ctx, snap2, CachePolicyRetain)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 2, 0)

	snap3, err := active2.Commit(ctx)
	require.NoError(t, err)

	err = snap2.Release(ctx)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 2, 0)

	err = snap3.Release(ctx)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 0, 2)

	buf := pruneResultBuffer()
	err = cm.Prune(ctx, buf.C, client.PruneInfo{})
	buf.close()
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 0, 0)

	require.Equal(t, len(buf.all), 2)

	err = cm.Close()
	require.NoError(t, err)

	dirs, err := ioutil.ReadDir(filepath.Join(tmpdir, "snapshots/snapshots"))
	require.NoError(t, err)
	require.Equal(t, 0, len(dirs))
}

func TestPrune(t *testing.T) {
	t.Parallel()
	ctx := namespaces.WithNamespace(context.Background(), "buildkit-test")

	tmpdir, err := ioutil.TempDir("", "cachemanager")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	snapshotter, err := native.NewSnapshotter(filepath.Join(tmpdir, "snapshots"))
	require.NoError(t, err)
	cm := getCacheManager(t, tmpdir, snapshotter)

	active, err := cm.New(ctx, nil)
	require.NoError(t, err)

	snap, err := active.Commit(ctx)
	require.NoError(t, err)

	active, err = cm.New(ctx, snap, CachePolicyRetain)
	require.NoError(t, err)

	snap2, err := active.Commit(ctx)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 2, 0)

	// prune with keeping refs does nothing
	buf := pruneResultBuffer()
	err = cm.Prune(ctx, buf.C, client.PruneInfo{})
	buf.close()
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 2, 0)
	require.Equal(t, len(buf.all), 0)

	dirs, err := ioutil.ReadDir(filepath.Join(tmpdir, "snapshots/snapshots"))
	require.NoError(t, err)
	require.Equal(t, 2, len(dirs))

	err = snap2.Release(ctx)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 1, 1)

	// prune with keeping single refs deletes one
	buf = pruneResultBuffer()
	err = cm.Prune(ctx, buf.C, client.PruneInfo{})
	buf.close()
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 1, 0)
	require.Equal(t, len(buf.all), 1)

	dirs, err = ioutil.ReadDir(filepath.Join(tmpdir, "snapshots/snapshots"))
	require.NoError(t, err)
	require.Equal(t, 1, len(dirs))

	err = snap.Release(ctx)
	require.NoError(t, err)

	active, err = cm.New(ctx, snap, CachePolicyRetain)
	require.NoError(t, err)

	snap2, err = active.Commit(ctx)
	require.NoError(t, err)

	err = snap.Release(ctx)
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 2, 0)

	// prune with parent released does nothing
	buf = pruneResultBuffer()
	err = cm.Prune(ctx, buf.C, client.PruneInfo{})
	buf.close()
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 2, 0)
	require.Equal(t, len(buf.all), 0)

	// releasing last reference
	err = snap2.Release(ctx)
	require.NoError(t, err)
	checkDiskUsage(ctx, t, cm, 0, 2)

	buf = pruneResultBuffer()
	err = cm.Prune(ctx, buf.C, client.PruneInfo{})
	buf.close()
	require.NoError(t, err)

	checkDiskUsage(ctx, t, cm, 0, 0)
	require.Equal(t, len(buf.all), 2)

	dirs, err = ioutil.ReadDir(filepath.Join(tmpdir, "snapshots/snapshots"))
	require.NoError(t, err)
	require.Equal(t, 0, len(dirs))
}

func TestLazyCommit(t *testing.T) {
	t.Parallel()
	ctx := namespaces.WithNamespace(context.Background(), "buildkit-test")

	tmpdir, err := ioutil.TempDir("", "cachemanager")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	snapshotter, err := native.NewSnapshotter(filepath.Join(tmpdir, "snapshots"))
	require.NoError(t, err)
	cm := getCacheManager(t, tmpdir, snapshotter)

	active, err := cm.New(ctx, nil, CachePolicyRetain)
	require.NoError(t, err)

	// after commit mutable is locked
	snap, err := active.Commit(ctx)
	require.NoError(t, err)

	_, err = cm.GetMutable(ctx, active.ID())
	require.Error(t, err)
	require.Equal(t, ErrLocked, errors.Cause(err))

	// immutable refs still work
	snap2, err := cm.Get(ctx, snap.ID())
	require.NoError(t, err)
	require.Equal(t, snap.ID(), snap2.ID())

	err = snap.Release(ctx)
	require.NoError(t, err)

	err = snap2.Release(ctx)
	require.NoError(t, err)

	// immutable work after final release as well
	snap, err = cm.Get(ctx, snap.ID())
	require.NoError(t, err)
	require.Equal(t, snap.ID(), snap2.ID())

	// active can't be get while immutable is held
	_, err = cm.GetMutable(ctx, active.ID())
	require.Error(t, err)
	require.Equal(t, ErrLocked, errors.Cause(err))

	err = snap.Release(ctx)
	require.NoError(t, err)

	// after release mutable becomes available again
	active2, err := cm.GetMutable(ctx, active.ID())
	require.NoError(t, err)
	require.Equal(t, active2.ID(), active.ID())

	// because ref was took mutable old immutable are cleared
	_, err = cm.Get(ctx, snap.ID())
	require.Error(t, err)
	require.Equal(t, errNotFound, errors.Cause(err))

	snap, err = active2.Commit(ctx)
	require.NoError(t, err)

	// this time finalize commit
	err = snap.Finalize(ctx, true)
	require.NoError(t, err)

	err = snap.Release(ctx)
	require.NoError(t, err)

	// mutable is gone after finalize
	_, err = cm.GetMutable(ctx, active2.ID())
	require.Error(t, err)
	require.Equal(t, errNotFound, errors.Cause(err))

	// immutable still works
	snap2, err = cm.Get(ctx, snap.ID())
	require.NoError(t, err)
	require.Equal(t, snap.ID(), snap2.ID())

	err = snap2.Release(ctx)
	require.NoError(t, err)

	// test restarting after commit
	active, err = cm.New(ctx, nil, CachePolicyRetain)
	require.NoError(t, err)

	// after commit mutable is locked
	snap, err = active.Commit(ctx)
	require.NoError(t, err)

	err = cm.Close()
	require.NoError(t, err)

	// we can't close snapshotter and open it twice (especially, its internal boltdb store)
	cm = getCacheManager(t, tmpdir, snapshotter)

	snap2, err = cm.Get(ctx, snap.ID())
	require.NoError(t, err)

	err = snap2.Release(ctx)
	require.NoError(t, err)

	active, err = cm.GetMutable(ctx, active.ID())
	require.NoError(t, err)

	_, err = cm.Get(ctx, snap.ID())
	require.Error(t, err)
	require.Equal(t, errNotFound, errors.Cause(err))

	snap, err = active.Commit(ctx)
	require.NoError(t, err)

	err = cm.Close()
	require.NoError(t, err)

	cm = getCacheManager(t, tmpdir, snapshotter)

	snap2, err = cm.Get(ctx, snap.ID())
	require.NoError(t, err)

	err = snap2.Finalize(ctx, true)
	require.NoError(t, err)

	err = snap2.Release(ctx)
	require.NoError(t, err)

	active, err = cm.GetMutable(ctx, active.ID())
	require.Error(t, err)
	require.Equal(t, errNotFound, errors.Cause(err))
}

func getCacheManager(t *testing.T, tmpdir string, snapshotter snapshots.Snapshotter) Manager {
	md, err := metadata.NewStore(filepath.Join(tmpdir, "metadata.db"))
	require.NoError(t, err)

	cm, err := NewManager(ManagerOpt{
		Snapshotter:   snapshot.FromContainerdSnapshotter(snapshotter),
		MetadataStore: md,
	})
	require.NoError(t, err, fmt.Sprintf("error: %+v", err))
	return cm
}

func checkDiskUsage(ctx context.Context, t *testing.T, cm Manager, inuse, unused int) {
	du, err := cm.DiskUsage(ctx, client.DiskUsageInfo{})
	require.NoError(t, err)
	var inuseActual, unusedActual int
	for _, r := range du {
		if r.InUse {
			inuseActual++
		} else {
			unusedActual++
		}
	}
	require.Equal(t, inuse, inuseActual)
	require.Equal(t, unused, unusedActual)
}

func pruneResultBuffer() *buf {
	b := &buf{C: make(chan client.UsageInfo), closed: make(chan struct{})}
	go func() {
		for c := range b.C {
			b.all = append(b.all, c)
		}
		close(b.closed)
	}()
	return b
}

type buf struct {
	C      chan client.UsageInfo
	closed chan struct{}
	all    []client.UsageInfo
}

func (b *buf) close() {
	close(b.C)
	<-b.closed
}
