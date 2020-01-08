package boltdbcachestorage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/moby/buildkit/solver"
	"github.com/moby/buildkit/solver/testutil"
	"github.com/stretchr/testify/require"
)

func TestBoltCacheStorage(t *testing.T) {
	testutil.RunCacheStorageTests(t, func() (solver.CacheKeyStorage, func()) {
		tmpDir, err := ioutil.TempDir("", "storage")
		require.NoError(t, err)

		cleanup := func() {
			os.RemoveAll(tmpDir)
		}

		st, err := NewStore(filepath.Join(tmpDir, "cache.db"))
		if err != nil {
			cleanup()
		}
		require.NoError(t, err)

		return st, cleanup
	})
}
