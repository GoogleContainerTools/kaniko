package containerdexecutor

import (
	"context"
	"io"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/contrib/seccomp"
	containerdoci "github.com/containerd/containerd/oci"
	"github.com/moby/buildkit/cache"
	"github.com/moby/buildkit/executor"
	"github.com/moby/buildkit/executor/oci"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/snapshot"
	"github.com/moby/buildkit/util/system"
	"github.com/pkg/errors"
)

type containerdExecutor struct {
	client *containerd.Client
	root   string
}

func New(client *containerd.Client, root string) executor.Executor {
	return containerdExecutor{
		client: client,
		root:   root,
	}
}

func (w containerdExecutor) Exec(ctx context.Context, meta executor.Meta, root cache.Mountable, mounts []executor.Mount, stdin io.ReadCloser, stdout, stderr io.WriteCloser) (err error) {
	id := identity.NewID()

	resolvConf, err := oci.GetResolvConf(ctx, w.root)
	if err != nil {
		return err
	}

	hostsFile, err := oci.GetHostsFile(ctx, w.root)
	if err != nil {
		return err
	}

	mountable, err := root.Mount(ctx, false)
	if err != nil {
		return err
	}

	rootMounts, err := mountable.Mount()
	if err != nil {
		return err
	}
	defer mountable.Release()

	var sgids []uint32
	uid, gid, err := oci.ParseUIDGID(meta.User)
	if err != nil {
		lm := snapshot.LocalMounterWithMounts(rootMounts)
		rootfsPath, err := lm.Mount()
		if err != nil {
			return err
		}
		uid, gid, sgids, err = oci.GetUser(ctx, rootfsPath, meta.User)
		if err != nil {
			lm.Unmount()
			return err
		}
		lm.Unmount()
	}

	opts := []containerdoci.SpecOpts{oci.WithUIDGID(uid, gid, sgids)}
	if meta.ReadonlyRootFS {
		opts = append(opts, containerdoci.WithRootFSReadonly())
	}
	if system.SeccompSupported() {
		opts = append(opts, seccomp.WithDefaultProfile())
	}
	spec, cleanup, err := oci.GenerateSpec(ctx, meta, mounts, id, resolvConf, hostsFile, opts...)
	if err != nil {
		return err
	}
	defer cleanup()

	container, err := w.client.NewContainer(ctx, id,
		containerd.WithSpec(spec),
	)
	if err != nil {
		return err
	}

	defer func() {
		if err1 := container.Delete(context.TODO()); err == nil && err1 != nil {
			err = errors.Wrapf(err1, "failed to delete container %s", id)
		}
	}()

	if stdin == nil {
		stdin = &emptyReadCloser{}
	}

	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStreams(stdin, stdout, stderr)), containerd.WithRootFS(rootMounts))
	if err != nil {
		return err
	}
	defer func() {
		if _, err1 := task.Delete(context.TODO()); err == nil && err1 != nil {
			err = errors.Wrapf(err1, "failed to delete task %s", id)
		}
	}()

	// TODO: Configure bridge networking

	if err := task.Start(ctx); err != nil {
		return err
	}

	statusCh, err := task.Wait(context.Background())
	if err != nil {
		return err
	}

	var cancel func()
	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			ctxDone = nil
			var killCtx context.Context
			killCtx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			task.Kill(killCtx, syscall.SIGKILL)
		case status := <-statusCh:
			if cancel != nil {
				cancel()
			}
			if status.ExitCode() != 0 {
				return errors.Errorf("process returned non-zero exit code: %d", status.ExitCode())
			}
			return nil
		}
	}

}

type emptyReadCloser struct{}

func (*emptyReadCloser) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (*emptyReadCloser) Close() error {
	return nil
}
