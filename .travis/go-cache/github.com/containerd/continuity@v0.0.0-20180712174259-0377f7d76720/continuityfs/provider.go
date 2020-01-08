// +build linux darwin freebsd

package continuityfs

import (
	"io"
	"path/filepath"

	"github.com/containerd/continuity/driver"
	"github.com/opencontainers/go-digest"
)

// FileContentProvider is an object which is used to fetch
// data and inode information about a path or digest.
// TODO(dmcgowan): Update GetContentPath to provide a
// filehandle or ReadWriteCloser.
type FileContentProvider interface {
	Path(string, digest.Digest) (string, error)
	Open(string, digest.Digest) (io.ReadCloser, error)
}

type fsContentProvider struct {
	root   string
	driver driver.Driver
}

// NewFSFileContentProvider creates a new content provider which
// gets content from a directory on an existing filesystem based
// on the resource path.
func NewFSFileContentProvider(root string, driver driver.Driver) FileContentProvider {
	return &fsContentProvider{
		root:   root,
		driver: driver,
	}
}

func (p *fsContentProvider) Path(path string, dgst digest.Digest) (string, error) {
	return filepath.Join(p.root, path), nil
}

func (p *fsContentProvider) Open(path string, dgst digest.Digest) (io.ReadCloser, error) {
	return p.driver.Open(filepath.Join(p.root, path))
}
