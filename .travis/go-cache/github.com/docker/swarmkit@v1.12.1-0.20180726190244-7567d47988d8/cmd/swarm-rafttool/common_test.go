package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/wal/walpb"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/manager"
	"github.com/docker/swarmkit/manager/encryption"
	"github.com/docker/swarmkit/manager/state/raft"
	"github.com/docker/swarmkit/manager/state/raft/storage"
	"github.com/stretchr/testify/require"
)

// writeFakeRaftData writes the given snapshot and some generated WAL data to given "snap" and "wal" directories
func writeFakeRaftData(t *testing.T, stateDir string, snapshot *raftpb.Snapshot, wf storage.WALFactory, sf storage.SnapFactory) {
	snapDir := filepath.Join(stateDir, "raft", "snap-v3-encrypted")
	walDir := filepath.Join(stateDir, "raft", "wal-v3-encrypted")
	require.NoError(t, os.MkdirAll(snapDir, 0755))

	wsn := walpb.Snapshot{}
	if snapshot != nil {
		require.NoError(t, sf.New(snapDir).SaveSnap(*snapshot))

		wsn.Index = snapshot.Metadata.Index
		wsn.Term = snapshot.Metadata.Term
	}

	var entries []raftpb.Entry
	for i := wsn.Index + 1; i < wsn.Index+6; i++ {
		entries = append(entries, raftpb.Entry{
			Term:  wsn.Term + 1,
			Index: i,
			Data:  []byte(fmt.Sprintf("v3Entry %d", i)),
		})
	}

	walWriter, err := wf.Create(walDir, []byte("v3metadata"))
	require.NoError(t, err)
	require.NoError(t, walWriter.SaveSnapshot(wsn))
	require.NoError(t, walWriter.Save(raftpb.HardState{}, entries))
	require.NoError(t, walWriter.Close())
}

func TestDecrypt(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "rafttool")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	kek := []byte("kek")
	dek := []byte("dek")
	unlockKey := encryption.HumanReadableKey(kek)

	// write a key to disk, else we won't be able to decrypt anything
	paths := certPaths(tempdir)
	krw := ca.NewKeyReadWriter(paths.Node, kek,
		manager.RaftDEKData{EncryptionKeys: raft.EncryptionKeys{CurrentDEK: dek}})
	cert, key, err := testutils.CreateRootCertAndKey("not really a root, just need cert and key")
	require.NoError(t, err)
	require.NoError(t, krw.Write(cert, key, nil))

	// create the encrypted v3 directory
	origSnapshot := raftpb.Snapshot{
		Data: []byte("snapshot"),
		Metadata: raftpb.SnapshotMetadata{
			Index: 1,
			Term:  1,
		},
	}
	e, d := encryption.Defaults(dek, false)
	writeFakeRaftData(t, tempdir, &origSnapshot, storage.NewWALFactory(e, d), storage.NewSnapFactory(e, d))

	outdir := filepath.Join(tempdir, "outdir")
	// if we use the wrong unlock key, we can't actually decrypt anything.  The output directory won't get created.
	err = decryptRaftData(tempdir, outdir, "")
	require.IsType(t, ca.ErrInvalidKEK{}, err)
	require.False(t, fileutil.Exist(outdir))

	// Using the right unlock key, we produce data that is unencrypted
	require.NoError(t, decryptRaftData(tempdir, outdir, unlockKey))
	require.True(t, fileutil.Exist(outdir))

	// The snapshot directory is readable by the regular snapshotter
	snapshot, err := storage.OriginalSnap.New(filepath.Join(outdir, "snap-decrypted")).Load()
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, origSnapshot, *snapshot)

	// The wals are readable by the regular wal
	walreader, err := storage.OriginalWAL.Open(filepath.Join(outdir, "wal-decrypted"), walpb.Snapshot{Index: 1, Term: 1})
	require.NoError(t, err)
	metadata, _, entries, err := walreader.ReadAll()
	require.NoError(t, err)
	require.Equal(t, []byte("v3metadata"), metadata)
	require.Len(t, entries, 5)
}
