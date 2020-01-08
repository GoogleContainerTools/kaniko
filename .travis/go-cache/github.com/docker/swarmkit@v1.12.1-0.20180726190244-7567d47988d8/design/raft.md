# Raft implementation

SwarmKit uses the Raft consensus protocol to synchronize state between manager
nodes and support high availability. The lowest level portions of this are
provided by the `github.com/coreos/etcd/raft` package. SwarmKit's
`github.com/docker/swarmkit/manager/state/raft` package builds a complete
solution on top of this, adding things like saving and loading state on disk,
an RPC layer so nodes can pass Raft messages over a network, and dynamic cluster
membership.

## A quick review of Raft

The details of the Raft protocol are outside the scope of this document, but
it's well worth reviewing the [raft paper](https://raft.github.io/raft.pdf).

Essentially, Raft gives us two things. It provides the mechanism to elect a
leader, which serves as the arbiter or all consensus decisions. It also provides
a distributed log that we can append entries to, subject to the leader's
approval. The distributed log is the basic building block for agreeing on and
distributing state. Once an entry in the log becomes *committed*, it becomes an
immutable part of the log that will survive any future leader elections and
changes to the cluster. We can think of a committed log entry as piece of state
that the cluster has reached agreement on.

## Role of the leader

The leader has special responsibilities in the Raft protocol, but we also assign
it special functions in SwarmKit outside the context of Raft. For example, the
scheduler, orchestrators, dispatcher, and CA run on the leader node. This is not
a design requirement, but simplifies things somewhat. If these components ran in
a distributed fashion, we would need some mechanism to resolve conflicts between
writes made by different nodes. Limiting decision-making to the leader avoids
the need for this, since we can be certain that there is at most one leader at
any time. The leader is also guaranteed to have the most up-to-date data in its
store, so it is best positioned to make decisions.

The basic rule is that anything which writes to the Raft-backed data store needs
to run on the leader. If a follower node tries to write to the data store, the
write will fail. Writes will also fail on a node that starts out as the leader
but loses its leadership position before the write finishes.

## Raft IDs vs. node IDs

Nodes in SwarmKit are identified by alphanumeric strings, but `etcd/raft` uses
integers to identify Raft nodes. Thus, managers have two distinct IDs. The Raft
IDs are assigned dynamically when a node joins the Raft consensus group. A node
could potentially leave the Raft consensus group (through demotion), then later
get promoted and rejoin under a different Raft ID. In this case, the node ID
would stay the same, because it's a cryptographically-verifiable property of the
node's certificate, but the Raft ID is assigned arbitrarily and would change.

It's important to note that a Raft ID can't be reused after a node that was
using the ID leaves the consensus group. These Raft IDs of nodes that are no
longer part of the cluster are saved (persisted on disk) in a list (a blacklist,
if you will) to make sure they aren't reused. If a node with a Raft ID on this list
tries to use Raft RPCs, other nodes won't honor these requests. etcd/raft doesn't allow
reuse of raft Id, which is likely done to avoid ambiguity.

The blacklist of demoted/removed nodes is used to restrict these nodes from
communicating and affecting cluster state. A membership list is also persisted,
however this does not restrict communication between nodes.
This is done to favor stability (and availability, by enabling faster return to
non-degraded state) over consistency, by allowing newly added nodes (which may not
have propagated to all the raft group members) to join and communicate with the group
even though the membership list may not consistent at the point in time (but eventually
will be). In case of node demotion/removal from the group, the affected node may be able
to communicate with the other members until the change is fully propagated.

## Logs and snapshots

There are two sets of files on disk that provide persistent state for Raft.
There is a set of WAL (write-ahead log files). These store a series of log
entries and Raft metadata, such as the current term, index, and committed index.
WAL files are automatically rotated when they reach a certain size.

To avoid having to retain every entry in the history of the log, snapshots
serialize a view of the state at a particular point in time. After a snapshot
gets taken, logs that predate the snapshot are no longer necessary, because the
snapshot captures all the information that's needed from the log up to that
point. The number of old snapshots and WALs to retain is configurable.

In SwarmKit's usage, WALs mostly contain protobuf-serialized data store
modifications. A log entry can contain a batch of creations, updates, and
deletions of objects from the data store. Some log entries contain other kinds
of metadata, like node additions or removals. Snapshots contain a complete dump
of the store, as well as any metadata from the log entries that needs to be
preserved. The saved metadata includes the Raft term and index, a list of nodes
in the cluster, and a list of nodes that have been removed from the cluster.

WALs and snapshots are both stored encrypted, even if the autolock feature is
disabled. With autolock turned off, the data encryption key is stored on disk in
plaintext, in a header inside the TLS key. When autolock is turned on, the data
encryption key is encrypted with a key encryption key.

## Initializing a Raft cluster

The first manager of a cluster (`swarm init`) assigns itself a random Raft ID.
It creates a new WAL with its own Raft identity stored in the metadata field.
The metadata field is the only part of the WAL that differs between nodes. By
storing information such as the local Raft ID, it's easy to restore this
node-specific information after a restart. In principle it could be stored in a
separate file, but embedding it inside the WAL is most convenient.

The node then starts the Raft state machine. From this point, it's a fully
functional single-node Raft instance. Writes to the data store actually go
through Raft, though this is a trivial case because reaching consensus doesn't
involve communicating with any other nodes. The `Run` loop sees these writes and
serializes them to disk as requested by the `etcd/raft` package.

## Adding and removing nodes

New nodes can join an existing Raft consensus group by invoking the `Join` RPC
on the leader node. This corresponds to joining a swarm with a manager-level
token, or promoting a worker node to a manager. If successful, `Join` returns a
Raft ID for the new node and a list of other members of the consensus group.

On the leader side, `Join` tries to append a configuration change entry to the
Raft log, and waits until that entry becomes committed.

A new node creates an empty Raft log with its own node information in the
metadata field. Then it starts the state machine. By running the Raft consensus
protocol, the leader will discover that the new node doesn't have any entries in
its log, and will synchronize these entries to the new node through some
combination of sending snapshots and log entries. It can take a little while for
a new node to become a functional member of the consensus group, because it
needs to receive this data first.

On the node receiving the log, code watching changes to the store will see log
entries replayed as if the changes to the store were happening at that moment.
This doesn't just apply when nodes receive logs for the first time - in
general, when followers receive log entries with changes to the store, those
are replayed in the follower's data store.

Removing a node through demotion is a bit different. This requires two
coordinated changes: the node must renew its certificate to get a worker
certificate, and it should also be cleanly removed from the Raft consensus
group. To avoid inconsistent states, particularly in cases like demoting the
leader, there is a reconciliation loop that handles this in
`manager/role_manager.go`. To initiate demotion, the user changes a node's
`DesiredRole` to `Worker`. The role manager detects any nodes that have been
demoted but are still acting as managers, and first removes them from the
consensus group by calling `RemoveMember`. Only once this has happened is it
safe to change the `Role` field to get a new certificate issued, because issuing
a worker certificate to a node participating in the Raft group could cause loss
of quorum.

`RemoveMember` works similarly to `Join`. It appends an entry to the Raft log
removing the member from the consensus group, and waits until this entry becomes
committed. Once a member is removed, its Raft ID can never be reused.

There is a special case when the leader is being demoted. It cannot reliably
remove itself, because this involves informing the other nodes that the removal
log entry has been committed, and if any of those messages are lost in transit,
the leader won't have an opportunity to retry sending them, since demotion
causes the Raft state machine to shut down. To solve this problem, the leader
demotes itself simply by transferring leadership to a different manager node.
When another node becomes the leader, the role manager will start up on that
node, and it will be able to demote the former leader without this complication.

## The main Raft loop

The `Run` method acts as a main loop. It receives ticks from a ticker, and
forwards these to the `etcd/raft` state machine, which relies on external code
for timekeeping. It also receives `Ready` structures from the `etcd/raft` state
machine on a channel.

A `Ready` message conveys the current state of the system, provides a set of
messages to send to peers, and includes any items that need to be acted on or
written to disk. It is basically `etcd/raft`'s mechanism for communicating with
the outside world and expressing its state to higher-level code.

There are five basic functions the `Run` function performs when it receives a
`Ready` message:

1. Write new entries or a new snapshot to disk.
2. Forward any messages for other peers to the right destinations over gRPC.
3. Update the data store based on new snapshots or newly-committed log entries.
4. Evaluate the current leadership status, and signal to other code if it
   changes (for example, so that components like the orchestrator can be started
   or stopped).
5. If enough entries have accumulated between snapshots, create a new snapshot
   to compact the WALs. The snapshot is written asynchronously and notifies the
   `Run` method on completion.

## Communication between nodes

The `etcd/raft` package does not implement communication over a network. It
references nodes by IDs, and it is up to higher-level code to convey messages to
the correct places.

SwarmKit uses gRPC to transfer these messages. The interface for this is very
simple. Messages are only conveyed through a single RPC named
`ProcessRaftMessage`.

There is an additional RPC called `ResolveAddress` that deals with a corner case
that can happen when nodes are added to a cluster dynamically. If a node was
down while the current cluster leader was added, or didn't mark the log entry
that added the leader as committed (which is done lazily), this node won't have
the leader's address. It would receive RPCs from the leader, but not be able to
invoke RPCs on the leader, so the communication would only happen in one
direction. It would normally be impossible for the node to catch up. With
`ResolveAddress`, it can query other cluster members for the leader's address,
and restore two-way communication. See
https://github.com/docker/swarmkit/issues/436 more details on this situation.

SwarmKit's `raft/transport` package abstracts the mechanism for keeping track of
peers, and sending messages to them over gRPC in a specific message order.

## Integration between Raft and the data store

The Raft `Node` object implements the `Proposer` interface which the data store
uses to propagate changes across the cluster. The key method is `ProposeValue`,
which appends information to the distributed log.

The guts of `ProposeValue` are inside `processInternalRaftRequest`. This method
appends the message to the log, and then waits for it to become committed. There
is only one way `ProposeValue` can fail, which is the node where it's running
losing its position as the leader. If the node remains the leader, there is no
way a proposal can fail, since the leader controls which new entries are added
to the log, and can't retract an entry once it has been appended. It can,
however, take an indefinitely long time for a quorum of members to acknowledge
the new entry. There is no timeout on `ProposeValue` because a timeout wouldn't
retract the log entry, so having a timeout could put us in a state where a
write timed out, but ends up going through later on. This would make the data
store inconsistent with what's actually in the Raft log, which would be very
bad.

When the log entry successfully becomes committed, `processEntry` triggers the
wait associated with this entry, which allows `processInternalRaftRequest` to
return. On a leadership change, all outstanding waits get cancelled.

## The Raft RPC proxy

As mentioned above, writes to the data store are only allowed on the leader
node. But any manager node can receive gRPC requests, and workers don't even
attempt to route those requests to the leaders. Somehow, requests that involve
writing to the data store or seeing a consistent view of it need to be
redirected to the leader.

We generate wrappers around RPC handlers using the code in
`protobuf/plugin/raftproxy`. These wrappers check if the current node is the
leader, and serve the RPC locally in that case. In the case where some other
node is the leader, the wrapper invokes the same RPC on the leader instead,
acting as a proxy. The proxy inserts identity information for the client node in
the gRPC headers of the request, so that clients can't achieve privilege
escalation by going through the proxy.

If one of these wrappers is registered with gRPC instead of the generated server
code itself, the server in question will automatically proxy its requests to the
leader. We use this for most APIs such as the dispatcher, control API, and CA.
However, there are some cases where RPCs need to be invoked directly instead of
being proxied to the leader, and in these cases, we don't use the wrappers. Raft
itself is a good example of this - if `ProcessRaftMessage` was always forwarded
to the leader, it would be impossible for the leader to communicate with other
nodes. Incidentally, this is why the Raft RPCs are split between a `Raft`
service and a `RaftMembership` service. The membership RPCs `Join` and `Leave`
need to run on the leader, but RPCs such as `ProcessRaftMessage` must not be
forwarded to the leader.
