// +build linux darwin freebsd

package continuityfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/containerd/continuity"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// File represents any file type (non directory) in the filesystem
type File struct {
	inode    uint64
	uid      uint32
	gid      uint32
	provider FileContentProvider
	resource continuity.Resource
}

// NewFile creates a new file with the given inode and content provider
func NewFile(inode uint64, provider FileContentProvider) *File {
	return &File{
		inode:    inode,
		provider: provider,
	}
}

func (f *File) setResource(r continuity.Resource) (err error) {
	// TODO: error out if uid excesses uint32?
	f.uid = uint32(r.UID())
	f.gid = uint32(r.GID())
	f.resource = r

	return
}

// Attr sets the fuse attribute for the file
func (f *File) Attr(ctx context.Context, attr *fuse.Attr) (err error) {
	// Set attributes from resource metadata
	attr.Mode = f.resource.Mode()
	attr.Uid = f.uid
	attr.Gid = f.gid

	if rf, ok := f.resource.(continuity.RegularFile); ok {
		attr.Nlink = uint32(len(rf.Paths()))
		attr.Size = uint64(rf.Size())
	} else {
		attr.Nlink = 1
	}

	attr.Inode = f.inode

	return nil
}

// Open opens the file for read
// currently only regular files can be opened
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	var dgst digest.Digest
	if rf, ok := f.resource.(continuity.RegularFile); ok {
		digests := rf.Digests()
		if len(digests) > 0 {
			dgst = digests[0]
		}
	}
	// TODO(dmcgowan): else check if device can be opened for read
	r, err := f.provider.Open(f.resource.Path(), dgst)
	if err != nil {
		logrus.Debugf("Error opening handle: %v", err)
		return nil, err
	}
	return &fileHandler{
		reader: r,
	}, nil

}

func (f *File) getDirent(name string) (fuse.Dirent, error) {
	var t fuse.DirentType
	switch f.resource.(type) {
	case continuity.RegularFile:
		t = fuse.DT_File
	case continuity.SymLink:
		t = fuse.DT_Link
	case continuity.Device:
		t = fuse.DT_Block
	case continuity.NamedPipe:
		t = fuse.DT_FIFO
	default:
		t = fuse.DT_Unknown
	}

	return fuse.Dirent{
		Inode: f.inode,
		Type:  t,
		Name:  name,
	}, nil
}

type fileHandler struct {
	offset int64
	reader io.ReadCloser
}

func (h *fileHandler) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if h.offset != req.Offset {
		if seeker, ok := h.reader.(io.Seeker); ok {
			if _, err := seeker.Seek(req.Offset, os.SEEK_SET); err != nil {
				logrus.Debugf("Error seeking: %v", err)
				return err
			}
			h.offset = req.Offset
		} else {
			return errors.New("unable to seek to offset")
		}
	}

	n, err := h.reader.Read(resp.Data[:req.Size])
	if err != nil {
		logrus.Debugf("Read error: %v", err)
		return err
	}
	h.offset = h.offset + int64(n)

	resp.Data = resp.Data[:n]

	return nil
}

func (h *fileHandler) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return h.reader.Close()
}

// Dir represents a file system directory
type Dir struct {
	inode    uint64
	uid      uint32
	gid      uint32
	nodes    map[string]fs.Node
	provider FileContentProvider
	resource continuity.Resource
}

// Attr sets the fuse attributes for the directory
func (d *Dir) Attr(ctx context.Context, attr *fuse.Attr) (err error) {
	if d.resource == nil {
		attr.Mode = os.ModeDir | 0555
	} else {
		attr.Mode = d.resource.Mode()
	}

	attr.Uid = d.uid
	attr.Gid = d.gid
	attr.Inode = d.inode

	return nil
}

func (d *Dir) getDirent(name string) (fuse.Dirent, error) {
	return fuse.Dirent{
		Inode: d.inode,
		Type:  fuse.DT_Dir,
		Name:  name,
	}, nil
}

type direnter interface {
	getDirent(name string) (fuse.Dirent, error)
}

// Lookup looks up the filesystem node for the name within the directory
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	node, ok := d.nodes[name]
	if !ok {
		return nil, fuse.ENOENT
	}
	return node, nil
}

// ReadDirAll reads all the directory entries
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	ents := make([]fuse.Dirent, 0, len(d.nodes))
	for name, node := range d.nodes {
		if nd, ok := node.(direnter); ok {
			de, err := nd.getDirent(name)
			if err != nil {
				return nil, err
			}
			ents = append(ents, de)
		} else {
			logrus.Errorf("%s does not have a directory entry", name)
		}
	}

	return ents, nil
}

func (d *Dir) setResource(r continuity.Resource) (err error) {
	d.uid = uint32(r.UID())
	d.gid = uint32(r.GID())
	d.resource = r

	return
}

// NewDir creates a new directory object
func NewDir(inode uint64, provider FileContentProvider) *Dir {
	return &Dir{
		inode:    inode,
		nodes:    map[string]fs.Node{},
		provider: provider,
	}
}

var (
	rootPath = fmt.Sprintf("%c", filepath.Separator)
)

func addNode(path string, node fs.Node, cache map[string]*Dir, provider FileContentProvider) {
	dirPath, file := filepath.Split(path)
	d, ok := cache[dirPath]
	if !ok {
		d = NewDir(0, provider)
		cache[dirPath] = d
		addNode(filepath.Clean(dirPath), d, cache, provider)
	}
	d.nodes[file] = node
	logrus.Debugf("%s (%#v) added to %s", file, node, dirPath)
}

type treeRoot struct {
	root *Dir
}

func (t treeRoot) Root() (fs.Node, error) {
	logrus.Debugf("Returning root with %#v", t.root.nodes)
	return t.root, nil
}

// NewFSFromManifest creates a fuse filesystem using the given manifest
// to create the node tree and the content provider to serve up
// content for regular files.
func NewFSFromManifest(manifest *continuity.Manifest, mountRoot string, provider FileContentProvider) (fs.FS, error) {
	tree := treeRoot{
		root: NewDir(0, provider),
	}

	fi, err := os.Stat(mountRoot)
	if err != nil {
		return nil, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, errors.New("could not access directory")
	}
	tree.root.uid = st.Uid
	tree.root.gid = st.Gid

	dirCache := map[string]*Dir{
		rootPath: tree.root,
	}

	for i, resource := range manifest.Resources {
		inode := uint64(i) + 1
		if _, ok := resource.(continuity.Directory); ok {
			cleanPath := filepath.Clean(resource.Path())
			keyPath := fmt.Sprintf("%s%c", cleanPath, filepath.Separator)
			d, ok := dirCache[keyPath]
			if !ok {
				d = NewDir(inode, provider)
				dirCache[keyPath] = d
				addNode(cleanPath, d, dirCache, provider)
			} else {
				d.inode = inode
			}
			if err := d.setResource(resource); err != nil {
				return nil, err
			}
			continue
		}
		f := NewFile(inode, provider)
		if err := f.setResource(resource); err != nil {
			return nil, err
		}
		if rf, ok := resource.(continuity.RegularFile); ok {

			for _, p := range rf.Paths() {
				addNode(p, f, dirCache, provider)
			}
		} else {
			addNode(resource.Path(), f, dirCache, provider)
		}
	}

	return tree, nil
}
