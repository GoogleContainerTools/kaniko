package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/continuity/fs/fstest"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/testutil/integration"
	"github.com/stretchr/testify/require"
)

func testBuildWithLocalFiles(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	dir, err := tmpdir(
		fstest.CreateFile("foo", []byte("bar"), 0600),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	st := llb.Image("busybox").
		Run(llb.Shlex("sh -c 'echo -n bar > foo2'")).
		Run(llb.Shlex("cmp -s /mnt/foo foo2"))

	st.AddMount("/mnt", llb.Local("src"), llb.Readonly)

	rdr, err := marshal(st.Root())
	require.NoError(t, err)

	cmd := sb.Cmd(fmt.Sprintf("build --no-progress --local src=%s", dir))
	cmd.Stdin = rdr

	err = cmd.Run()
	require.NoError(t, err)
}

func testBuildLocalExporter(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	st := llb.Image("busybox").
		Run(llb.Shlex("sh -c 'echo -n bar > /out/foo'"))

	out := st.AddMount("/out", llb.Scratch())

	rdr, err := marshal(out)
	require.NoError(t, err)

	tmpdir, err := ioutil.TempDir("", "buildkit-buildctl")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	cmd := sb.Cmd(fmt.Sprintf("build --no-progress --exporter=local --exporter-opt output=%s", tmpdir))
	cmd.Stdin = rdr
	err = cmd.Run()

	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(tmpdir, "foo"))
	require.NoError(t, err)
	require.Equal(t, string(dt), "bar")
}

func testBuildContainerdExporter(t *testing.T, sb integration.Sandbox) {
	t.Parallel()

	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		t.Skip("only for containerd worker")
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	st := llb.Image("busybox").
		Run(llb.Shlex("sh -c 'echo -n bar > /foo'"))

	rdr, err := marshal(st.Root())
	require.NoError(t, err)

	cmd := sb.Cmd("build --no-progress --exporter=image --exporter-opt name=example.com/moby/imageexporter:test")
	cmd.Stdin = rdr
	err = cmd.Run()
	require.NoError(t, err)

	client, err := containerd.New(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	_, err = client.ImageService().Get(ctx, "example.com/moby/imageexporter:test")
	require.NoError(t, err)
}

func marshal(st llb.State) (io.Reader, error) {
	def, err := st.Marshal()
	if err != nil {
		return nil, err
	}
	dt, err := def.ToPB().Marshal()
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(dt), nil
}

func tmpdir(appliers ...fstest.Applier) (string, error) {
	tmpdir, err := ioutil.TempDir("", "buildkit-buildctl")
	if err != nil {
		return "", err
	}
	if err := fstest.Apply(appliers...).Apply(tmpdir); err != nil {
		return "", err
	}
	return tmpdir, nil
}
