package dockerfile2llb

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectives(t *testing.T) {
	t.Parallel()

	dt := `#escape=\
# key = FOO bar

# smth
`

	d := ParseDirectives(bytes.NewBuffer([]byte(dt)))
	require.Equal(t, len(d), 2, fmt.Sprintf("%+v", d))

	v, ok := d["escape"]
	require.True(t, ok)
	require.Equal(t, v, "\\")

	v, ok = d["key"]
	require.True(t, ok)
	require.Equal(t, v, "FOO bar")

	// for some reason Moby implementation in case insensitive for escape
	dt = `# EScape=\
# KEY = FOO bar

# smth
`

	d = ParseDirectives(bytes.NewBuffer([]byte(dt)))
	require.Equal(t, len(d), 2, fmt.Sprintf("%+v", d))

	v, ok = d["escape"]
	require.True(t, ok)
	require.Equal(t, v, "\\")

	v, ok = d["key"]
	require.True(t, ok)
	require.Equal(t, v, "FOO bar")
}

func TestSyntaxDirective(t *testing.T) {
	t.Parallel()

	dt := `# syntax = dockerfile:experimental // opts
FROM busybox
`

	ref, cmdline, ok := DetectSyntax(bytes.NewBuffer([]byte(dt)))
	require.True(t, ok)
	require.Equal(t, ref, "dockerfile:experimental")
	require.Equal(t, cmdline, "dockerfile:experimental // opts")

	dt = `FROM busybox
RUN ls
`
	ref, cmdline, ok = DetectSyntax(bytes.NewBuffer([]byte(dt)))
	require.False(t, ok)
	require.Equal(t, ref, "")
	require.Equal(t, cmdline, "")

}
