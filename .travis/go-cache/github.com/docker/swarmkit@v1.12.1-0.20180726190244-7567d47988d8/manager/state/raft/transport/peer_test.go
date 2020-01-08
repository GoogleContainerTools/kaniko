package transport

import (
	"math"
	"testing"

	"golang.org/x/net/context"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/stretchr/testify/assert"
)

// Test SplitSnapshot() for different snapshot sizes.
func TestSplitSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var raftMsg raftpb.Message
	raftMsg.Type = raftpb.MsgSnap
	snaphotSize := 8 << 20
	raftMsg.Snapshot.Data = make([]byte, snaphotSize)

	raftMessagePayloadSize := raftMessagePayloadSize(&raftMsg)

	check := func(size, expectedNumMsgs int) {
		raftMsg.Snapshot.Data = make([]byte, size)
		msgs := splitSnapshotData(ctx, &raftMsg)
		assert.Equal(t, expectedNumMsgs, len(msgs), "unexpected number of messages")
	}

	check(snaphotSize, int(math.Ceil(float64(snaphotSize)/float64(raftMessagePayloadSize))))
	check(raftMessagePayloadSize, 1)
	check(raftMessagePayloadSize-1, 1)
	check(raftMessagePayloadSize*2, 2)
	check(0, 0)

	raftMsg.Type = raftpb.MsgApp
	check(0, 0)
}
