package watchapi

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestWatch(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	ctx := context.Background()

	// Watch for node creates
	watch, err := ts.Client.Watch(ctx, &api.WatchRequest{
		Entries: []*api.WatchRequest_WatchEntry{
			{
				Kind:   "node",
				Action: api.WatchActionKindCreate,
			},
		},
	})
	assert.NoError(t, err)

	// Should receive an initial message that indicates the watch is ready
	msg, err := watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, &api.WatchMessage{}, msg)

	createNode(t, ts, "id1", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)
	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, api.WatchActionKindCreate, msg.Events[0].Action)
	require.NotNil(t, msg.Events[0].Object.GetNode())
	assert.Equal(t, "id1", msg.Events[0].Object.GetNode().ID)

	watch.CloseSend()

	// Watch for node creates that match a name prefix and a custom index, or
	// are managers
	watch, err = ts.Client.Watch(ctx, &api.WatchRequest{
		Entries: []*api.WatchRequest_WatchEntry{
			{
				Kind:   "node",
				Action: api.WatchActionKindCreate,
				Filters: []*api.SelectBy{
					{
						By: &api.SelectBy_NamePrefix{
							NamePrefix: "east",
						},
					},
					{
						By: &api.SelectBy_Custom{
							Custom: &api.SelectByCustom{
								Index: "myindex",
								Value: "myval",
							},
						},
					},
				},
			},
			{
				Kind:   "node",
				Action: api.WatchActionKindCreate,
				Filters: []*api.SelectBy{
					{
						By: &api.SelectBy_Role{
							Role: api.NodeRoleManager,
						},
					},
				},
			},
		},
	})
	assert.NoError(t, err)

	// Should receive an initial message that indicates the watch is ready
	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, &api.WatchMessage{}, msg)

	createNode(t, ts, "id2", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)
	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, api.WatchActionKindCreate, msg.Events[0].Action)
	require.NotNil(t, msg.Events[0].Object.GetNode())
	assert.Equal(t, "id2", msg.Events[0].Object.GetNode().ID)

	// Shouldn't be seen by the watch
	createNode(t, ts, "id3", api.NodeRoleWorker, api.NodeMembershipAccepted, api.NodeStatus_READY)

	// Shouldn't be seen either - no hostname
	node := &api.Node{
		ID: "id4",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Indices: []api.IndexEntry{
					{Key: "myindex", Val: "myval"},
				},
			},
		},
		Role: api.NodeRoleWorker,
	}
	err = ts.Store.Update(func(tx store.Tx) error {
		return store.CreateNode(tx, node)
	})
	assert.NoError(t, err)

	// Shouldn't be seen either - hostname doesn't match filter
	node = &api.Node{
		ID: "id5",
		Description: &api.NodeDescription{
			Hostname: "west-40",
		},
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Indices: []api.IndexEntry{
					{Key: "myindex", Val: "myval"},
				},
			},
		},
		Role: api.NodeRoleWorker,
	}
	err = ts.Store.Update(func(tx store.Tx) error {
		return store.CreateNode(tx, node)
	})
	assert.NoError(t, err)

	// This one should be seen
	node = &api.Node{
		ID: "id6",
		Description: &api.NodeDescription{
			Hostname: "east-95",
		},
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Indices: []api.IndexEntry{
					{Key: "myindex", Val: "myval"},
				},
			},
		},
		Role: api.NodeRoleWorker,
	}
	err = ts.Store.Update(func(tx store.Tx) error {
		return store.CreateNode(tx, node)
	})
	assert.NoError(t, err)

	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, api.WatchActionKindCreate, msg.Events[0].Action)
	require.NotNil(t, msg.Events[0].Object.GetNode())
	assert.Equal(t, "id6", msg.Events[0].Object.GetNode().ID)

	watch.CloseSend()
}

func TestWatchMultipleActions(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	ctx := context.Background()

	// Watch for node creates
	watch, err := ts.Client.Watch(ctx, &api.WatchRequest{
		Entries: []*api.WatchRequest_WatchEntry{
			{
				Kind:   "node",
				Action: api.WatchActionKindCreate | api.WatchActionKindRemove,
			},
		},
	})
	assert.NoError(t, err)

	// Should receive an initial message that indicates the watch is ready
	msg, err := watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, &api.WatchMessage{}, msg)

	createNode(t, ts, "id1", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)
	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, api.WatchActionKindCreate, msg.Events[0].Action)
	require.NotNil(t, msg.Events[0].Object.GetNode())
	assert.Equal(t, "id1", msg.Events[0].Object.GetNode().ID)

	// Update should not be seen
	err = ts.Store.Update(func(tx store.Tx) error {
		node := store.GetNode(tx, "id1")
		require.NotNil(t, node)
		node.Role = api.NodeRoleWorker
		return store.UpdateNode(tx, node)
	})
	assert.NoError(t, err)

	// Delete should be seen
	err = ts.Store.Update(func(tx store.Tx) error {
		return store.DeleteNode(tx, "id1")
	})
	assert.NoError(t, err)
	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, api.WatchActionKindRemove, msg.Events[0].Action)
	require.NotNil(t, msg.Events[0].Object.GetNode())
	assert.Equal(t, "id1", msg.Events[0].Object.GetNode().ID)

	watch.CloseSend()
}

func TestWatchIncludeOldObject(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	ctx := context.Background()

	// Watch for node updates
	watch, err := ts.Client.Watch(ctx, &api.WatchRequest{
		Entries: []*api.WatchRequest_WatchEntry{
			{
				Kind:   "node",
				Action: api.WatchActionKindUpdate,
			},
		},
		IncludeOldObject: true,
	})
	assert.NoError(t, err)

	// Should receive an initial message that indicates the watch is ready
	msg, err := watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, &api.WatchMessage{}, msg)

	createNode(t, ts, "id1", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)

	err = ts.Store.Update(func(tx store.Tx) error {
		node := store.GetNode(tx, "id1")
		require.NotNil(t, node)
		node.Role = api.NodeRoleWorker
		return store.UpdateNode(tx, node)
	})
	assert.NoError(t, err)

	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, api.WatchActionKindUpdate, msg.Events[0].Action)
	require.NotNil(t, msg.Events[0].Object.GetNode())
	assert.Equal(t, "id1", msg.Events[0].Object.GetNode().ID)
	assert.Equal(t, api.NodeRoleWorker, msg.Events[0].Object.GetNode().Role)
	require.NotNil(t, msg.Events[0].OldObject.GetNode())
	assert.Equal(t, "id1", msg.Events[0].OldObject.GetNode().ID)
	assert.Equal(t, api.NodeRoleManager, msg.Events[0].OldObject.GetNode().Role)

	watch.CloseSend()
}

func TestWatchResumeFrom(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	ctx := context.Background()

	createNode(t, ts, "id1", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)
	node2 := createNode(t, ts, "id2", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)

	// Watch for node creates, starting from after the first node creation.
	watch, err := ts.Client.Watch(ctx, &api.WatchRequest{
		Entries: []*api.WatchRequest_WatchEntry{
			{
				Kind:   "node",
				Action: api.WatchActionKindCreate,
			},
		},
		ResumeFrom: &node2.Meta.Version,
	})
	assert.NoError(t, err)

	// Should receive an initial message that indicates the watch is ready
	msg, err := watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, &api.WatchMessage{}, msg)

	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, api.WatchActionKindCreate, msg.Events[0].Action)
	require.NotNil(t, msg.Events[0].Object.GetNode())
	assert.Equal(t, "id2", msg.Events[0].Object.GetNode().ID)
	assert.Equal(t, node2.Meta.Version.Index+3, msg.Version.Index)

	// Create a new node
	node3 := createNode(t, ts, "id3", api.NodeRoleManager, api.NodeMembershipAccepted, api.NodeStatus_READY)

	msg, err = watch.Recv()
	assert.NoError(t, err)
	assert.Equal(t, api.WatchActionKindCreate, msg.Events[0].Action)
	require.NotNil(t, msg.Events[0].Object.GetNode())
	assert.Equal(t, "id3", msg.Events[0].Object.GetNode().ID)
	assert.Equal(t, node3.Meta.Version.Index+3, msg.Version.Index)

	watch.CloseSend()
}
