package llbtest

import (
	"testing"

	"github.com/containerd/containerd/platforms"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver"
	"github.com/moby/buildkit/solver/llbsolver"
	"github.com/moby/buildkit/solver/pb"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

func TestCustomPlatform(t *testing.T) {
	t.Parallel()

	s := llb.Image("foo", llb.LinuxArmhf).
		Run(llb.Shlex("baz")).
		Run(llb.Shlex("bar")).
		Run(llb.Shlex("bax"), llb.Windows).
		Run(llb.Shlex("bay"))

	def, err := s.Marshal()
	require.NoError(t, err)

	e, err := llbsolver.Load(def.ToPB())
	require.NoError(t, err)

	require.Equal(t, depth(e), 5)

	expected := specs.Platform{OS: "windows", Architecture: "amd64"}
	require.Equal(t, expected, platform(e))
	e = parent(e, 0)
	require.Equal(t, expected, platform(e))
	e = parent(e, 0)

	expected = specs.Platform{OS: "linux", Architecture: "arm", Variant: "v7"}
	require.Equal(t, expected, platform(e))
	e = parent(e, 0)
	require.Equal(t, expected, platform(e))
	require.Equal(t, []string{"baz"}, args(e))
	e = parent(e, 0)
	require.Equal(t, expected, platform(e))
	require.Equal(t, "docker-image://docker.io/library/foo:latest", id(e))
}

func TestDefaultPlatform(t *testing.T) {
	t.Parallel()

	s := llb.Image("foo").Run(llb.Shlex("bar"))

	def, err := s.Marshal()
	require.NoError(t, err)

	e, err := llbsolver.Load(def.ToPB())
	require.NoError(t, err)

	require.Equal(t, depth(e), 2)

	expected := platforms.DefaultSpec()
	require.Equal(t, expected, platform(e))
	require.Equal(t, []string{"bar"}, args(e))
	e = parent(e, 0)
	require.Equal(t, expected, platform(e))
	require.Equal(t, "docker-image://docker.io/library/foo:latest", id(e))
}

func TestPlatformOnMarshal(t *testing.T) {
	t.Parallel()

	s := llb.Image("image1").Run(llb.Shlex("bar"))

	def, err := s.Marshal(llb.Windows)
	require.NoError(t, err)

	e, err := llbsolver.Load(def.ToPB())
	require.NoError(t, err)

	expected := specs.Platform{OS: "windows", Architecture: "amd64"}
	require.Equal(t, expected, platform(e))
	e = parent(e, 0)
	require.Equal(t, expected, platform(e))
	require.Equal(t, "docker-image://docker.io/library/image1:latest", id(e))
}

func TestPlatformMixed(t *testing.T) {
	t.Parallel()

	s1 := llb.Image("image1").Run(llb.Shlex("cmd-main"))
	s2 := llb.Image("image2", llb.LinuxArmel).Run(llb.Shlex("cmd-sub"))
	s1.AddMount("/mnt", s2.Root())

	def, err := s1.Marshal(llb.LinuxAmd64)
	require.NoError(t, err)

	e, err := llbsolver.Load(def.ToPB())
	require.NoError(t, err)

	require.Equal(t, depth(e), 4)

	expectedAmd := specs.Platform{OS: "linux", Architecture: "amd64"}
	require.Equal(t, []string{"cmd-main"}, args(e))
	require.Equal(t, expectedAmd, platform(e))

	e1 := mount(e, "/")
	require.Equal(t, "docker-image://docker.io/library/image1:latest", id(e1))
	require.Equal(t, expectedAmd, platform(e1))

	expectedArm := specs.Platform{OS: "linux", Architecture: "arm", Variant: "v6"}
	e2 := mount(e, "/mnt")
	require.Equal(t, []string{"cmd-sub"}, args(e2))
	require.Equal(t, expectedArm, platform(e2))
	e2 = parent(e2, 0)
	require.Equal(t, "docker-image://docker.io/library/image2:latest", id(e2))
	require.Equal(t, expectedArm, platform(e2))
}

func toOp(e solver.Edge) *pb.Op {
	return e.Vertex.Sys().(*pb.Op)
}

func platform(e solver.Edge) specs.Platform {
	op := toOp(e)
	p := *op.Platform
	return specs.Platform{
		OS:           p.OS,
		Architecture: p.Architecture,
		Variant:      p.Variant,
		OSVersion:    p.OSVersion,
		OSFeatures:   p.OSFeatures,
	}
}

func depth(e solver.Edge) int {
	i := 1
	for _, inp := range e.Vertex.Inputs() {
		i += depth(inp)
	}
	return i
}

func parent(e solver.Edge, i int) solver.Edge {
	return e.Vertex.Inputs()[i]
}

func id(e solver.Edge) string {
	return toOp(e).GetSource().Identifier
}

func args(e solver.Edge) []string {
	return toOp(e).GetExec().Meta.Args
}

func mount(e solver.Edge, target string) solver.Edge {
	op := toOp(e).GetExec()
	for _, m := range op.Mounts {
		if m.Dest == target {
			return e.Vertex.Inputs()[int(m.Input)]
		}
	}
	panic("could not find mount " + target)
}
