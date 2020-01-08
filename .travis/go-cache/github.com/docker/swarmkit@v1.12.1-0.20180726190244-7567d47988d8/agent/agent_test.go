package agent

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	events "github.com/docker/go-events"
	agentutils "github.com/docker/swarmkit/agent/testutils"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/connectionbroker"
	"github.com/docker/swarmkit/log"
	"github.com/docker/swarmkit/remotes"
	"github.com/docker/swarmkit/testutils"
	"github.com/docker/swarmkit/xnet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

var localDispatcher = false

// TestMain runs every test in this file twice - once with a local dispatcher, and
// once again with a remote dispatcher
func TestMain(m *testing.M) {
	localDispatcher = false
	dispatcherRPCTimeout = 500 * time.Millisecond
	if status := m.Run(); status != 0 {
		os.Exit(status)
	}

	localDispatcher = true
	os.Exit(m.Run())
}

func TestAgent(t *testing.T) {
	// TODO(stevvooe): The current agent is fairly monolithic, making it hard
	// to test without implementing or mocking an entire master. We'd like to
	// avoid this, as these kinds of tests are expensive to maintain.
	//
	// To support a proper testing program, the plan is to decouple the agent
	// into the following components:
	//
	// 	Connection: Manages the RPC connection and the available managers. Must
	// 	follow lazy grpc style but also expose primitives to force reset, which
	// 	is currently exposed through remotes.
	//
	//	Session: Manages the lifecycle of an agent from Register to a failure.
	//	Currently, this is implemented as Agent.session but we'd prefer to
	//	encapsulate it to keep the agent simple.
	//
	// 	Agent: With the above scaffolding, the agent reduces to Agent.Assign
	// 	and Agent.Watch. Testing becomes as simple as assigning tasks sets and
	// 	checking that the appropriate events come up on the watch queue.
	//
	// We may also move the Assign/Watch to a Driver type and have the agent
	// oversee everything.
}

func TestAgentStartStop(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	agentSecurityConfig, err := tc.NewNodeConfig(ca.WorkerRole)
	require.NoError(t, err)

	addr := "localhost:4949"
	remotes := remotes.NewRemotes(api.Peer{Addr: addr})

	db, cleanup := storageTestEnv(t)
	defer cleanup()

	agent, err := New(&Config{
		Executor:    &agentutils.TestExecutor{},
		ConnBroker:  connectionbroker.New(remotes),
		Credentials: agentSecurityConfig.ClientTLSCreds,
		DB:          db,
		NodeTLSInfo: &api.NodeTLSInfo{},
	})
	require.NoError(t, err)
	assert.NotNil(t, agent)

	ctx, _ := context.WithTimeout(tc.Context, 5000*time.Millisecond)

	assert.Equal(t, errAgentNotStarted, agent.Stop(ctx))
	assert.NoError(t, agent.Start(ctx))

	if err := agent.Start(ctx); err != errAgentStarted {
		t.Fatalf("expected agent started error: %v", err)
	}

	assert.NoError(t, agent.Stop(ctx))
}

func TestHandleSessionMessageNetworkManagerChanges(t *testing.T) {
	nodeChangeCh := make(chan *NodeChanges, 1)
	defer close(nodeChangeCh)
	tester := agentTestEnv(t, nodeChangeCh, nil)
	defer tester.cleanup()
	defer tester.StartAgent(t)()

	currSession, closedSessions := tester.dispatcher.GetSessions()
	require.NotNil(t, currSession)
	require.NotNil(t, currSession.Description)
	require.Empty(t, closedSessions)

	var messages = []*api.SessionMessage{
		{
			Managers: []*api.WeightedPeer{
				{Peer: &api.Peer{NodeID: "node1", Addr: "10.0.0.1"}, Weight: 1.0}},
			NetworkBootstrapKeys: []*api.EncryptionKey{{}},
		},
		{
			Managers: []*api.WeightedPeer{
				{Peer: &api.Peer{NodeID: "node1", Addr: ""}, Weight: 1.0}},
			NetworkBootstrapKeys: []*api.EncryptionKey{{}},
		},
		{
			Managers: []*api.WeightedPeer{
				{Peer: &api.Peer{NodeID: "node1", Addr: "10.0.0.1"}, Weight: 1.0}},
			NetworkBootstrapKeys: nil,
		},
		{
			Managers: []*api.WeightedPeer{
				{Peer: &api.Peer{NodeID: "", Addr: "10.0.0.1"}, Weight: 1.0}},
			NetworkBootstrapKeys: []*api.EncryptionKey{{}},
		},
		{
			Managers: []*api.WeightedPeer{
				{Peer: &api.Peer{NodeID: "node1", Addr: "10.0.0.1"}, Weight: 0.0}},
			NetworkBootstrapKeys: []*api.EncryptionKey{{}},
		},
	}

	for _, m := range messages {
		m.SessionID = currSession.SessionID
		tester.dispatcher.SessionMessageChannel() <- m
		select {
		case nodeChange := <-nodeChangeCh:
			require.FailNow(t, "there should be no node changes with these messages: %v", nodeChange)
		case <-time.After(100 * time.Millisecond):
		}
	}

	currSession, closedSessions = tester.dispatcher.GetSessions()
	require.NotEmpty(t, currSession)
	require.Empty(t, closedSessions)
}

func TestHandleSessionMessageNodeChanges(t *testing.T) {
	nodeChangeCh := make(chan *NodeChanges, 1)
	defer close(nodeChangeCh)
	tester := agentTestEnv(t, nodeChangeCh, nil)
	defer tester.cleanup()
	defer tester.StartAgent(t)()

	currSession, closedSessions := tester.dispatcher.GetSessions()
	require.NotNil(t, currSession)
	require.NotNil(t, currSession.Description)
	require.Empty(t, closedSessions)

	var testcases = []struct {
		msg      *api.SessionMessage
		change   *NodeChanges
		errorMsg string
	}{
		{
			msg: &api.SessionMessage{
				Node: &api.Node{},
			},
			change:   &NodeChanges{Node: &api.Node{}},
			errorMsg: "the node changed, but no notification of node change",
		},
		{
			msg: &api.SessionMessage{
				RootCA: []byte("new root CA"),
			},
			change:   &NodeChanges{RootCert: []byte("new root CA")},
			errorMsg: "the root cert changed, but no notification of node change",
		},
		{
			msg: &api.SessionMessage{
				Node:   &api.Node{ID: "something"},
				RootCA: []byte("new root CA"),
			},
			change: &NodeChanges{
				Node:     &api.Node{ID: "something"},
				RootCert: []byte("new root CA"),
			},
			errorMsg: "the root cert and node both changed, but no notification of node change",
		},
		{
			msg: &api.SessionMessage{
				Node:   &api.Node{ID: "something"},
				RootCA: tester.testCA.RootCA.Certs,
			},
			errorMsg: "while a node and root cert were provided, nothing has changed so no node changed",
		},
	}

	for _, tc := range testcases {
		tc.msg.SessionID = currSession.SessionID
		tester.dispatcher.SessionMessageChannel() <- tc.msg
		if tc.change != nil {
			select {
			case nodeChange := <-nodeChangeCh:
				require.Equal(t, tc.change, nodeChange, tc.errorMsg)
			case <-time.After(100 * time.Millisecond):
				require.FailNow(t, tc.errorMsg)
			}
		} else {
			select {
			case nodeChange := <-nodeChangeCh:
				require.FailNow(t, "%s: but got change: %v", tc.errorMsg, nodeChange)
			case <-time.After(100 * time.Millisecond):
			}
		}
	}

	currSession, closedSessions = tester.dispatcher.GetSessions()
	require.NotEmpty(t, currSession)
	require.Empty(t, closedSessions)
}

// when the node description changes, the session is restarted and propagated up to the dispatcher.
// the node description includes the FIPSness of the agent.
func TestSessionRestartedOnNodeDescriptionChange(t *testing.T) {
	tlsCh := make(chan events.Event, 1)
	defer close(tlsCh)
	tester := agentTestEnv(t, nil, tlsCh)
	tester.agent.config.FIPS = true // start out with the agent in FIPS-enabled mode
	defer tester.cleanup()
	defer tester.StartAgent(t)()

	currSession, closedSessions := tester.dispatcher.GetSessions()
	require.NotNil(t, currSession)
	require.NotNil(t, currSession.Description)
	require.True(t, currSession.Description.FIPS)
	require.Empty(t, closedSessions)

	tester.executor.UpdateNodeDescription(&api.NodeDescription{
		Hostname: "testAgent",
	})
	var gotSession *api.SessionRequest
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		gotSession, closedSessions = tester.dispatcher.GetSessions()
		if gotSession == nil {
			return errors.New("no current session")
		}
		if len(closedSessions) != 1 {
			return fmt.Errorf("expecting 1 closed sessions, got %d", len(closedSessions))
		}
		return nil
	}, 2*time.Second))
	require.NotEqual(t, currSession, gotSession)
	require.NotNil(t, gotSession.Description)
	require.Equal(t, "testAgent", gotSession.Description.Hostname)
	require.True(t, gotSession.Description.FIPS)
	currSession = gotSession

	// If nothing changes, the session is not re-established
	tlsCh <- gotSession.Description.TLSInfo
	time.Sleep(1 * time.Second)
	gotSession, closedSessions = tester.dispatcher.GetSessions()
	require.Equal(t, currSession, gotSession)
	require.Len(t, closedSessions, 1)

	newTLSInfo := &api.NodeTLSInfo{
		TrustRoot:           cautils.ECDSA256SHA256Cert,
		CertIssuerPublicKey: []byte("public key"),
		CertIssuerSubject:   []byte("subject"),
	}
	tlsCh <- newTLSInfo
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		gotSession, closedSessions = tester.dispatcher.GetSessions()
		if gotSession == nil {
			return errors.New("no current session")
		}
		if len(closedSessions) != 2 {
			return fmt.Errorf("expecting 2 closed sessions, got %d", len(closedSessions))
		}
		return nil
	}, 2*time.Second))
	require.NotEqual(t, currSession, gotSession)
	require.NotNil(t, gotSession.Description)
	require.Equal(t, "testAgent", gotSession.Description.Hostname)
	require.Equal(t, newTLSInfo, gotSession.Description.TLSInfo)
	require.True(t, gotSession.Description.FIPS)
}

// If the dispatcher returns an error, if it times out, or if it's unreachable, no matter
// what the agent attempts to reconnect and rebuild a new session.
func TestSessionReconnectsIfDispatcherErrors(t *testing.T) {
	tlsCh := make(chan events.Event, 1)
	defer close(tlsCh)

	tester := agentTestEnv(t, nil, tlsCh)
	defer tester.cleanup()
	defer tester.StartAgent(t)()

	// create a second dispatcher we can fall back on
	anotherConfig, err := tester.testCA.NewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)
	anotherDispatcher, stop := agentutils.NewMockDispatcher(t, anotherConfig, false) // this one is not local, because the other one may be
	defer stop()

	var counter int
	anotherDispatcher.SetSessionHandler(func(r *api.SessionRequest, stream api.Dispatcher_SessionServer) error {
		if counter == 0 {
			counter++
			return errors.New("terminate immediately")
		}
		// hang forever until the other side cancels, and then set the session to nil so we use the default one
		defer anotherDispatcher.SetSessionHandler(nil)
		<-stream.Context().Done()
		return stream.Context().Err()
	})

	// ok, agent should have connect to the first dispatcher by now - if it has, kill the first dispatcher and ensure
	// the agent connects to the second one
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		gotSession, closedSessions := tester.dispatcher.GetSessions()
		if gotSession == nil {
			return errors.New("no current session")
		}
		if len(closedSessions) != 0 {
			return fmt.Errorf("expecting 0 closed sessions, got %d", len(closedSessions))
		}
		return nil
	}, 2*time.Second))
	tester.stopDispatcher()
	tester.remotes.setPeer(api.Peer{Addr: anotherDispatcher.Addr})
	tester.agent.config.ConnBroker.SetLocalConn(nil)

	// It should have connected with the second dispatcher 3 times - first because the first dispatcher died,
	// second because the dispatcher returned an error, third time because the session timed out.  So there should
	// be 2 closed sessions.
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		gotSession, closedSessions := anotherDispatcher.GetSessions()
		if gotSession == nil {
			return errors.New("no current session")
		}
		if len(closedSessions) != 2 {
			return fmt.Errorf("expecting 2 closed sessions, got %d", len(closedSessions))
		}
		return nil
	}, 5*time.Second))
}

type testSessionTracker struct {
	closeCounter, errCounter, establishedSessions int
	err                                           error
	mu                                            sync.Mutex
}

func (t *testSessionTracker) SessionError(err error) {
	t.mu.Lock()
	t.err = err
	t.errCounter++
	t.mu.Unlock()
}

func (t *testSessionTracker) SessionClosed() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closeCounter++
	if t.closeCounter >= 3 {
		return t.err
	}
	return nil
}

func (t *testSessionTracker) SessionEstablished() {
	t.mu.Lock()
	t.establishedSessions++
	t.mu.Unlock()
}

func (t *testSessionTracker) Stats() (int, int, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.establishedSessions, t.errCounter, t.closeCounter
}

// If we pass a session tracker, and OnSessionClosed returns an error, the agent should exit with that error
// as opposed to rebuilding
func TestAgentExitsBasedOnSessionTracker(t *testing.T) {
	tlsCh := make(chan events.Event, 1)
	defer close(tlsCh)
	tester := agentTestEnv(t, nil, tlsCh)
	defer tester.cleanup()

	// set the dispatcher to always error
	tester.dispatcher.SetSessionHandler(func(r *api.SessionRequest, stream api.Dispatcher_SessionServer) error {
		return errors.New("I always error")
	})

	// add a hook to the agent to exit after 3 session rebuilds
	tracker := testSessionTracker{}
	tester.agent.config.SessionTracker = &tracker

	go tester.agent.Start(tester.testCA.Context)
	defer tester.agent.Stop(tester.testCA.Context)

	getErr := make(chan error)
	go func() {
		getErr <- tester.agent.Err(tester.testCA.Context)
	}()

	select {
	case err := <-getErr:
		require.Error(t, err)
		require.Contains(t, err.Error(), "I always error")
	case <-tester.agent.Ready():
		require.FailNow(t, "agent should have failed to connect")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "agent didn't fail within 5 seconds")
	}

	establishedSessions, errCounter, closeClounter := tracker.Stats()
	require.Equal(t, establishedSessions, 0)
	require.Equal(t, errCounter, 3)
	require.Equal(t, closeClounter, 3)
	currSession, closedSessions := tester.dispatcher.GetSessions()
	require.Nil(t, currSession)
	require.Len(t, closedSessions, 3)
}

// If we pass a session tracker, established sessions get tracked.
func TestAgentRegistersSessionsWithSessionTracker(t *testing.T) {
	tlsCh := make(chan events.Event, 1)
	defer close(tlsCh)
	tester := agentTestEnv(t, nil, tlsCh)
	defer tester.cleanup()

	// add a hook to the agent to exit after 3 session rebuilds
	tracker := testSessionTracker{}
	tester.agent.config.SessionTracker = &tracker

	defer tester.StartAgent(t)()

	var establishedSessions, errCounter, closeCounter int
	// poll because session tracker gets called after the ready channel is closed
	// (so there may be edge cases where the stats are called before the session
	// tracker is called)
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		establishedSessions, errCounter, closeCounter = tracker.Stats()
		if establishedSessions != 1 {
			return errors.New("sessiontracker hasn't been called yet")
		}
		return nil
	}, 3*time.Millisecond))
	require.Equal(t, errCounter, 0)
	require.Equal(t, closeCounter, 0)
	currSession, closedSessions := tester.dispatcher.GetSessions()
	require.NotNil(t, currSession)
	require.Len(t, closedSessions, 0)
}

type agentTester struct {
	agent                   *Agent
	dispatcher              *agentutils.MockDispatcher
	executor                *agentutils.TestExecutor
	stopDispatcher, cleanup func()
	testCA                  *cautils.TestCA
	remotes                 *fakeRemotes
}

func (a *agentTester) StartAgent(t *testing.T) func() {
	go a.agent.Start(a.testCA.Context)

	getErr := make(chan error)
	go func() {
		getErr <- a.agent.Err(a.testCA.Context)
	}()
	select {
	case err := <-getErr:
		require.FailNow(t, "starting agent errored with: %v", err)
	case <-a.agent.Ready():
	case <-time.After(5 * time.Second):
		require.FailNow(t, "agent not ready within 5 seconds")
	}

	return func() {
		a.agent.Stop(a.testCA.Context)
	}
}

func agentTestEnv(t *testing.T, nodeChangeCh chan *NodeChanges, tlsChangeCh chan events.Event) *agentTester {
	var cleanup []func()
	tc := cautils.NewTestCA(t)
	cleanup = append(cleanup, tc.Stop)
	tc.Context = log.WithLogger(tc.Context, log.G(tc.Context).WithField("localDispatcher", localDispatcher))

	agentSecurityConfig, err := tc.NewNodeConfig(ca.WorkerRole)
	require.NoError(t, err)
	managerSecurityConfig, err := tc.NewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)

	mockDispatcher, mockDispatcherStop := agentutils.NewMockDispatcher(t, managerSecurityConfig, localDispatcher)
	cleanup = append(cleanup, mockDispatcherStop)

	fr := &fakeRemotes{}
	broker := connectionbroker.New(fr)
	if localDispatcher {
		insecureCreds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		conn, err := grpc.Dial(
			mockDispatcher.Addr,
			grpc.WithTransportCredentials(insecureCreds),
			grpc.WithDialer(
				func(addr string, timeout time.Duration) (net.Conn, error) {
					return xnet.DialTimeoutLocal(addr, timeout)
				}),
		)
		require.NoError(t, err)
		cleanup = append(cleanup, func() { conn.Close() })

		broker.SetLocalConn(conn)
	} else {
		fr.setPeer(api.Peer{Addr: mockDispatcher.Addr})
	}

	db, cleanupStorage := storageTestEnv(t)
	cleanup = append(cleanup, func() { cleanupStorage() })

	executor := &agentutils.TestExecutor{}

	agent, err := New(&Config{
		Executor:         executor,
		ConnBroker:       broker,
		Credentials:      agentSecurityConfig.ClientTLSCreds,
		DB:               db,
		NotifyNodeChange: nodeChangeCh,
		NotifyTLSChange:  tlsChangeCh,
		NodeTLSInfo: &api.NodeTLSInfo{
			TrustRoot:           tc.RootCA.Certs,
			CertIssuerPublicKey: agentSecurityConfig.IssuerInfo().PublicKey,
			CertIssuerSubject:   agentSecurityConfig.IssuerInfo().Subject,
		},
	})
	require.NoError(t, err)
	agent.nodeUpdatePeriod = 200 * time.Millisecond

	return &agentTester{
		agent:          agent,
		dispatcher:     mockDispatcher,
		stopDispatcher: mockDispatcherStop,
		executor:       executor,
		testCA:         tc,
		cleanup: func() {
			// go in reverse order
			for i := len(cleanup) - 1; i >= 0; i-- {
				cleanup[i]()
			}
		},
		remotes: fr,
	}
}

// fakeRemotes is a Remotes interface that just always selects the current remote until
// it is switched out
type fakeRemotes struct {
	mu   sync.Mutex
	peer api.Peer
}

func (f *fakeRemotes) Weights() map[api.Peer]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return map[api.Peer]int{f.peer: 1}
}

func (f *fakeRemotes) Select(...string) (api.Peer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.peer, nil
}

// do nothing
func (f *fakeRemotes) Observe(peer api.Peer, weight int)         {}
func (f *fakeRemotes) ObserveIfExists(peer api.Peer, weight int) {}
func (f *fakeRemotes) Remove(addrs ...api.Peer)                  {}

func (f *fakeRemotes) setPeer(p api.Peer) {
	f.mu.Lock()
	f.peer = p
	f.mu.Unlock()
}

var _ remotes.Remotes = &fakeRemotes{}
