This is a TLA+ model of SwarmKit. Even if you don't know TLA+, you should be able to
get the general idea. This section gives a very brief overview of the syntax.

Declare `x' to be something that changes as the system runs:

  VARIABLE x

Define `Init' to be a state predicate (== means ``is defined to be''):

  Init ==
    x = 0

`Init' is true for states in which `x = 0'. This can be used to define
the possible initial states of the system. For example, the state
[ x |-> 0, y |-> 2, ... ] satisfies this.

Define `Next' to be an action:

  Next ==
    /\ x' \in Nat
    /\ x' > x

An action takes a pair of states, representing an atomic step of the system.
Unprimed expressions (e.g. `x') refer to the old state, and primed ones to
the new state. This example says that a step is a `Next' step iff the new
value of `x' is a natural number and greater than the previous value.
For example, [ x |-> 3, ... ] -> [x |-> 10, ... ] is a `Next' step.

/\ is logical ``and''. This example uses TLA's ``bulleted-list'' syntax, which makes
writing these easier. It is indentation-sensitive. TLA also has \/ lists (``or'').

See `.http://lamport.azurewebsites.net/tla/summary.pdf.' for a more complete summary
of the syntax.

This specification can be read as documentation, but it can also be executed by the TLC
model checker. See the model checking section below for details about that.

The rest of the document is organised as follows:

1. Parameters to the model
2. Types and definitions
3. How to run the model checker
4. Actions performed by the user
5. Actions performed by the components of SwarmKit
6. The complete system
7. Properties of the system

-------------------------------- MODULE SwarmKit --------------------------------

(* Import some libraries we use.
   Common SwarmKit types are defined in Types.tla. You should probably read that before continuing. *)
EXTENDS Integers, TLC, FiniteSets,  \* From the TLA+ standard library
        Types,                      \* SwarmKit types
        Tasks,                      \* The `tasks' variable
        WorkerSpec,                 \* High-level spec for worker nodes
        EventCounter                \* Event limiting, for modelling purposes

(* The maximum number of terminated tasks to keep for each slot. *)
CONSTANT maxTerminated
ASSUME maxTerminated \in Nat

(* In the model, we share taskIDs (see ModelTaskId), which means that
   we can cover most behaviours with only enough task IDs
   for one running task and maxTerminated finished ones. *)
ASSUME Cardinality(TaskId) >= 1 + maxTerminated

-------------------------------------------------------------------------------
\* Services

VARIABLE services       \* A map of currently-allocated services, indexed by ServiceId

(* A replicated service is one that specifies some number of replicas it wants. *)
IsReplicated(sid) ==
  services[sid].replicas \in Nat

(* A global service is one that wants one task running on each node. *)
IsGlobal(sid) ==
  services[sid].replicas = global

(* TasksOf(sid) is the set of tasks for service `sid'. *)
TasksOf(sid) ==
  { t \in tasks : t.service = sid }

(* All tasks of service `sid' in `vslot'. *)
TasksOfVSlot(sid, vslot) ==
  { t \in TasksOf(sid) : VSlot(t) = vslot }

(* All vslots of service `sid'. *)
VSlotsOf(sid) ==
  { VSlot(t) : t \in TasksOf(sid) }

-------------------------------------------------------------------------------
\* Types

(* The expected type of each variable. TLA+ is an untyped language, but the model checker
   can check that TypeOK is true for every reachable state. *)
TypeOK ==
  \* `services' is a mapping from service IDs to ServiceSpecs:
  /\ DOMAIN services \subseteq ServiceId
  /\ services \in [ DOMAIN services -> ServiceSpec ]
  /\ TasksTypeOK    \* Defined in Types.tla
  /\ WorkerTypeOK   \* Defined in WorkerSpec.tla

-------------------------------------------------------------------------------
(*
`^ \textbf{Model checking} ^'

   You can test this specification using the TLC model checker.
   This section describes how to do that. If you don't want to run TLC,
   you can skip this section.

   To use TLC, load this specification file in the TLA+ toolbox (``Open Spec'')
   and create a new model using the menu.

   You will be prompted to enter values for the various CONSTANTS.
   A suitable set of initial values is:

      `.
      Node          <- [ model value ] {n1}
      ServiceId     <- [ model value ] {s1}
      TaskId        <- [ model value ] {t1, t2}
      maxReplicas   <- 1
      maxTerminated <- 1
      .'

   For the [ model value ] ones, select `Set of model values'.

   This says that we have one node, `n1', at most one service, and at most
   two tasks per vslot. TLC can explore all possible behaviours of this system
   in a couple of seconds on my laptop.

   You should also specify some things to check (under ``What to check?''):

   - Add `TypeOK' and `Inv' under ``Invariants''
   - Add `TransitionsOK' and `EventuallyAsDesired' under ``Properties''

   Running the model should report ``No errors''.

   If the model fails, TLC will show you an example sequence of actions that lead to
   the failure and you can inspect the state at each step. You can try this out by
   commenting out any important-looking condition in the model (e.g. the requirement
   in UpdateService that you can't change the mode of an existing service).

   Although the above model is very small, it should detect most errors that you might
   accidentally introduce when modifying the specification. Increasing the number of nodes,
   services, replicas or terminated tasks will check more behaviours of the system,
   but will be MUCH slower.

   The rest of this section describes techniques to make model checking faster by reducing
   the number of states that must be considered in various ways. Feel free to skip it.

`^ \textbf{Symmetry sets} ^'

   You should configure any model sets (e.g. `TaskId') as `symmetry sets'.
   For example, if you have a model with two nodes {n1, n2} then this tells TLC that
   two states which are the same except that n1 and n2 are swapped are equivalent
   and it only needs to continue exploring from one of them.
   TLC will warn that checking temporal properties may not work correctly,
   but it's much faster and I haven't had any problems with it.

`^ \textbf{Limiting the maximum number of setbacks to consider} ^'

   Another way to speed things up is to reduce the number of failures that TLC must consider.
   By default, it checks every possible combination of failures at every point, which
   is very expensive.
   In the `Advanced Options' panel of the model, add a ``Definition Override'' of e.g.
   `maxEvents = 2'. Actions that represent unnecessary extra work (such as the user
   changing the configuration or a worker node going down) are tagged with `CountEvent'.
   Any run of the system cannot have more than `maxEvents' such events.

   See `EventCounter.tla' for details.

`^ \textbf{Preventing certain failures} ^'

   If you're not interested in some actions then you can block them. For example,
   adding these two constraints in the ``Action Constraint'' box of the
   ``Advanced Options'' tab tells TLC not to consider workers going down or
   workers rejecting tasks as possible actions:

   /\ ~WorkerDown
   /\ ~RejectTask
*)

(*
`^ \textbf{Combining task states} ^'

   A finished task can be either in the `complete' or `failed' state, depending on
   its exit status. If we have 4 finished tasks, that's 16 different states. For
   modelling, we might not care about exit codes and we can treat this as a single
   state with another definition override:

   `.failed <- complete.'

   In a similar way, we can combine { assigned, accepted, preparing, ready } into a single
   state:

   `.accepted <- assigned
     preparing <- assigned
     ready <- assigned.'
*)

---------------------------- MODULE User  --------------------------------------------
\* Actions performed by users

(* Create a new service with any ServiceSpec.

   This says that a single atomic step of the system from an old state
   to a new one is a CreateService step iff `tasks', `nodes' and `nEvents' don't change
   and the new value of `services' is the same as before except that some
   service ID that wasn't used in the old state is now mapped to some
   ServiceSpec.

   Note: A \ B means { x \in A : x \notin B } --
         i.e. the set A with all elements in B removed.
   *)
CreateService ==
  /\ UNCHANGED << tasks, nodes, nEvents >>
  /\ \E sid \in ServiceId \ DOMAIN services,     \* `sid' is an unused ServiceId
       spec \in ServiceSpec :                    \* `spec' is any ServiceSpec
          /\ spec.remove = FALSE                 \* (not flagged for removal)
          /\ services' = services @@ sid :> spec \* Add `sid |-> spec' to `services'

(* Update an existing service's spec. *)
UpdateService ==
  /\ UNCHANGED << tasks, nodes >>
  /\ CountEvent \* Flag as an event for model-checking purposes
  /\ \E sid     \in DOMAIN services,   \* `sid' is an existing ServiceId
        newSpec \in ServiceSpec :      \* `newSpec' is any `ServiceSpec'
       /\ services[sid].remove = FALSE \* We weren't trying to remove sid
       /\ newSpec.remove = FALSE       \* and we still aren't.
       \* You can't change a service's mode:
       /\ (services[sid].replicas = global) <=> (newSpec.replicas = global)
       /\ services' = [ services EXCEPT ![sid] = newSpec ]

(* The user removes a service.

   Note: Currently, SwarmKit deletes the service from its records immediately.
   However, this isn't right because we need to wait for service-level resources
   such as Virtual IPs to be freed.
   Here we model the proposed fix, in which we just flag the service for removal. *)
RemoveService ==
  /\ UNCHANGED << nodes >>
  /\ CountEvent
  /\ \E sid \in DOMAIN services : \* sid is some existing service
       \* Flag service for removal:
       /\ services' = [services EXCEPT ![sid].remove = TRUE]
       \* Flag every task of the service for removal:
       /\ UpdateTasks([ t \in TasksOf(sid) |->
                          [t EXCEPT !.desired_state = remove] ])

(* A user action is one of these. *)
User ==
  \/ CreateService
  \/ UpdateService
  \/ RemoveService

=============================================================================

---------------------------- MODULE Orchestrator ----------------------------

\* Actions performed the orchestrator

\* Note: This is by far the most complicated component in the model.
\* You might want to read this section last...

(* The set of tasks for service `sid' that should be considered as active.
   This is any task that is running or on its way to running. *)
RunnableTasks(sid) ==
  { t \in TasksOf(sid) : Runnable(t) }

(* Candidates for shutting down when we have too many. We don't want to count tasks that are shutting down
   towards the total count when deciding whether we need to kill anything. *)
RunnableWantedTasks(sid) ==
  { t \in RunnableTasks(sid) : t.desired_state \preceq running  }

(* The set of possible new vslots for `sid'. *)
UnusedVSlot(sid) ==
  IF IsReplicated(sid) THEN Slot \ VSlotsOf(sid)
                       ELSE Node \ VSlotsOf(sid)

(* The set of possible IDs for a new task in a vslot.

   The complexity here is just a side-effect of the modelling (where we need to
   share and reuse task IDs for performance).
   In the real system, choosing an unused ID is easy. *)
UnusedId(sid, vslot) ==
  LET swarmTaskIds == { t.id : t \in TasksOfVSlot(sid, vslot) }
  IN  TaskId \ swarmTaskIds

(* Create a new task/slot if the number of runnable tasks is less than the number requested. *)
CreateSlot ==
  /\ UNCHANGED << services, nodes, nEvents >>
  /\ \E sid \in DOMAIN services :          \* `sid' is an existing service
     /\ ~services[sid].remove              \* that we're not trying to remove
     (* For replicated tasks, only create as many slots as we need.
        For global tasks, we want all possible vslots (nodes). *)
     /\ IsReplicated(sid) =>
          services[sid].replicas > Cardinality(VSlotsOf(sid))  \* Desired > actual
     /\ \E slot \in UnusedVSlot(sid) :
        \E id   \in UnusedId(sid, slot) :
           tasks' = tasks \union { NewTask(sid, slot, id, running) }

(* Add a task if a slot exists, contains no runnable tasks, and we weren't trying to remove it.
   Note: if we are trying to remove it, the slot will eventually disappear and CreateSlot will
   then make a new one if we later need it again.

   Currently in SwarmKit, slots do not actually exist as objects in the store.
   Instead, we just infer that a slot exists because there exists a task with that slot ID.
   This has the odd effect that if `maxTerminated = 0' then we may create new slots rather than reusing
   existing ones, depending on exactly when the reaper runs.
   *)
ReplaceTask ==
  /\ UNCHANGED << services, nodes, nEvents >>
  /\ \E sid  \in DOMAIN services :
     \E slot \in VSlotsOf(sid) :
     /\ \A task \in TasksOfVSlot(sid, slot) :    \* If all tasks in `slot' are
           ~Runnable(task)                       \* dead (not runnable) and
     /\ \E task \in TasksOfVSlot(sid, slot) :    \* there is some task that
           task.desired_state # remove           \* we're not trying to remove,
     /\ \E id \in UnusedId(sid, slot) :          \* then create a replacement task:
        tasks' = tasks \union { NewTask(sid, slot, id, running) }

(* If we have more replicas than the spec asks for, remove one of them. *)
RequestRemoval ==
  /\ UNCHANGED << services, nodes, nEvents >>
  /\ \E sid \in DOMAIN services :
       LET current == RunnableWantedTasks(sid)
       IN \* Note: `current' excludes tasks we're already trying to kill
       /\ IsReplicated(sid)
       /\ services[sid].replicas < Cardinality(current)   \* We have too many replicas
       /\ \E slot \in { t.slot : t \in current } :        \* Choose an allocated slot
            \* Mark all tasks for that slot for removal:
            UpdateTasks( [ t \in TasksOfVSlot(sid, slot) |->
                            [t EXCEPT !.desired_state = remove] ] )

(* Mark a terminated task for removal if we already have `maxTerminated' terminated tasks for this slot. *)
CleanupTerminated ==
  /\ UNCHANGED << services, nodes, nEvents >>
  /\ \E sid  \in DOMAIN services :
     \E slot \in VSlotsOf(sid) :
     LET termTasksInSlot == { t \in TasksOfVSlot(sid, slot) :
                              State(t) \in { complete, shutdown, failed, rejected } }
     IN
     /\ Cardinality(termTasksInSlot) > maxTerminated    \* Too many tasks for slot
     /\ \E t \in termTasksInSlot :                      \* Pick a victim to remove
        UpdateTasks(t :> [t EXCEPT !.desired_state = remove])

(* We don't model the updater explicitly, but we allow any task to be restarted (perhaps with
   a different image) at any time, which should cover the behaviours of the restart supervisor.

   TODO: SwarmKit also allows ``start-first'' mode updates where we first get the new task to
   `running' and then mark the old task for shutdown. Add this to the model. *)
RestartTask ==
  /\ UNCHANGED << services, nodes >>
  /\ CountEvent
  /\ \E oldT  \in tasks :
     \E newId \in UnusedId(oldT.service, VSlot(oldT)) :
        /\ Runnable(oldT)                           \* Victim must be runnable
        /\ oldT.desired_state \prec shutdown        \* and we're not trying to kill it
        \* Create the new task in the `ready' state (see ReleaseReady below):
        /\ LET replacement == NewTask(oldT.service, VSlot(oldT), newId, ready)
           IN  tasks' =
                (tasks \ {oldT}) \union {
                  [oldT EXCEPT !.desired_state = shutdown],
                  replacement
                }

(* A task is set to wait at `ready' and the previous task for that slot has now finished.
   Allow it to proceed to `running'. *)
ReleaseReady ==
  /\ UNCHANGED << services, nodes, nEvents >>
  /\ \E t \in tasks :
       /\ t.desired_state = ready         \* (and not e.g. `remove')
       /\ State(t) = ready
       /\ \A other \in TasksOfVSlot(t.service, VSlot(t)) \ {t} :
             ~Runnable(other)             \* All other tasks have finished
       /\ UpdateTasks(t :> [t EXCEPT !.desired_state = running])

(* The user asked to remove a service, and now all its tasks have been cleaned up. *)
CleanupService ==
  /\ UNCHANGED << tasks, nodes, nEvents >>
  /\ \E sid \in DOMAIN services :
     /\ services[sid].remove = TRUE
     /\ TasksOf(sid) = {}
     /\ services' = [ i \in DOMAIN services \ {sid} |-> services[i] ]

(* Tasks that the orchestrator must always do eventually if it can: *)
OrchestratorProgress ==
  \/ CreateSlot
  \/ ReplaceTask
  \/ RequestRemoval
  \/ CleanupTerminated
  \/ ReleaseReady
  \/ CleanupService

(* All actions that the orchestrator can perform *)
Orchestrator ==
  \/ OrchestratorProgress
  \/ RestartTask

=============================================================================

---------------------------- MODULE Allocator -------------------------------
\*  Actions performed the allocator

(* Pick a `new' task and move it to `pending'.

   The spec says the allocator will ``allocate resources such as network attachments
   which are necessary for the tasks to run''. However, we don't model any resources here. *)
AllocateTask ==
  /\ UNCHANGED << services, nodes, nEvents >>
  /\ \E t \in tasks :
     /\ State(t) = new
     /\ UpdateTasks(t :> [t EXCEPT !.status.state = pending])

AllocatorProgress ==
  \/ AllocateTask

Allocator ==
  \/ AllocatorProgress

=============================================================================

---------------------------- MODULE Scheduler -------------------------------

\*  Actions performed by the scheduler

(* The scheduler assigns a node to a `pending' task and moves it to `assigned'
   once sufficient resources are available (we don't model resources here). *)
Scheduler ==
  /\ UNCHANGED << services, nodes, nEvents >>
  /\ \E t \in tasks :
     /\ State(t) = pending
     /\ LET candidateNodes == IF t.node = unassigned
                                THEN Node  \* (all nodes)
                                ELSE { t.node }
        IN
        \E node \in candidateNodes :
           UpdateTasks(t :> [t EXCEPT !.status.state = assigned,
                                      !.node = node ])

=============================================================================

---------------------------- MODULE Reaper ----------------------------------

\*  Actions performed by the reaper

(* Forget about tasks in remove or orphan states.

   Orphaned tasks belong to nodes that we are assuming are lost forever (or have crashed
   and will come up with nothing running, which is an equally fine outcome). *)
Reaper ==
  /\ UNCHANGED << services, nodes, nEvents >>
  /\ \E t \in tasks :
      /\ \/ /\ t.desired_state = remove
            /\ (State(t) \prec assigned \/ ~Runnable(t)) \* Not owned by agent
         \/ State(t) = orphaned
      /\ tasks' = tasks \ {t}

=============================================================================

\*  The complete system

\* Import definitions from the various modules
INSTANCE User
INSTANCE Orchestrator
INSTANCE Allocator
INSTANCE Scheduler
INSTANCE Reaper

\* All the variables
vars == << tasks, services, nodes, nEvents >>

\* Initially there are no tasks and no services, and all nodes are up.
Init ==
  /\ tasks = {}
  /\ services = << >>
  /\ nodes = [ n \in Node |-> nodeUp ]
  /\ InitEvents

(* WorkerSpec doesn't mention `services'. To combine it with this spec, we need to say
   that every action of the agent leaves `services' unchanged. *)
AgentReal ==
  Agent /\ UNCHANGED services

(* Unfortunately, `AgentReal' causes TLC to report all problems of the agent
   as simply `AgentReal' steps, which isn't very helpful. We can get better
   diagnostics by expanding it, like this: *)
AgentTLC ==
  \/ (ProgressTask     /\ UNCHANGED services)
  \/ (ShutdownComplete /\ UNCHANGED services)
  \/ (OrphanTasks      /\ UNCHANGED services)
  \/ (WorkerUp         /\ UNCHANGED services)
  \/ (RejectTask       /\ UNCHANGED services)
  \/ (ContainerExit    /\ UNCHANGED services)
  \/ (WorkerDown       /\ UNCHANGED services)

(* To avoid the risk of `AgentTLC' getting out of sync,
   TLAPS can check that the definitions are equivalent. *)
THEOREM AgentTLC = AgentReal
BY DEF AgentTLC, AgentReal, Agent, AgentProgress

(* A next step is one in which any of these sub-components takes a step: *)
Next ==
  \/ User
  \/ Orchestrator
  \/ Allocator
  \/ Scheduler
  \/ AgentTLC
  \/ Reaper
  \* For model checking: don't report deadlock if we're limiting events
  \/ (nEvents = maxEvents /\ UNCHANGED vars)

(* This is a ``temporal formula''. It takes a sequence of states representing the
   changing state of the world and evaluates to TRUE if that sequences of states is
   a possible behaviour of SwarmKit. *)
Spec ==
  \* The first state in the behaviour must satisfy Init:
  /\ Init
  \* All consecutive pairs of states must satisfy Next or leave `vars' unchanged:
  /\ [][Next]_vars
  (* Some actions are required to happen eventually. For example, a behaviour in
     which SwarmKit stops doing anything forever, even though it could advance some task
     from the `new' state, isn't a valid behaviour of the system.
     This property is called ``weak fairness''. *)
  /\ WF_vars(OrchestratorProgress)
  /\ WF_vars(AllocatorProgress)
  /\ WF_vars(Scheduler)
  /\ WF_vars(AgentProgress /\ UNCHANGED services)
  /\ WF_vars(Reaper)
  /\ WF_vars(WorkerUp /\ UNCHANGED services)
     (* We don't require fairness of:
        - User (we don't control them),
        - RestartTask (services aren't required to be updated),
        - RejectTask (tasks aren't required to be rejected),
        - ContainerExit (we don't specify image behaviour) or
        - WorkerDown (workers aren't required to fail). *)

-------------------------------------------------------------------------------
\* Properties to verify

(* These are properties that should follow automatically if the system behaves as
   described by `Spec' in the previous section. *)

\* A state invariant (things that should be true in every state).
Inv ==
  \A t \in tasks :
    (* Every task has a service:

       TODO: The spec says: ``In some cases, there are tasks that exist independent of any service.
             These do not have a value set in service_id.''. Add an example of one. *)
    /\ t.service \in DOMAIN services
    \* Tasks have nodes once they reach `assigned', except maybe if rejected:
    /\ assigned \preceq State(t) => t.node \in Node \/ State(t) = rejected
    \* `remove' is only used as a desired state, not an actual one:
    /\ State(t) # remove
    \* Task IDs are unique
    /\ \A t2 \in tasks : Id(t) = Id(t2) => t = t2

(* The state of task `i' in `S', or `null' if it doesn't exist *)
Get(S, i) ==
  LET cand == { x \in S : Id(x) = i }
  IN  IF cand = {} THEN null
                   ELSE State(CHOOSE x \in cand : TRUE)

(* An action in which all transitions were valid. *)
StepTransitionsOK ==
  LET permitted == { << x, x >> : x \in TaskState } \union  \* No change is always OK
    CASE Orchestrator -> Transitions.orchestrator
      [] Allocator    -> Transitions.allocator
      [] Scheduler    -> Transitions.scheduler
      [] Agent        -> Transitions.agent
      [] Reaper       -> Transitions.reaper
      [] OTHER        -> {}
    oldIds == IdSet(tasks)
    newIds == IdSet(tasks')
  IN
  \A id \in newIds \union oldIds :
     << Get(tasks, id), Get(tasks', id) >> \in permitted

(* Some of the expressions below are ``temporal formulas''. Unlike state expressions and actions,
   these look at a complete behaviour (sequence of states). Summary of notation:

   [] means ``always''. e.g. []x=3 means that `x = 3' in all states.

   <> means ``eventually''. e.g. <>x=3 means that `x = 3' in some state.

   `x=3' on its own means that `x=3' in the initial state.
*)

\* A temporal formula that checks every step satisfies StepTransitionsOK (or `vars' is unchanged)
TransitionsOK ==
  [][StepTransitionsOK]_vars

(* Every service has the right number of running tasks (the system is in the desired state). *)
InDesiredState ==
  \A sid \in DOMAIN services :
    \* We're not trying to remove the service:
    /\ ~services[sid].remove
    \* The service has the correct set of running replicas:
    /\ LET runningTasks  == { t \in TasksOf(sid) : State(t) = running }
           nRunning      == Cardinality(runningTasks)
       IN
       CASE IsReplicated(sid) ->
              /\ nRunning = services[sid].replicas
         [] IsGlobal(sid) ->
              \* We have as many tasks as nodes:
              /\ nRunning = Cardinality(Node)
              \* We have a task for every node:
              /\ { t.node : t \in runningTasks } = Node
    \* The service does not have too many terminated tasks
    /\ \A slot \in VSlotsOf(sid) :
       LET terminated == { t \in TasksOfVSlot(sid, slot) : ~Runnable(t) }
       IN  Cardinality(terminated) <= maxTerminated

(* The main property we want to check.

   []<> means ``always eventually'' (``infinitely-often'')

   <>[] means ``eventually always'' (always true after some point)

   This temporal formula says that if we only experience a finite number of
   problems then the system will eventually settle on InDesiredState.
*)
EventuallyAsDesired ==
  \/ []<> <<User>>_vars               \* Either the user keeps changing the configuration,
  \/ []<> <<RestartTask>>_vars        \* or restarting/updating tasks,
  \/ []<> <<WorkerDown>>_vars         \* or workers keep failing,
  \/ []<> <<RejectTask>>_vars         \* or workers keep rejecting tasks,
  \/ []<> <<ContainerExit>>_vars      \* or the containers keep exiting,
  \/ <>[] InDesiredState              \* or we eventually get to the desired state and stay there.

=============================================================================
