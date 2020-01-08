# SwarmKit task model

This document explains some important properties of tasks in SwarmKit. It
covers the types of state that exist for a task, a task's lifecycle, and the
slot model that associates a task with a particular replica or node.

## Task message

Tasks are defined by the `Task` protobuf message. A simplified version of this
message, showing only the fields described in this document, is presented below:

```
// Task specifies the parameters for implementing a Spec. A task is effectively
// immutable and idempotent. Once it is dispatched to a node, it will not be
// dispatched to another node.
message Task {
        string id = 1 [(gogoproto.customname) = "ID"];

        // Spec defines the desired state of the task as specified by the user.
        // The system will honor this and will *never* modify it.
        TaskSpec spec = 3 [(gogoproto.nullable) = false];

        // ServiceID indicates the service under which this task is
        // orchestrated. This should almost always be set.
        string service_id = 4 [(gogoproto.customname) = "ServiceID"];

        // Slot is the service slot number for a task.
        // For example, if a replicated service has replicas = 2, there will be
        // a task with slot = 1, and another with slot = 2.
        uint64 slot = 5;

        // NodeID indicates the node to which the task is assigned. If this
        // field is empty or not set, the task is unassigned.
        string node_id = 6 [(gogoproto.customname) = "NodeID"];

        TaskStatus status = 9 [(gogoproto.nullable) = false];

        // DesiredState is the target state for the task. It is set to
        // TaskStateRunning when a task is first created, and changed to
        // TaskStateShutdown if the manager wants to terminate the task. This
        // field is only written by the manager.
        TaskState desired_state = 10;
}
```

### ID

The `id` field contains a unique ID string for the task.

### Spec

The `spec` field contains the specification for the task. This is a part of the
service spec, which is copied to the task object when the task is created. The
spec is entirely specified by the user through the service spec. It will never
be modified by the system.

### Service ID

`service_id` links a task to the associated service. Tasks link back to the
service that created them, rather than services maintaining a list of all
associated tasks. Generally, a service's tasks are listed by querying for tasks
where service_id has a specific value. In some cases, there are tasks that exist
independent of any service. These do not have a value set in `service_id`.

### Slot

`slot` is used for replicated tasks to identify which slot the task satisfies.
The slot model is discussed in more detail below.

### Node ID

`node_id` assigns the task to a specific node. This is used by both replicated
tasks and global tasks. For global tasks, the node ID is assigned when the task
is first created. For replicated tasks, it is assigned by the scheduler when
the task gets scheduled.

### Status

`status` contains *observed* state of the task as reported by the agent. The
most important field inside `status` is `state`, which indicates where the task
is in its lifecycle (assigned, running, complete, and so on). The status
information in this field may become out of date if the node that the task is
assigned to is unresponsive. In this case, it's up to the orchestrator to
replace the task with a new one.

### Desired state

Desired state is the state that the orchestrator would like the task to progress
to. This field provides a way for the orchestrator to control when the task can
advance in state. For example, the orchestrator may create a task with desired
state set to `READY` during a rolling update, and then advance the desired state
to `RUNNING` once the old task it is replacing has stopped. This gives it a way
to get the new task ready to start (for example, pulling the new image), without
actually starting it.

## Properties of tasks

A task is a "one-shot" execution unit. Once a task stops running, it is never
executed again. A new task may be created to replace it.

Tasks states are changed in a monotonic progression. Tasks may move to states
beyond the current state, but their states may never move backwards.

## Task history

Once a task stops running, the task object is not necessarily removed from the
distributed data store. Generally, a few historic tasks for each slot of each
service are retained to provide task history. The task reaper will garbage
collect old tasks if the limit of historic tasks for a given slot is reached.
Currently, retention of containers on the workers is tied to the presence of the
old task objects in the distributed data store, but this may change in the
future.

## Task lifecycle

Tasks are created by the orchestrator. They may be created for a new service, or
to scale up an existing service, or to replace tasks for an existing service
that are no longer running for whatever reason. The orchestrator creates tasks
in the `NEW` state.

Tasks next run through the allocator, which allocate resources such as network
attachments which are necessary for the tasks to run. When the allocator has
processed a task, it moves the task to the `PENDING` state.

The scheduler takes `PENDING` tasks and assigns them to nodes (or verifies
that the requested node has the necessary resources, in the case of global
services' tasks). It changes their state to `ASSIGNED`.

From this point, control over the state passes to the agent. A task will
progress through the `ACCEPTED`, `PREPARING`, `READY', and `STARTING` states on
the way to `RUNNING`. If a task exits without an error code, it moves to the
`COMPLETE` state. If it fails, it moves to the `FAILED` state instead.

A task may alternatively end up in the `SHUTDOWN` state if its shutdown was
requested by the orchestrator (by setting desired state to `SHUTDOWN`),
the `REJECTED` state if the agent rejected the
task, or the `ORPHANED` state if the node on which the task is scheduled is
down for too long. The orchestrator will also set desired state for a task not
already in a terminal state to
`REMOVE` when the service associated with the task was removed or scaled down
by the user. When this happens, the agent proceeds to shut the task down.
The task is removed from the store by the task reaper only after the shutdown succeeds.
This ensures that resources associated with the task are not released before
the task has shut down.
Tasks that were removed becacuse of service removal or scale down
are not kept around in task history.

The task state can never move backwards - it only increases monotonically.

## Slot model

Replicated tasks have a slot number assigned to them. This allows the system to
track the history of a particular replica over time.

For example, a replicated service with three replicas would lead to three tasks,
with slot numbers 1, 2, and 3. If the task in slot 2 fails, a new task would be
started with `Slot = 2`. Through the slot numbers, the administrator would be
able to see that the new task was a replacement for the previous one in slot 2
that failed.

The orchestrator for replicated services tries to make sure the correct number
of slots have a running task in them. For example, if this 3-replica service
only has running tasks with two distinct slot numbers, it will create a third
task with a different slot number. Also, if there are 4 slot numbers represented
among the tasks in the running state, it will kill one or more tasks so that
there are only 3 slot numbers between the running tasks.

Slot numbers may be noncontiguous. For example, when a service is scaled down,
the task that's removed may not be the one with the highest slot number.

It's normal for a slot to have multiple tasks. Generally, there will be a single
task with the desired state of `RUNNING`, and also some historic tasks with a
desired state of `SHUTDOWN` that are no longer active in the system. However,
there are also cases where a slot may have multiple tasks with the desired state
of `RUNNING`. This can happen during rolling updates when the updates are
configured to start the new task before stopping the old one. The orchestrator
isn't confused by this situation, because it only cares about which slots are
satisfied by at least one running task, not the detailed makeup of those slots.
The updater takes care of making sure that each slot converges to having a
single running task.

Also, for application availability, multiple tasks can share the single slot
number when a network partition occurs between nodes. If a node is split from
manager nodes, the tasks that were running on the node will be recreated on
another node.  However, the tasks on the split node can still continue
running. So the old tasks and the new ones can share identical slot
numbers. These tasks may be considered "orphaned" by the manager, after some
time. Upon recovering the split, these tasks will be killed.

Global tasks do not have slot numbers, but the concept is similar. Each node in
the system should have a single running task associated with it. If this is not
the case, the orchestrator and updater work together to create or destroy tasks
as necessary.
