package isolation

import (
	"fmt"

	"github.com/GoogleContainerTools/kaniko/pkg/chroot"
)

type Isolator interface {
	NewRoot() (newRoot string, exitFunc func() error, err error)
}

type Chroot struct{}

var _ Isolator = Chroot{}

func (c Chroot) NewRoot() (newRoot string, exitFunc func() error, err error) {
	newRoot, err = chroot.TmpDirInHome()
	if err != nil {
		return "", nil, fmt.Errorf("getting newRoot: %w", err)
	}
	revertMounts, err := chroot.PrepareMounts(newRoot)
	if err != nil {
		return "", nil, fmt.Errorf("creating mounts: %w", err)
	}
	exitFunc = func() error {
		errRevert := revertMounts()
		if errRevert != nil {
			return errRevert
		}
		return nil
	}
	return newRoot, exitFunc, nil
}

type None struct{}

func (n None) NewRoot() (newRoot string, exitFunc func() error, err error) {
	return "/", func() error { return nil }, nil
}
