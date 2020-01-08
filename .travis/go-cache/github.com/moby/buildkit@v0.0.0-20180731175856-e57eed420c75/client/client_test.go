package client

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/continuity/fs/fstest"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/moby/buildkit/util/testutil"
	"github.com/moby/buildkit/util/testutil/httpserver"
	"github.com/moby/buildkit/util/testutil/integration"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestClientIntegration(t *testing.T) {
	integration.Run(t, []integration.Test{
		testRelativeWorkDir,
		testCallDiskUsage,
		testBuildMultiMount,
		testBuildHTTPSource,
		testBuildPushAndValidate,
		testResolveAndHosts,
		testUser,
		testOCIExporter,
		testWhiteoutParentDir,
		testDuplicateWhiteouts,
		testSchema1Image,
		testMountWithNoSource,
		testInvalidExporter,
		testReadonlyRootFS,
		testBasicCacheImportExport,
		testCachedMounts,
		testProxyEnv,
		testLocalSymlinkEscape,
		testTmpfsMounts,
		testSharedCacheMounts,
		testLockedCacheMounts,
		testDuplicateCacheMount,
		testParallelLocalBuilds,
		testSecretMounts,
	})
}

func testSecretMounts(t *testing.T, sb integration.Sandbox) {
	t.Parallel()

	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	st := llb.Image("busybox:latest").
		Run(llb.Shlex(`sh -c 'mount | grep mysecret | grep "type tmpfs" && [ "$(cat /run/secrets/mysecret)" = 'foo-secret' ]'`), llb.AddSecret("/run/secrets/mysecret"))

	def, err := st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Session: []session.Attachable{secretsprovider.FromMap(map[string][]byte{
			"/run/secrets/mysecret": []byte("foo-secret"),
		})},
	}, nil)
	require.NoError(t, err)

	// test optional
	st = llb.Image("busybox:latest").
		Run(llb.Shlex(`echo secret2`), llb.AddSecret("/run/secrets/mysecret2", llb.SecretOptional))

	def, err = st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Session: []session.Attachable{secretsprovider.FromMap(map[string][]byte{})},
	}, nil)
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)

	st = llb.Image("busybox:latest").
		Run(llb.Shlex(`echo secret3`), llb.AddSecret("/run/secrets/mysecret3"))

	def, err = st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Session: []session.Attachable{secretsprovider.FromMap(map[string][]byte{})},
	}, nil)
	require.Error(t, err)

	// test id,perm,uid
	st = llb.Image("busybox:latest").
		Run(llb.Shlex(`sh -c '[ "$(stat -c "%%u %%g %%f" /run/secrets/mysecret4)" = "1 1 81ff" ]' `), llb.AddSecret("/run/secrets/mysecret4", llb.SecretID("mysecret"), llb.SecretFileOpt(1, 1, 0777)))

	def, err = st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Session: []session.Attachable{secretsprovider.FromMap(map[string][]byte{
			"mysecret": []byte("pw"),
		})},
	}, nil)
	require.NoError(t, err)
}

func testTmpfsMounts(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	requiresLinux(t)
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	st := llb.Image("busybox:latest").
		Run(llb.Shlex(`sh -c 'mount | grep /foobar | grep "type tmpfs" && touch /foobar/test'`), llb.AddMount("/foobar", llb.Scratch(), llb.Tmpfs()))

	def, err := st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)
}

func testLocalSymlinkEscape(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	requiresLinux(t)
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	test := []byte(`set -ex
[[ -L /mount/foo ]]
[[ -L /mount/sub/bar ]]
[[ -L /mount/bax ]]
[[ -f /mount/bay ]]
[[ -f /mount/sub/sub2/file ]]
[[ ! -f /mount/baz ]]
[[ ! -f /mount/etc/passwd ]]
[[ ! -f /mount/etc/group ]]
[[ $(readlink /mount/foo) == "/etc/passwd" ]]
[[ $(readlink /mount/sub/bar) == "../../../etc/group" ]]
`)

	dir, err := tmpdir(
		// point to absolute path that is not part of dir
		fstest.Symlink("/etc/passwd", "foo"),
		fstest.CreateDir("sub", 0700),
		// point outside of the dir
		fstest.Symlink("../../../etc/group", "sub/bar"),
		// regular valid symlink
		fstest.Symlink("bay", "bax"),
		// target for symlink (not requested)
		fstest.CreateFile("bay", []byte{}, 0600),
		// file with many subdirs
		fstest.CreateDir("sub/sub2", 0700),
		fstest.CreateFile("sub/sub2/file", []byte{}, 0600),
		// unused file that shouldn't be included
		fstest.CreateFile("baz", []byte{}, 0600),
		fstest.CreateFile("test.sh", test, 0700),
	)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	local := llb.Local("mylocal", llb.FollowPaths([]string{
		"test.sh", "foo", "sub/bar", "bax", "sub/sub2/file",
	}))

	st := llb.Image("busybox:latest").
		Run(llb.Shlex(`sh /mount/test.sh`), llb.AddMount("/mount", local, llb.Readonly))

	def, err := st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		LocalDirs: map[string]string{
			"mylocal": dir,
		},
	}, nil)
	require.NoError(t, err)
}

func testRelativeWorkDir(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	requiresLinux(t)
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	pwd := llb.Image("docker.io/library/busybox:latest").
		Dir("test1").
		Dir("test2").
		Run(llb.Shlex(`sh -c "pwd > /out/pwd"`)).
		AddMount("/out", llb.Scratch())

	def, err := pwd.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "pwd"))
	require.NoError(t, err)
	require.Equal(t, []byte("/test1/test2\n"), dt)
}

func testCallDiskUsage(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()
	_, err = c.DiskUsage(context.TODO())
	require.NoError(t, err)
}

func testBuildMultiMount(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	requiresLinux(t)
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	alpine := llb.Image("docker.io/library/alpine:latest")
	ls := alpine.Run(llb.Shlex("/bin/ls -l"))
	busybox := llb.Image("docker.io/library/busybox:latest")
	cp := ls.Run(llb.Shlex("/bin/cp -a /busybox/etc/passwd baz"))
	cp.AddMount("/busybox", busybox)

	def, err := cp.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)

	checkAllReleasable(t, c, sb, true)
}

func testBuildHTTPSource(t *testing.T, sb integration.Sandbox) {
	t.Parallel()

	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	modTime := time.Now().Add(-24 * time.Hour) // avoid falso positive with current time

	resp := httpserver.Response{
		Etag:         identity.NewID(),
		Content:      []byte("content1"),
		LastModified: &modTime,
	}

	server := httpserver.NewTestServer(map[string]httpserver.Response{
		"/foo": resp,
	})
	defer server.Close()

	// invalid URL first
	st := llb.HTTP(server.URL + "/bar")

	def, err := st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid response status 404")

	// first correct request
	st = llb.HTTP(server.URL + "/foo")

	def, err = st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)

	require.Equal(t, server.Stats("/foo").AllRequests, 1)
	require.Equal(t, server.Stats("/foo").CachedRequests, 0)

	tmpdir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: tmpdir,
	}, nil)
	require.NoError(t, err)

	require.Equal(t, server.Stats("/foo").AllRequests, 2)
	require.Equal(t, server.Stats("/foo").CachedRequests, 1)

	dt, err := ioutil.ReadFile(filepath.Join(tmpdir, "foo"))
	require.NoError(t, err)
	require.Equal(t, []byte("content1"), dt)

	// test extra options
	st = llb.HTTP(server.URL+"/foo", llb.Filename("bar"), llb.Chmod(0741), llb.Chown(1000, 1000))

	def, err = st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: tmpdir,
	}, nil)
	require.NoError(t, err)

	require.Equal(t, server.Stats("/foo").AllRequests, 3)
	require.Equal(t, server.Stats("/foo").CachedRequests, 1)

	dt, err = ioutil.ReadFile(filepath.Join(tmpdir, "bar"))
	require.NoError(t, err)
	require.Equal(t, []byte("content1"), dt)

	fi, err := os.Stat(filepath.Join(tmpdir, "bar"))
	require.NoError(t, err)
	require.Equal(t, fi.ModTime().Format(http.TimeFormat), modTime.Format(http.TimeFormat))
	require.Equal(t, int(fi.Mode()&0777), 0741)

	checkAllReleasable(t, c, sb, true)

	// TODO: check that second request was marked as cached
}

func testResolveAndHosts(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	st := llb.Scratch()

	run := func(cmd string) {
		st = busybox.Run(llb.Shlex(cmd), llb.Dir("/wd")).AddMount("/wd", st)
	}

	run(`sh -c "cp /etc/resolv.conf ."`)
	run(`sh -c "cp /etc/hosts ."`)

	def, err := st.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "resolv.conf"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "nameserver")

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "hosts"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "127.0.0.1	localhost")

}

func testUser(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	st := llb.Image("busybox:latest").Run(llb.Shlex(`sh -c "mkdir -m 0777 /wd"`))

	run := func(user, cmd string) {
		st = st.Run(llb.Shlex(cmd), llb.Dir("/wd"), llb.User(user))
	}

	run("daemon", `sh -c "id -nu > user"`)
	run("daemon:daemon", `sh -c "id -ng > group"`)
	run("daemon:nogroup", `sh -c "id -ng > nogroup"`)
	run("1:1", `sh -c "id -g > userone"`)

	st = st.Run(llb.Shlex("cp -a /wd/. /out/"))
	out := st.AddMount("/out", llb.Scratch())

	def, err := out.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "user"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "daemon")

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "group"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "daemon")

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "nogroup"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "nogroup")

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "userone"))
	require.NoError(t, err)
	require.Contains(t, string(dt), "1")

	checkAllReleasable(t, c, sb, true)
}

func testOCIExporter(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	st := llb.Scratch()

	run := func(cmd string) {
		st = busybox.Run(llb.Shlex(cmd), llb.Dir("/wd")).AddMount("/wd", st)
	}

	run(`sh -c "echo -n first > foo"`)
	run(`sh -c "echo -n second > bar"`)

	def, err := st.Marshal()
	require.NoError(t, err)

	for _, exp := range []string{ExporterOCI, ExporterDocker} {

		destDir, err := ioutil.TempDir("", "buildkit")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		out := filepath.Join(destDir, "out.tar")
		outW, err := os.Create(out)
		require.NoError(t, err)
		target := "example.com/buildkit/testoci:latest"

		_, err = c.Solve(context.TODO(), def, SolveOpt{
			Exporter: exp,
			ExporterAttrs: map[string]string{
				"name": target,
			},
			ExporterOutput: outW,
		}, nil)
		require.NoError(t, err)

		dt, err := ioutil.ReadFile(out)
		require.NoError(t, err)

		m, err := testutil.ReadTarToMap(dt, false)
		require.NoError(t, err)

		_, ok := m["oci-layout"]
		require.True(t, ok)

		var index ocispec.Index
		err = json.Unmarshal(m["index.json"].Data, &index)
		require.NoError(t, err)
		require.Equal(t, 2, index.SchemaVersion)
		require.Equal(t, 1, len(index.Manifests))

		var mfst ocispec.Manifest
		err = json.Unmarshal(m["blobs/sha256/"+index.Manifests[0].Digest.Hex()].Data, &mfst)
		require.NoError(t, err)
		require.Equal(t, 2, len(mfst.Layers))

		var ociimg ocispec.Image
		err = json.Unmarshal(m["blobs/sha256/"+mfst.Config.Digest.Hex()].Data, &ociimg)
		require.NoError(t, err)
		require.Equal(t, "layers", ociimg.RootFS.Type)
		require.Equal(t, 2, len(ociimg.RootFS.DiffIDs))

		_, ok = m["blobs/sha256/"+mfst.Layers[0].Digest.Hex()]
		require.True(t, ok)
		_, ok = m["blobs/sha256/"+mfst.Layers[1].Digest.Hex()]
		require.True(t, ok)

		if exp != ExporterDocker {
			continue
		}

		var dockerMfst []struct {
			Config   string
			RepoTags []string
			Layers   []string
		}
		err = json.Unmarshal(m["manifest.json"].Data, &dockerMfst)
		require.NoError(t, err)
		require.Equal(t, 1, len(dockerMfst))

		_, ok = m[dockerMfst[0].Config]
		require.True(t, ok)
		require.Equal(t, 2, len(dockerMfst[0].Layers))
		require.Equal(t, 1, len(dockerMfst[0].RepoTags))
		require.Equal(t, target, dockerMfst[0].RepoTags[0])

		for _, l := range dockerMfst[0].Layers {
			_, ok := m[l]
			require.True(t, ok)
		}
	}

	checkAllReleasable(t, c, sb, true)
}

func testBuildPushAndValidate(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	st := llb.Scratch()

	run := func(cmd string) {
		st = busybox.Run(llb.Shlex(cmd), llb.Dir("/wd")).AddMount("/wd", st)
	}

	run(`sh -e -c "mkdir -p foo/sub; echo -n first > foo/sub/bar; chmod 0741 foo;"`)
	run(`true`) // this doesn't create a layer
	run(`sh -c "echo -n second > foo/sub/baz"`)

	def, err := st.Marshal()
	require.NoError(t, err)

	registry, err := sb.NewRegistry()
	if errors.Cause(err) == integration.ErrorRequirements {
		t.Skip(err.Error())
	}
	require.NoError(t, err)

	target := registry + "/buildkit/testpush:latest"

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter: ExporterImage,
		ExporterAttrs: map[string]string{
			"name": target,
			"push": "true",
		},
	}, nil)
	require.NoError(t, err)

	// test existence of the image with next build
	firstBuild := llb.Image(target)

	def, err = firstBuild.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "foo/sub/bar"))
	require.NoError(t, err)
	require.Equal(t, dt, []byte("first"))

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "foo/sub/baz"))
	require.NoError(t, err)
	require.Equal(t, dt, []byte("second"))

	fi, err := os.Stat(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, 0741, int(fi.Mode()&0777))

	checkAllReleasable(t, c, sb, false)

	// examine contents of exported tars (requires containerd)
	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		return
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	// TODO: make public pull helper function so this can be checked for standalone as well

	client, err := containerd.New(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")

	// check image in containerd
	_, err = client.ImageService().Get(ctx, target)
	require.NoError(t, err)

	// deleting image should release all content
	err = client.ImageService().Delete(ctx, target, images.SynchronousDelete())
	require.NoError(t, err)

	checkAllReleasable(t, c, sb, true)

	img, err := client.Pull(ctx, target)
	require.NoError(t, err)

	desc, err := img.Config(ctx)
	require.NoError(t, err)

	dt, err = content.ReadBlob(ctx, img.ContentStore(), desc)
	require.NoError(t, err)

	var ociimg ocispec.Image
	err = json.Unmarshal(dt, &ociimg)
	require.NoError(t, err)

	require.NotEqual(t, "", ociimg.OS)
	require.NotEqual(t, "", ociimg.Architecture)
	require.NotEqual(t, "", ociimg.Config.WorkingDir)
	require.Equal(t, "layers", ociimg.RootFS.Type)
	require.Equal(t, 2, len(ociimg.RootFS.DiffIDs))
	require.NotNil(t, ociimg.Created)
	require.True(t, time.Since(*ociimg.Created) < 2*time.Minute)
	require.Condition(t, func() bool {
		for _, env := range ociimg.Config.Env {
			if strings.HasPrefix(env, "PATH=") {
				return true
			}
		}
		return false
	})

	require.Equal(t, 3, len(ociimg.History))
	require.Contains(t, ociimg.History[0].CreatedBy, "foo/sub/bar")
	require.Contains(t, ociimg.History[1].CreatedBy, "true")
	require.Contains(t, ociimg.History[2].CreatedBy, "foo/sub/baz")
	require.False(t, ociimg.History[0].EmptyLayer)
	require.True(t, ociimg.History[1].EmptyLayer)
	require.False(t, ociimg.History[2].EmptyLayer)

	dt, err = content.ReadBlob(ctx, img.ContentStore(), img.Target())
	require.NoError(t, err)

	var mfst = struct {
		MediaType string `json:"mediaType,omitempty"`
		ocispec.Manifest
	}{}

	err = json.Unmarshal(dt, &mfst)
	require.NoError(t, err)

	require.Equal(t, images.MediaTypeDockerSchema2Manifest, mfst.MediaType)
	require.Equal(t, 2, len(mfst.Layers))

	dt, err = content.ReadBlob(ctx, img.ContentStore(), ocispec.Descriptor{Digest: mfst.Layers[0].Digest})
	require.NoError(t, err)

	m, err := testutil.ReadTarToMap(dt, true)
	require.NoError(t, err)

	item, ok := m["foo/"]
	require.True(t, ok)
	require.Equal(t, int32(item.Header.Typeflag), tar.TypeDir)
	require.Equal(t, 0741, int(item.Header.Mode&0777))

	item, ok = m["foo/sub/"]
	require.True(t, ok)
	require.Equal(t, int32(item.Header.Typeflag), tar.TypeDir)

	item, ok = m["foo/sub/bar"]
	require.True(t, ok)
	require.Equal(t, int32(item.Header.Typeflag), tar.TypeReg)
	require.Equal(t, []byte("first"), item.Data)

	_, ok = m["foo/sub/baz"]
	require.False(t, ok)

	dt, err = content.ReadBlob(ctx, img.ContentStore(), ocispec.Descriptor{Digest: mfst.Layers[1].Digest})
	require.NoError(t, err)

	m, err = testutil.ReadTarToMap(dt, true)
	require.NoError(t, err)

	item, ok = m["foo/sub/baz"]
	require.True(t, ok)
	require.Equal(t, int32(item.Header.Typeflag), tar.TypeReg)
	require.Equal(t, []byte("second"), item.Data)

	item, ok = m["foo/"]
	require.True(t, ok)
	require.Equal(t, int32(item.Header.Typeflag), tar.TypeDir)
	require.Equal(t, 0741, int(item.Header.Mode&0777))

	item, ok = m["foo/sub/"]
	require.True(t, ok)
	require.Equal(t, int32(item.Header.Typeflag), tar.TypeDir)

	_, ok = m["foo/sub/bar"]
	require.False(t, ok)
}

func testBasicCacheImportExport(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()

	registry, err := sb.NewRegistry()
	if errors.Cause(err) == integration.ErrorRequirements {
		t.Skip(err.Error())
	}
	require.NoError(t, err)

	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	st := llb.Scratch()

	run := func(cmd string) {
		st = busybox.Run(llb.Shlex(cmd), llb.Dir("/wd")).AddMount("/wd", st)
	}

	run(`sh -c "echo -n foobar > const"`)
	run(`sh -c "cat /dev/urandom | head -c 100 | sha256sum > unique"`)

	def, err := st.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	target := registry + "/buildkit/testexport:latest"

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
		ExportCache:       target,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "const"))
	require.NoError(t, err)
	require.Equal(t, string(dt), "foobar")

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "unique"))
	require.NoError(t, err)

	err = c.Prune(context.TODO(), nil, PruneAll)
	require.NoError(t, err)

	checkAllRemoved(t, c, sb)

	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
		ImportCache:       []string{target},
	}, nil)
	require.NoError(t, err)

	dt2, err := ioutil.ReadFile(filepath.Join(destDir, "const"))
	require.NoError(t, err)
	require.Equal(t, string(dt2), "foobar")

	dt2, err = ioutil.ReadFile(filepath.Join(destDir, "unique"))
	require.NoError(t, err)
	require.Equal(t, string(dt), string(dt2))
}

func testCachedMounts(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	// setup base for one of the cache sources
	st := busybox.Run(llb.Shlex(`sh -c "echo -n base > baz"`), llb.Dir("/wd"))
	base := st.AddMount("/wd", llb.Scratch())

	st = busybox.Run(llb.Shlex(`sh -c "echo -n first > foo"`), llb.Dir("/wd"))
	st.AddMount("/wd", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountShared))
	st = st.Run(llb.Shlex(`sh -c "cat foo && echo -n second > /wd2/bar"`), llb.Dir("/wd"))
	st.AddMount("/wd", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountShared))
	st.AddMount("/wd2", base, llb.AsPersistentCacheDir("mycache2", llb.CacheMountShared))

	def, err := st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)

	// repeat to make sure cache works
	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)

	// second build using cache directories
	st = busybox.Run(llb.Shlex(`sh -c "cp /src0/foo . && cp /src1/bar . && cp /src1/baz ."`), llb.Dir("/wd"))
	out := st.AddMount("/wd", llb.Scratch())
	st.AddMount("/src0", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountShared))
	st.AddMount("/src1", base, llb.AsPersistentCacheDir("mycache2", llb.CacheMountShared))

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	def, err = out.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "foo"))
	require.NoError(t, err)
	require.Equal(t, string(dt), "first")

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "bar"))
	require.NoError(t, err)
	require.Equal(t, string(dt), "second")

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "baz"))
	require.NoError(t, err)
	require.Equal(t, string(dt), "base")

	checkAllReleasable(t, c, sb, true)
}

func testSharedCacheMounts(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	st := busybox.Run(llb.Shlex(`sh -e -c "touch one; while [[ ! -f two ]]; do ls -l; usleep 500000; done"`), llb.Dir("/wd"))
	st.AddMount("/wd", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountShared))

	st2 := busybox.Run(llb.Shlex(`sh -e -c "touch two; while [[ ! -f one ]]; do ls -l; usleep 500000; done"`), llb.Dir("/wd"))
	st2.AddMount("/wd", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountShared))

	out := busybox.Run(llb.Shlex("true"))
	out.AddMount("/m1", st.Root())
	out.AddMount("/m2", st2.Root())

	def, err := out.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)
}

func testLockedCacheMounts(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	st := busybox.Run(llb.Shlex(`sh -e -c "touch one; if [[ -f two ]]; then exit 0; fi; for i in $(seq 10); do if [[ -f two ]]; then exit 1; fi; usleep 200000; done"`), llb.Dir("/wd"))
	st.AddMount("/wd", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountLocked))

	st2 := busybox.Run(llb.Shlex(`sh -e -c "touch two; if [[ -f one ]]; then exit 0; fi; for i in $(seq 10); do if [[ -f one ]]; then exit 1; fi; usleep 200000; done"`), llb.Dir("/wd"))
	st2.AddMount("/wd", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountLocked))

	out := busybox.Run(llb.Shlex("true"))
	out.AddMount("/m1", st.Root())
	out.AddMount("/m2", st2.Root())

	def, err := out.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)
}

func testDuplicateCacheMount(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")

	out := busybox.Run(llb.Shlex(`sh -e -c "[[ ! -f /m2/foo ]]; touch /m1/foo; [[ -f /m2/foo ]];"`))
	out.AddMount("/m1", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountLocked))
	out.AddMount("/m2", llb.Scratch(), llb.AsPersistentCacheDir("mycache1", llb.CacheMountLocked))

	def, err := out.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)
}

// containerd/containerd#2119
func testDuplicateWhiteouts(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	st := llb.Scratch()

	run := func(cmd string) {
		st = busybox.Run(llb.Shlex(cmd), llb.Dir("/wd")).AddMount("/wd", st)
	}

	run(`sh -e -c "mkdir -p d0 d1; echo -n first > d1/bar;"`)
	run(`sh -c "rm -rf d0 d1"`)

	def, err := st.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	out := filepath.Join(destDir, "out.tar")
	outW, err := os.Create(out)
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:       ExporterOCI,
		ExporterOutput: outW,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(out)
	require.NoError(t, err)

	m, err := testutil.ReadTarToMap(dt, false)
	require.NoError(t, err)

	var index ocispec.Index
	err = json.Unmarshal(m["index.json"].Data, &index)
	require.NoError(t, err)

	var mfst ocispec.Manifest
	err = json.Unmarshal(m["blobs/sha256/"+index.Manifests[0].Digest.Hex()].Data, &mfst)
	require.NoError(t, err)

	lastLayer := mfst.Layers[len(mfst.Layers)-1]

	layer, ok := m["blobs/sha256/"+lastLayer.Digest.Hex()]
	require.True(t, ok)

	m, err = testutil.ReadTarToMap(layer.Data, true)
	require.NoError(t, err)

	_, ok = m[".wh.d0"]
	require.True(t, ok)

	_, ok = m[".wh.d1"]
	require.True(t, ok)

	// check for a bug that added whiteout for subfile
	_, ok = m["d1/.wh.bar"]
	require.True(t, !ok)
}

// #276
func testWhiteoutParentDir(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("busybox:latest")
	st := llb.Scratch()

	run := func(cmd string) {
		st = busybox.Run(llb.Shlex(cmd), llb.Dir("/wd")).AddMount("/wd", st)
	}

	run(`sh -c "mkdir -p foo; echo -n first > foo/bar;"`)
	run(`rm foo/bar`)

	def, err := st.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	out := filepath.Join(destDir, "out.tar")
	outW, err := os.Create(out)
	require.NoError(t, err)
	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:       ExporterOCI,
		ExporterOutput: outW,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(out)
	require.NoError(t, err)

	m, err := testutil.ReadTarToMap(dt, false)
	require.NoError(t, err)

	var index ocispec.Index
	err = json.Unmarshal(m["index.json"].Data, &index)
	require.NoError(t, err)

	var mfst ocispec.Manifest
	err = json.Unmarshal(m["blobs/sha256/"+index.Manifests[0].Digest.Hex()].Data, &mfst)
	require.NoError(t, err)

	lastLayer := mfst.Layers[len(mfst.Layers)-1]

	layer, ok := m["blobs/sha256/"+lastLayer.Digest.Hex()]
	require.True(t, ok)

	m, err = testutil.ReadTarToMap(layer.Data, true)
	require.NoError(t, err)

	_, ok = m["foo/.wh.bar"]
	require.True(t, ok)

	_, ok = m["foo/"]
	require.True(t, ok)
}

// #296
func testSchema1Image(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	st := llb.Image("gcr.io/google_containers/pause:3.0@sha256:0d093c962a6c2dd8bb8727b661e2b5f13e9df884af9945b4cc7088d9350cd3ee")

	def, err := st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)

	checkAllReleasable(t, c, sb, true)
}

// #319
func testMountWithNoSource(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("docker.io/library/busybox:latest")
	st := llb.Scratch()

	var nilState llb.State

	// This should never actually be run, but we want to succeed
	// if it was, because we expect an error below, or a daemon
	// panic if the issue has regressed.
	run := busybox.Run(
		llb.Args([]string{"/bin/true"}),
		llb.AddMount("/nil", nilState, llb.SourcePath("/"), llb.Readonly))

	st = run.AddMount("/mnt", st)

	def, err := st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.NoError(t, err)

	checkAllReleasable(t, c, sb, true)
}

// #324
func testReadonlyRootFS(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	busybox := llb.Image("docker.io/library/busybox:latest")
	st := llb.Scratch()

	// The path /foo should be unwriteable.
	run := busybox.Run(
		llb.ReadonlyRootFS(),
		llb.Args([]string{"/bin/touch", "/foo"}))
	st = run.AddMount("/mnt", st)

	def, err := st.Marshal()
	require.NoError(t, err)

	_, err = c.Solve(context.TODO(), def, SolveOpt{}, nil)
	require.Error(t, err)
	// Would prefer to detect more specifically "Read-only file
	// system" but that isn't exposed here (it is on the stdio
	// which we don't see).
	require.Contains(t, err.Error(), "executor failed running [/bin/touch /foo]:")

	checkAllReleasable(t, c, sb, true)
}

func testProxyEnv(t *testing.T, sb integration.Sandbox) {
	t.Parallel()

	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	base := llb.Image("docker.io/library/busybox:latest").Dir("/out")
	cmd := `sh -c "echo -n $HTTP_PROXY-$HTTPS_PROXY-$NO_PROXY-$no_proxy > env"`

	st := base.Run(llb.Shlex(cmd), llb.WithProxy(llb.ProxyEnv{
		HttpProxy:  "httpvalue",
		HttpsProxy: "httpsvalue",
		NoProxy:    "noproxyvalue",
	}))
	out := st.AddMount("/out", llb.Scratch())

	def, err := out.Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(destDir, "env"))
	require.NoError(t, err)
	require.Equal(t, string(dt), "httpvalue-httpsvalue-noproxyvalue-noproxyvalue")

	// repeat to make sure proxy doesn't change cache
	st = base.Run(llb.Shlex(cmd), llb.WithProxy(llb.ProxyEnv{
		HttpsProxy: "httpsvalue2",
		NoProxy:    "noproxyvalue2",
	}))
	out = st.AddMount("/out", llb.Scratch())

	def, err = out.Marshal()
	require.NoError(t, err)

	destDir, err = ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:          ExporterLocal,
		ExporterOutputDir: destDir,
	}, nil)
	require.NoError(t, err)

	dt, err = ioutil.ReadFile(filepath.Join(destDir, "env"))
	require.NoError(t, err)
	require.Equal(t, string(dt), "httpvalue-httpsvalue-noproxyvalue-noproxyvalue")
}

func requiresLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("unsupported GOOS: %s", runtime.GOOS)
	}
}

func checkAllRemoved(t *testing.T, c *Client, sb integration.Sandbox) {
	retries := 0
	for {
		require.True(t, 20 > retries)
		retries++
		du, err := c.DiskUsage(context.TODO())
		require.NoError(t, err)
		if len(du) > 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
}

func checkAllReleasable(t *testing.T, c *Client, sb integration.Sandbox, checkContent bool) {
	retries := 0
loop0:
	for {
		require.True(t, 20 > retries)
		retries++
		du, err := c.DiskUsage(context.TODO())
		require.NoError(t, err)
		for _, d := range du {
			if d.InUse {
				time.Sleep(500 * time.Millisecond)
				continue loop0
			}
		}
		break
	}

	err := c.Prune(context.TODO(), nil, PruneAll)
	require.NoError(t, err)

	du, err := c.DiskUsage(context.TODO())
	require.NoError(t, err)
	require.Equal(t, 0, len(du))

	// examine contents of exported tars (requires containerd)
	var cdAddress string
	if cd, ok := sb.(interface {
		ContainerdAddress() string
	}); !ok {
		return
	} else {
		cdAddress = cd.ContainerdAddress()
	}

	// TODO: make public pull helper function so this can be checked for standalone as well

	client, err := containerd.New(cdAddress)
	require.NoError(t, err)
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "buildkit")
	snapshotService := client.SnapshotService("overlayfs")

	retries = 0
	for {
		count := 0
		err = snapshotService.Walk(ctx, func(context.Context, snapshots.Info) error {
			count++
			return nil
		})
		require.NoError(t, err)
		if count == 0 {
			break
		}
		require.True(t, 20 > retries)
		retries++
		time.Sleep(500 * time.Millisecond)
	}

	if !checkContent {
		return
	}

	retries = 0
	for {
		count := 0
		err = client.ContentStore().Walk(ctx, func(content.Info) error {
			count++
			return nil
		})
		require.NoError(t, err)
		if count == 0 {
			break
		}
		require.True(t, 20 > retries)
		retries++
		time.Sleep(500 * time.Millisecond)
	}
}

func testInvalidExporter(t *testing.T, sb integration.Sandbox) {
	requiresLinux(t)
	t.Parallel()
	c, err := New(context.TODO(), sb.Address())
	require.NoError(t, err)
	defer c.Close()

	def, err := llb.Image("busybox:latest").Marshal()
	require.NoError(t, err)

	destDir, err := ioutil.TempDir("", "buildkit")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	target := "example.com/buildkit/testoci:latest"
	attrs := map[string]string{
		"name": target,
	}
	for _, exp := range []string{ExporterOCI, ExporterDocker} {
		_, err = c.Solve(context.TODO(), def, SolveOpt{
			Exporter:      exp,
			ExporterAttrs: attrs,
		}, nil)
		// output file writer is required
		require.Error(t, err)
		_, err = c.Solve(context.TODO(), def, SolveOpt{
			Exporter:          exp,
			ExporterAttrs:     attrs,
			ExporterOutputDir: destDir,
		}, nil)
		// output directory is not supported
		require.Error(t, err)
	}

	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:      ExporterLocal,
		ExporterAttrs: attrs,
	}, nil)
	// output directory is required
	require.Error(t, err)

	f, err := os.Create(filepath.Join(destDir, "a"))
	require.NoError(t, err)
	defer f.Close()
	_, err = c.Solve(context.TODO(), def, SolveOpt{
		Exporter:       ExporterLocal,
		ExporterAttrs:  attrs,
		ExporterOutput: f,
	}, nil)
	// output file writer is not supported
	require.Error(t, err)

	checkAllReleasable(t, c, sb, true)
}

// moby/buildkit#492
func testParallelLocalBuilds(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	c, err := New(ctx, sb.Address())
	require.NoError(t, err)
	defer c.Close()

	eg, ctx := errgroup.WithContext(ctx)

	for i := 0; i < 3; i++ {
		func(i int) {
			eg.Go(func() error {
				fn := fmt.Sprintf("test%d", i)
				srcDir, err := tmpdir(
					fstest.CreateFile(fn, []byte("contents"), 0600),
				)
				require.NoError(t, err)
				defer os.RemoveAll(srcDir)

				def, err := llb.Local("source").Marshal()
				require.NoError(t, err)

				destDir, err := ioutil.TempDir("", "buildkit")
				require.NoError(t, err)
				defer os.RemoveAll(destDir)

				_, err = c.Solve(ctx, def, SolveOpt{
					Exporter:          ExporterLocal,
					ExporterOutputDir: destDir,
					LocalDirs: map[string]string{
						"source": srcDir,
					},
				}, nil)
				require.NoError(t, err)

				act, err := ioutil.ReadFile(filepath.Join(destDir, fn))
				require.NoError(t, err)

				require.Equal(t, "contents", string(act))
				return nil
			})
		}(i)
	}

	err = eg.Wait()
	require.NoError(t, err)
}

func tmpdir(appliers ...fstest.Applier) (string, error) {
	tmpdir, err := ioutil.TempDir("", "buildkit-client")
	if err != nil {
		return "", err
	}
	if err := fstest.Apply(appliers...).Apply(tmpdir); err != nil {
		return "", err
	}
	return tmpdir, nil
}
