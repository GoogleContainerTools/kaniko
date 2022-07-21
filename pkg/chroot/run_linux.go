//go:build linux
// +build linux

package chroot

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	mobymount "github.com/moby/sys/mount"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func Chroot(newRoot string, additionalMounts ...string) (func() error, error) {
	// root fd for reverting
	root, err := os.Open("/")
	if err != nil {
		return nil, err
	}

	unmountFunc, err := prepareMounts(newRoot, additionalMounts...)
	if err != nil {
		return nil, err
	}

	revertFunc := func() error {
		logrus.Debug("exit chroot")
		defer root.Close()
		defer func() {
			err2 := unmountFunc()
			if err2 != nil {
				err = err2
			}
		}()
		if err2 := root.Chdir(); err2 != nil {
			err = err2
		}
		// check for errors first instead of returning, because unmount needs to be called after chroot
		err2 := unix.Chroot(".")
		if err2 != nil {
			err = fmt.Errorf("chroot back to old root: %w", err2)
		}
		return nil
	}
	logrus.Debugf("chdir into %v", newRoot)
	err = unix.Chdir(newRoot)
	if err != nil {
		return nil, fmt.Errorf("chdir to %v before chroot: %w", newRoot, err)
	}
	err = unix.Chroot(newRoot)
	if err != nil {
		return nil, fmt.Errorf("chroot to %v: %w", newRoot, err)
	}
	return revertFunc, nil
}

func prepareMounts(base string, additionalMounts ...string) (undoMount func() error, err error) {
	bindFlags := uintptr(unix.MS_BIND | unix.MS_REC | unix.MS_PRIVATE)
	devFlags := bindFlags | unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_RDONLY
	procFlags := devFlags | unix.MS_NODEV
	sysFlags := devFlags | unix.MS_NODEV
	type mountOpts struct {
		flags     uintptr
		mountType string
	}
	mounts := map[string]mountOpts{
		"/etc/resolv.conf": {flags: unix.MS_RDONLY | bindFlags},
		"/etc/hostname":    {flags: unix.MS_RDONLY | bindFlags},
		"/etc/hosts":       {flags: unix.MS_RDONLY | bindFlags},
		"/dev":             {flags: devFlags},
		"/sys":             {flags: sysFlags},
		"/proc":            {flags: procFlags},
	}
	for _, add := range additionalMounts {
		mounts[add] = mountOpts{flags: bindFlags}
	}

	for src, opts := range mounts {
		srcinfo, err := os.Lstat(src)
		if err != nil {
			return nil, fmt.Errorf("src %v for mount doesn't exist: %w", src, err)
		}
		dest := filepath.Join(base, src)
		err = createDest(srcinfo, dest)
		if err != nil {
			return nil, fmt.Errorf("creating dest %v: %w", dest, err)
		}
		err = mount(src, dest, opts.mountType, opts.flags)
		if err != nil {
			return nil, err
		}
		err = makeMountPrivate(dest)
		if err != nil {
			return nil, err
		}
	}

	undoMount = func() error {
		for src := range mounts {
			dest := filepath.Join(base, src)
			logrus.Debugf("unmounting %v", dest)
			err := unmount(dest)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return undoMount, nil
}

// createNewMountNamespace unshares into a new mount namespace for this process.
// This is not working with the current implementation because we need to exit this mount namespace
// before unmount the bind mounts (/proc, /dev etc.).
// For more information, see: https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#NOTES - Chapter: Resitrctions on mount namespaces
func createNewMountNamespace() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// Create a new mount namespace in which to do the things we're doing.
	if err := unix.Unshare(unix.CLONE_NEWNS); err != nil {
		return fmt.Errorf("error creating new mount namespace: %w", err)
	}
	return nil
}

// makeMountPrivate sets target to a rprivate mount.
func makeMountPrivate(target string) error {
	// Make all of our mounts private to our namespace.
	err := mobymount.MakeRPrivate(target)
	if err != nil {
		return fmt.Errorf("making %v private for new mnt namespace: %w", target, err)
	}
	return nil
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
			return fmt.Errorf("unmounting %q (retried %d times): %v", dest, retries, err)
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
	err := unix.Mount(src, dest, mountType, uintptr(flags), "")
	if err != nil {
		return fmt.Errorf("mounting %v to %v: %w", src, dest, err)
	}
	return nil
}
