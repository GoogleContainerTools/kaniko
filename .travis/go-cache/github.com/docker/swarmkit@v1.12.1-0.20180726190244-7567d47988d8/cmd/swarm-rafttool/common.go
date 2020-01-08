package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/net/context"

	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/coreos/etcd/wal/walpb"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/manager"
	"github.com/docker/swarmkit/manager/encryption"
	"github.com/docker/swarmkit/manager/state/raft/storage"
	"github.com/docker/swarmkit/node"
)

func certPaths(swarmdir string) *ca.SecurityConfigPaths {
	return ca.NewConfigPaths(filepath.Join(swarmdir, "certificates"))
}

func getDEKData(krw *ca.KeyReadWriter) (manager.RaftDEKData, error) {
	h, _ := krw.GetCurrentState()
	dekData, ok := h.(manager.RaftDEKData)
	if !ok {
		return manager.RaftDEKData{}, errors.New("cannot read raft dek headers in TLS key")
	}

	if dekData.CurrentDEK == nil {
		return manager.RaftDEKData{}, errors.New("no raft DEKs available")
	}

	return dekData, nil
}

func getKRW(swarmdir, unlockKey string) (*ca.KeyReadWriter, error) {
	var (
		kek []byte
		err error
	)
	if unlockKey != "" {
		kek, err = encryption.ParseHumanReadableKey(unlockKey)
		if err != nil {
			return nil, err
		}
	}
	krw := ca.NewKeyReadWriter(certPaths(swarmdir).Node, kek, manager.RaftDEKData{})
	_, _, err = krw.Read() // loads all the key data into the KRW object
	if err != nil {
		return nil, err
	}
	return krw, nil
}

func moveDirAside(dirname string) error {
	if fileutil.Exist(dirname) {
		tempdir, err := ioutil.TempDir(filepath.Dir(dirname), filepath.Base(dirname))
		if err != nil {
			return err
		}
		return os.Rename(dirname, tempdir)
	}
	return nil
}

func decryptRaftData(swarmdir, outdir, unlockKey string) error {
	krw, err := getKRW(swarmdir, unlockKey)
	if err != nil {
		return err
	}
	deks, err := getDEKData(krw)
	if err != nil {
		return err
	}

	// always use false for FIPS, since we want to be able to decrypt logs written using
	// any algorithm (not just FIPS-compatible ones)
	_, d := encryption.Defaults(deks.CurrentDEK, false)
	if deks.PendingDEK == nil {
		_, d2 := encryption.Defaults(deks.PendingDEK, false)
		d = encryption.NewMultiDecrypter(d, d2)
	}

	snapDir := filepath.Join(outdir, "snap-decrypted")
	if err := moveDirAside(snapDir); err != nil {
		return err
	}
	if err := storage.MigrateSnapshot(
		filepath.Join(swarmdir, "raft", "snap-v3-encrypted"), snapDir,
		storage.NewSnapFactory(encryption.NoopCrypter, d), storage.OriginalSnap); err != nil {
		return err
	}

	var walsnap walpb.Snapshot
	snap, err := storage.OriginalSnap.New(snapDir).Load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if snap != nil {
		walsnap.Index = snap.Metadata.Index
		walsnap.Term = snap.Metadata.Term
	}

	walDir := filepath.Join(outdir, "wal-decrypted")
	if err := moveDirAside(walDir); err != nil {
		return err
	}
	return storage.MigrateWALs(context.Background(),
		filepath.Join(swarmdir, "raft", "wal-v3-encrypted"), walDir,
		storage.NewWALFactory(encryption.NoopCrypter, d), storage.OriginalWAL, walsnap)
}

func downgradeKey(swarmdir, unlockKey string) error {
	var (
		kek []byte
		err error
	)
	if unlockKey != "" {
		kek, err = encryption.ParseHumanReadableKey(unlockKey)
		if err != nil {
			return err
		}
	}

	n, err := node.New(&node.Config{
		StateDir:  swarmdir,
		UnlockKey: kek,
	})
	if err != nil {
		return err
	}

	return n.DowngradeKey()
}
