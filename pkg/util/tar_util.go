/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/system"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Tar knows how to write files to a tar file.
type Tar struct {
	hardlinks map[uint64]string
	w         *tar.Writer
}

// NewTar will create an instance of Tar that can write files to the writer at f.
func NewTar(f io.Writer) Tar {
	w := tar.NewWriter(f)
	return Tar{
		w:         w,
		hardlinks: map[uint64]string{},
	}
}

func CreateTarballOfDirectory(pathToDir string, f io.Writer) error {
	if !filepath.IsAbs(pathToDir) {
		return errors.New("pathToDir is not absolute")
	}
	tarWriter := NewTar(f)
	defer tarWriter.Close()

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !filepath.IsAbs(path) {
			return fmt.Errorf("path %v is not absolute, cant read file", path)
		}
		return tarWriter.AddFileToTar(path)
	}

	return filepath.WalkDir(pathToDir, walkFn)
}

// Close will close any open streams used by Tar.
func (t *Tar) Close() {
	t.w.Close()
}

// AddFileToTar adds the file at path p to the tar
func (t *Tar) AddFileToTar(p string) error {
	i, err := os.Lstat(p)
	if err != nil {
		return fmt.Errorf("Failed to get file info for %s: %s", p, err)
	}
	linkDst := ""
	if i.Mode()&os.ModeSymlink != 0 {
		var err error
		linkDst, err = os.Readlink(p)
		if err != nil {
			return err
		}
	}
	if i.Mode()&os.ModeSocket != 0 {
		logrus.Infof("Ignoring socket %s, not adding to tar", i.Name())
		return nil
	}
	hdr, err := tar.FileInfoHeader(i, linkDst)
	if err != nil {
		return err
	}
	err = readSecurityXattrToTarHeader(p, hdr)
	if err != nil {
		return err
	}

	if p == config.RootDir {
		// allow entry for / to preserve permission changes etc. (currently ignored anyway by Docker runtime)
		hdr.Name = "/"
	} else {
		// Docker uses no leading / in the tarball
		hdr.Name = strings.TrimPrefix(p, config.RootDir)
		hdr.Name = strings.TrimLeft(hdr.Name, "/")
	}
	if hdr.Typeflag == tar.TypeDir && !strings.HasSuffix(hdr.Name, "/") {
		hdr.Name = hdr.Name + "/"
	}
	// rootfs may not have been extracted when using cache, preventing uname/gname from resolving
	// this makes this layer unnecessarily differ from a cached layer which does contain this information
	hdr.Uname = ""
	hdr.Gname = ""
	// use PAX format to preserve accurate mtime (match Docker behavior)
	hdr.Format = tar.FormatPAX

	hardlink, linkDst := t.checkHardlink(p, i)
	if hardlink {
		hdr.Linkname = linkDst
		hdr.Typeflag = tar.TypeLink
		hdr.Size = 0
	}
	if err := t.w.WriteHeader(hdr); err != nil {
		return err
	}
	if !(i.Mode().IsRegular()) || hardlink {
		return nil
	}
	r, err := os.Open(p)
	if err != nil {
		return err
	}
	defer r.Close()
	if _, err := io.Copy(t.w, r); err != nil {
		return err
	}
	return nil
}

const (
	securityCapabilityXattr = "security.capability"
)

// writeSecurityXattrToTarHeader writes security.capability
// xattrs from a tar header to filesystem
func writeSecurityXattrToToFile(path string, hdr *tar.Header) error {
	if hdr.Xattrs == nil {
		return nil
	}
	if capability, ok := hdr.Xattrs[securityCapabilityXattr]; ok {
		err := system.Lsetxattr(path, securityCapabilityXattr, []byte(capability), 0)
		if err != nil && !errors.Is(err, syscall.EOPNOTSUPP) && err != system.ErrNotSupportedPlatform {
			return errors.Wrapf(err, "failed to write %q attribute to %q", securityCapabilityXattr, path)
		}
	}
	return nil
}

// readSecurityXattrToTarHeader reads security.capability
// xattrs from filesystem to a tar header
func readSecurityXattrToTarHeader(path string, hdr *tar.Header) error {
	if hdr.Xattrs == nil {
		hdr.Xattrs = make(map[string]string)
	}
	capability, err := system.Lgetxattr(path, securityCapabilityXattr)
	if err != nil && !errors.Is(err, syscall.EOPNOTSUPP) && err != system.ErrNotSupportedPlatform {
		return errors.Wrapf(err, "failed to read %q attribute from %q", securityCapabilityXattr, path)
	}
	if capability != nil {
		hdr.Xattrs[securityCapabilityXattr] = string(capability)
	}
	return nil
}

func (t *Tar) Whiteout(p string) error {
	dir := filepath.Dir(p)
	name := archive.WhiteoutPrefix + filepath.Base(p)

	th := &tar.Header{
		// Docker uses no leading / in the tarball
		Name: strings.TrimLeft(filepath.Join(dir, name), "/"),
		Size: 0,
	}
	if err := t.w.WriteHeader(th); err != nil {
		return err
	}

	return nil
}

// Returns true if path is hardlink, and the link destination
func (t *Tar) checkHardlink(p string, i os.FileInfo) (bool, string) {
	hardlink := false
	linkDst := ""
	stat := getSyscallStatT(i)
	if stat != nil {
		nlinks := stat.Nlink
		if nlinks > 1 {
			inode := stat.Ino
			if original, exists := t.hardlinks[inode]; exists && original != p {
				hardlink = true
				logrus.Debugf("%s inode exists in hardlinks map, linking to %s", p, original)
				linkDst = original
			} else {
				t.hardlinks[inode] = p
			}
		}
	}
	return hardlink, linkDst
}

func getSyscallStatT(i os.FileInfo) *syscall.Stat_t {
	if sys := i.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			return stat
		}
	}
	return nil
}

// UnpackLocalTarArchive unpacks the tar archive at path to the directory dest
// Returns the files extracted from the tar archive
func UnpackLocalTarArchive(path, dest string) ([]string, error) {
	// First, we need to check if the path is a local tar archive
	if compressed, compressionLevel := fileIsCompressedTar(path); compressed {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		if compressionLevel == archive.Gzip {
			return nil, UnpackCompressedTar(path, dest)
		} else if compressionLevel == archive.Bzip2 {
			bzr := bzip2.NewReader(file)
			return UnTar(bzr, dest)
		}
	}
	if fileIsUncompressedTar(path) {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		return UnTar(file, dest)
	}
	return nil, errors.New("path does not lead to local tar archive")
}

// IsFileLocalTarArchive returns true if the file is a local tar archive
func IsFileLocalTarArchive(src string) bool {
	compressed, _ := fileIsCompressedTar(src)
	uncompressed := fileIsUncompressedTar(src)
	return compressed || uncompressed
}

func fileIsCompressedTar(src string) (bool, archive.Compression) {
	r, err := os.Open(src)
	if err != nil {
		return false, -1
	}
	defer r.Close()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return false, -1
	}
	compressionLevel := archive.DetectCompression(buf)
	return (compressionLevel > 0), compressionLevel
}

func fileIsUncompressedTar(src string) bool {
	r, err := os.Open(src)
	if err != nil {
		return false
	}
	defer r.Close()
	fi, err := os.Lstat(src)
	if err != nil {
		return false
	}
	if fi.Size() == 0 {
		return false
	}
	tr := tar.NewReader(r)
	if tr == nil {
		return false
	}
	_, err = tr.Next()
	return err == nil
}

// UnpackCompressedTar unpacks the compressed tar at path to dir
func UnpackCompressedTar(path, dir string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()
	_, err = UnTar(gzr, dir)
	return err
}
