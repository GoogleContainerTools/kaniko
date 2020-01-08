package testutils

import (
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	etcdraft "github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/manager/health"
	"github.com/docker/swarmkit/manager/state/raft"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/testutils"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNode represents a raft test node
type TestNode struct {
	*raft.Node
	Server         *grpc.Server
	Listener       *WrappedListener
	SecurityConfig *ca.SecurityConfig
	Address        string
	StateDir       string
	cancel         context.CancelFunc
	KeyRotator     *SimpleKeyRotator
}

// Leader is wrapper around real Leader method to suppress error.
// TODO: tests should use Leader method directly.
func (n *TestNode) Leader() uint64 {
	id, _ := n.Node.Leader()
	return id
}

// AdvanceTicks advances the raft state machine fake clock
func AdvanceTicks(clockSource *fakeclock.FakeClock, ticks int) {
	// A FakeClock timer won't fire multiple times if time is advanced
	// more than its interval.
	for i := 0; i != ticks; i++ {
		clockSource.Increment(time.Second)
	}
}

// WaitForCluster waits until leader will be one of specified nodes
func WaitForCluster(t *testing.T, clockSource *fakeclock.FakeClock, nodes map[uint64]*TestNode) {
	err := testutils.PollFunc(clockSource, func() error {
		var prev *etcdraft.Status
	nodeLoop:
		for _, n := range nodes {
			if prev == nil {
				prev = new(etcdraft.Status)
				*prev = n.Status()
				for _, n2 := range nodes {
					if n2.Config.ID == prev.Lead && n2.ReadyForProposals() {
						continue nodeLoop
					}
				}
				return errors.New("did not find a ready leader in member list")
			}
			cur := n.Status()

			for _, n2 := range nodes {
				if n2.Config.ID == cur.Lead {
					if cur.Lead != prev.Lead || cur.Term != prev.Term || cur.Applied != prev.Applied {
						return errors.New("state does not match on all nodes")
					}
					continue nodeLoop
				}
			}
			return errors.New("did not find leader in member list")
		}
		return nil
	})
	require.NoError(t, err)
}

// WaitForPeerNumber waits until peers in cluster converge to specified number
func WaitForPeerNumber(t *testing.T, clockSource *fakeclock.FakeClock, nodes map[uint64]*TestNode, count int) {
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		for _, n := range nodes {
			if len(n.GetMemberlist()) != count {
				return errors.New("unexpected number of members")
			}
		}
		return nil
	}))
}

// WrappedListener disables the Close method to make it possible to reuse a
// socket. close must be called to release the socket.
type WrappedListener struct {
	net.Listener
	acceptConn chan net.Conn
	acceptErr  chan error
	closed     chan struct{}
}

// NewWrappedListener creates a new wrapped listener to register the raft server
func NewWrappedListener(l net.Listener) *WrappedListener {
	wrappedListener := WrappedListener{
		Listener:   l,
		acceptConn: make(chan net.Conn, 10),
		acceptErr:  make(chan error, 1),
		closed:     make(chan struct{}, 10), // grpc closes multiple times
	}
	// Accept connections
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				wrappedListener.acceptErr <- err
				return
			}
			wrappedListener.acceptConn <- conn
		}
	}()

	return &wrappedListener
}

// Accept accepts new connections on a wrapped listener
func (l *WrappedListener) Accept() (net.Conn, error) {
	// closure must take precedence over taking a connection
	// from the channel
	select {
	case <-l.closed:
		return nil, errors.New("listener closed")
	default:
	}

	select {
	case conn := <-l.acceptConn:
		return conn, nil
	case err := <-l.acceptErr:
		return nil, err
	case <-l.closed:
		return nil, errors.New("listener closed")
	}
}

// Close notifies that the listener can't accept any more connections
func (l *WrappedListener) Close() error {
	l.closed <- struct{}{}
	return nil
}

// CloseListener closes the underlying listener
func (l *WrappedListener) CloseListener() error {
	return l.Listener.Close()
}

// RecycleWrappedListener creates a new wrappedListener that uses the same
// listening socket as the supplied wrappedListener.
func RecycleWrappedListener(old *WrappedListener) *WrappedListener {
	return &WrappedListener{
		Listener:   old.Listener,
		acceptConn: old.acceptConn,
		acceptErr:  old.acceptErr,
		closed:     make(chan struct{}, 10), // grpc closes multiple times
	}
}

// SimpleKeyRotator does some DEK rotation
type SimpleKeyRotator struct {
	mu                 sync.Mutex
	rotateCh           chan struct{}
	updateFunc         func() error
	overrideNeedRotate *bool
	raft.EncryptionKeys
}

// GetKeys returns the current set of keys
func (s *SimpleKeyRotator) GetKeys() raft.EncryptionKeys {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.EncryptionKeys
}

// NeedsRotation returns whether we need to rotate
func (s *SimpleKeyRotator) NeedsRotation() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.overrideNeedRotate != nil {
		return *s.overrideNeedRotate
	}
	return s.EncryptionKeys.PendingDEK != nil
}

// UpdateKeys updates the current encryption keys
func (s *SimpleKeyRotator) UpdateKeys(newKeys raft.EncryptionKeys) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updateFunc != nil {
		return s.updateFunc()
	}
	s.EncryptionKeys = newKeys
	return nil
}

// RotationNotify returns the rotation notification channel
func (s *SimpleKeyRotator) RotationNotify() chan struct{} {
	return s.rotateCh
}

// QueuePendingKey lets us rotate the key
func (s *SimpleKeyRotator) QueuePendingKey(key []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EncryptionKeys.PendingDEK = key
}

// SetUpdateFunc enables you to inject an error when updating keys
func (s *SimpleKeyRotator) SetUpdateFunc(updateFunc func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateFunc = updateFunc
}

// SetNeedsRotation enables you to inject a value for NeedsRotation
func (s *SimpleKeyRotator) SetNeedsRotation(override *bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overrideNeedRotate = override
}

// NewSimpleKeyRotator returns a basic EncryptionKeyRotator
func NewSimpleKeyRotator(keys raft.EncryptionKeys) *SimpleKeyRotator {
	return &SimpleKeyRotator{
		rotateCh:       make(chan struct{}),
		EncryptionKeys: keys,
	}
}

var _ raft.EncryptionKeyRotator = NewSimpleKeyRotator(raft.EncryptionKeys{})

// NewNode creates a new raft node to use for tests
func NewNode(t *testing.T, clockSource *fakeclock.FakeClock, tc *cautils.TestCA, opts ...raft.NodeOptions) *TestNode {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "can't bind to raft service port")
	wrappedListener := NewWrappedListener(l)

	securityConfig, err := tc.NewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)

	serverOpts := []grpc.ServerOption{grpc.Creds(securityConfig.ServerTLSCreds)}
	s := grpc.NewServer(serverOpts...)

	cfg := raft.DefaultNodeConfig()

	stateDir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err, "can't create temporary state directory")

	keyRotator := NewSimpleKeyRotator(raft.EncryptionKeys{CurrentDEK: []byte("current")})
	newNodeOpts := raft.NodeOptions{
		ID:             securityConfig.ClientTLSCreds.NodeID(),
		Addr:           l.Addr().String(),
		Config:         cfg,
		StateDir:       stateDir,
		ClockSource:    clockSource,
		TLSCredentials: securityConfig.ClientTLSCreds,
		KeyRotator:     keyRotator,
	}

	if len(opts) > 1 {
		panic("more than one optional argument provided")
	}
	if len(opts) == 1 {
		newNodeOpts.JoinAddr = opts[0].JoinAddr
		if opts[0].Addr != "" {
			newNodeOpts.Addr = opts[0].Addr
		}
		newNodeOpts.DisableStackDump = opts[0].DisableStackDump
	}

	n := raft.NewNode(newNodeOpts)

	healthServer := health.NewHealthServer()
	api.RegisterHealthServer(s, healthServer)
	raft.Register(s, n)

	go s.Serve(wrappedListener)

	healthServer.SetServingStatus("Raft", api.HealthCheckResponse_SERVING)

	return &TestNode{
		Node:           n,
		Listener:       wrappedListener,
		SecurityConfig: securityConfig,
		Address:        newNodeOpts.Addr,
		StateDir:       newNodeOpts.StateDir,
		Server:         s,
		KeyRotator:     keyRotator,
	}
}

// NewInitNode creates a new raft node initiating the cluster
// for other members to join
func NewInitNode(t *testing.T, tc *cautils.TestCA, raftConfig *api.RaftConfig, opts ...raft.NodeOptions) (*TestNode, *fakeclock.FakeClock) {
	clockSource := fakeclock.NewFakeClock(time.Now())
	n := NewNode(t, clockSource, tc, opts...)
	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	err := n.Node.JoinAndStart(ctx)
	require.NoError(t, err, "can't join cluster")

	leadershipCh, cancel := n.SubscribeLeadership()
	defer cancel()

	go n.Run(ctx)

	// Wait for the node to become the leader.
	<-leadershipCh

	if raftConfig != nil {
		assert.NoError(t, n.MemoryStore().Update(func(tx store.Tx) error {
			return store.CreateCluster(tx, &api.Cluster{
				ID: identity.NewID(),
				Spec: api.ClusterSpec{
					Annotations: api.Annotations{
						Name: store.DefaultClusterName,
					},
					Raft: *raftConfig,
				},
			})
		}))
	}

	return n, clockSource
}

// NewJoinNode creates a new raft node joining an existing cluster
func NewJoinNode(t *testing.T, clockSource *fakeclock.FakeClock, join string, tc *cautils.TestCA, opts ...raft.NodeOptions) *TestNode {
	var derivedOpts raft.NodeOptions
	if len(opts) == 1 {
		derivedOpts = opts[0]
	}
	derivedOpts.JoinAddr = join
	n := NewNode(t, clockSource, tc, derivedOpts)

	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel
	err := n.Node.JoinAndStart(ctx)
	require.NoError(t, err, "can't join cluster")

	go n.Run(ctx)

	return n
}

// CopyNode returns a copy of a node
func CopyNode(t *testing.T, clockSource *fakeclock.FakeClock, oldNode *TestNode, forceNewCluster bool, kr *SimpleKeyRotator) (*TestNode, context.Context) {
	wrappedListener := RecycleWrappedListener(oldNode.Listener)
	securityConfig := oldNode.SecurityConfig
	serverOpts := []grpc.ServerOption{grpc.Creds(securityConfig.ServerTLSCreds)}
	s := grpc.NewServer(serverOpts...)

	cfg := raft.DefaultNodeConfig()

	if kr == nil {
		kr = oldNode.KeyRotator
	}

	newNodeOpts := raft.NodeOptions{
		ID:              securityConfig.ClientTLSCreds.NodeID(),
		Addr:            oldNode.Address,
		Config:          cfg,
		StateDir:        oldNode.StateDir,
		ForceNewCluster: forceNewCluster,
		ClockSource:     clockSource,
		SendTimeout:     2 * time.Second,
		TLSCredentials:  securityConfig.ClientTLSCreds,
		KeyRotator:      kr,
	}

	ctx, cancel := context.WithCancel(context.Background())
	n := raft.NewNode(newNodeOpts)

	healthServer := health.NewHealthServer()
	api.RegisterHealthServer(s, healthServer)
	raft.Register(s, n)

	go s.Serve(wrappedListener)

	healthServer.SetServingStatus("Raft", api.HealthCheckResponse_SERVING)

	return &TestNode{
		Node:           n,
		Listener:       wrappedListener,
		SecurityConfig: securityConfig,
		Address:        newNodeOpts.Addr,
		StateDir:       newNodeOpts.StateDir,
		cancel:         cancel,
		Server:         s,
		KeyRotator:     kr,
	}, ctx
}

// RestartNode restarts a raft test node
func RestartNode(t *testing.T, clockSource *fakeclock.FakeClock, oldNode *TestNode, forceNewCluster bool) *TestNode {
	n, ctx := CopyNode(t, clockSource, oldNode, forceNewCluster, nil)

	err := n.Node.JoinAndStart(ctx)
	require.NoError(t, err, "can't join cluster")

	go n.Node.Run(ctx)

	return n
}

// NewRaftCluster creates a new raft cluster with 3 nodes for testing
func NewRaftCluster(t *testing.T, tc *cautils.TestCA, config ...*api.RaftConfig) (map[uint64]*TestNode, *fakeclock.FakeClock) {
	var (
		raftConfig  *api.RaftConfig
		clockSource *fakeclock.FakeClock
	)
	if len(config) > 1 {
		panic("more than one optional argument provided")
	}
	if len(config) == 1 {
		raftConfig = config[0]
	}
	nodes := make(map[uint64]*TestNode)
	nodes[1], clockSource = NewInitNode(t, tc, raftConfig)
	AddRaftNode(t, clockSource, nodes, tc)
	AddRaftNode(t, clockSource, nodes, tc)
	return nodes, clockSource
}

// AddRaftNode adds an additional raft test node to an existing cluster
func AddRaftNode(t *testing.T, clockSource *fakeclock.FakeClock, nodes map[uint64]*TestNode, tc *cautils.TestCA, opts ...raft.NodeOptions) {
	n := uint64(len(nodes) + 1)
	nodes[n] = NewJoinNode(t, clockSource, nodes[1].Address, tc, opts...)
	WaitForCluster(t, clockSource, nodes)
}

// TeardownCluster destroys a raft cluster used for tests
func TeardownCluster(nodes map[uint64]*TestNode) {
	for _, node := range nodes {
		ShutdownNode(node)
	}
}

// ShutdownNode shuts down a raft test node and deletes the content
// of the state directory
func ShutdownNode(node *TestNode) {
	node.Server.Stop()
	if node.cancel != nil {
		node.cancel()
		<-node.Done()
	}
	os.RemoveAll(node.StateDir)
	node.Listener.CloseListener()
}

// ShutdownRaft shutdowns only raft part of node.
func (n *TestNode) ShutdownRaft() {
	if n.cancel != nil {
		n.cancel()
		<-n.Done()
	}
}

// CleanupNonRunningNode frees resources associated with a node which is not
// running.
func CleanupNonRunningNode(node *TestNode) {
	node.Server.Stop()
	os.RemoveAll(node.StateDir)
	node.Listener.CloseListener()
}

// Leader determines who is the leader amongst a set of raft nodes
// belonging to the same cluster
func Leader(nodes map[uint64]*TestNode) *TestNode {
	for _, n := range nodes {
		if n.Config.ID == n.Leader() {
			return n
		}
	}
	panic("could not find a leader")
}

// ProposeValue proposes a value to a raft test cluster
func ProposeValue(t *testing.T, raftNode *TestNode, time time.Duration, nodeID ...string) (*api.Node, error) {
	nodeIDStr := "id1"
	if len(nodeID) != 0 {
		nodeIDStr = nodeID[0]
	}
	node := &api.Node{
		ID: nodeIDStr,
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: nodeIDStr,
			},
		},
	}

	storeActions := []api.StoreAction{
		{
			Action: api.StoreActionKindCreate,
			Target: &api.StoreAction_Node{
				Node: node,
			},
		},
	}

	ctx, _ := context.WithTimeout(context.Background(), time)

	err := raftNode.ProposeValue(ctx, storeActions, func() {
		err := raftNode.MemoryStore().ApplyStoreActions(storeActions)
		assert.NoError(t, err, "error applying actions")
	})
	if err != nil {
		return nil, err
	}

	return node, nil
}

// CheckValue checks that the value has been propagated between raft members
func CheckValue(t *testing.T, clockSource *fakeclock.FakeClock, raftNode *TestNode, createdNode *api.Node) {
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		var err error
		raftNode.MemoryStore().View(func(tx store.ReadTx) {
			var allNodes []*api.Node
			allNodes, err = store.FindNodes(tx, store.All)
			if err != nil {
				return
			}
			if len(allNodes) != 1 {
				err = errors.Errorf("expected 1 node, got %d nodes", len(allNodes))
				return
			}
			if !reflect.DeepEqual(allNodes[0], createdNode) {
				err = errors.New("node did not match expected value")
			}
		})
		return err
	}))
}

// CheckNoValue checks that there is no value replicated on nodes, generally
// used to test the absence of a leader
func CheckNoValue(t *testing.T, clockSource *fakeclock.FakeClock, raftNode *TestNode) {
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		var err error
		raftNode.MemoryStore().View(func(tx store.ReadTx) {
			var allNodes []*api.Node
			allNodes, err = store.FindNodes(tx, store.All)
			if err != nil {
				return
			}
			if len(allNodes) != 0 {
				err = errors.Errorf("expected no nodes, got %d", len(allNodes))
			}
		})
		return err
	}))
}

// CheckValuesOnNodes checks that all the nodes in the cluster have the same
// replicated data, generally used to check if a node can catch up with the logs
// correctly
func CheckValuesOnNodes(t *testing.T, clockSource *fakeclock.FakeClock, checkNodes map[uint64]*TestNode, ids []string, values []*api.Node) {
	iteration := 0
	for checkNodeID, node := range checkNodes {
		assert.NoError(t, testutils.PollFunc(clockSource, func() error {
			var err error
			node.MemoryStore().View(func(tx store.ReadTx) {
				var allNodes []*api.Node
				allNodes, err = store.FindNodes(tx, store.All)
				if err != nil {
					return
				}
				for i, id := range ids {
					n := store.GetNode(tx, id)
					if n == nil {
						err = errors.Errorf("node %s not found on %d (iteration %d)", id, checkNodeID, iteration)
						return
					}
					if !reflect.DeepEqual(values[i], n) {
						err = errors.Errorf("node %s did not match expected value on %d (iteration %d)", id, checkNodeID, iteration)
						return
					}
				}
				if len(allNodes) != len(ids) {
					err = errors.Errorf("expected %d nodes, got %d (iteration %d)", len(ids), len(allNodes), iteration)
					return
				}
			})
			return err
		}))
		iteration++
	}
}

// GetAllValuesOnNode returns all values on this node
func GetAllValuesOnNode(t *testing.T, clockSource *fakeclock.FakeClock, raftNode *TestNode) ([]string, []*api.Node) {
	ids := []string{}
	values := []*api.Node{}
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		var err error
		raftNode.MemoryStore().View(func(tx store.ReadTx) {
			var allNodes []*api.Node
			allNodes, err = store.FindNodes(tx, store.All)
			if err != nil {
				return
			}
			for _, node := range allNodes {
				ids = append(ids, node.ID)
				values = append(values, node)
			}
		})
		return err
	}))

	return ids, values
}

// NewSnapshotMessage creates and returns a raftpb.Message of type MsgSnap
// where the snapshot data is of the given size and the value of each byte
// is (index of the byte) % 256.
func NewSnapshotMessage(from, to uint64, size int) *raftpb.Message {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i % (1 << 8))
	}

	return &raftpb.Message{
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

// VerifySnapshot verifies that the snapshot data where each byte is
// of the value (index % sizeof(byte)).
func VerifySnapshot(raftMsg *raftpb.Message) bool {
	for i, b := range raftMsg.Snapshot.Data {
		if int(b) != i%(1<<8) {
			return false
		}
	}

	return len(raftMsg.Snapshot.Data) == int(raftMsg.Snapshot.Metadata.Index)
}
