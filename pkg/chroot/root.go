package chroot

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func TmpDirInHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting homeDir: %w", err)
	}
	id := uuid.New()
	tmpDir := filepath.Join(home, id.String())
	err = os.Mkdir(tmpDir, 0755)
	if err != nil {
		return "", err
	}
	return tmpDir, nil
}
