package llb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelativeWd(t *testing.T) {
	st := Scratch().Dir("foo")
	require.Equal(t, st.GetDir(), "/foo")

	st = st.Dir("bar")
	require.Equal(t, st.GetDir(), "/foo/bar")

	st = st.Dir("..")
	require.Equal(t, st.GetDir(), "/foo")

	st = st.Dir("/baz")
	require.Equal(t, st.GetDir(), "/baz")

	st = st.Dir("../../..")
	require.Equal(t, st.GetDir(), "/")
}
