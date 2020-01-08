package watchapi

import (
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/docker/swarmkit/api"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/manager/state/store"
	stateutils "github.com/docker/swarmkit/manager/state/testutils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type testServer struct {
	Server *Server
	Client api.WatchClient
	Store  *store.MemoryStore

	grpcServer *grpc.Server
	clientConn *grpc.ClientConn

	tempUnixSocket string
}

func (ts *testServer) Stop() {
	ts.Server.Stop()
	ts.clientConn.Close()
	ts.grpcServer.Stop()
	ts.Store.Close()
	os.RemoveAll(ts.tempUnixSocket)
}

func newTestServer(t *testing.T) *testServer {
	ts := &testServer{}

	// Create a testCA just to get a usable RootCA object
	tc := cautils.NewTestCA(nil)
	tc.Stop()

	ts.Store = store.NewMemoryStore(&stateutils.MockProposer{})
	assert.NotNil(t, ts.Store)
	ts.Server = NewServer(ts.Store)
	assert.NotNil(t, ts.Server)

	require.NoError(t, ts.Server.Start(context.Background()))

	temp, err := ioutil.TempFile("", "test-socket")
	assert.NoError(t, err)
	assert.NoError(t, temp.Close())
	assert.NoError(t, os.Remove(temp.Name()))

	ts.tempUnixSocket = temp.Name()

	lis, err := net.Listen("unix", temp.Name())
	assert.NoError(t, err)

	ts.grpcServer = grpc.NewServer()
	api.RegisterWatchServer(ts.grpcServer, ts.Server)
	go func() {
		// Serve will always return an error (even when properly stopped).
		// Explicitly ignore it.
		_ = ts.grpcServer.Serve(lis)
	}()

	conn, err := grpc.Dial(temp.Name(), grpc.WithInsecure(), grpc.WithTimeout(10*time.Second),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	assert.NoError(t, err)
	ts.clientConn = conn

	ts.Client = api.NewWatchClient(conn)

	return ts
}

func createNode(t *testing.T, ts *testServer, id string, role api.NodeRole, membership api.NodeSpec_Membership, state api.NodeStatus_State) *api.Node {
	node := &api.Node{
		ID: id,
		Spec: api.NodeSpec{
			Membership: membership,
		},
		Status: api.NodeStatus{
			State: state,
		},
		Role: role,
	}
	err := ts.Store.Update(func(tx store.Tx) error {
		return store.CreateNode(tx, node)
	})
	assert.NoError(t, err)
	return node
}

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, ioutil.Discard))
	logrus.SetOutput(ioutil.Discard)
}
