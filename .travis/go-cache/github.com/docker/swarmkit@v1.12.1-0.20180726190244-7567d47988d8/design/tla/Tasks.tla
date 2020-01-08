---------------------------- MODULE Tasks ----------------------------------

EXTENDS TLC, Types

VARIABLE tasks          \* The set of currently-allocated tasks

(* The expected type of each variable. TLA+ is an untyped language, but the model checker
   can check that TasksTypeOK is true for every reachable state. *)
TasksTypeOK ==
  \* `tasks' is a subset of the set of all possible tasks
  /\ tasks \in SUBSET Task

(* Update `tasks' by performing each update in `f', which is a function
   mapping old tasks to new ones. *)
UpdateTasks(f) ==
  /\ Assert(\A t \in DOMAIN f : t \in tasks, "An old task does not exist!")
  /\ Assert(\A t \in DOMAIN f :
                LET t2 == f[t]
                IN                        \* The updated version of `t' must have
                /\ t.id      = t2.id      \* the same task ID,
                /\ t.service = t2.service \* the same service ID,
                /\ VSlot(t)  = VSlot(t2), \* and the same vslot.
            "An update changes a task's identity!")
  \* Remove all the old tasks and add the new ones:
  /\ tasks' = (tasks \ DOMAIN f) \union Range(f)

(* A `new' task belonging to service `sid' with the given slot, id, and desired state. *)
NewTask(sid, vslot, id, desired_state) ==
  [
    id            |-> id,
    service       |-> sid,
    status        |-> [ state |-> new ],
    desired_state |-> desired_state,
    node          |-> IF vslot \in Node THEN vslot ELSE unassigned,
    slot          |-> IF vslot \in Slot THEN vslot ELSE global
  ]


\* A special ``state'' used when a task doesn't exist.
null == "null"

(* All the possible transitions, grouped by the component that performs them. *)
Transitions == [
  orchestrator |-> {
    << null, new >>
  },

  allocator |-> {
    << new, pending >>
  },

  scheduler |-> {
    << pending, assigned >>
  },

  agent |-> {
    << assigned, accepted >>,
    << accepted, preparing >>,
    << preparing, ready >>,
    << ready, starting >>,
    << starting, running >>,

    << assigned, rejected >>,
    << accepted, rejected >>,
    << preparing, rejected >>,
    << ready, rejected >>,
    << starting, rejected >>,

    << running, complete >>,
    << running, failed >>,

    << running, shutdown >>,

    << assigned, orphaned >>,
    << accepted, orphaned >>,
    << preparing, orphaned >>,
    << ready, orphaned >>,
    << starting, orphaned >>,
    << running, orphaned >>
  },

  reaper |-> {
    << new, null >>,
    << pending, null >>,
    << rejected, null >>,
    << complete, null >>,
    << failed, null >>,
    << shutdown, null >>,
    << orphaned, null >>
  }
]

(* Check that `Transitions' itself is OK. *)
TransitionTableOK ==
  \* No transition moves to a lower-ranked state:
  /\ \A actor \in DOMAIN Transitions :
     \A trans \in Transitions[actor] :
        \/ trans[1] = null
        \/ trans[2] = null
        \/ trans[1] \preceq trans[2]
  (* Every source state has exactly one component which handles transitions out of that state.
     Except for the case of the reaper removing `new' and `pending' tasks that are flagged
     for removal. *)
  /\ \A a1, a2 \in DOMAIN Transitions :
     LET exceptions == { << new, null >>, << pending, null >> }
          Source(a) == { s[1] : s \in Transitions[a] \ exceptions}
     IN  a1 # a2 =>
           Source(a1) \intersect Source(a2) = {}

ASSUME TransitionTableOK  \* Note: ASSUME means ``check'' to TLC

=============================================================================