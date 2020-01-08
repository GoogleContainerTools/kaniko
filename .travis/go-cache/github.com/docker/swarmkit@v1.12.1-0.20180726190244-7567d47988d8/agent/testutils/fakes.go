package testutils

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/docker/swarmkit/agent/exec"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

// TestExecutor is executor for integration tests
type TestExecutor struct {
	mu   sync.Mutex
	desc *api.NodeDescription
}

// Describe just returns empty NodeDescription.
func (e *TestExecutor) Describe(ctx context.Context) (*api.NodeDescription, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.desc == nil {
		return &api.NodeDescription{}, nil
	}
	return e.desc.Copy(), nil
}

// Configure does nothing.
func (e *TestExecutor) Configure(ctx context.Context, node *api.Node) error {
	return nil
}

// SetNetworkBootstrapKeys does nothing.
func (e *TestExecutor) SetNetworkBootstrapKeys([]*api.EncryptionKey) error {
	return nil
}

// Controller returns TestController.
func (e *TestExecutor) Controller(t *api.Task) (exec.Controller, error) {
	return &TestController{
		ch: make(chan struct{}),
	}, nil
}

// UpdateNodeDescription sets the node description on the test executor
func (e *TestExecutor) UpdateNodeDescription(newDesc *api.NodeDescription) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.desc = newDesc
}

// TestController is dummy channel based controller for tests.
type TestController struct {
	ch        chan struct{}
	closeOnce sync.Once
}

// Update does nothing.
func (t *TestController) Update(ctx context.Context, task *api.Task) error {
	return nil
}

// Prepare does nothing.
func (t *TestController) Prepare(ctx context.Context) error {
	return nil
}

// Start does nothing.
func (t *TestController) Start(ctx context.Context) error {
	return nil
}

// Wait waits on internal channel.
func (t *TestController) Wait(ctx context.Context) error {
	select {
	case <-t.ch:
	case <-ctx.Done():
	}
	return nil
}

// Shutdown closes internal channel
func (t *TestController) Shutdown(ctx context.Context) error {
	t.closeOnce.Do(func() {
		close(t.ch)
	})
	return nil
}

// Terminate closes internal channel if it wasn't closed before.
func (t *TestController) Terminate(ctx context.Context) error {
	t.closeOnce.Do(func() {
		close(t.ch)
	})
	return nil
}

// Remove does nothing.
func (t *TestController) Remove(ctx context.Context) error {
	return nil
}

// Close does nothing.
func (t *TestController) Close() error {
	t.closeOnce.Do(func() {
		close(t.ch)
	})
	return nil
}

// SessionHandler is an injectable function that can be used handle session requests
type SessionHandler func(*api.SessionRequest, api.Dispatcher_SessionServer) error

// MockDispatcher is a fake dispatcher that one agent at a time can connect to
type MockDispatcher struct {
	mu             sync.Mutex
	sessionCh      chan *api.SessionMessage
	openSession    *api.SessionRequest
	closedSessions []*api.SessionRequest
	sessionHandler SessionHandler

	Addr string
}

// UpdateTaskStatus is not implemented
func (m *MockDispatcher) UpdateTaskStatus(context.Context, *api.UpdateTaskStatusRequest) (*api.UpdateTaskStatusResponse, error) {
	panic("not implemented")
}

// Tasks keeps an open stream until canceled
func (m *MockDispatcher) Tasks(_ *api.TasksRequest, stream api.Dispatcher_TasksServer) error {
	select {
	case <-stream.Context().Done():
	}
	return nil
}

// Assignments keeps an open stream until canceled
func (m *MockDispatcher) Assignments(_ *api.AssignmentsRequest, stream api.Dispatcher_AssignmentsServer) error {
	select {
	case <-stream.Context().Done():
	}
	return nil
}

// Heartbeat always successfully heartbeats
func (m *MockDispatcher) Heartbeat(context.Context, *api.HeartbeatRequest) (*api.HeartbeatResponse, error) {
	return &api.HeartbeatResponse{Period: time.Second * 5}, nil
}

// Session allows a session to be established, and sends the node info
func (m *MockDispatcher) Session(r *api.SessionRequest, stream api.Dispatcher_SessionServer) error {
	m.mu.Lock()
	handler := m.sessionHandler
	m.openSession = r
	m.mu.Unlock()
	sessionID := identity.NewID()

	defer func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		log.G(stream.Context()).Debugf("non-dispatcher side closed session: %s", sessionID)
		m.closedSessions = append(m.closedSessions, r)
		if m.openSession == r { // only overwrite session if it hasn't changed
			m.openSession = nil
		}
	}()

	if handler != nil {
		return handler(r, stream)
	}

	// send the initial message first
	if err := stream.Send(&api.SessionMessage{
		SessionID: sessionID,
		Managers: []*api.WeightedPeer{
			{
				Peer: &api.Peer{Addr: m.Addr},
			},
		},
	}); err != nil {
		return err
	}

	ctx := stream.Context()
	for {
		select {
		case msg := <-m.sessionCh:
			msg.SessionID = sessionID
			if err := stream.Send(msg); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// GetSessions return all the established and closed sessions
func (m *MockDispatcher) GetSessions() (*api.SessionRequest, []*api.SessionRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.openSession, m.closedSessions
}

// SessionMessageChannel returns a writable channel to inject session messages
func (m *MockDispatcher) SessionMessageChannel() chan<- *api.SessionMessage {
	return m.sessionCh
}

// SetSessionHandler lets you inject a custom function to handle session requests
func (m *MockDispatcher) SetSessionHandler(s SessionHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionHandler = s
}

// NewMockDispatcher starts and returns a mock dispatcher instance that can be connected to
func NewMockDispatcher(t *testing.T, secConfig *ca.SecurityConfig, local bool) (*MockDispatcher, func()) {
	var (
		l       net.Listener
		err     error
		addr    string
		cleanup func()
	)
	if local {
		tempDir, err := ioutil.TempDir("", "local-dispatcher-socket")
		require.NoError(t, err)
		addr = filepath.Join(tempDir, "socket")
		l, err = net.Listen("unix", addr)
		require.NoError(t, err)
		cleanup = func() {
			os.RemoveAll(tempDir)
		}
	} else {
		l, err = net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr = l.Addr().String()
	}

	serverOpts := []grpc.ServerOption{grpc.Creds(secConfig.ServerTLSCreds)}
	s := grpc.NewServer(serverOpts...)

	m := &MockDispatcher{
		Addr:      addr,
		sessionCh: make(chan *api.SessionMessage, 1),
	}
	api.RegisterDispatcherServer(s, m)
	go s.Serve(l)
	return m, func() {
		l.Close()
		s.Stop()
		if cleanup != nil {
			cleanup()
		}
	}
}
