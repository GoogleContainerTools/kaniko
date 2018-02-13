package snapshot

import (
	"archive/tar"
	"fmt"
	"github.com/priyawadhwa/kbuild/pkg/constants"
	"github.com/sirupsen/logrus"

	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Snapshotter holds the root directory from which to take snapshots, and a list of snapshots taken
type Snapshotter struct {
	l         *LayeredMap
	directory string
	snapshots []string
}

// NewSnapshotter creates a new snapshotter rooted at d
func NewSnapshotter(l *LayeredMap, d string) *Snapshotter {
	return &Snapshotter{l: l, directory: d, snapshots: []string{}}
}

// Init initializes a new snapshotter
func (s *Snapshotter) Init() error {
	if _, err := s.snapShotFS(ioutil.Discard); err != nil {
		return err
	}
	return nil
}

// TakeSnapshot takes a snapshot of the filesystem, avoiding directories in the whitelist, and creates
// a tarball of the changed files
func (s *Snapshotter) TakeSnapshot() error {
	fmt.Println("taking snapshots in ", s.directory)
	path := filepath.Join(s.directory+constants.WorkDir, fmt.Sprintf("layer-%d.tar", len(s.snapshots)))
	fmt.Println("Generating a snapshot in: ", path)
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	}

	added, err := s.snapShotFS(f)
	if err != nil {
		return err
	}
	if !added {
		logrus.Infof("No files were changed in this command, this layer will not be appended.")
		return os.Remove(path)
	}
	s.snapshots = append(s.snapshots, path)
	return nil
}

func (s *Snapshotter) snapShotFS(f io.Writer) (bool, error) {
	s.l.Snapshot()
	added := false
	w := tar.NewWriter(f)
	defer w.Close()

	err := filepath.Walk(s.directory, func(path string, info os.FileInfo, err error) error {
		if ignorePath(path, s.directory) {
			return nil
		}

		// Only add to the tar if we add it to the layeredmap.
		if s.l.MaybeAdd(path) {
			added = true
			return addToTar(path, info, w)
		}
		return nil
	})
	return added, err
}

// TODO: ignore anything in /proc/self/mounts
// ignore anything in the whitelist
func ignorePath(p, directory string) bool {
	for _, d := range constants.Whitelist {
		dirPath := filepath.Join(directory, d)
		if strings.HasPrefix(p, dirPath) {
			return true
		}
	}
	return false
}

func addToTar(p string, i os.FileInfo, w *tar.Writer) error {
	linkDst := ""
	fmt.Println("looking at ", i.Name())
	fmt.Println(os.Readlink(p))
	if i.Mode()&os.ModeSymlink != 0 {
		var err error
		linkDst, err = os.Readlink(p)
		if err != nil {
			return err
		}
	}
	hdr, err := tar.FileInfoHeader(i, linkDst)
	if err != nil {
		return err
	}
	hdr.Name = p
	w.WriteHeader(hdr)
	if !i.Mode().IsRegular() {
		return nil
	}
	r, err := os.Open(p)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}
