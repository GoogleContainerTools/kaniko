package fsutil

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestWalkerSimple(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo file",
		"ADD foo2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, string(b.Bytes()), `file foo
file foo2
`)

}

func TestWalkerInclude(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar dir",
		"ADD bar/foo file",
		"ADD foo2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"bar"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"bar/foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"b*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"bar/f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"bar/g*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file foo2
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"b*/f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"b*/foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"b*/"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))
}

func TestWalkerExclude(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar file",
		"ADD foo dir",
		"ADD foo2 file",
		"ADD foo/bar2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		ExcludePatterns: []string{"foo*", "!foo/bar2"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file bar
dir foo
file foo/bar2
`, string(b.Bytes()))

}

func TestWalkerFollowLinks(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar file",
		"ADD foo dir",
		"ADD foo/l1 symlink /baz/one",
		"ADD foo/l2 symlink /baz/two",
		"ADD baz dir",
		"ADD baz/one file",
		"ADD baz/two symlink ../bax",
		"ADD bax file",
		"ADD bay file", // not included
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		FollowPaths: []string{"foo/l*", "bar"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file bar
file bax
dir baz
file baz/one
symlink:../bax baz/two
dir foo
symlink:/baz/one foo/l1
symlink:/baz/two foo/l2
`, string(b.Bytes()))
}

func TestWalkerFollowLinksToRoot(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo symlink .",
		"ADD bar file",
		"ADD bax file",
		"ADD bay dir",
		"ADD bay/baz file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		FollowPaths: []string{"foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file bar
file bax
dir bay
file bay/baz
symlink:. foo
`, string(b.Bytes()))
}

func TestWalkerMap(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar file",
		"ADD foo dir",
		"ADD foo2 file",
		"ADD foo/bar2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		Map: func(s *Stat) bool {
			if strings.HasPrefix(s.Path, "foo") {
				s.Path = "_" + s.Path
				return true
			}
			return false
		},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir _foo
file _foo/bar2
file _foo2
`, string(b.Bytes()))
}

func TestMatchPrefix(t *testing.T) {
	ok, partial := matchPrefix("foo", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = matchPrefix("foo/bar/baz", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = matchPrefix("foo/bar/baz", "foo/bar")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = matchPrefix("foo/bar/baz", "foo/bax")
	assert.Equal(t, false, ok)

	ok, partial = matchPrefix("foo/bar/baz", "foo/bar/baz")
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = matchPrefix("f*", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = matchPrefix("foo/bar/*", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = matchPrefix("foo/*/baz", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = matchPrefix("*/*/baz", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = matchPrefix("*/bar/baz", "foo/bar")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = matchPrefix("*/bar/baz", "foo/bax")
	assert.Equal(t, false, ok)

	ok, partial = matchPrefix("*/*/baz", "foo/bar/baz")
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)
}

func bufWalk(buf *bytes.Buffer) filepath.WalkFunc {
	return func(path string, fi os.FileInfo, err error) error {
		stat, ok := fi.Sys().(*Stat)
		if !ok {
			return errors.Errorf("invalid symlink %s", path)
		}
		t := "file"
		if fi.IsDir() {
			t = "dir"
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			t = "symlink:" + stat.Linkname
		}
		fmt.Fprintf(buf, "%s %s", t, path)
		if fi.Mode()&os.ModeSymlink == 0 && stat.Linkname != "" {
			fmt.Fprintf(buf, " >%s", stat.Linkname)
		}
		fmt.Fprintln(buf)
		return nil
	}
}

func tmpDir(inp []*change) (dir string, retErr error) {
	tmpdir, err := ioutil.TempDir("", "diff")
	if err != nil {
		return "", err
	}
	defer func() {
		if retErr != nil {
			os.RemoveAll(tmpdir)
		}
	}()
	for _, c := range inp {
		if c.kind == ChangeKindAdd {
			p := filepath.Join(tmpdir, c.path)
			stat, ok := c.fi.Sys().(*Stat)
			if !ok {
				return "", errors.Errorf("invalid symlink change %s", p)
			}
			if c.fi.IsDir() {
				if err := os.Mkdir(p, 0700); err != nil {
					return "", err
				}
			} else if c.fi.Mode()&os.ModeSymlink != 0 {
				if err := os.Symlink(stat.Linkname, p); err != nil {
					return "", err
				}
			} else if len(stat.Linkname) > 0 {
				if err := os.Link(filepath.Join(tmpdir, stat.Linkname), p); err != nil {
					return "", err
				}
			} else {
				f, err := os.Create(p)
				if err != nil {
					return "", err
				}
				if len(c.data) > 0 {
					if _, err := f.Write([]byte(c.data)); err != nil {
						return "", err
					}
				}
				f.Close()
			}
		}
	}
	return tmpdir, nil
}
