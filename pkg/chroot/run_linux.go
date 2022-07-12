//go:build linux
// +build linux

package chroot

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func Chroot(newRoot string, additionalMounts ...string) (func() error, error) {
	// root fd for reverting
	root, err := os.Open("/")
	if err != nil {
		return nil, err
	}
	// run chroot to a tempDir to isolate the rootDir
	var revertFunc func() error
	unmountFunc, err := prepareMounts(newRoot, additionalMounts...)
	if err != nil {
		return revertFunc, err
	}

	revertFunc = func() error {
		logrus.Debug("exit chroot")
		defer root.Close()
		defer func() {
			err := unmountFunc()
			if err != nil {
				logrus.Fatalf("unmounting: %v", err)
			}
		}()
		if err := root.Chdir(); err != nil {
			return err
		}
		// check for errors first instead of returning, because unmount needs to be called after chroot
		err := unix.Chroot(".")
		if err != nil {
			return fmt.Errorf("chroot back to old root: %w", err)
		}
		return nil
	}
	logrus.Debugf("chdir into %v", newRoot)
	err = unix.Chdir(newRoot)
	if err != nil {
		return revertFunc, fmt.Errorf("chroot to %v: %w", newRoot, err)
	}
	// chroot doesn't change the current directory
	return revertFunc, unix.Chroot(newRoot)
}

func prepareMounts(base string, additionalMounts ...string) (undoMount func() error, err error) {
	var unmountFunc func() error
	bindFlags := uintptr(unix.MS_BIND | unix.MS_REC | unix.MS_PRIVATE)
	devFlags := bindFlags | unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_RDONLY
	procFlags := devFlags | unix.MS_NODEV
	sysFlags := devFlags | unix.MS_NODEV
	type mountOpts struct {
		flags     uintptr
		mountType string
	}
	// use a seperate map for move mounts because handling unmount is different
	moveMounts := map[string]mountOpts{
		"/etc/resolv.conf": {flags: unix.MS_RDONLY | unix.MS_MOVE},
		"/etc/hostname":    {flags: unix.MS_RDONLY | unix.MS_MOVE},
		"/etc/hosts":       {flags: unix.MS_RDONLY | unix.MS_MOVE},
	}
	mounts := map[string]mountOpts{
		"/dev":  {flags: devFlags},
		"/sys":  {flags: sysFlags},
		"/proc": {flags: procFlags},
	}
	for _, add := range additionalMounts {
		mounts[add] = mountOpts{flags: bindFlags}
	}
	for src, opt := range moveMounts {
		mounts[src] = opt
	}

	for src, opts := range mounts {
		srcinfo, err := os.Lstat(src)
		if err != nil {
			return unmountFunc, fmt.Errorf("src %v for mount doesn't exist: %w", src, err)
		}
		dest := filepath.Join(base, src)
		err = createDest(srcinfo, dest)
		if err != nil {
			return unmountFunc, fmt.Errorf("creating dest %v: %w", dest, err)
		}
		err = mount(src, dest, opts.mountType, opts.flags)
		if err != nil {
			return unmountFunc, err
		}
	}

	unmountFunc = func() error {
		for src, opts := range mounts {
			// check for MS_MOVE flag
			if unix.MS_MOVE&opts.flags == unix.MS_MOVE {
				logrus.Debugf("found move mount, moving mount at %s back to old src", src)
				mountedOn := filepath.Join(base, src)
				err = mount(mountedOn, src, opts.mountType, opts.flags)
				if err != nil {
					return fmt.Errorf("moving mount %v back to %v: %w", mountedOn, src, err)
				}

			} else {
				dest := filepath.Join(base, src)
				logrus.Debugf("unmounting %v", dest)
				err := unmount(dest)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	return unmountFunc, nil
}

func unmount(dest string) error {
	// perform lazy detaching if bind mount
	err := unix.Unmount(dest, unix.MNT_DETACH)
	if err != nil {
		retries := 0
		for (err == unix.EBUSY || err == unix.EAGAIN) && retries < 50 {
			time.Sleep(50 * time.Millisecond)
			err = unix.Unmount(dest, unix.MNT_DETACH)
			retries++
		}
		if err != nil {
			logrus.Warnf("pkg/chroot: error unmounting %q (retried %d times): %v", dest, retries, err)
			return fmt.Errorf("unmounting %v: %w", dest, err)
		}
	}
	return nil
}

func createDest(srcinfo fs.FileInfo, dest string) error {
	// Check if target is a symlink
	_, err := os.Lstat(dest)
	if err != nil {
		// If the target can't be stat()ted, check the error.
		if !os.IsNotExist(err) {
			return fmt.Errorf("error examining %q for mounting: %w", dest, err)
		}
		// The target isn't there yet, so create it.
		if srcinfo.IsDir() {
			if err = os.MkdirAll(dest, 0755); err != nil {
				return fmt.Errorf("error creating mountpoint %q in mount namespace: %w", dest, err)
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return fmt.Errorf("error ensuring parent of mountpoint %q (%q) is present in new root: %w", dest, filepath.Dir(dest), err)
			}
			var file *os.File
			if file, err = os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
				return fmt.Errorf("error creating mountpoint %q: %w", dest, err)
			}
			file.Close()
		}
	}
	return nil
}

func mount(src, dest, mountType string, flags uintptr) error {
	logrus.Debugf("mounting %v to %v", src, dest)
	if err := unix.Mount(src, dest, mountType, uintptr(flags), ""); err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(dest, 0755)
			if err == nil {
				err = unix.Mount(src, dest, mountType, uintptr(flags), "")
			}
		}
		if err != nil {
			return fmt.Errorf("mounting %v to %v: %w", src, dest, err)
		}
	}
	return nil
}