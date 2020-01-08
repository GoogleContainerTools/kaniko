package copy

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/otiai10/mint"
)

func TestMain(m *testing.M) {
	os.MkdirAll("testdata.copy", os.ModePerm)
	code := m.Run()
	os.RemoveAll("testdata.copy")
	os.Exit(code)
}

func TestCopy(t *testing.T) {

	err := Copy("./testdata/case00", "./testdata.copy/case00")
	Expect(t, err).ToBe(nil)
	info, err := os.Stat("./testdata.copy/case00/README.md")
	Expect(t, err).ToBe(nil)
	Expect(t, info.IsDir()).ToBe(false)

	When(t, "specified src doesn't exist", func(t *testing.T) {
		err := Copy("NOT/EXISTING/SOURCE/PATH", "anywhere")
		Expect(t, err).Not().ToBe(nil)
	})

	When(t, "specified src is just a file", func(t *testing.T) {
		err := Copy("testdata/case01/README.md", "testdata.copy/case01/README.md")
		Expect(t, err).ToBe(nil)
	})

	When(t, "too long name is given", func(t *testing.T) {
		dest := "foobar"
		for i := 0; i < 8; i++ {
			dest = dest + dest
		}
		err := Copy("testdata/case00", filepath.Join("testdata/case00", dest))
		Expect(t, err).Not().ToBe(nil)
		Expect(t, err).TypeOf("*os.PathError")
	})

	When(t, "try to create not permitted location", func(t *testing.T) {
		err := Copy("testdata/case00", "/case00")
		Expect(t, err).Not().ToBe(nil)
		Expect(t, err).TypeOf("*os.PathError")
	})

	When(t, "try to create a directory on existing file name", func(t *testing.T) {
		err := Copy("testdata/case02", "testdata.copy/case00/README.md")
		Expect(t, err).Not().ToBe(nil)
		Expect(t, err).TypeOf("*os.PathError")
	})

	When(t, "source directory includes symbolic link", func(t *testing.T) {
		err := Copy("testdata/case03", "testdata.copy/case03")
		Expect(t, err).ToBe(nil)
		info, err := os.Lstat("testdata.copy/case03/case01")
		Expect(t, err).ToBe(nil)
		Expect(t, info.Mode()&os.ModeSymlink).Not().ToBe(0)
	})

	When(t, "try to copy a file to existing path", func(t *testing.T) {
		err := Copy("testdata/case04/README.md", "testdata/case04")
		Expect(t, err).Not().ToBe(nil)
		err = Copy("testdata/case04/README.md", "testdata/case04/README.md/foobar")
		Expect(t, err).Not().ToBe(nil)
	})
}
