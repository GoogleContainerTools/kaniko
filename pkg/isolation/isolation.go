package isolation

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/GoogleContainerTools/kaniko/pkg/chroot"
	"github.com/pkg/errors"
)

type Isolator interface {
	NewRoot() (newRoot string, err error)
	ExecRunCommand(cmd *exec.Cmd) error
}

type Chroot struct {
	// rootDir holds the dir created by NewRoot
	rootDir string
}

var _ Isolator = &Chroot{}

func (c *Chroot) NewRoot() (newRoot string, err error) {
	newRoot, err = chroot.TmpDirInHome()
	if err != nil {
		return "", fmt.Errorf("getting newRoot: %w", err)
	}
	c.rootDir = newRoot
	return newRoot, nil
}

func (c *Chroot) ExecRunCommand(cmd *exec.Cmd) (err error) {
	if c.rootDir == "" {
		return errors.New("NewRoot() was not executed beforehand")
	}
	err = chroot.Run(cmd, c.rootDir)
	if err != nil {
		return fmt.Errorf("running command in chroot env: %w", err)
	}
	return nil
}

type None struct{}

func (n None) NewRoot() (newRoot string, err error) {
	return "/", nil
}

func (n None) ExecRunCommand(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "starting command")
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return errors.Wrap(err, "getting group id for process")
	}
	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "waiting for process to exit")
	}

	//it's not an error if there are no grandchildren
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && err.Error() != "no such process" {
		return err
	}
	return nil
}
