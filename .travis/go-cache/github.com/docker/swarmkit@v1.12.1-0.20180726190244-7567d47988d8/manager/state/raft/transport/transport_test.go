package transport

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Build a snapshot message where each byte in the data is of the value (index % sizeof(byte))
func newSnapshotMessage(from uint64, to uint64) raftpb.Message {
	data := make([]byte, GRPCMaxMsgSize)
	for i := 0; i < GRPCMaxMsgSize; i++ {
		data[i] = byte(i % (1 << 8))
	}

	return raftpb.Message{
		Type: raftpb.MsgSnap,
		From: from,
		To:   to,
		Snapshot: raftpb.Snapshot{
			Data: data,
			// Include the snapshot size in the Index field for testing.
			Metadata: raftpb.SnapshotMetadata{
				Index: uint64(len(data)),
			},
		},
	}
}

// Verify that the snapshot data where each byte is of the value (index % sizeof(byte)).
func verifySnapshot(raftMsg *raftpb.Message) bool {
	for i, b := range raftMsg.Snapshot.Data {
		if int(b) != i%(1<<8) {
			return false
		}
	}

	return len(raftMsg.Snapshot.Data) == int(raftMsg.Snapshot.Metadata.Index)
}

func sendMessages(ctx context.Context, c *mockCluster, from uint64, to []uint64, msgType raftpb.MessageType) error {
	var firstErr error
	for _, id := range to {
		var err error
		if msgType == raftpb.MsgSnap {
			err = c.Get(from).tr.Send(newSnapshotMessage(from, id))
		} else {
			err = c.Get(from).tr.Send(raftpb.Message{
				Type: msgType,
				From: from,
				To:   id,
			})
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func testSend(ctx context.Context, c *mockCluster, from uint64, to []uint64, msgType raftpb.MessageType) func(*testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
		require.NoError(t, sendMessages(ctx, c, from, to, msgType))

		for _, id := range to {
			select {
			case msg := <-c.Get(id).processedMessages:
				assert.Equal(t, msg.To, id)
				assert.Equal(t, msg.From, from)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}

		if msgType == raftpb.MsgSnap {
			var snaps []snapshotReport
			for i := 0; i < len(to); i++ {
				select {
				case snap := <-c.Get(from).processedSnapshots:
					snaps = append(snaps, snap)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
		loop:
			for _, id := range to {
				for _, s := range snaps {
					if s.id == id {
						assert.Equal(t, s.status, raft.SnapshotFinish)
						continue loop
					}
				}
				t.Fatalf("snapshot id %d is not reported", id)
			}
		}
	}
}

func TestSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newCluster()
	defer func() {
		cancel()
		c.Stop()
	}()
	require.NoError(t, c.Add(1))
	require.NoError(t, c.Add(2))
	require.NoError(t, c.Add(3))

	t.Run("Send Message", testSend(ctx, c, 1, []uint64{2, 3}, raftpb.MsgHup))
	t.Run("Send_Snapshot_Message", testSend(ctx, c, 1, []uint64{2, 3}, raftpb.MsgSnap))

	// Return error on streaming.
	for _, raft := range c.rafts {
		raft.forceErrorStream = true
	}

	// Messages should still be delivered.
	t.Run("Send Message", testSend(ctx, c, 1, []uint64{2, 3}, raftpb.MsgHup))
}

func TestSendRemoved(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newCluster()
	defer func() {
		cancel()
		c.Stop()
	}()
	require.NoError(t, c.Add(1))
	require.NoError(t, c.Add(2))
	require.NoError(t, c.Add(3))
	require.NoError(t, c.Get(1).RemovePeer(2))

	err := sendMessages(ctx, c, 1, []uint64{2, 3}, raftpb.MsgHup)
	require.Error(t, err)
	require.Contains(t, err.Error(), "to removed member")
}

func TestSendSnapshotFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newCluster()
	defer func() {
		cancel()
		c.Stop()
	}()
	require.NoError(t, c.Add(1))
	require.NoError(t, c.Add(2))

	// stop peer server to emulate error
	c.Get(2).s.Stop()

	msgCtx, msgCancel := context.WithTimeout(ctx, 4*time.Second)
	defer msgCancel()

	require.NoError(t, sendMessages(msgCtx, c, 1, []uint64{2}, raftpb.MsgSnap))

	select {
	case snap := <-c.Get(1).processedSnapshots:
		assert.Equal(t, snap.id, uint64(2))
		assert.Equal(t, snap.status, raft.SnapshotFailure)
	case <-msgCtx.Done():
		t.Fatal(ctx.Err())
	}

	select {
	case id := <-c.Get(1).reportedUnreachables:
		assert.Equal(t, id, uint64(2))
	case <-msgCtx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestSendUnknown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newCluster()
	defer func() {
		cancel()
		c.Stop()
	}()
	require.NoError(t, c.Add(1))
	require.NoError(t, c.Add(2))
	require.NoError(t, c.Add(3))

	// remove peer from 1 transport to make it "unknown" to it
	oldPeer := c.Get(1).tr.peers[2]
	delete(c.Get(1).tr.peers, 2)
	oldPeer.cancel()
	<-oldPeer.done

	// give peers time to mark each other as active
	time.Sleep(1 * time.Second)

	msgCtx, msgCancel := context.WithTimeout(ctx, 4*time.Second)
	defer msgCancel()

	require.NoError(t, sendMessages(msgCtx, c, 1, []uint64{2}, raftpb.MsgHup))

	select {
	case msg := <-c.Get(2).processedMessages:
		assert.Equal(t, msg.To, uint64(2))
		assert.Equal(t, msg.From, uint64(1))
	case <-msgCtx.Done():
		t.Fatal(msgCtx.Err())
	}
}

func TestUpdatePeerAddr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newCluster()
	defer func() {
		cancel()
		c.Stop()
	}()
	require.NoError(t, c.Add(1))
	require.NoError(t, c.Add(2))
	require.NoError(t, c.Add(3))

	t.Run("Send Message Before Address Update", testSend(ctx, c, 1, []uint64{2, 3}, raftpb.MsgHup))

	nr, err := newMockRaft()
	require.NoError(t, err)

	c.Get(3).Stop()
	c.rafts[3] = nr

	require.NoError(t, c.Get(1).tr.UpdatePeer(3, nr.Addr()))
	require.NoError(t, c.Get(1).tr.UpdatePeer(3, nr.Addr()))

	t.Run("Send Message After Address Update", testSend(ctx, c, 1, []uint64{2, 3}, raftpb.MsgHup))
}

func TestUpdatePeerAddrDelayed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newCluster()
	defer func() {
		cancel()
		c.Stop()
	}()
	require.NoError(t, c.Add(1))
	require.NoError(t, c.Add(2))
	require.NoError(t, c.Add(3))

	t.Run("Send Message Before Address Update", testSend(ctx, c, 1, []uint64{2, 3}, raftpb.MsgHup))

	nr, err := newMockRaft()
	require.NoError(t, err)

	c.Get(3).Stop()
	c.rafts[3] = nr

	require.NoError(t, c.Get(1).tr.UpdatePeerAddr(3, nr.Addr()))

	// initiate failure to replace connection, and wait for it
	sendMessages(ctx, c, 1, []uint64{3}, raftpb.MsgHup)
	updateCtx, updateCancel := context.WithTimeout(ctx, 4*time.Second)
	defer updateCancel()
	select {
	case update := <-c.Get(1).updatedNodes:
		require.Equal(t, update.id, uint64(3))
		require.Equal(t, update.addr, nr.Addr())
	case <-updateCtx.Done():
		t.Fatal(updateCtx.Err())
	}

	t.Run("Send Message After Address Update", testSend(ctx, c, 1, []uint64{2, 3}, raftpb.MsgHup))
}

func TestSendUnreachable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newCluster()
	defer func() {
		cancel()
		c.Stop()
	}()
	require.NoError(t, c.Add(1))
	require.NoError(t, c.Add(2))

	// set channel to nil to emulate full queue
	// we need to reset some fields after cancel
	p2 := c.Get(1).tr.peers[2]
	p2.cancel()
	<-p2.done
	p2.msgc = nil
	p2.done = make(chan struct{})
	p2.ctx = ctx
	go p2.run(ctx)

	msgCtx, msgCancel := context.WithTimeout(ctx, 4*time.Second)
	defer msgCancel()

	err := sendMessages(msgCtx, c, 1, []uint64{2}, raftpb.MsgSnap)
	require.Error(t, err)
	require.Contains(t, err.Error(), "peer is unreachable")
	select {
	case id := <-c.Get(1).reportedUnreachables:
		assert.Equal(t, id, uint64(2))
	case <-msgCtx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestSendNodeRemoved(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newCluster()
	defer func() {
		cancel()
		c.Stop()
	}()
	require.NoError(t, c.Add(1))
	require.NoError(t, c.Add(2))

	require.NoError(t, c.Get(1).RemovePeer(2))

	msgCtx, msgCancel := context.WithTimeout(ctx, 4*time.Second)
	defer msgCancel()

	require.NoError(t, sendMessages(msgCtx, c, 2, []uint64{1}, raftpb.MsgSnap))
	select {
	case <-c.Get(2).nodeRemovedSignal:
	case <-msgCtx.Done():
		t.Fatal(msgCtx.Err())
	}
}
