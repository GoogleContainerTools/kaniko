package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/wal/walpb"
	"github.com/docker/swarmkit/manager/encryption"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestBootstrapFromDisk(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "raft-storage")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	logger := EncryptedRaftLogger{
		StateDir:      tempdir,
		EncryptionKey: []byte("key1"),
	}
	err = logger.BootstrapNew([]byte("metadata"))
	require.NoError(t, err)

	// everything should be saved with "key1"
	_, entries, _ := makeWALData(0, 0)
	err = logger.SaveEntries(raftpb.HardState{}, entries)
	require.NoError(t, err)
	logger.Close(context.Background())

	// now we can bootstrap from disk, even if there is no snapshot
	logger = EncryptedRaftLogger{
		StateDir:      tempdir,
		EncryptionKey: []byte("key1"),
	}
	readSnap, waldata, err := logger.BootstrapFromDisk(context.Background())
	require.NoError(t, err)
	require.Nil(t, readSnap)
	require.Equal(t, entries, waldata.Entries)

	// save a snapshot
	snapshot := fakeSnapshotData
	err = logger.SaveSnapshot(snapshot)
	require.NoError(t, err)
	_, entries, _ = makeWALData(snapshot.Metadata.Index, snapshot.Metadata.Term)
	err = logger.SaveEntries(raftpb.HardState{}, entries)
	require.NoError(t, err)
	logger.Close(context.Background())

	// load snapshots and wals
	logger = EncryptedRaftLogger{
		StateDir:      tempdir,
		EncryptionKey: []byte("key1"),
	}
	readSnap, waldata, err = logger.BootstrapFromDisk(context.Background())
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, snapshot, *readSnap)
	require.Equal(t, entries, waldata.Entries)

	// start writing more wals and rotate in the middle
	_, entries, _ = makeWALData(snapshot.Metadata.Index, snapshot.Metadata.Term)
	err = logger.SaveEntries(raftpb.HardState{}, entries[:1])
	require.NoError(t, err)
	logger.RotateEncryptionKey([]byte("key2"))
	err = logger.SaveEntries(raftpb.HardState{}, entries[1:])
	require.NoError(t, err)
	logger.Close(context.Background())

	// we can't bootstrap from disk using only the first or second key
	for _, key := range []string{"key1", "key2"} {
		logger := EncryptedRaftLogger{
			StateDir:      tempdir,
			EncryptionKey: []byte(key),
		}
		_, _, err := logger.BootstrapFromDisk(context.Background())
		require.IsType(t, encryption.ErrCannotDecrypt{}, errors.Cause(err))
	}

	// but we can if we combine the two keys, we can bootstrap just fine
	logger = EncryptedRaftLogger{
		StateDir:      tempdir,
		EncryptionKey: []byte("key2"),
	}
	readSnap, waldata, err = logger.BootstrapFromDisk(context.Background(), []byte("key1"))
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, snapshot, *readSnap)
	require.Equal(t, entries, waldata.Entries)
}

// Ensure that we can change encoding and not have a race condition
func TestRaftLoggerRace(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "raft-storage")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	logger := EncryptedRaftLogger{
		StateDir:      tempdir,
		EncryptionKey: []byte("Hello"),
	}
	err = logger.BootstrapNew([]byte("metadata"))
	require.NoError(t, err)

	_, entries, _ := makeWALData(fakeSnapshotData.Metadata.Index, fakeSnapshotData.Metadata.Term)

	done1 := make(chan error)
	done2 := make(chan error)
	done3 := make(chan error)
	done4 := make(chan error)
	go func() {
		done1 <- logger.SaveSnapshot(fakeSnapshotData)
	}()
	go func() {
		done2 <- logger.SaveEntries(raftpb.HardState{}, entries)
	}()
	go func() {
		logger.RotateEncryptionKey([]byte("Hello 2"))
		done3 <- nil
	}()
	go func() {
		done4 <- logger.SaveSnapshot(fakeSnapshotData)
	}()

	err = <-done1
	require.NoError(t, err, "unable to save snapshot")

	err = <-done2
	require.NoError(t, err, "unable to save entries")

	err = <-done3
	require.NoError(t, err, "unable to rotate key")

	err = <-done4
	require.NoError(t, err, "unable to save snapshot a second time")
}

func TestMigrateToV3EncryptedForm(t *testing.T) {
	t.Parallel()

	tempdir, err := ioutil.TempDir("", "raft-storage")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	dek := []byte("key")

	writeDataTo := func(suffix string, snapshot raftpb.Snapshot, walFactory WALFactory, snapFactory SnapFactory) []raftpb.Entry {
		snapDir := filepath.Join(tempdir, "snap"+suffix)
		walDir := filepath.Join(tempdir, "wal"+suffix)
		require.NoError(t, os.MkdirAll(snapDir, 0755))
		require.NoError(t, snapFactory.New(snapDir).SaveSnap(snapshot))

		_, entries, _ := makeWALData(snapshot.Metadata.Index, snapshot.Metadata.Term)
		walWriter, err := walFactory.Create(walDir, []byte("metadata"))
		require.NoError(t, err)
		require.NoError(t, walWriter.SaveSnapshot(walpb.Snapshot{Index: snapshot.Metadata.Index, Term: snapshot.Metadata.Term}))
		require.NoError(t, walWriter.Save(raftpb.HardState{}, entries))
		require.NoError(t, walWriter.Close())
		return entries
	}

	requireLoadedData := func(expectedSnap raftpb.Snapshot, expectedEntries []raftpb.Entry) {
		logger := EncryptedRaftLogger{
			StateDir:      tempdir,
			EncryptionKey: dek,
		}
		readSnap, waldata, err := logger.BootstrapFromDisk(context.Background())
		require.NoError(t, err)
		require.NotNil(t, readSnap)
		require.Equal(t, expectedSnap, *readSnap)
		require.Equal(t, expectedEntries, waldata.Entries)
		logger.Close(context.Background())
	}

	v2Snapshot := fakeSnapshotData
	v3Snapshot := fakeSnapshotData
	v3Snapshot.Metadata.Index += 100
	v3Snapshot.Metadata.Term += 10
	v3EncryptedSnapshot := fakeSnapshotData
	v3EncryptedSnapshot.Metadata.Index += 200
	v3EncryptedSnapshot.Metadata.Term += 20

	encoder, decoders := encryption.Defaults(dek, false)
	walFactory := NewWALFactory(encoder, decoders)
	snapFactory := NewSnapFactory(encoder, decoders)

	// generate both v2 and v3 unencrypted snapshot data directories, as well as an encrypted directory
	v2Entries := writeDataTo("", v2Snapshot, OriginalWAL, OriginalSnap)
	v3Entries := writeDataTo("-v3", v3Snapshot, OriginalWAL, OriginalSnap)
	v3EncryptedEntries := writeDataTo("-v3-encrypted", v3EncryptedSnapshot, walFactory, snapFactory)

	// bootstrap from disk - the encrypted directory exists, so we should just read from
	// it instead of from the legacy directories
	requireLoadedData(v3EncryptedSnapshot, v3EncryptedEntries)

	// remove the newest dirs - should try to migrate from v3
	require.NoError(t, os.RemoveAll(filepath.Join(tempdir, "snap-v3-encrypted")))
	require.NoError(t, os.RemoveAll(filepath.Join(tempdir, "wal-v3-encrypted")))
	requireLoadedData(v3Snapshot, v3Entries)
	// it can recover from partial migrations
	require.NoError(t, os.RemoveAll(filepath.Join(tempdir, "snap-v3-encrypted")))
	requireLoadedData(v3Snapshot, v3Entries)
	// v3 dirs still there
	_, err = os.Stat(filepath.Join(tempdir, "snap-v3"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempdir, "wal-v3"))
	require.NoError(t, err)

	// remove the v3 dirs - should try to migrate from v2
	require.NoError(t, os.RemoveAll(filepath.Join(tempdir, "snap-v3-encrypted")))
	require.NoError(t, os.RemoveAll(filepath.Join(tempdir, "wal-v3-encrypted")))
	require.NoError(t, os.RemoveAll(filepath.Join(tempdir, "snap-v3")))
	require.NoError(t, os.RemoveAll(filepath.Join(tempdir, "wal-v3")))
	requireLoadedData(v2Snapshot, v2Entries)
}
