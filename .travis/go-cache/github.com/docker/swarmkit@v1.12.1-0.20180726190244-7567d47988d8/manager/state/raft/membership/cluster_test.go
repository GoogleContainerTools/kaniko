package membership_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc/grpclog"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/docker/swarmkit/api"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/manager/state/raft"
	"github.com/docker/swarmkit/manager/state/raft/membership"
	raftutils "github.com/docker/swarmkit/manager/state/raft/testutils"
	"github.com/docker/swarmkit/testutils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var tc *cautils.TestCA

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, ioutil.Discard))
	logrus.SetOutput(ioutil.Discard)
}

func TestMain(m *testing.M) {
	tc = cautils.NewTestCA(nil)
	res := m.Run()
	tc.Stop()
	os.Exit(res)
}

func newTestMember(id uint64) *membership.Member {
	return &membership.Member{
		RaftMember: &api.RaftMember{RaftID: id},
	}
}

func newTestCluster(members []*membership.Member, removed []*membership.Member) *membership.Cluster {
	c := membership.NewCluster()
	for _, m := range members {
		c.AddMember(m)
	}
	for _, m := range removed {
		c.AddMember(m)
		c.RemoveMember(m.RaftID)
	}
	return c
}

func TestClusterMember(t *testing.T) {
	members := []*membership.Member{
		newTestMember(1),
		newTestMember(2),
	}
	tests := []struct {
		id    uint64
		match bool
	}{
		{1, true},
		{2, true},
		{3, false},
	}
	for i, tt := range tests {
		c := newTestCluster(members, nil)
		m := c.GetMember(tt.id)
		if g := m != nil; g != tt.match {
			t.Errorf("#%d: find member = %v, want %v", i, g, tt.match)
		}
		if m != nil && m.RaftID != tt.id {
			t.Errorf("#%d: id = %x, want %x", i, m.RaftID, tt.id)
		}
	}
}

func TestMembers(t *testing.T) {
	cls := membership.NewCluster()
	defer cls.Clear()
	cls.AddMember(&membership.Member{RaftMember: &api.RaftMember{RaftID: 1}})
	cls.AddMember(&membership.Member{RaftMember: &api.RaftMember{RaftID: 5}})
	cls.AddMember(&membership.Member{RaftMember: &api.RaftMember{RaftID: 20}})
	cls.AddMember(&membership.Member{RaftMember: &api.RaftMember{RaftID: 50}})
	cls.AddMember(&membership.Member{RaftMember: &api.RaftMember{RaftID: 10}})

	assert.Len(t, cls.Members(), 5)
}

func TestGetMember(t *testing.T) {
	members := []*membership.Member{
		newTestMember(1),
	}
	removed := []*membership.Member{
		newTestMember(2),
	}
	cls := newTestCluster(members, removed)

	m := cls.GetMember(1)
	assert.NotNil(t, m)
	assert.Equal(t, m.RaftID, uint64(1))

	m = cls.GetMember(2)
	assert.Nil(t, m)

	m = cls.GetMember(3)
	assert.Nil(t, m)
}

func TestClusterAddMember(t *testing.T) {
	members := []*membership.Member{
		newTestMember(1),
	}
	removed := []*membership.Member{
		newTestMember(2),
	}
	cls := newTestCluster(members, removed)

	// Cannot add a node present in the removed set
	err := cls.AddMember(&membership.Member{RaftMember: &api.RaftMember{RaftID: 2}})
	assert.Error(t, err)
	assert.Equal(t, err, membership.ErrIDRemoved)
	assert.Nil(t, cls.GetMember(2))

	err = cls.AddMember(&membership.Member{RaftMember: &api.RaftMember{RaftID: 3}})
	assert.NoError(t, err)
	assert.NotNil(t, cls.GetMember(3))
}

func TestClusterRemoveMember(t *testing.T) {
	members := []*membership.Member{
		newTestMember(1),
	}
	removed := []*membership.Member{
		newTestMember(2),
	}
	cls := newTestCluster(members, removed)

	// Can remove a node whose ID is not yet in the member list
	err := cls.RemoveMember(3)
	assert.NoError(t, err)
	assert.Nil(t, cls.GetMember(3))

	err = cls.RemoveMember(1)
	assert.NoError(t, err)
	assert.Nil(t, cls.GetMember(1))
}

func TestIsIDRemoved(t *testing.T) {
	members := []*membership.Member{
		newTestMember(1),
	}
	removed := []*membership.Member{
		newTestMember(2),
	}
	cls := newTestCluster(members, removed)

	assert.False(t, cls.IsIDRemoved(1))
	assert.True(t, cls.IsIDRemoved(2))
}

func TestClear(t *testing.T) {
	members := []*membership.Member{
		newTestMember(1),
		newTestMember(2),
		newTestMember(3),
	}
	removed := []*membership.Member{
		newTestMember(4),
		newTestMember(5),
		newTestMember(6),
	}
	cls := newTestCluster(members, removed)

	cls.Clear()
	assert.Equal(t, len(cls.Members()), 0)
	assert.Equal(t, len(cls.Removed()), 0)
}

func TestValidateConfigurationChange(t *testing.T) {
	members := []*membership.Member{
		newTestMember(1),
		newTestMember(2),
		newTestMember(3),
	}
	removed := []*membership.Member{
		newTestMember(4),
		newTestMember(5),
		newTestMember(6),
	}
	cls := newTestCluster(members, removed)

	m := &api.RaftMember{RaftID: 1}
	existingMember, err := m.Marshal()
	assert.NoError(t, err)
	assert.NotNil(t, existingMember)

	m = &api.RaftMember{RaftID: 7}
	newMember, err := m.Marshal()
	assert.NoError(t, err)
	assert.NotNil(t, newMember)

	m = &api.RaftMember{RaftID: 4}
	removedMember, err := m.Marshal()
	assert.NoError(t, err)
	assert.NotNil(t, removedMember)

	n := &api.Node{}
	node, err := n.Marshal()
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// Add node but ID exists
	cc := raftpb.ConfChange{ID: 1, Type: raftpb.ConfChangeAddNode, NodeID: 1, Context: existingMember}
	err = cls.ValidateConfigurationChange(cc)
	assert.Error(t, err)
	assert.Equal(t, err, membership.ErrIDExists)

	// Any configuration change but ID in remove set
	cc = raftpb.ConfChange{ID: 4, Type: raftpb.ConfChangeAddNode, NodeID: 4, Context: removedMember}
	err = cls.ValidateConfigurationChange(cc)
	assert.Error(t, err)
	assert.Equal(t, err, membership.ErrIDRemoved)

	// Remove Node but ID not found in memberlist
	cc = raftpb.ConfChange{ID: 7, Type: raftpb.ConfChangeRemoveNode, NodeID: 7, Context: newMember}
	err = cls.ValidateConfigurationChange(cc)
	assert.Error(t, err)
	assert.Equal(t, err, membership.ErrIDNotFound)

	// Update Node but ID not found in memberlist
	cc = raftpb.ConfChange{ID: 7, Type: raftpb.ConfChangeUpdateNode, NodeID: 7, Context: newMember}
	err = cls.ValidateConfigurationChange(cc)
	assert.Error(t, err)
	assert.Equal(t, err, membership.ErrIDNotFound)

	// Any configuration change but can't unmarshal config
	cc = raftpb.ConfChange{ID: 7, Type: raftpb.ConfChangeAddNode, NodeID: 7, Context: []byte("abcdef")}
	err = cls.ValidateConfigurationChange(cc)
	assert.Error(t, err)
	assert.Equal(t, err, membership.ErrCannotUnmarshalConfig)

	// Invalid configuration change
	cc = raftpb.ConfChange{ID: 1, Type: 10, NodeID: 1, Context: newMember}
	err = cls.ValidateConfigurationChange(cc)
	assert.Error(t, err)
	assert.Equal(t, err, membership.ErrConfigChangeInvalid)
}

func TestCanRemoveMember(t *testing.T) {
	nodes, clockSource := raftutils.NewRaftCluster(t, tc)
	defer raftutils.TeardownCluster(nodes)

	// Stop node 2 and node 3 (2 nodes out of 3)
	nodes[2].Server.Stop()
	nodes[2].ShutdownRaft()
	nodes[3].Server.Stop()
	nodes[3].ShutdownRaft()

	// Node 2 and Node 3 should be listed as Unreachable
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		members := nodes[1].GetMemberlist()
		if len(members) != 3 {
			return fmt.Errorf("expected 3 nodes, got %d", len(members))
		}
		if members[nodes[2].Config.ID].Status.Reachability == api.RaftMemberStatus_REACHABLE {
			return errors.New("expected node 2 to be unreachable")
		}
		if members[nodes[3].Config.ID].Status.Reachability == api.RaftMemberStatus_REACHABLE {
			return errors.New("expected node 3 to be unreachable")
		}
		return nil
	}))

	// Removing nodes at this point fails because we lost quorum
	for i := 1; i <= 3; i++ {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		err := nodes[1].RemoveMember(ctx, uint64(i))
		assert.Error(t, err)
		members := nodes[1].GetMemberlist()
		assert.Equal(t, len(members), 3)
	}

	// Restart node 2 and node 3
	nodes[2] = raftutils.RestartNode(t, clockSource, nodes[2], false)
	nodes[3] = raftutils.RestartNode(t, clockSource, nodes[3], false)
	raftutils.WaitForCluster(t, clockSource, nodes)

	var leader uint64
	leaderIndex := func() uint64 {
		for i, n := range nodes {
			if n.Config.ID == n.Leader() {
				return i
			}
		}
		return 0
	}

	// Node 2 and Node 3 should be listed as Reachable
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		leader = leaderIndex()
		if leader == 0 {
			return errors.New("no leader")
		}
		members := nodes[leader].GetMemberlist()
		if len(members) != 3 {
			return fmt.Errorf("expected 3 nodes, got %d", len(members))
		}
		if members[nodes[2].Config.ID].Status.Reachability == api.RaftMemberStatus_UNREACHABLE {
			return errors.New("expected node 2 to be reachable")
		}
		if members[nodes[3].Config.ID].Status.Reachability == api.RaftMemberStatus_UNREACHABLE {
			return errors.New("expected node 3 to be reachable")
		}
		return nil
	}))

	// Stop Node 3 (1 node out of 3)
	nodes[3].Server.Stop()
	nodes[3].ShutdownRaft()

	// Node 3 should be listed as Unreachable
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		leader = leaderIndex()
		if leader == 0 {
			return errors.New("no leader")
		}
		members := nodes[leader].GetMemberlist()
		if len(members) != 3 {
			return fmt.Errorf("expected 3 nodes, got %d", len(members))
		}
		if members[nodes[3].Config.ID].Status.Reachability == api.RaftMemberStatus_REACHABLE {
			return errors.New("expected node 3 to be unreachable")
		}
		return nil
	}))

	// Removing node 2 should fail (this would break the quorum)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := nodes[leader].RemoveMember(ctx, nodes[2].Config.ID)
	assert.EqualError(t, err, raft.ErrCannotRemoveMember.Error())
	members := nodes[leader].GetMemberlist()
	assert.Equal(t, len(members), 3)

	// Removing node 3 works fine because it is already unreachable
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	err = nodes[leader].RemoveMember(ctx, nodes[3].Config.ID)
	assert.NoError(t, err)
	members = nodes[leader].GetMemberlist()
	assert.Nil(t, members[nodes[3].Config.ID])
	assert.Equal(t, len(members), 2)

	// Add back node 3
	raftutils.ShutdownNode(nodes[3])
	nodes[3] = raftutils.NewJoinNode(t, clockSource, nodes[leader].Address, tc)
	raftutils.WaitForCluster(t, clockSource, nodes)

	// Node 2 and Node 3 should be listed as Reachable
	assert.NoError(t, testutils.PollFunc(clockSource, func() error {
		leader = leaderIndex()
		if leader == 0 {
			return errors.New("no leader")
		}
		members := nodes[leader].GetMemberlist()
		if len(members) != 3 {
			return fmt.Errorf("expected 3 nodes, got %d", len(members))
		}
		if members[nodes[2].Config.ID].Status.Reachability != api.RaftMemberStatus_REACHABLE {
			return errors.New("expected node 2 to be reachable")
		}
		if members[nodes[3].Config.ID].Status.Reachability != api.RaftMemberStatus_REACHABLE {
			return errors.New("expected node 3 to be reachable")
		}
		return nil
	}))

	// Removing node 3 should succeed
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	err = nodes[leader].RemoveMember(ctx, nodes[3].Config.ID)
	assert.NoError(t, err)
	members = nodes[leader].GetMemberlist()
	assert.Nil(t, members[nodes[3].Config.ID])
	assert.Equal(t, len(members), 2)

	// Removing node 2 should succeed
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	err = nodes[leader].RemoveMember(ctx, nodes[2].Config.ID)
	assert.NoError(t, err)
	members = nodes[leader].GetMemberlist()
	assert.Nil(t, members[nodes[2].Config.ID])
	assert.Equal(t, len(members), 1)
}
