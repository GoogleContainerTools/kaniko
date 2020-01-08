package controlapi

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/swarmkit/api"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/identity"
	raftutils "github.com/docker/swarmkit/manager/state/raft/testutils"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/testutils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
)

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

func TestGetNode(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	_, err := ts.Client.GetNode(context.Background(), &api.GetNodeRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	_, err = ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: "invalid"})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err))

	node := createNode(t, ts, "id", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)
	r, err := ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: node.ID})
	assert.NoError(t, err)
	assert.Equal(t, node.ID, r.Node.ID)
}

func TestListNodes(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	r, err := ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Empty(t, r.Nodes)

	createNode(t, ts, "id1", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)
	r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Nodes))

	createNode(t, ts, "id2", api.NodeRoleWorker, api.NodeMembershipAccepted, api.NodeStatus_READY)
	createNode(t, ts, "id3", api.NodeRoleWorker, api.NodeMembershipPending, api.NodeStatus_READY)
	r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Nodes))

	// List by role.
	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Roles: []api.NodeRole{api.NodeRoleManager},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Nodes))
	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Roles: []api.NodeRole{api.NodeRoleWorker},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(r.Nodes))
	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Roles: []api.NodeRole{api.NodeRoleManager, api.NodeRoleWorker},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Nodes))

	// List by membership.
	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Memberships: []api.NodeSpec_Membership{api.NodeMembershipAccepted},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(r.Nodes))
	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Memberships: []api.NodeSpec_Membership{api.NodeMembershipPending},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Nodes))
	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Memberships: []api.NodeSpec_Membership{api.NodeMembershipAccepted, api.NodeMembershipPending},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Nodes))
	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Roles:       []api.NodeRole{api.NodeRoleWorker},
				Memberships: []api.NodeSpec_Membership{api.NodeMembershipPending},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Nodes))
}

func TestRemoveNodes(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	ts.Store.Update(func(tx store.Tx) error {
		store.CreateCluster(tx, &api.Cluster{
			ID: identity.NewID(),
			Spec: api.ClusterSpec{
				Annotations: api.Annotations{
					Name: store.DefaultClusterName,
				},
			},
		})
		return nil
	})

	r, err := ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Empty(t, r.Nodes)

	createNode(t, ts, "id1", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)
	r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Len(t, r.Nodes, 1)

	createNode(t, ts, "id2", api.NodeRoleWorker, api.NodeMembershipAccepted, api.NodeStatus_READY)
	createNode(t, ts, "id3", api.NodeRoleWorker, api.NodeMembershipPending, api.NodeStatus_UNKNOWN)
	r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Len(t, r.Nodes, 3)

	// Attempt to remove a ready node without force
	_, err = ts.Client.RemoveNode(context.Background(),
		&api.RemoveNodeRequest{
			NodeID: "id2",
			Force:  false,
		},
	)
	assert.Error(t, err)

	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Roles: []api.NodeRole{api.NodeRoleManager, api.NodeRoleWorker},
			},
		},
	)
	assert.NoError(t, err)
	assert.Len(t, r.Nodes, 3)

	// Attempt to remove a ready node with force
	_, err = ts.Client.RemoveNode(context.Background(),
		&api.RemoveNodeRequest{
			NodeID: "id2",
			Force:  true,
		},
	)
	assert.NoError(t, err)

	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Roles: []api.NodeRole{api.NodeRoleManager, api.NodeRoleWorker},
			},
		},
	)
	assert.NoError(t, err)
	assert.Len(t, r.Nodes, 2)

	clusterResp, err := ts.Client.ListClusters(context.Background(), &api.ListClustersRequest{})
	assert.NoError(t, err)
	require.Len(t, clusterResp.Clusters, 1)
	require.Len(t, clusterResp.Clusters[0].BlacklistedCertificates, 1)
	_, ok := clusterResp.Clusters[0].BlacklistedCertificates["id2"]
	assert.True(t, ok)

	// Attempt to remove a non-ready node without force
	_, err = ts.Client.RemoveNode(context.Background(),
		&api.RemoveNodeRequest{
			NodeID: "id3",
			Force:  false,
		},
	)
	assert.NoError(t, err)

	r, err = ts.Client.ListNodes(context.Background(),
		&api.ListNodesRequest{
			Filters: &api.ListNodesRequest_Filters{
				Roles: []api.NodeRole{api.NodeRoleManager, api.NodeRoleWorker},
			},
		},
	)
	assert.NoError(t, err)
	assert.Len(t, r.Nodes, 1)
}

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, ioutil.Discard))
	logrus.SetOutput(ioutil.Discard)
}

func getMap(t *testing.T, nodes []*api.Node) map[uint64]*api.ManagerStatus {
	m := make(map[uint64]*api.ManagerStatus)
	for _, n := range nodes {
		if n.ManagerStatus != nil {
			m[n.ManagerStatus.RaftID] = n.ManagerStatus
		}
	}
	return m
}

func TestListManagerNodes(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(nil)
	defer tc.Stop()
	ts := newTestServer(t)
	defer ts.Stop()

	nodes, clockSource := raftutils.NewRaftCluster(t, tc)
	defer raftutils.TeardownCluster(nodes)

	// Create a node object for each of the managers
	assert.NoError(t, nodes[1].MemoryStore().Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, &api.Node{ID: nodes[1].SecurityConfig.ClientTLSCreds.NodeID()}))
		assert.NoError(t, store.CreateNode(tx, &api.Node{ID: nodes[2].SecurityConfig.ClientTLSCreds.NodeID()}))
		assert.NoError(t, store.CreateNode(tx, &api.Node{ID: nodes[3].SecurityConfig.ClientTLSCreds.NodeID()}))
		return nil
	}))

	// Assign one of the raft node to the test server
	ts.Server.raft = nodes[1].Node
	ts.Server.store = nodes[1].MemoryStore()

	// There should be 3 reachable managers listed
	r, err := ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, r)
	managers := getMap(t, r.Nodes)
	assert.Len(t, ts.Server.raft.GetMemberlist(), 3)
	assert.Len(t, r.Nodes, 3)

	// Node 1 should be the leader
	for i := 1; i <= 3; i++ {
		if i == 1 {
			assert.True(t, managers[nodes[uint64(i)].Config.ID].Leader)
			continue
		}
		assert.False(t, managers[nodes[uint64(i)].Config.ID].Leader)
	}

	// All nodes should be reachable
	for i := 1; i <= 3; i++ {
		assert.Equal(t, api.RaftMemberStatus_REACHABLE, managers[nodes[uint64(i)].Config.ID].Reachability)
	}

	// Add two more nodes to the cluster
	raftutils.AddRaftNode(t, clockSource, nodes, tc)
	raftutils.AddRaftNode(t, clockSource, nodes, tc)
	raftutils.WaitForCluster(t, clockSource, nodes)

	// Add node entries for these
	assert.NoError(t, nodes[1].MemoryStore().Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, &api.Node{ID: nodes[4].SecurityConfig.ClientTLSCreds.NodeID()}))
		assert.NoError(t, store.CreateNode(tx, &api.Node{ID: nodes[5].SecurityConfig.ClientTLSCreds.NodeID()}))
		return nil
	}))

	// There should be 5 reachable managers listed
	r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, r)
	managers = getMap(t, r.Nodes)
	assert.Len(t, ts.Server.raft.GetMemberlist(), 5)
	assert.Len(t, r.Nodes, 5)
	for i := 1; i <= 5; i++ {
		assert.Equal(t, api.RaftMemberStatus_REACHABLE, managers[nodes[uint64(i)].Config.ID].Reachability)
	}

	// Stops 2 nodes
	nodes[4].Server.Stop()
	nodes[4].ShutdownRaft()
	nodes[5].Server.Stop()
	nodes[5].ShutdownRaft()

	// Node 4 and Node 5 should be listed as Unreachable
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
		if err != nil {
			return err
		}

		managers = getMap(t, r.Nodes)

		if len(r.Nodes) != 5 {
			return fmt.Errorf("expected 5 nodes, got %d", len(r.Nodes))
		}

		if managers[nodes[4].Config.ID].Reachability == api.RaftMemberStatus_REACHABLE {
			return fmt.Errorf("expected node 4 to be unreachable")
		}

		if managers[nodes[5].Config.ID].Reachability == api.RaftMemberStatus_REACHABLE {
			return fmt.Errorf("expected node 5 to be unreachable")
		}

		return nil
	}))

	// Restart the 2 nodes
	nodes[4] = raftutils.RestartNode(t, clockSource, nodes[4], false)
	nodes[5] = raftutils.RestartNode(t, clockSource, nodes[5], false)
	raftutils.WaitForCluster(t, clockSource, nodes)

	assert.Len(t, ts.Server.raft.GetMemberlist(), 5)
	// All the nodes should be reachable again
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
		if err != nil {
			return err
		}
		managers = getMap(t, r.Nodes)
		for i := 1; i <= 5; i++ {
			if managers[nodes[uint64(i)].Config.ID].Reachability != api.RaftMemberStatus_REACHABLE {
				return fmt.Errorf("node %x is unreachable", nodes[uint64(i)].Config.ID)
			}
		}
		return nil
	}))

	// Stop node 1 (leader)
	nodes[1].Server.Stop()
	nodes[1].ShutdownRaft()

	newCluster := map[uint64]*raftutils.TestNode{
		2: nodes[2],
		3: nodes[3],
		4: nodes[4],
		5: nodes[5],
	}

	// Wait for the re-election to occur
	raftutils.WaitForCluster(t, clockSource, newCluster)

	var leaderNode *raftutils.TestNode
	for _, node := range newCluster {
		if node.IsLeader() {
			leaderNode = node
		}
	}

	// Switch the raft node used by the server
	ts.Server.raft = leaderNode.Node

	// Node 1 should not be the leader anymore
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
		if err != nil {
			return err
		}

		managers = getMap(t, r.Nodes)

		if managers[nodes[1].Config.ID].Leader {
			return fmt.Errorf("expected node 1 not to be the leader")
		}

		if managers[nodes[1].Config.ID].Reachability == api.RaftMemberStatus_REACHABLE {
			return fmt.Errorf("expected node 1 to be unreachable")
		}

		return nil
	}))

	// Restart node 1
	nodes[1].ShutdownRaft()
	nodes[1] = raftutils.RestartNode(t, clockSource, nodes[1], false)
	raftutils.WaitForCluster(t, clockSource, nodes)

	// Ensure that node 1 is not the leader
	assert.False(t, managers[nodes[uint64(1)].Config.ID].Leader)

	// Check that another node got the leader status
	var leader uint64
	leaderCount := 0
	for i := 1; i <= 5; i++ {
		if managers[nodes[uint64(i)].Config.ID].Leader {
			leader = nodes[uint64(i)].Config.ID
			leaderCount++
		}
	}

	// There should be only one leader after node 1 recovery and it
	// should be different than node 1
	assert.Equal(t, 1, leaderCount)
	assert.NotEqual(t, leader, nodes[1].Config.ID)
}

func TestUpdateNode(t *testing.T) {
	tc := cautils.NewTestCA(nil)
	defer tc.Stop()
	ts := newTestServer(t)
	defer ts.Stop()

	nodes := make(map[uint64]*raftutils.TestNode)
	nodes[1], _ = raftutils.NewInitNode(t, tc, nil)
	defer raftutils.TeardownCluster(nodes)

	nodeID := nodes[1].SecurityConfig.ClientTLSCreds.NodeID()

	// Assign one of the raft node to the test server
	ts.Server.raft = nodes[1].Node
	ts.Server.store = nodes[1].MemoryStore()

	_, err := ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{
		NodeID: nodeID,
		Spec: &api.NodeSpec{
			Availability: api.NodeAvailabilityDrain,
		},
		NodeVersion: &api.Version{},
	})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err))

	// Create a node object for the manager
	assert.NoError(t, nodes[1].MemoryStore().Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, &api.Node{
			ID: nodes[1].SecurityConfig.ClientTLSCreds.NodeID(),
			Spec: api.NodeSpec{
				Membership: api.NodeMembershipAccepted,
			},
			Role: api.NodeRoleManager,
		}))
		return nil
	}))

	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{NodeID: "invalid", Spec: &api.NodeSpec{}, NodeVersion: &api.Version{}})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err))

	r, err := ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: nodeID})
	assert.NoError(t, err)
	if !assert.NotNil(t, r) {
		assert.FailNow(t, "got unexpected nil response from GetNode")
	}
	assert.NotNil(t, r.Node)

	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{NodeID: nodeID})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	spec := r.Node.Spec.Copy()
	spec.Availability = api.NodeAvailabilityDrain
	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{
		NodeID: nodeID,
		Spec:   spec,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{
		NodeID:      nodeID,
		Spec:        spec,
		NodeVersion: &r.Node.Meta.Version,
	})
	assert.NoError(t, err)

	r, err = ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: nodeID})
	assert.NoError(t, err)
	if !assert.NotNil(t, r) {
		assert.FailNow(t, "got unexpected nil response from GetNode")
	}
	assert.NotNil(t, r.Node)
	assert.NotNil(t, r.Node.Spec)
	assert.Equal(t, api.NodeAvailabilityDrain, r.Node.Spec.Availability)

	version := &r.Node.Meta.Version
	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{NodeID: nodeID, Spec: &r.Node.Spec, NodeVersion: version})
	assert.NoError(t, err)

	// Perform an update with the "old" version.
	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{NodeID: nodeID, Spec: &r.Node.Spec, NodeVersion: version})
	assert.Error(t, err)
}

func testUpdateNodeDemote(t *testing.T) {
	tc := cautils.NewTestCA(nil)
	defer tc.Stop()
	ts := newTestServer(t)
	defer ts.Stop()

	nodes, clockSource := raftutils.NewRaftCluster(t, tc)
	defer raftutils.TeardownCluster(nodes)

	// Assign one of the raft node to the test server
	ts.Server.raft = nodes[1].Node
	ts.Server.store = nodes[1].MemoryStore()

	// Create a node object for each of the managers
	assert.NoError(t, nodes[1].MemoryStore().Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, &api.Node{
			ID: nodes[1].SecurityConfig.ClientTLSCreds.NodeID(),
			Spec: api.NodeSpec{
				DesiredRole: api.NodeRoleManager,
				Membership:  api.NodeMembershipAccepted,
			},
			Role: api.NodeRoleManager,
		}))
		assert.NoError(t, store.CreateNode(tx, &api.Node{
			ID: nodes[2].SecurityConfig.ClientTLSCreds.NodeID(),
			Spec: api.NodeSpec{
				DesiredRole: api.NodeRoleManager,
				Membership:  api.NodeMembershipAccepted,
			},
			Role: api.NodeRoleManager,
		}))
		assert.NoError(t, store.CreateNode(tx, &api.Node{
			ID: nodes[3].SecurityConfig.ClientTLSCreds.NodeID(),
			Spec: api.NodeSpec{
				DesiredRole: api.NodeRoleManager,
				Membership:  api.NodeMembershipAccepted,
			},
			Role: api.NodeRoleManager,
		}))
		return nil
	}))

	// Stop Node 3 (1 node out of 3)
	nodes[3].Server.Stop()
	nodes[3].ShutdownRaft()

	// Node 3 should be listed as Unreachable
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		members := nodes[1].GetMemberlist()
		if len(members) != 3 {
			return fmt.Errorf("expected 3 nodes, got %d", len(members))
		}
		if members[nodes[3].Config.ID].Status.Reachability == api.RaftMemberStatus_REACHABLE {
			return fmt.Errorf("expected node 3 to be unreachable")
		}
		return nil
	}))

	// Try to demote Node 2, this should fail because of the quorum safeguard
	r, err := ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: nodes[2].SecurityConfig.ClientTLSCreds.NodeID()})
	assert.NoError(t, err)
	spec := r.Node.Spec.Copy()
	spec.DesiredRole = api.NodeRoleWorker
	version := &r.Node.Meta.Version
	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{
		NodeID:      nodes[2].SecurityConfig.ClientTLSCreds.NodeID(),
		Spec:        spec,
		NodeVersion: version,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpc.Code(err))

	// Restart Node 3
	nodes[3] = raftutils.RestartNode(t, clockSource, nodes[3], false)
	raftutils.WaitForCluster(t, clockSource, nodes)

	// Node 3 should be listed as Reachable
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		members := nodes[1].GetMemberlist()
		if len(members) != 3 {
			return fmt.Errorf("expected 3 nodes, got %d", len(members))
		}
		if members[nodes[3].Config.ID].Status.Reachability == api.RaftMemberStatus_UNREACHABLE {
			return fmt.Errorf("expected node 3 to be reachable")
		}
		return nil
	}))

	raftMember := ts.Server.raft.GetMemberByNodeID(nodes[3].SecurityConfig.ClientTLSCreds.NodeID())
	assert.NotNil(t, raftMember)

	// Try to demote Node 3, this should succeed
	r, err = ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: nodes[3].SecurityConfig.ClientTLSCreds.NodeID()})
	assert.NoError(t, err)
	spec = r.Node.Spec.Copy()
	spec.DesiredRole = api.NodeRoleWorker
	version = &r.Node.Meta.Version
	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{
		NodeID:      nodes[3].SecurityConfig.ClientTLSCreds.NodeID(),
		Spec:        spec,
		NodeVersion: version,
	})
	assert.NoError(t, err)

	newCluster := map[uint64]*raftutils.TestNode{
		1: nodes[1],
		2: nodes[2],
	}

	ts.Server.raft.RemoveMember(context.Background(), raftMember.RaftID)

	raftutils.WaitForCluster(t, clockSource, newCluster)

	// Server should list 2 members
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		members := nodes[1].GetMemberlist()
		if len(members) != 2 {
			return fmt.Errorf("expected 2 nodes, got %d", len(members))
		}
		return nil
	}))

	demoteNode := nodes[2]
	lastNode := nodes[1]

	raftMember = ts.Server.raft.GetMemberByNodeID(demoteNode.SecurityConfig.ClientTLSCreds.NodeID())
	assert.NotNil(t, raftMember)

	// Try to demote a Node and scale down to 1
	r, err = ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: demoteNode.SecurityConfig.ClientTLSCreds.NodeID()})
	assert.NoError(t, err)
	spec = r.Node.Spec.Copy()
	spec.DesiredRole = api.NodeRoleWorker
	version = &r.Node.Meta.Version
	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{
		NodeID:      demoteNode.SecurityConfig.ClientTLSCreds.NodeID(),
		Spec:        spec,
		NodeVersion: version,
	})
	assert.NoError(t, err)

	ts.Server.raft.RemoveMember(context.Background(), raftMember.RaftID)

	// Update the server
	ts.Server.raft = lastNode.Node
	ts.Server.store = lastNode.MemoryStore()

	newCluster = map[uint64]*raftutils.TestNode{
		1: lastNode,
	}

	raftutils.WaitForCluster(t, clockSource, newCluster)

	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		members := lastNode.GetMemberlist()
		if len(members) != 1 {
			return fmt.Errorf("expected 1 node, got %d", len(members))
		}
		return nil
	}))

	// Make sure we can't demote the last manager.
	r, err = ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: lastNode.SecurityConfig.ClientTLSCreds.NodeID()})
	assert.NoError(t, err)
	spec = r.Node.Spec.Copy()
	spec.DesiredRole = api.NodeRoleWorker
	version = &r.Node.Meta.Version
	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{
		NodeID:      lastNode.SecurityConfig.ClientTLSCreds.NodeID(),
		Spec:        spec,
		NodeVersion: version,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpc.Code(err))

	// Propose a change in the spec and check if the remaining node can still process updates
	r, err = ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: lastNode.SecurityConfig.ClientTLSCreds.NodeID()})
	assert.NoError(t, err)
	spec = r.Node.Spec.Copy()
	spec.Availability = api.NodeAvailabilityDrain
	version = &r.Node.Meta.Version
	_, err = ts.Client.UpdateNode(context.Background(), &api.UpdateNodeRequest{
		NodeID:      lastNode.SecurityConfig.ClientTLSCreds.NodeID(),
		Spec:        spec,
		NodeVersion: version,
	})
	assert.NoError(t, err)

	// Get node information and check that the availability is set to drain
	r, err = ts.Client.GetNode(context.Background(), &api.GetNodeRequest{NodeID: lastNode.SecurityConfig.ClientTLSCreds.NodeID()})
	assert.NoError(t, err)
	assert.Equal(t, r.Node.Spec.Availability, api.NodeAvailabilityDrain)
}

func TestUpdateNodeDemote(t *testing.T) {
	t.Parallel()
	testUpdateNodeDemote(t)
}

// TestRemoveNodeAttachments tests the unexported removeNodeAttachments
// function. This avoids us having to update the TestRemoveNodes function to
// test all of this logic
func TestRemoveNodeAttachments(t *testing.T) {
	// first, set up a store and all that
	ts := newTestServer(t)
	defer ts.Stop()

	ts.Store.Update(func(tx store.Tx) error {
		store.CreateCluster(tx, &api.Cluster{
			ID: identity.NewID(),
			Spec: api.ClusterSpec{
				Annotations: api.Annotations{
					Name: store.DefaultClusterName,
				},
			},
		})
		return nil
	})

	// make sure before we start that our server is in a good (empty) state
	r, err := ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Empty(t, r.Nodes)

	// create a manager
	createNode(t, ts, "id1", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)
	r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Len(t, r.Nodes, 1)

	// create a worker. put it in the DOWN state, which is the state it will be
	// in to remove it anyway
	createNode(t, ts, "id2", api.NodeRoleWorker, api.NodeMembershipAccepted, api.NodeStatus_DOWN)
	r, err = ts.Client.ListNodes(context.Background(), &api.ListNodesRequest{})
	assert.NoError(t, err)
	assert.Len(t, r.Nodes, 2)

	// create a network we can "attach" to
	err = ts.Store.Update(func(tx store.Tx) error {
		n := &api.Network{
			ID: "net1id",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "net1name",
				},
				Attachable: true,
			},
		}
		return store.CreateNetwork(tx, n)
	})
	require.NoError(t, err)

	// create some tasks:
	err = ts.Store.Update(func(tx store.Tx) error {
		// 1.) A network attachment on the node we're gonna remove
		task1 := &api.Task{
			ID:           "task1",
			NodeID:       "id2",
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Attachment{
					Attachment: &api.NetworkAttachmentSpec{
						ContainerID: "container1",
					},
				},
				Networks: []*api.NetworkAttachmentConfig{
					{
						Target:    "net1id",
						Addresses: []string{}, // just leave this empty, we don't need it
					},
				},
			},
			// we probably don't care about the rest of the fields.
		}
		if err := store.CreateTask(tx, task1); err != nil {
			return err
		}

		// 2.) A network attachment on the node we're not going to remove
		task2 := &api.Task{
			ID:           "task2",
			NodeID:       "id1",
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Attachment{
					Attachment: &api.NetworkAttachmentSpec{
						ContainerID: "container2",
					},
				},
				Networks: []*api.NetworkAttachmentConfig{
					{
						Target:    "net1id",
						Addresses: []string{}, // just leave this empty, we don't need it
					},
				},
			},
			// we probably don't care about the rest of the fields.
		}
		if err := store.CreateTask(tx, task2); err != nil {
			return err
		}

		// 3.) A regular task on the node we're going to remove
		task3 := &api.Task{
			ID:           "task3",
			NodeID:       "id2",
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
			},
		}
		if err := store.CreateTask(tx, task3); err != nil {
			return err
		}

		// 4.) A regular task on the node we're not going to remove
		task4 := &api.Task{
			ID:           "task4",
			NodeID:       "id1",
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
			},
		}
		return store.CreateTask(tx, task4)
	})
	require.NoError(t, err)

	// Now, call the function with our nodeID. make sure it returns no error
	err = ts.Store.Update(func(tx store.Tx) error {
		return removeNodeAttachments(tx, "id2")
	})
	require.NoError(t, err)

	// Now, make sure only task1, the network-attacahed task on id2, was
	// removed
	ts.Store.View(func(tx store.ReadTx) {
		tasks, err := store.FindTasks(tx, store.All)
		require.NoError(t, err)
		// should only be 3 tasks left
		require.Len(t, tasks, 4)
		// and the list should not contain task1
		for _, task := range tasks {
			require.NotNil(t, task)
			if task.ID == "task1" {
				require.Equal(t, task.Status.State, api.TaskStateOrphaned)
			}
		}
	})
}
