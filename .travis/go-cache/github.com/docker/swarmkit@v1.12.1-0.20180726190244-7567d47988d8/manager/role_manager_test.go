package manager

import (
	"errors"
	"testing"

	"github.com/docker/swarmkit/api"
	cautils "github.com/docker/swarmkit/ca/testutils"
	raftutils "github.com/docker/swarmkit/manager/state/raft/testutils"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/testutils"
	"github.com/stretchr/testify/require"
)

// While roleManager is running, if a node is demoted, it is removed from the raft cluster.  If a node is
// promoted, it is not added to the cluster but its observed role will change to manager.
func TestRoleManagerRemovesDemotedNodesAndAddsPromotedNodes(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(nil)
	defer tc.Stop()

	nodes, fc := raftutils.NewRaftCluster(t, tc)
	defer raftutils.TeardownCluster(nodes)

	// nodes is not a list, but a map.  The IDs are 1, 2, 3
	require.Len(t, nodes[1].GetMemberlist(), 3)

	// create node objects in the memory store
	for _, node := range nodes {
		s := raftutils.Leader(nodes).MemoryStore()
		// Create a new node object
		require.NoError(t, s.Update(func(tx store.Tx) error {
			return store.CreateNode(tx, &api.Node{
				Role: api.NodeRoleManager,
				ID:   node.SecurityConfig.ClientTLSCreds.NodeID(),
				Spec: api.NodeSpec{
					DesiredRole:  api.NodeRoleManager,
					Membership:   api.NodeMembershipAccepted,
					Availability: api.NodeAvailabilityActive,
				},
			})
		}))
	}

	lead := raftutils.Leader(nodes)
	var nonLead *raftutils.TestNode
	for _, n := range nodes {
		if n != lead {
			nonLead = n
			break
		}
	}
	rm := newRoleManager(lead.MemoryStore(), lead.Node)
	rm.clocksource = fc
	go rm.Run(tc.Context)
	defer rm.Stop()

	// demote the node
	require.NoError(t, lead.MemoryStore().Update(func(tx store.Tx) error {
		n := store.GetNode(tx, nonLead.SecurityConfig.ClientTLSCreds.NodeID())
		n.Spec.DesiredRole = api.NodeRoleWorker
		return store.UpdateNode(tx, n)
	}))
	require.NoError(t, testutils.PollFuncWithTimeout(fc, func() error {
		memberlist := lead.GetMemberlist()
		if len(memberlist) != 2 {
			return errors.New("raft node hasn't been removed yet")
		}
		for _, m := range memberlist {
			if m.NodeID == nonLead.SecurityConfig.ClientTLSCreds.NodeID() {
				return errors.New("wrong member was removed")
			}
		}
		// use Update just because it returns an error
		return lead.MemoryStore().Update(func(tx store.Tx) error {
			if n := store.GetNode(tx, nonLead.SecurityConfig.ClientTLSCreds.NodeID()); n.Role != api.NodeRoleWorker {
				return errors.New("raft node hasn't been marked as a worker yet")
			}
			return nil
		})
	}, roleReconcileInterval/2))

	// now promote the node
	require.NoError(t, lead.MemoryStore().Update(func(tx store.Tx) error {
		n := store.GetNode(tx, nonLead.SecurityConfig.ClientTLSCreds.NodeID())
		n.Spec.DesiredRole = api.NodeRoleManager
		return store.UpdateNode(tx, n)
	}))
	require.NoError(t, testutils.PollFuncWithTimeout(fc, func() error {
		if len(lead.GetMemberlist()) != 2 {
			return errors.New("raft nodes in membership should not have changed")
		}
		// use Update just because it returns an error
		return lead.MemoryStore().Update(func(tx store.Tx) error {
			if n := store.GetNode(tx, nonLead.SecurityConfig.ClientTLSCreds.NodeID()); n.Role != api.NodeRoleManager {
				return errors.New("raft node hasn't been marked as a manager yet")
			}
			return nil
		})
	}, roleReconcileInterval/2))
}

// If a node was demoted before the roleManager starts up, roleManger will remove
// the node from the cluster membership.
func TestRoleManagerRemovesDemotedNodesOnStartup(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(nil)
	defer tc.Stop()

	nodes, fc := raftutils.NewRaftCluster(t, tc)
	defer raftutils.TeardownCluster(nodes)

	// nodes is not a list, but a map.  The IDs are 1, 2, 3
	require.Len(t, nodes[1].GetMemberlist(), 3)

	// create node objects in the memory store
	for i, node := range nodes {
		s := raftutils.Leader(nodes).MemoryStore()
		desired := api.NodeRoleManager
		if i == 3 {
			desired = api.NodeRoleWorker
		}
		// Create a new node object
		require.NoError(t, s.Update(func(tx store.Tx) error {
			return store.CreateNode(tx, &api.Node{
				Role: api.NodeRoleManager,
				ID:   node.SecurityConfig.ClientTLSCreds.NodeID(),
				Spec: api.NodeSpec{
					DesiredRole:  desired,
					Membership:   api.NodeMembershipAccepted,
					Availability: api.NodeAvailabilityActive,
				},
			})
		}))
	}
	demoted := nodes[3]

	lead := raftutils.Leader(nodes)
	rm := newRoleManager(lead.MemoryStore(), lead.Node)
	rm.clocksource = fc
	go rm.Run(tc.Context)
	defer rm.Stop()

	require.NoError(t, testutils.PollFuncWithTimeout(fc, func() error {
		memberlist := lead.GetMemberlist()
		if len(memberlist) != 2 {
			return errors.New("raft node hasn't been removed yet")
		}
		for _, m := range memberlist {
			if m.NodeID == demoted.SecurityConfig.ClientTLSCreds.NodeID() {
				return errors.New("wrong member was removed")
			}
		}
		// use Update just because it returns an error
		return lead.MemoryStore().Update(func(tx store.Tx) error {
			if n := store.GetNode(tx, demoted.SecurityConfig.ClientTLSCreds.NodeID()); n.Role != api.NodeRoleWorker {
				return errors.New("raft node hasn't been marked as a worker yet")
			}
			return nil
		})
	}, roleReconcileInterval/2))
}

// While roleManager is running, if a node is deleted, it is removed from the raft cluster.
func TestRoleManagerRemovesDeletedNodes(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(nil)
	defer tc.Stop()

	nodes, fc := raftutils.NewRaftCluster(t, tc)
	defer raftutils.TeardownCluster(nodes)

	// nodes is not a list, but a map.  The IDs are 1, 2, 3
	require.Len(t, nodes[1].GetMemberlist(), 3)

	// create node objects in the memory store
	for _, node := range nodes {
		s := raftutils.Leader(nodes).MemoryStore()
		// Create a new node object
		require.NoError(t, s.Update(func(tx store.Tx) error {
			return store.CreateNode(tx, &api.Node{
				Role: api.NodeRoleManager,
				ID:   node.SecurityConfig.ClientTLSCreds.NodeID(),
				Spec: api.NodeSpec{
					DesiredRole:  api.NodeRoleManager,
					Membership:   api.NodeMembershipAccepted,
					Availability: api.NodeAvailabilityActive,
				},
			})
		}))
	}

	lead := raftutils.Leader(nodes)
	var nonLead *raftutils.TestNode
	for _, n := range nodes {
		if n != lead {
			nonLead = n
			break
		}
	}
	rm := newRoleManager(lead.MemoryStore(), lead.Node)
	rm.clocksource = fc
	go rm.Run(tc.Context)
	defer rm.Stop()

	// delete the node
	require.NoError(t, lead.MemoryStore().Update(func(tx store.Tx) error {
		return store.DeleteNode(tx, nonLead.SecurityConfig.ClientTLSCreds.NodeID())
	}))
	require.NoError(t, testutils.PollFuncWithTimeout(fc, func() error {
		memberlist := lead.GetMemberlist()
		if len(memberlist) != 2 {
			return errors.New("raft node hasn't been removed yet")
		}
		for _, m := range memberlist {
			if m.NodeID == nonLead.SecurityConfig.ClientTLSCreds.NodeID() {
				return errors.New("wrong member was removed")
			}
		}
		return nil
	}, roleReconcileInterval/2))

}

// If a node was removed before the roleManager starts up, roleManger will remove
// the node from the cluster membership.
func TestRoleManagerRemovesDeletedNodesOnStartup(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(nil)
	defer tc.Stop()

	nodes, fc := raftutils.NewRaftCluster(t, tc)
	defer raftutils.TeardownCluster(nodes)

	// nodes is not a list, but a map.  The IDs are 1, 2, 3
	require.Len(t, nodes[1].GetMemberlist(), 3)

	// create node objects in the memory store
	for i, node := range nodes {
		s := raftutils.Leader(nodes).MemoryStore()
		if i == 3 {
			continue
		}
		// Create a new node object
		require.NoError(t, s.Update(func(tx store.Tx) error {
			return store.CreateNode(tx, &api.Node{
				Role: api.NodeRoleManager,
				ID:   node.SecurityConfig.ClientTLSCreds.NodeID(),
				Spec: api.NodeSpec{
					DesiredRole:  api.NodeRoleManager,
					Membership:   api.NodeMembershipAccepted,
					Availability: api.NodeAvailabilityActive,
				},
			})
		}))
	}

	lead := raftutils.Leader(nodes)
	rm := newRoleManager(lead.MemoryStore(), lead.Node)
	rm.clocksource = fc
	go rm.Run(tc.Context)
	defer rm.Stop()

	require.NoError(t, testutils.PollFuncWithTimeout(fc, func() error {
		memberlist := lead.GetMemberlist()
		if len(memberlist) != 2 {
			return errors.New("raft node hasn't been removed yet")
		}
		for _, m := range memberlist {
			if m.NodeID == nodes[3].SecurityConfig.ClientTLSCreds.NodeID() {
				return errors.New("wrong member was removed")
			}
		}
		return nil
	}, roleReconcileInterval/2))
}
