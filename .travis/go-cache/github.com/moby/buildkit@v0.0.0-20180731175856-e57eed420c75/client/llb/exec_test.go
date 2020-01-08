package llb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTmpfsMountError(t *testing.T) {
	t.Parallel()

	st := Image("foo").Run(Shlex("args")).AddMount("/tmp", Scratch(), Tmpfs())
	_, err := st.Marshal()

	require.Error(t, err)
	require.Contains(t, err.Error(), "can't be used as a parent")

	st = Image("foo").Run(Shlex("args"), AddMount("/tmp", Scratch(), Tmpfs())).Root()
	_, err = st.Marshal()
	require.NoError(t, err)

	st = Image("foo").Run(Shlex("args"), AddMount("/tmp", Image("bar"), Tmpfs())).Root()
	_, err = st.Marshal()
	require.Error(t, err)
	require.Contains(t, err.Error(), "must use scratch")
}
