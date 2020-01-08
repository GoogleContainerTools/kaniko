---------------------------- MODULE WorkerImpl ----------------------------------

EXTENDS TLC, Types, Tasks, EventCounter

(*
`WorkerSpec' provides a high-level specification of worker nodes that only refers to
the state of the tasks recorded in SwarmKit's store. This specification (WorkerImpl)
refines WorkerSpec by also modelling the state of the containers running on a node.
It should be easier to see that this lower-level specification corresponds to what
actually happens on worker nodes.

The reason for having this in a separate specification is that including the container
state greatly increases the number of states to be considered and so slows down model
checking. Instead of checking

  SwarmKit /\ WorkerImpl => EventuallyAsDesired

(which is very slow), we check two separate expressions:

  SwarmKit /\ WorkerSpec => EventuallyAsDesired
  WorkerImpl => WorkerSpec

TLAPS can check that separating the specification in this way makes sense: *)
THEOREM ASSUME TEMPORAL SwarmKit, TEMPORAL WorkerSpec,
               TEMPORAL WorkerImpl, TEMPORAL EventuallyAsDesired,
               TEMPORAL Env,  \* A simplified version of SwarmKit
               SwarmKit /\ WorkerSpec => EventuallyAsDesired,
               Env /\ WorkerImpl => WorkerSpec,
               SwarmKit => Env
        PROVE  SwarmKit /\ WorkerImpl => EventuallyAsDesired
OBVIOUS

\* This worker's node ID
CONSTANT node
ASSUME node \in Node

VARIABLES nodes         \* Defined in WorkerSpec.tla
VARIABLE containers     \* The actual container state on the node, indexed by ModelTaskId

(* The high-level specification of worker nodes.
   This module should be a refinement of `WS'. *)
WS == INSTANCE WorkerSpec

terminating == "terminating"        \* A container which we're trying to stop

(* The state of an actual container on a worker node. *)
ContainerState == { running, terminating, complete, failed }

(* A running container finishes running on its own (or crashes). *)
ContainerExit ==
  /\ UNCHANGED << nodes, tasks >>
  /\ CountEvent
  /\ \E id \in DOMAIN containers,
        s2 \in {failed, complete} :      \* Either a successful or failed exit status
        /\ containers[id] = running
        /\ containers' = [containers EXCEPT ![id] = s2]

(* A running container finishes because we stopped it. *)
ShutdownComplete ==
  /\ UNCHANGED << nodes, tasks, nEvents >>
  /\ \E id \in DOMAIN containers :
        /\ containers[id] = terminating
        /\ containers' = [containers EXCEPT ![id] = failed]

(* SwarmKit thinks the node is up. i.e. the agent is connected to a manager. *)
IsUp(n) == WS!IsUp(n)

(* The new value that `containers' should take after getting an update from the
   managers. If the managers asked us to run a container and then stop mentioning
   that task, we shut the container down and (once stopped) remove it. *)
DesiredContainers ==
  LET WantShutdown(id) ==
        \* The managers stop mentioning the task, or ask for it to be stopped.
        \/ id \notin IdSet(tasks)
        \/ running \prec (CHOOSE t \in tasks : Id(t) = id).desired_state
      (* Remove containers that no longer have tasks, once they've stopped. *)
      rm == { id \in DOMAIN containers :
                  /\ containers[id] \in { complete, failed }
                  /\ id \notin IdSet(tasks) }
  IN [ id \in DOMAIN containers \ rm |->
           IF containers[id] = running /\ WantShutdown(id) THEN terminating
           ELSE containers[id]
     ]

(* The updates that SwarmKit should apply to its store to bring it up-to-date
   with the real state of the containers. *)
RequiredTaskUpdates ==
  LET \* Tasks the manager is expecting news about:
      oldTasks == { t \in tasks : t.node = node /\ State(t) = running }
      \* The state to report for task `t':
      ReportFor(t) ==
        IF Id(t) \notin DOMAIN containers THEN \* We were asked to forget about this container.
          shutdown \* SwarmKit doesn't care which terminal state we finish in.
        ELSE IF /\ containers[Id(t)] = failed       \* It's terminated and
                /\ t.desired_state = shutdown THEN  \* we wanted to shut it down,
          shutdown \* Report a successful shutdown
        ELSE IF containers[Id(t)] = terminating THEN
          running  \* SwarmKit doesn't record progress of the shutdown
        ELSE
          containers[Id(t)]  \* Report the actual state
  IN [ t \in oldTasks |-> [ t EXCEPT !.status.state = ReportFor(t) ]]

(* Our node synchronises its state with a manager. *)
DoSync ==
   /\ containers' = DesiredContainers
   /\ UpdateTasks(RequiredTaskUpdates)

(* Try to advance containers towards `desired_state' if we're not there yet.

   XXX: do we need a connection to the manager to do this, or can we make progress
   while disconnected and just report the final state?
*)
ProgressTask ==
  /\ UNCHANGED << nodes, nEvents >>
  /\ \E t  \in tasks,
        s2 \in TaskState :   \* The state we want to move to
        LET t2 == [t EXCEPT !.status.state = s2]
        IN
        /\ s2 \preceq t.desired_state       \* Can't be after the desired state
        /\ << State(t), State(t2) >> \in {  \* Possible ``progress'' (desirable) transitions
             << assigned, accepted >>,
             << accepted, preparing >>,
             << preparing, ready >>,
             << ready, starting >>,
             << starting, running >>
           }
        /\ IsUp(t.node)                     \* Node must be connected to SwarmKit
        /\ IF s2 = running THEN
              \* The container started running
              containers' = Id(t) :> running @@ containers
           ELSE
              UNCHANGED containers
        /\ UpdateTasks(t :> t2)

(* The agent on the node synchronises with a manager. *)
SyncWithManager ==
  /\ UNCHANGED << nodes, nEvents >>
  /\ IsUp(node)
  /\ DoSync

(* We can reject a task once we're responsible for it (it has reached `assigned')
   until it reaches the `running' state.
   Note that an ``accepted'' task can still be rejected. *)
RejectTask ==
  /\ UNCHANGED << nodes, containers >>
  /\ CountEvent
  /\ \E t \in tasks :
       /\ State(t) \in { assigned, accepted, preparing, ready, starting }
       /\ t.node = node
       /\ IsUp(node)
       /\ UpdateTasks(t :> [t EXCEPT !.status.state = rejected])

(* The dispatcher notices that the worker is down (the connection is lost). *)
WorkerDown ==
  /\ UNCHANGED << tasks, containers >>
  /\ CountEvent
  /\ \E n \in Node :
       /\ IsUp(n)
       /\ nodes' = [nodes EXCEPT ![n] = WS!nodeDown]

(* When the node reconnects to the cluster, it gets an assignment set from the dispatcher
   which does not include any tasks that have been marked orphaned and then deleted.
   Any time an agent gets an assignment set that does not include some task it has running,
   it shuts down those tasks.

   We model this separately with the `SyncWithManager' action. *)
WorkerUp ==
  /\ UNCHANGED << nEvents, containers, tasks >>
  /\ \E n \in Node :
       /\ ~IsUp(n)
       /\ nodes' = [nodes EXCEPT ![n] = WS!nodeUp]

(* Tasks assigned to a node and for which the node is responsible. *)
TasksOwnedByNode(n) == { t \in tasks :
  /\ t.node = n
  /\ assigned \preceq State(t)
  /\ State(t) \prec remove
}

(* If SwarmKit sees a node as down for a long time (48 hours or so) then
   it marks all the node's tasks as orphaned.
   Note that this sets the actual state, not the desired state.

   ``Moving a task to the Orphaned state is not desirable,
   because it's the one case where we break the otherwise invariant
   that the agent sets all states past ASSIGNED.''
*)
OrphanTasks ==
  /\ UNCHANGED << nodes, containers, nEvents >>
  /\ LET affected == { t \in TasksOwnedByNode(node) : Runnable(t) }
     IN
     /\ ~IsUp(node)    \* Our connection to the agent is down
     /\ UpdateTasks([ t \in affected |->
                         [t EXCEPT !.status.state = orphaned] ])

(* The worker reboots. All containers are terminated. *)
WorkerReboot ==
  /\ UNCHANGED << nodes, tasks >>
  /\ CountEvent
  /\ containers' = [ id \in DOMAIN containers |->
                       LET state == containers[id]
                       IN  CASE state \in {running, terminating} -> failed
                             [] state \in {complete, failed}     -> state
                   ]

(* Actions we require to happen eventually when possible. *)
AgentProgress ==
  \/ ProgressTask
  \/ OrphanTasks
  \/ WorkerUp
  \/ ShutdownComplete
  \/ SyncWithManager

(* All actions of the agent/worker. *)
Agent ==
  \/ AgentProgress
  \/ RejectTask
  \/ WorkerDown
  \/ ContainerExit
  \/ WorkerReboot

-------------------------------------------------------------------------------
\* A simplified model of the rest of the system

(* A new task is created. *)
CreateTask ==
  /\ UNCHANGED << containers, nEvents, nodes >>
  /\ \E t \in Task :    \* `t' is the new task
      \* Don't reuse IDs (only really an issue for model checking)
      /\ Id(t) \notin IdSet(tasks)
      /\ Id(t) \notin DOMAIN containers
      /\ State(t) = new
      /\ t.desired_state \in { ready, running }
      /\ \/ /\ t.node = unassigned  \* A task for a replicated service
            /\ t.slot \in Slot
         \/ /\ t.node \in Node      \* A task for a global service
            /\ t.slot = global
      /\ ~\E t2 \in tasks : \* All tasks of a service have the same mode
            /\ t.service = t2.service
            /\ (t.slot = global) # (t2.slot = global)
      /\ tasks' = tasks \union {t}

(* States before `assigned' aren't shared with worker nodes, so modelling them
   isn't very useful. You can use this in a model to override `CreateTask' to
   speed things up a bit. It creates tasks directly in the `assigned' state. *)
CreateTaskQuick ==
  /\ UNCHANGED << containers, nEvents, nodes >>
  /\ \E t \in Task :
      /\ Id(t) \notin IdSet(tasks)
      /\ Id(t) \notin DOMAIN containers
      /\ State(t) = assigned
      /\ t.desired_state \in { ready, running }
      /\ t.node \in Node
      /\ t.slot \in Slot \union {global}
      /\ ~\E t2 \in tasks : \* All tasks of a service have the same mode
            /\ t.service = t2.service
            /\ (t.slot = global) # (t2.slot = global)
      /\ tasks' = tasks \union {t}

(* The state or desired_state of a task is updated. *)
UpdateTask ==
  /\ UNCHANGED << containers, nEvents, nodes >>
  /\ \E t \in tasks, t2 \in Task :  \* `t' becomes `t2'
        /\ Id(t) = Id(t2)           \* The ID can't change
        /\ State(t) # State(t2) =>  \* If the state changes then
             \E actor \in DOMAIN Transitions :  \* it is a legal transition
                 /\ actor = "agent"  =>  t.node # node    \* and not one our worker does
                 /\ << State(t), State(t2) >> \in Transitions[actor]
        \* When tasks reach the `assigned' state, they must have a node
        /\ IF State(t2) = assigned /\ t.node = unassigned THEN t2.node \in Node
                                                          ELSE t2.node = t.node
        /\ UpdateTasks(t :> t2)

(* The reaper removes a task. *)
RemoveTask ==
  /\ UNCHANGED << containers, nEvents, nodes >>
  /\ \E t \in tasks :
      /\ << State(t), null >> \in Transitions.reaper
      /\ tasks' = tasks \ {t}

(* Actions of our worker's environment (i.e. SwarmKit and other workers). *)
OtherComponent ==
  \/ CreateTask
  \/ UpdateTask
  \/ RemoveTask

-------------------------------------------------------------------------------
\* A complete system

vars == << tasks, nEvents, nodes, containers >>

Init ==
  /\ tasks = {}
  /\ containers = << >>
  /\ nodes = [ n \in Node |-> WS!nodeUp ]
  /\ InitEvents

Next ==
  \/ OtherComponent
  \/ Agent

(* The specification for our worker node. *)
Impl == Init /\ [][Next]_vars /\ WF_vars(AgentProgress)

-------------------------------------------------------------------------------

TypeOK ==
  /\ TasksTypeOK
  \* The node's container map maps IDs to states
  /\ DOMAIN containers \in SUBSET ModelTaskId
  /\ containers \in [ DOMAIN containers -> ContainerState ]

wsVars == << tasks, nEvents, nodes >>

(* We want to check that a worker implementing `Impl' is also implementing
   `WorkerSpec'. i.e. we need to check that Impl => WSSpec. *)
WSSpec ==
  /\ [][WS!Agent \/ OtherComponent]_wsVars
  /\ WF_wsVars(WS!AgentProgress)

=============================================================================
