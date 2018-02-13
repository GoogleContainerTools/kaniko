package util

import (
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

// CreateFile creates a file at path with contents specified
func CreateFile(path string, contents []byte, chown string) error {
	// Create directory path if it doesn't exist
	baseDir := filepath.Dir(path)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		logrus.Debugf("baseDir %s for file %s does not exist. Creating.", baseDir, path)
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return err
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = f.Write(contents)
	// TODO: Figure out chown for ADD/COPY commands
	if chown != "" {
	}
	return err
}

// Files returns a list of all files that stem from root
func Files(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(path, "vendor") || strings.HasPrefix(path, ".") {
			return err
		}
		files = append(files, path)
		return err
	})
	return files, err
}

// IsDir checks if path is a directory
func IsDir(path string) (bool, error) {
	f, err := os.Stat(path)
	return f.IsDir(), err
}
