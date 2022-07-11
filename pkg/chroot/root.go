package chroot

import (
	"fmt"
	"os"
)

func TmpDirInHome() (string, error) {
  home, err := os.UserHomeDir()
  if err != nil {
    return "", fmt.Errorf("getting homeDir: %w", err)
	}
	tmpDir, err := os.MkdirTemp(home, "*")
	if err != nil {
		return "", err
	}
	return tmpDir, nil
}
