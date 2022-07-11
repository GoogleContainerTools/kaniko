package chroot

import (
	"errors"
	"os"
)

func TmpDirInHome() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("HOME environment variable is not set, needed for chroot")
	}
	tmpDir, err := os.MkdirTemp(home, "*")
	if err != nil {
		return "", err
	}
	return tmpDir, nil
}
