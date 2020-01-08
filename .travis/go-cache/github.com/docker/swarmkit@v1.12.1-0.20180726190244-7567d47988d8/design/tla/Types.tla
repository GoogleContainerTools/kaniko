------------------------------- MODULE Types -------------------------------

EXTENDS Naturals, FiniteSets

(* A generic operator to get the range of a function (the set of values in a map): *)
Range(S) == { S[i] : i \in DOMAIN S }

(* The set of worker nodes.

   Note: a CONSTANT is an input to the model. The model should work with any set of nodes you provide.

   TODO: should cope with this changing at runtime, and with draining nodes. *)
CONSTANT Node

(* A special value indicating that a task is not yet assigned to a node.

   Note: this TLA+ CHOOSE idiom just says to pick some value that isn't a Node (e.g. `null'). *)
unassigned == CHOOSE n : n \notin Node

(* The type (set) of service IDs (e.g. `Int' or `String').
   When model checking, this will be some small set (e.g. {"s1", "s2"}). *)
CONSTANT ServiceId

(* The type of task IDs. *)
CONSTANT TaskId

(* The maximum possible value for `replicas' in ServiceSpec. *)
CONSTANT maxReplicas
ASSUME maxReplicas \in Nat
Slot == 1..maxReplicas  \* Possible slot numbers

(* A special value (e.g. `-1') indicating that we want one replica running on each node: *)
global == CHOOSE x : x \notin Nat

(* The type of a description of a service (a struct/record).
   This is provided by, and only changed by, the user. *)
ServiceSpec == [
  (* The replicas field is either a count giving the desired number of replicas,
     or the special value `global'. *)
  replicas : 0..maxReplicas \union {global},
  remove   : BOOLEAN    \* The user wants to remove this service
]

(* The possible states for a task: *)
new == "new"
pending == "pending"
assigned == "assigned"
accepted == "accepted"
preparing == "preparing"
ready == "ready"
starting == "starting"
running == "running"
complete == "complete"
shutdown == "shutdown"
failed == "failed"
rejected == "rejected"
remove == "remove"      \* Only used as a ``desired state'', not an actual state
orphaned == "orphaned"

(* Every state has a rank. It is only possible for a task to change
   state to a state with a higher rank (later in this sequence). *)
order == << new, pending, assigned, accepted,
             preparing, ready, starting,
             running,
             complete, shutdown, failed, rejected,
             remove, orphaned >>

(* Maps a state to its position in `order' (e.g. StateRank(new) = 1): *)
StateRank(s) == CHOOSE i \in DOMAIN order : order[i] = s

(* Convenient notation for comparing states: *)
s1 \prec s2   == StateRank(s1) < StateRank(s2)
s1 \preceq s2 == StateRank(s1) <= StateRank(s2)

(* The set of possible states ({new, pending, ...}): *)
TaskState == Range(order) \ {remove}

(* Possibly this doesn't need to be a record, but we might want to add extra fields later. *)
TaskStatus == [
  state : TaskState
]

(* The state that SwarmKit wants to the task to be in. *)
DesiredState == { ready, running, shutdown, remove }

(* This has every field mentioned in `task_model.md' except for `spec', which
   it doesn't seem to use for anything.

   `desired_state' can be any state, although currently we only ever set it to one of
    {ready, running, shutdown, remove}. *)
Task == [
  id : TaskId,                      \* To uniquely identify this task
  service : ServiceId,              \* The service that owns the task
  status : TaskStatus,              \* The current state
  desired_state : DesiredState,     \* The state requested by the orchestrator
  node : Node \union {unassigned},  \* The node on which the task should be run
  slot : Slot \union {global}       \* A way of tracking related tasks
]

(* The current state of task `t'. *)
State(t) == t.status.state

(* A task is runnable if it is running or could become running in the future. *)
Runnable(t) == State(t) \preceq running

(* A task's ``virtual slot'' is its actual slot for replicated services,
   but its node for global ones. *)
VSlot(t) ==
  IF t.slot = global THEN t.node ELSE t.slot

(* In the real SwarmKit, a task's ID is just its taskId field.
   However, this requires lots of IDs, which is expensive for model checking.
   So instead, we will identify tasks by their << serviceId, vSlot, taskId >>
   triple, and only require taskId to be unique within its vslot. *)
ModelTaskId == ServiceId \X (Slot \union Node) \X TaskId

(* A unique identifier for a task, which never changes. *)
Id(t) ==
  << t.service, VSlot(t), t.id >>   \* A ModelTaskId

(* The ModelTaskIds of a set of tasks. *)
IdSet(S) == { Id(t) : t \in S }

=============================================================================
