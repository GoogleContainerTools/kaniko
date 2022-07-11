//go:build linux
// +build linux

package chroot

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func Chroot(newRoot, kanikoDir, contextDir string) (func() error, error) {
	// root fd for reverting
	root, err := os.Open("/")
	if err != nil {
		return nil, err
	}
	// run chroot to a tempDir to isolate the rootDir
	var revertFunc func() error
	unmountFunc, err := prepareMounts(newRoot, kanikoDir, contextDir)
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
	logrus.Debugf("chroot into %v", newRoot)
	return revertFunc, unix.Chroot(newRoot)
}

func prepareMounts(base, kanikoDir, contextDir string) (func() error, error) {
	var unmountFunc func() error
	mounts := []struct {
		src   string
		flags uint
	}{
		{
			src:   "/dev",
			flags: unix.MS_BIND,
		},
		{
			src:   "/proc",
			flags: unix.MS_BIND | unix.MS_RDONLY,
		},
		{
			src:   contextDir,
			flags: unix.MS_BIND,
		},
	}
	for _, m := range mounts {
		err := mount(m.src, filepath.Join(base, m.src), m.flags)
		if err != nil {
			return unmountFunc, err
		}
	}
	unmountFunc = func() error {
		for _, m := range mounts {
      dest := filepath.Join(base, m.src)
      logrus.Debugf("unmounting %v", dest)
      // perform lazy detaching
			err := unix.Unmount(dest, unix.MNT_DETACH)
			if err != nil {
        return fmt.Errorf("unmounting %v: %w", dest, err)
			}
		}
		return nil
	}
	// create kanikoDir in new root for snapshots
	err := os.Mkdir(filepath.Join(base, kanikoDir), 0755)
	if err != nil {
		return unmountFunc, err
	}
	return unmountFunc, nil
}

func mount(src, dest string, flags uint) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("creating %v for mount: %w", dest, err)
	}
  logrus.Debugf("mounting %v to %v", src, dest)
	if err := unix.Mount(src, dest, "", uintptr(flags), ""); err != nil {
		return fmt.Errorf("mounting %v to %v: %w", src, dest, err)
	}
	return nil
}
