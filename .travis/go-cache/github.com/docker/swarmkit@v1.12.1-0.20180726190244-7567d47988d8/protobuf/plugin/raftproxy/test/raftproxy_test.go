package test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type testRouteGuide struct{}

func (testRouteGuide) GetFeature(context.Context, *Point) (*Feature, error) {
	panic("not implemented")
}

func (testRouteGuide) ListFeatures(*Rectangle, RouteGuide_ListFeaturesServer) error {
	panic("not implemented")
}

func (testRouteGuide) RecordRoute(RouteGuide_RecordRouteServer) error {
	panic("not implemented")
}

func (testRouteGuide) RouteChat(RouteGuide_RouteChatServer) error {
	panic("not implemented")
}

type mockCluster struct {
	conn *grpc.ClientConn
}

func (m *mockCluster) LeaderConn(ctx context.Context) (*grpc.ClientConn, error) {
	return m.conn, nil
}

func TestSimpleRedirect(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithTimeout(5*time.Second))
	require.NoError(t, err)
	defer conn.Close()

	cluster := &mockCluster{conn: conn}

	forwardAsOwnRequest := func(ctx context.Context) (context.Context, error) { return ctx, nil }
	api := NewRaftProxyRouteGuideServer(testRouteGuide{}, cluster, nil, forwardAsOwnRequest)
	srv := grpc.NewServer()
	RegisterRouteGuideServer(srv, api)
	go srv.Serve(l)
	defer srv.Stop()

	client := NewRouteGuideClient(conn)
	_, err = client.GetFeature(context.Background(), &Point{})
	assert.NotNil(t, err)
	assert.Equal(t, codes.ResourceExhausted, grpc.Code(err))
}
