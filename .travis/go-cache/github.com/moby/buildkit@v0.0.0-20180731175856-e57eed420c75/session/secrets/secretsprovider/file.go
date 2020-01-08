package secretsprovider

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/moby/buildkit/session/secrets"
	"github.com/pkg/errors"
)

type FileSource struct {
	ID       string
	FilePath string
}

func NewFileStore(files []FileSource) (secrets.SecretStore, error) {
	m := map[string]FileSource{}
	for _, f := range files {
		if f.ID == "" {
			return nil, errors.Errorf("secret missing ID")
		}
		if f.FilePath == "" {
			f.FilePath = f.ID
		}
		fi, err := os.Stat(f.FilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to stat %s", f.FilePath)
		}
		if fi.Size() > MaxSecretSize {
			return nil, errors.Errorf("secret %s too big. max size 500KB", f.ID)
		}
		m[f.ID] = f
	}
	return &fileStore{
		m: m,
	}, nil
}

type fileStore struct {
	m map[string]FileSource
}

func (fs *fileStore) GetSecret(ctx context.Context, id string) ([]byte, error) {
	v, ok := fs.m[id]
	if !ok {
		return nil, errors.WithStack(secrets.ErrNotFound)
	}
	dt, err := ioutil.ReadFile(v.FilePath)
	if err != nil {
		return nil, err
	}
	return dt, nil
}
