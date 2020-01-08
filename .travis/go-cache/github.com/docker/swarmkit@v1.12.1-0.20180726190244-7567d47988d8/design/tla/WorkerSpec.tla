----------------------------- MODULE WorkerSpec -----------------------------

EXTENDS Types, Tasks, EventCounter

VARIABLE nodes          \* Maps nodes to SwarmKit's view of their NodeState

(* The possible states of a node, as recorded by SwarmKit. *)
nodeUp   == "up"
nodeDown == "down"
NodeState == { nodeUp, nodeDown }

WorkerTypeOK ==
  \* Nodes are up or down
  /\ nodes \in [ Node -> NodeState ]

-----------------------------------------------------------------------------

\*  Actions performed by worker nodes (actually, by the dispatcher on their behalf)

(* SwarmKit thinks the node is up. i.e. the agent is connected to a manager. *)
IsUp(n) == nodes[n] = nodeUp

(* Try to advance containers towards `desired_state' if we're not there yet. *)
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
        /\ UpdateTasks(t :> t2)

(* A running container finishes because we stopped it. *)
ShutdownComplete ==
  /\ UNCHANGED << nodes, nEvents >>
  /\ \E t \in tasks :
     /\ t.desired_state \in {shutdown, remove}                  \* We are trying to stop it
     /\ State(t) = running                                      \* It is currently running
     /\ IsUp(t.node)
     /\ UpdateTasks(t :> [t EXCEPT !.status.state = shutdown])  \* It becomes shutdown

(* A node can reject a task once it's responsible for it (it has reached `assigned')
   until it reaches the `running' state.
   Note that an ``accepted'' task can still be rejected. *)
RejectTask ==
  /\ UNCHANGED << nodes >>
  /\ CountEvent
  /\ \E t \in tasks :
       /\ State(t) \in { assigned, accepted, preparing, ready, starting }
       /\ IsUp(t.node)
       /\ UpdateTasks(t :> [t EXCEPT !.status.state = rejected])

(* We notify the managers that some running containers have finished.
   There might be several updates at once (e.g. if we're reconnecting). *)
ContainerExit ==
  /\ UNCHANGED << nodes >>
  /\ CountEvent
  /\ \E n \in Node :
        /\ IsUp(n)
        /\ \E ts \in SUBSET { t \in tasks : t.node = n /\ State(t) = running } :
           \* Each container could have ended in either state:
           \E s2 \in [ ts -> { failed, complete } ] :
             UpdateTasks( [ t \in ts |->
                             [t EXCEPT !.status.state =
                               \* Report `failed' as `shutdown' if we wanted to shut down
                               IF s2[t] = failed /\ t.desired_state = shutdown THEN shutdown
                               ELSE s2[t]]
                        ] )

(* Tasks assigned to a node and for which the node is responsible. *)
TasksOwnedByNode(n) == { t \in tasks :
  /\ t.node = n
  /\ assigned \preceq State(t)
  /\ State(t) \prec remove
}

(* The dispatcher notices that the worker is down (the connection is lost). *)
WorkerDown ==
  /\ UNCHANGED << tasks >>
  /\ CountEvent
  /\ \E n \in Node :
       /\ IsUp(n)
       /\ nodes' = [nodes EXCEPT ![n] = nodeDown]

(* When the node reconnects to the cluster, it gets an assignment set from the dispatcher
   which does not include any tasks that have been marked orphaned and then deleted.
   Any time an agent gets an assignment set that does not include some task it has running,
   it shuts down those tasks. *)
WorkerUp ==
  /\ UNCHANGED << tasks, nEvents >>
  /\ \E n \in Node :
       /\ ~IsUp(n)
       /\ nodes' = [nodes EXCEPT ![n] = nodeUp]

(* If SwarmKit sees a node as down for a long time (48 hours or so) then
   it marks all the node's tasks as orphaned.

   ``Moving a task to the Orphaned state is not desirable,
   because it's the one case where we break the otherwise invariant
   that the agent sets all states past ASSIGNED.''
*)
OrphanTasks ==
  /\ UNCHANGED << nodes, nEvents >>
  /\ \E n \in Node :
       LET affected == { t \in TasksOwnedByNode(n) : Runnable(t) }
       IN
       /\ ~IsUp(n)    \* Node `n' is still detected as down
       /\ UpdateTasks([ t \in affected |->
                         [t EXCEPT !.status.state = orphaned] ])

(* Actions we require to happen eventually when possible. *)
AgentProgress ==
  \/ ProgressTask
  \/ ShutdownComplete
  \/ OrphanTasks
  \/ WorkerUp

(* All actions of the agent/worker. *)
Agent ==
  \/ AgentProgress
  \/ RejectTask
  \/ ContainerExit
  \/ WorkerDown

=============================================================================
