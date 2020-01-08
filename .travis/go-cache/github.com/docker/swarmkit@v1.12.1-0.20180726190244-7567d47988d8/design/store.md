# Data store design

SwarmKit has an embedded data store for configuration and state. This store is
usually backed by the raft protocol, but is abstracted from the underlying
consensus protocol, and in principle could use other means to synchronize data
across the cluster. This document focuses on the design of the store itself,
such as the programmer-facing APIs and consistency guarantees, and does not
cover distributed consensus.

## Structure of stored data

The SwarmKit data store is built on top of go-memdb, which stores data in radix
trees.

There are separate tables for each data type, for example nodes, tasks, and so
on. Each table has its own set of indices, which always includes an ID index,
but may include other indices as well. For example, tasks can be indexed by
their service ID and node ID, among several other things.

Under the hood, go-memdb implements an index by adding keys for each index to
the radix tree, prefixed with the index's name. A single object in the data
store may have several keys corresponding to it, because it will have a
different key (and possibly multiple keys) within each index.

There are several advantages to using radix trees in this way. The first is that
it makes prefix matching easy. A second powerful feature of this design is
copy-on-write snapshotting. Since the radix tree consists of a hierarchy of
pointers, the root pointer always a fully consistent state at that moment in
time. Making a change to the tree involves replacing a leaf node with a new
value, and "bubbling up" that change to the root through the intermediate
pointers. To make the change visible to other readers, all it takes is a single
atomic pointer swap that replaces the root of the tree with a new root that
incorporates the changed nodes. The text below will discuss how this is used to
support transactions.

## Transactions

Code that uses the store can only use it inside a *transaction*. There are two
kinds of transactions: view transactions (read-only) and update transactions
(read/write).

A view transaction runs in a callback passed to the `View` method:

```
	s.View(func(tx store.ReadTx) {
		nodes, err = store.FindNodes(tx, store.All)
	})
```

This callback can call functions defined in the `store` package that retrieve
and list the various types of objects. `View` operates on an atomic snapshot of
the data store, so changes made while the callback is running won't be visible
to code inside the callback using the supplied `ReadTx`.

An update transaction works similarly, but provides the ability to create,
update, and delete objects:

```
	s.Update(func(tx store.Tx) error {
		t2 := &api.Task{
			ID: "testTaskID2",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			ServiceID:    "testServiceID2",
			DesiredState: api.TaskStateRunning,
		}
		return store.CreateTask(tx, t2)
	})
```

If the callback returns `nil`, the changes made inside the callback function are
committed atomically. If it returns any other error value, the transaction gets
rolled back. The changes are never visible to any other readers before the
commit happens, but they are visible to code inside the callback using the
`Tx` argument.

There is an exclusive lock for updates, so only one can happen at once. Take
care not to do expensive or blocking operations inside an `Update` callback.

## Batching

Sometimes it's necessary to create or update many objects in the store, but we
want to do this without holding the update lock for an arbitrarily long period
of time, or generating a huge set of changes from the transaction that would
need to be serialized in a Raft write. For this situation, the store provides
primitives to batch iterated operations that don't require atomicity into
transactions of an appropriate size.

Here is an example of a batch operation:

```
	err = d.store.Batch(func(batch *store.Batch) error {
		for _, n := range nodes {
			err := batch.Update(func(tx store.Tx) error {
				// check if node is still here
				node := store.GetNode(tx, n.ID)
				if node == nil {
					return nil
				}

				// [...]

				node.Status.State = api.NodeStatus_UNKNOWN
				node.Status.Message = `Node moved to "unknown" state due to leadership change in cluster`

				if err := d.nodes.AddUnknown(node, expireFunc); err != nil {
					return errors.Wrap(err, `adding node in "unknown" state to node store failed`)
				}
				if err := store.UpdateNode(tx, node); err != nil {
					return errors.Wrap(err, "update failed")
				}
				return nil
			})
			if err != nil {
				log.WithField("node", n.ID).WithError(err).Error(`failed to move node to "unknown" state`)
			}
		}
		return nil
	})
```

This is a slightly abbreviated version of code in the dispatcher that moves a
set of nodes to the "unknown" state. If there were many nodes in the system,
doing this inside a single Update transaction might block updates to the store
for a long time, or exceed the size limit of a serialized transaction. By using
`Batch`, the changes are automatically broken up into a set of transactions.

`Batch` takes a callback which generally contains a loop that iterates over a
set of objects. Every iteration can call `batch.Update` with another nested
callback that performs the actual changes. Changes performed inside a single
`batch.Update` call are guaranteed to land in the same transaction, and
therefore be applied atomically. However, changes different calls to
`batch.Update` may end up in different transactions.

## Watches

The data store provides a real-time feed of insertions, deletions, and
modifications. Any number of listeners can subscribe to this feed, optionally
applying filters to the set of events. This is very useful for building control
loops. For example, the orchestrators watch changes to services to trigger
reconciliation.

To start a watch, use the `state.Watch` function. The first argument is the
watch queue, which can be obtained with the store instance's `WatchQueue`
method. Extra arguments are events to be matched against the incoming event when
filtering. For example, this call returns only tasks creations, updates, and
deletions that affect a specific task ID:


```
	nodeTasks, err := store.Watch(s.WatchQueue(),
		api.EventCreateTask{Task: &api.Task{NodeID: nodeID},
			Checks: []api.TaskCheckFunc{api.TaskCheckNodeID}},
		api.EventUpdateTask{Task: &api.Task{NodeID: nodeID},
			Checks: []api.TaskCheckFunc{api.TaskCheckNodeID}},
		api.EventDeleteTask{Task: &api.Task{NodeID: nodeID},
			Checks: []api.TaskCheckFunc{api.TaskCheckNodeID}},
	)
```

There is also a `ViewAndWatch` method on the store that provides access to a
snapshot of the store at just before the moment the watch starts receiving
events. It guarantees that events following this snapshot won't be missed, and
events that are already incorporated in the snapshot won't be received.
`ViewAndWatch` involves holding the store update lock while its callback runs,
so it's preferable to use `View` and `Watch` separately instead if the use case
isn't sensitive to redundant events. `Watch` should be called before `View` so
that events aren't missed in between viewing a snapshot and starting the event
stream.

## Distributed operation

Data written to the store is automatically replicated to the other managers in
the cluster through the underlying consensus protocol. All active managers have
local in-memory copies of all the data in the store, accessible through
go-memdb.

The current consensus implementation, based on Raft, only allows writes to
happen on the leader. This avoids potentially conflicting writes ending up in
the log, which would have to be reconciled later on. The leader's copy of the
data in the store is the most up-to-date. Other nodes may lag behind this copy,
if there are replication delays, but will never diverge from it.

## Sequencer

It's important not to overwrite current data with stale data. In some
situations, we might want to take data from the store, hand it to the user, and
then write it back to the store with the user's modifications. The store has
a safeguard to make sure this fails if the data has been updated since the copy
was retrieved.

Every top-level object has a `Meta` field which contains a `Version` object. The
`Version` is managed automatically by the store. When an object is updated, its
`Version` field is increased to distinguish the old version from the new
version. Trying to update an object will fail if the object passed into an
update function has a `Version` which doesn't match the current `Version` of
that object in the store.

`Meta` also contains timestamps that are automatically updated by the store.

To keep version numbers consistent across the cluster, version numbers are
provided by the underlying consensus protocol through the `Proposer` interface.
In the case of the Raft consensus implementation, the version number is simply
the current Raft index at the time that the object was last updated. Note that
the index is queried before the change is actually written to Raft, so an object
created with `Version.Index = 5` would most likely be appended to the Raft log
at index 6.

The `Proposer` interface also provides the mechanism for the store code to
synchronize changes to the rest of the cluster. `ProposeValue` sends a set of
changes to the other managers in the cluster through the consensus protocol.

## RPC API

In addition to the Go API discussed above, the store exposes watches over gRPC.
There is a watch server that provides a very similar interface to the `Watch`
call. See `api/watch.proto` for the relevant protobuf definitions.

A full gRPC API for the store has been proposed, but not yet merged at the time
this document was written. See https://github.com/docker/swarmkit/pull/1998 for
draft code. In this proposal, the gRPC store API did not support full
transactions, but did allow creations and updates to happen in atomic sets.
Implementing full transactions over gRPC presents some challenges, because of
the store update lock. If a streaming RPC could hold the update lock, a
misbehaving client or severed network connection might cause this lock to be
held too long. Transactional APIs might need very short timeouts or other
safeguards.

The purpose of exposing an external gRPC API for the store would be to support
externally-implemented control loops. This would make swarmkit more extensible
because code that works with objects directly wouldn't need to be implemented
inside the swarmkit repository anymore.

## Generated code

For type safety, the store exposes type-safe helper functions such as
`DeleteNode` and `FindSecrets`. These functions wrap internal methods that are
not type-specific. However, providing these wrappers ended up involving a lot of
boilerplate code. There was also code that had to be duplicated for things like
saving and restoring snapshots of the store, defining events, and indexing
objects in the store.

To make this more manageable, a lot of store code is now automatically
generated by `protobuf/plugin/storeobject/storeobject.go`. It's now a lot easier
to add a new object type to the store. There is scope for further improvements
through code generation.

The plugin uses the presence of the `docker.protobuf.plugin.store_object` option
to detect top-level objects that can be stored inside the store. There is a
`watch_selectors` field inside this option that specifies which functions should
be generated for matching against specific fields of an object in a `Watch`
call.
