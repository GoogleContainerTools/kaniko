package fs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var bufferPool = &sync.Pool{
	New: func() interface{} {
		buffer := make([]byte, 32*1024)
		return &buffer
	},
}

// Copy copies files using `cp -a` semantics.
// Copy is likely unsafe to be used in non-containerized environments.
func Copy(ctx context.Context, src, dst string, opts ...Opt) error {
	var ci CopyInfo
	for _, o := range opts {
		o(&ci)
	}
	ensureDstPath := dst
	if d, f := filepath.Split(dst); f != "" && f != "." {
		ensureDstPath = d
	}
	if ensureDstPath != "" {
		if err := os.MkdirAll(ensureDstPath, 0700); err != nil {
			return err
		}
	}
	dst = filepath.Clean(dst)

	c := newCopier(ci.Chown)
	srcs := []string{src}

	destRequiresDir := false
	if ci.AllowWildcards {
		d1, d2 := splitWildcards(src)
		if d2 != "" {
			matches, err := resolveWildcards(d1, d2)
			if err != nil {
				return err
			}
			srcs = matches
			destRequiresDir = true
		}
	}

	for _, src := range srcs {
		srcFollowed, err := filepath.EvalSymlinks(src)
		if err != nil {
			return err
		}
		dst, err := normalizedCopyInputs(srcFollowed, src, dst, destRequiresDir)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			return err
		}
		if err := c.copy(ctx, srcFollowed, dst, false); err != nil {
			return err
		}
	}

	return nil
}

func normalizedCopyInputs(srcFollowed, src, destPath string, forceDir bool) (string, error) {
	fiSrc, err := os.Lstat(srcFollowed)
	if err != nil {
		return "", err
	}

	fiDest, err := os.Stat(destPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrap(err, "failed to lstat destination path")
		}
	}

	if (forceDir && fiSrc.IsDir()) || (!fiSrc.IsDir() && fiDest != nil && fiDest.IsDir()) {
		destPath = filepath.Join(destPath, filepath.Base(src))
	}

	return destPath, nil
}

type ChownOpt struct {
	Uid, Gid int
}

type CopyInfo struct {
	Chown          *ChownOpt
	AllowWildcards bool
}

type Opt func(*CopyInfo)

func WithChown(uid, gid int) Opt {
	return func(ci *CopyInfo) {
		ci.Chown = &ChownOpt{Uid: uid, Gid: gid}
	}
}

func AllowWildcards(ci *CopyInfo) {
	ci.AllowWildcards = true
}

type copier struct {
	chown  *ChownOpt
	inodes map[uint64]string
}

func newCopier(chown *ChownOpt) *copier {
	return &copier{inodes: map[uint64]string{}, chown: chown}
}

// dest is always clean
func (c *copier) copy(ctx context.Context, src, target string, overwriteTargetMetadata bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	fi, err := os.Lstat(src)
	if err != nil {
		return errors.Wrapf(err, "failed to stat %s", src)
	}

	if !fi.IsDir() {
		if err := ensureEmptyFileTarget(target); err != nil {
			return err
		}
	}

	copyFileInfo := true

	switch {
	case fi.IsDir():
		if created, err := c.copyDirectory(ctx, src, target, fi, overwriteTargetMetadata); err != nil {
			return err
		} else if !overwriteTargetMetadata {
			copyFileInfo = created
		}
	case (fi.Mode() & os.ModeType) == 0:
		link, err := getLinkSource(target, fi, c.inodes)
		if err != nil {
			return errors.Wrap(err, "failed to get hardlink")
		}
		if link != "" {
			if err := os.Link(link, target); err != nil {
				return errors.Wrap(err, "failed to create hard link")
			}
		} else if err := copyFile(src, target); err != nil {
			return errors.Wrap(err, "failed to copy files")
		}
	case (fi.Mode() & os.ModeSymlink) == os.ModeSymlink:
		link, err := os.Readlink(src)
		if err != nil {
			return errors.Wrapf(err, "failed to read link: %s", src)
		}
		if err := os.Symlink(link, target); err != nil {
			return errors.Wrapf(err, "failed to create symlink: %s", target)
		}
	case (fi.Mode() & os.ModeDevice) == os.ModeDevice:
		if err := copyDevice(target, fi); err != nil {
			return errors.Wrapf(err, "failed to create device")
		}
	default:
		// TODO: Support pipes and sockets
		return errors.Wrapf(err, "unsupported mode %s", fi.Mode())
	}

	if copyFileInfo {
		if err := c.copyFileInfo(fi, target); err != nil {
			return errors.Wrap(err, "failed to copy file info")
		}

		if err := copyXAttrs(target, src); err != nil {
			return errors.Wrap(err, "failed to copy xattrs")
		}
	}
	return nil
}

func (c *copier) copyDirectory(ctx context.Context, src, dst string, stat os.FileInfo, overwriteTargetMetadata bool) (bool, error) {
	if !stat.IsDir() {
		return false, errors.Errorf("source is not directory")
	}

	created := false

	if st, err := os.Lstat(dst); err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		created = true
		if err := os.Mkdir(dst, stat.Mode()); err != nil {
			return created, errors.Wrapf(err, "failed to mkdir %s", dst)
		}
	} else if !st.IsDir() {
		return false, errors.Errorf("cannot copy to non-directory: %s", dst)
	} else if overwriteTargetMetadata {
		if err := os.Chmod(dst, stat.Mode()); err != nil {
			return false, errors.Wrapf(err, "failed to chmod on %s", dst)
		}
	}

	fis, err := ioutil.ReadDir(src)
	if err != nil {
		return false, errors.Wrapf(err, "failed to read %s", src)
	}

	for _, fi := range fis {
		if err := c.copy(ctx, filepath.Join(src, fi.Name()), filepath.Join(dst, fi.Name()), true); err != nil {
			return false, err
		}
	}

	return created, nil
}

func ensureEmptyFileTarget(dst string) error {
	fi, err := os.Lstat(dst)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "failed to lstat file target")
	}
	if fi.IsDir() {
		return errors.Errorf("cannot replace to directory %s with file", dst)
	}
	return os.Remove(dst)
}

func copyFile(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return errors.Wrapf(err, "failed to open source %s", source)
	}
	defer src.Close()
	tgt, err := os.Create(target)
	if err != nil {
		return errors.Wrapf(err, "failed to open target %s", target)
	}
	defer tgt.Close()

	return copyFileContent(tgt, src)
}

func containsWildcards(name string) bool {
	isWindows := runtime.GOOS == "windows"
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '\\' && !isWindows {
			i++
		} else if ch == '*' || ch == '?' || ch == '[' {
			return true
		}
	}
	return false
}

func splitWildcards(p string) (d1, d2 string) {
	parts := strings.Split(filepath.Join(p), string(filepath.Separator))
	var p1, p2 []string
	var found bool
	for _, p := range parts {
		if !found && containsWildcards(p) {
			found = true
		}
		if p == "" {
			p = "/"
		}
		if !found {
			p1 = append(p1, p)
		} else {
			p2 = append(p2, p)
		}
	}
	return filepath.Join(p1...), filepath.Join(p2...)
}

func resolveWildcards(basePath, comp string) ([]string, error) {
	var out []string
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := rel(basePath, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if match, _ := filepath.Match(comp, rel); !match {
			return nil
		}
		out = append(out, path)
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.Errorf("no matches found: %s", filepath.Join(basePath, comp))
	}
	return out, nil
}

// rel makes a path relative to base path. Same as `filepath.Rel` but can also
// handle UUID paths in windows.
func rel(basepath, targpath string) (string, error) {
	// filepath.Rel can't handle UUID paths in windows
	if runtime.GOOS == "windows" {
		pfx := basepath + `\`
		if strings.HasPrefix(targpath, pfx) {
			p := strings.TrimPrefix(targpath, pfx)
			if p == "" {
				p = "."
			}
			return p, nil
		}
	}
	return filepath.Rel(basepath, targpath)
}
