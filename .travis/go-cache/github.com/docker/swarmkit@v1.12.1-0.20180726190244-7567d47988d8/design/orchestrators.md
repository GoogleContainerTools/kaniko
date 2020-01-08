# Orchestrators

When we talk about an *orchestrator* in SwarmKit, we're not talking about
SwarmKit as a whole, but a specific component that creates and shuts down tasks.
In SwarmKit's [task model](task_model.md), a *service* gets translated into some
number of *tasks*. The service is an abstract description of the workload, and
the tasks are individual units that can be dispatched to specific nodes. An
orchestrator manages these tasks.

The scope of an orchestrator is fairly limited. It creates the corresponding
tasks when a service is created, adds or removes tasks when a service is scaled,
and deletes the linked tasks when a service is deleted. In general, it does not
make scheduling decisions, which are left to the [scheduler](scheduler.md).
However, the *global orchestrator* does create tasks that are bound to specific
nodes, because tasks from global services can't be scheduled freely.

## Event handling

There are two general types of events an orchestrator handles: service-level events
and task-level events.

Some examples of service-level events are a new service being created, or an
existing service being updated. In these cases, the orchestrator will create
and shut down tasks as necessary to satisfy the service definition.

An example of a task-level event is a failure being reported for a particular
task instance. In this case, the orchestrator will restart this task, if
appropriate. (Note that *restart* in this context means starting a new task to
replace the old one.) Node events are similar: if a node fails, the orchestrator
can restart tasks which ran on that node.

This combination of events makes the orchestrator more efficient. A simple,
naive design would involve reconciling the service every time a relevant event
is received. Scaling a service and replacing a failed task could be handled
through the same code, which would compare the set of running tasks with the set
of tasks that are supposed to be running, and create or shut down tasks as
necessary. This would be quite inefficient though. Every time something needed
to trigger a task restart, we'd have to look at every task in the service. By
handling task events separately, an orchestrator can avoid looking at the whole
service except when the service itself changes.

## Initialization

When an orchestrator starts up, it needs to do an initial reconciliation pass to
make sure tasks are consistent with the service definitions. In steady-state
operation, actions like restarting failed tasks and deleting tasks when a
service is deleted happen in response to events. However, if there is a
leadership change or cluster restart, some events may have gone unhandled by the
orchestrator. At startup, `CheckTasks` iterates over all the tasks in the store
and takes care of anything that should normally have been handled by an event
handler.

## Replicated orchestrator

The replicated orchestrator only acts on replicated services, and tasks
associated with replicated services. It ignores other services and tasks.

There's not much magic to speak of. The replicated orchestrator responds to some
task events by triggering restarts through the restart supervisor, which is also
used by the global orchestrator. The restart supervisor is explained in more
detail below. The replicated orchestrator responds to service creations and
updates by reconciling the service, a process that relies on the update
supervisor, also shared by the global orchestrator. When a replicated service is
deleted, the replicated orchestrator deletes all of its tasks.

The service reconciliation process starts by grouping a service's tasks by slot
number (see the explanation of slots in the [task model](task_model.md)
document). These slots are marked either runnable or dead - runnable if at least
one task has a desired state of `Running` or below, and dead otherwise.

If there are fewer runnable slots than the number of replicas specified in the
service spec, the orchestrator creates the right number of tasks to make up the
difference, assigning them slot numbers that don't conflict with any runnable
slots.

If there are more runnable slots than the number of replicas specified in the
service spec, the orchestrator deletes extra tasks. It attempts to remove tasks
on nodes that have the most instances of this service running, to maintain
balance in the way tasks are assigned to nodes. When there's a tie between the
number of tasks running on multiple nodes, it prefers to remove tasks that
aren't running (in terms of observed state) over tasks that are currently
running. Note that scale-down decisions are made by the orchestrator, and don't
quite match the state the scheduler would arrive at when scaling up. This is an
area for future improvement; see https://github.com/docker/swarmkit/issues/2320
for more details.

In both of these cases, and also in the case where the number of replicas is
already correct, the orchestrator calls the update supervisor to ensure that the
existing tasks (or tasks being kept, in the case of a scale-down) are
up-to-date. The update supervisor does the heavy lifting involved in rolling
updates and automatic rollbacks, but this is all abstracted from the
orchestrator.

## Global orchestrator

The global orchestrator works similarly to the replicated orchestrator, but
tries to maintain one task per active node meeting the constraints, instead of a
specific total number of replicas. It ignores services that aren't global
services and tasks that aren't associated with global services.

The global orchestrator responds to task events in much the same way that the
replicated orchestrator does. If a task fails, the global orchestrator will
indicate to the restart supervisor that a restart may be needed.

When a service is created, updated, or deleted, this triggers a reconciliation.
The orchestrator has to check whether each node meets the constraints for the
service, and create or update tasks on that node if it does. The tasks are
created with a specific node ID pre-filled. They pass through the scheduler so
that the scheduler can wait for the nodes to have sufficient resources before
moving the desired state to `Assigned`, but the scheduler does not make the
actual scheduling decision.

The global orchestrator also responds to node events. These trigger
reconciliations much like service events do. A new node might mean creating a
task from each service on that node, and a deleted node would mean deleting any
global service tasks from that node. When a node gets drained, the global
orchestrator shuts down any global service tasks running on that node. It also
does this when a node goes down, which avoids stuck rolling updates that would
otherwise want to update the task on the unavailable node before proceeding.

Like the replicated orchestrator, the global orchestrator uses the update
supervisor to implement rolling updates and automatic rollbacks. Instead of
passing tasks to the update supervisor by slot, it groups them by node. This
means rolling updates will go node-by-node instead of slot-by-slot.

## Restart supervisor

The restart supervisor manages the process of shutting down a task, and
possibly starting a replacement task. Its entry point is a `Restart` method
which is called inside a store write transaction in one of the orchestrators.
It atomically changes the desired state of the old task to `Shutdown`, and, if
it's appropriate to start a replacement task based on the service's restart
policy, creates a new task in the same slot (replicated service) or on the same
node (global service).

If the service is set up with a restart delay, the restart supervisor handles
this delay too. It initially creates the new task with the desired state
`Ready`, and only changes the desired state to `Running` after the delay has
elapsed. One of the things the orchestrators do when they start up is check for
tasks that were in this delay phase of being restarted, and make sure they get
advanced to `Running`.

In some cases, a task can fail or be rejected before its desired state reaches
`Running`. One example is a failure to pull an image from a registry. The
restart supervisor tries to make sure this doesn't result in fast restart loops
that effectively ignore the restart delay. If `Restart` is called on a task that
the restart supervisor is still in the process of starting up - i.e. it hasn't
moved the task to `Running` yet - it will wait for the restart delay to elapse
before triggering this second restart.

The restart supervisor implements the logic to decide whether a task should be
restarted, and since this can be dependent on restart history (when
`MaxAttempts`) is set, the restart supervisor keeps track of this history. The
history isn't persisted, so some restart behavior may be slightly off after a
restart or leader election.

Note that a call to `Restart` doesn't always end up with the task being
restarted - this depends on the service's configuration. `Restart` can be
understood as "make sure this task gets shut down, and maybe start a replacement
if the service configuration says to".

## Update supervisor

The update supervisor is the component that updates existing tasks to match the
latest version of the service. This means shutting down the old task and
starting a new one to replace it. The update supervisor implements rolling
updates and automatic rollback.

The update supervisor operates on an abstract notion of slots, which are either
slot numbers for replicated services, or node IDs for global services. You can
think of it as reconciling the contents of each slot with the service. If a slot
has more than one task or fewer than one task, it corrects that. If the task (or
tasks) in a slot are out of date, they are replaced with a single task that's up
to date.

Every time the update supervisor is called to start an update of a service, it
spawns an `Updater` set up to work toward this goal. Each service can only have
one `Updater` at once, so if the service already had a different update in
progress, it is interrupted and replaced by the new one. The `Updater` runs in
its own goroutine, going through the slots and reconciling them with the
current service. It starts by checking which of the slots are dirty. If they
are all up to date and have a single task, it can finish immediately.
Otherwise, it starts as many worker goroutines as the update parallelism
setting allows, and lets them consume dirty slots from a channel.

The workers do the work of reconciling an individual slot with the service. If
there is a runnable task in the slot which is up to date, this may only involve
starting up the up-to-date task and shutting down the other tasks. Otherwise,
the worker will shut down all tasks in the slot and create a new one that's
up-to-date. It can either do this atomically, or start the new task before the
old one shuts down, depending on the update settings.

The updater watches task events to see if any of the new tasks it created fail
while the update is still running. If enough fail, and the update is set up to
pause or roll back after a certain threshold of failures, the updater will pause
or roll back the update. Pausing involves setting `UpdateStatus.State` on the
service to "paused". This is recognized as a paused update by the updater, and
it won't try to update the service again until the flag gets cleared by
`controlapi` the next time a client updates the service. Rolling back involves
setting `UpdateStatus.State` to "rollback started", then copying `PreviousSpec`
into `Spec`, updating `SpecVersion` accordingly, and clearing `PreviousSpec`.
This triggers a reconciliation in the replicated or global orchestrator, which
ends up calling the update supervisor again to "update" the tasks to the
previous version of the service. Effectively, the updater just gets called again
in reverse. The updater knows when it's being used in a rollback scenario, based
on `UpdateStatus.State`, so it can choose the appropriate update parameters and
avoid rolling back a rollback, but other than that, the logic is the same
whether an update is moving forward or in reverse.

The updater waits the time interval given by `Monitor` after the update
completes. This allows it to notice problems after it's done updating tasks, and
take actions that were requested for failure cases. For example, if a service
only has one task, has `Monitor` set to 5 seconds, and `FailureAction` set to
"rollback", the updater will wait 5 seconds after updating the task. Then, if
the new task fails within 5 seconds, the updater will be able to trigger a
rollback. Without waiting, the updater would end up finishing immediately after
creating and starting the new task, and probably wouldn't be around to respond
by the time the task failed.

## Task reaper

As discussed above, restarting a task involves shutting down the old task and
starting a new one. If restarts happen frequently, a lot of old tasks that
aren't actually running might accumulate.

The task reaper implements configurable garbage collection of these
no-longer-running tasks. The number of old tasks to keep per slot or node is
controlled by `Orchestration.TaskHistoryRetentionLimit` in the cluster's
`ClusterSpec`.

The task reaper watches for task creation events, and adds the slots or nodes
from these events to a watchlist. It periodically iterates over the watchlist
and deletes tasks from referenced slots or nodes which exceed the retention
limit. It prefers to delete tasks with the oldest `Status` timestamps.
