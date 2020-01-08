package transport

import (
	"io"
	"net"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/log"
	"github.com/docker/swarmkit/manager/health"
	"github.com/docker/swarmkit/manager/state/raft/membership"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type snapshotReport struct {
	id     uint64
	status raft.SnapshotStatus
}

type updateInfo struct {
	id   uint64
	addr string
}

type mockRaft struct {
	lis net.Listener
	s   *grpc.Server
	tr  *Transport

	nodeRemovedSignal chan struct{}

	removed map[uint64]bool

	processedMessages  chan *raftpb.Message
	processedSnapshots chan snapshotReport

	reportedUnreachables chan uint64
	updatedNodes         chan updateInfo

	forceErrorStream bool
}

func newMockRaft() (*mockRaft, error) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}
	mr := &mockRaft{
		lis:                  l,
		s:                    grpc.NewServer(),
		removed:              make(map[uint64]bool),
		nodeRemovedSignal:    make(chan struct{}),
		processedMessages:    make(chan *raftpb.Message, 4096),
		processedSnapshots:   make(chan snapshotReport, 4096),
		reportedUnreachables: make(chan uint64, 4096),
		updatedNodes:         make(chan updateInfo, 4096),
	}
	cfg := &Config{
		HeartbeatInterval: 3 * time.Second,
		SendTimeout:       2 * time.Second,
		Raft:              mr,
	}
	tr := New(cfg)
	mr.tr = tr
	hs := health.NewHealthServer()
	hs.SetServingStatus("Raft", api.HealthCheckResponse_SERVING)
	api.RegisterRaftServer(mr.s, mr)
	api.RegisterHealthServer(mr.s, hs)
	go mr.s.Serve(l)
	return mr, nil
}

func (r *mockRaft) Addr() string {
	return r.lis.Addr().String()
}

func (r *mockRaft) Stop() {
	r.tr.Stop()
	r.s.Stop()
}

func (r *mockRaft) RemovePeer(id uint64) error {
	r.removed[id] = true
	return r.tr.RemovePeer(id)
}

func (r *mockRaft) ProcessRaftMessage(ctx context.Context, req *api.ProcessRaftMessageRequest) (*api.ProcessRaftMessageResponse, error) {
	if r.removed[req.Message.From] {
		return nil, status.Errorf(codes.NotFound, "%s", membership.ErrMemberRemoved.Error())
	}
	r.processedMessages <- req.Message
	return &api.ProcessRaftMessageResponse{}, nil
}

// StreamRaftMessage is the mock server endpoint for streaming messages of type StreamRaftMessageRequest.
func (r *mockRaft) StreamRaftMessage(stream api.Raft_StreamRaftMessageServer) error {
	if r.forceErrorStream {
		return grpc.Errorf(codes.Unimplemented, "streaming not supported")
	}
	var recvdMsg, assembledMessage *api.StreamRaftMessageRequest
	var err error
	for {
		recvdMsg, err = stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			log.G(context.Background()).WithError(err).Error("error while reading from stream")
			return err
		}

		if r.removed[recvdMsg.Message.From] {
			return status.Errorf(codes.NotFound, "%s", membership.ErrMemberRemoved.Error())
		}

		if assembledMessage == nil {
			assembledMessage = recvdMsg
			continue
		}

		// For all message types except raftpb.MsgSnap,
		// we don't expect more than a single message
		// on the stream.
		if recvdMsg.Message.Type != raftpb.MsgSnap {
			panic("Unexpected message type received on stream: " + string(recvdMsg.Message.Type))
		}

		// Append received snapshot chunk to the chunk that was already received.
		assembledMessage.Message.Snapshot.Data = append(assembledMessage.Message.Snapshot.Data, recvdMsg.Message.Snapshot.Data...)
	}

	// We should have the complete snapshot. Verify and process.
	if err == io.EOF {
		if assembledMessage.Message.Type == raftpb.MsgSnap {
			if !verifySnapshot(assembledMessage.Message) {
				log.G(context.Background()).Error("snapshot data mismatch")
				panic("invalid snapshot data")
			}
		}

		r.processedMessages <- assembledMessage.Message

		return stream.SendAndClose(&api.StreamRaftMessageResponse{})
	}

	return nil
}

func (r *mockRaft) ResolveAddress(ctx context.Context, req *api.ResolveAddressRequest) (*api.ResolveAddressResponse, error) {
	addr, err := r.tr.PeerAddr(req.RaftID)
	if err != nil {
		return nil, err
	}
	return &api.ResolveAddressResponse{
		Addr: addr,
	}, nil
}

func (r *mockRaft) ReportUnreachable(id uint64) {
	r.reportedUnreachables <- id
}

func (r *mockRaft) IsIDRemoved(id uint64) bool {
	return r.removed[id]
}

func (r *mockRaft) ReportSnapshot(id uint64, status raft.SnapshotStatus) {
	r.processedSnapshots <- snapshotReport{
		id:     id,
		status: status,
	}
}

func (r *mockRaft) UpdateNode(id uint64, addr string) {
	r.updatedNodes <- updateInfo{
		id:   id,
		addr: addr,
	}
}

func (r *mockRaft) NodeRemoved() {
	close(r.nodeRemovedSignal)
}

type mockCluster struct {
	rafts map[uint64]*mockRaft
}

func newCluster() *mockCluster {
	return &mockCluster{
		rafts: make(map[uint64]*mockRaft),
	}
}

func (c *mockCluster) Stop() {
	for _, r := range c.rafts {
		r.s.Stop()
	}
}

func (c *mockCluster) Add(id uint64) error {
	mr, err := newMockRaft()
	if err != nil {
		return err
	}
	for otherID, otherRaft := range c.rafts {
		if err := mr.tr.AddPeer(otherID, otherRaft.Addr()); err != nil {
			return err
		}
		if err := otherRaft.tr.AddPeer(id, mr.Addr()); err != nil {
			return err
		}
	}
	c.rafts[id] = mr
	return nil
}

func (c *mockCluster) Get(id uint64) *mockRaft {
	return c.rafts[id]
}
