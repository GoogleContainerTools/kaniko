# Nomenclature

To keep track of the software components in swarm, this document defines
various aspects of the swarm system as referenced in _this_ code base.

Several of these definitions may be a part of the product, while others are
simply for communicating about backend components. Where this distinction is
important, it will be called out.

## Overview

There are several moving parts in a swarm cluster. This section attempts to
define the high-level aspects that can provide context to the specifics.

To begin, we'll define the concept of a _cluster_.

### Cluster

A _cluster_ is made up of an organized set of Docker _Engines_ configured in a
manner to allow the dispatch of _services_.

### Node

A _Node_ refers to an active member in a cluster. Nodes can execute work and / or
act as a cluster _manager_.

### Manager

A _manager_ accepts _services_ defined by users through the cluster API. When a
valid _service_ is provided, the manager will generate tasks, allocate resources
and dispatch _tasks_ to an available _node_.

_Managers_ operate in a coordinated group, organized via the Raft protocol.
When a quorum is available, a leader will be elected to handle all API requests
and all other members of the quorum will proxy to the leader.

#### Orchestrator

The _Orchestrator_ ensures that services have the appropriate set of tasks
running in the _cluster_ according to the _service_ configuration and polices.

#### Allocator

The _allocator_ dispenses resources, such as volumes and networks to tasks, as required.

#### Scheduler

The _scheduler_ assigns _tasks_ to available nodes.

#### Dispatcher

The _dispatcher_ directly handles all _agent_ connections. This includes
registration, session management, and notification of task assignment.

### Worker

A _worker_ is a complete _Engine_ joined to a _cluster_. It receives and executes
_tasks_ while reporting on their status.

A worker's _agent_ coordinates the receipt of task assignments and ensures status
is correctly reported to the _dispatcher_.

#### Engine

The _Engine_ is shorthand for the _Docker Engine_. It runs containers
distributed via the _scheduler_ -> _dispatcher_ -> _agent_ pipeline.

#### Agent

The _agent_ coordinates the dispatch of work for a _worker_. The _agent_
maintains a connection to the _dispatcher_, waiting for the current set of
tasks assigned to the node. Assigned tasks are then dispatched to the Engine.
The agent notifies the _dispatcher_ of the current state of assigned tasks.

This is roughly analogous to a real life talent agent who ensures the worker
has the correct set of _tasks_ and lets others know what the worker is doing.

While we refer to a cluster Engine as a "worker", the term _agent_ encompasses
only the component of a worker that communicates with the dispatcher.

## Objects

An _object_ is any configuration component accessed at the top-level. These
typically include a set of APIs to inspect the objects and manipulate them
through a _spec_. 

_Objects_ are typically broken up into a _spec_ component and a set of fields
to keep track of the implementation of the _spec_. The _spec_ represents the
users intent. When a user wants to modify an object, only the spec portion is
provided. When an object flows through the system, the spec portion is left
untouched by all cluster components.

Examples of _objects_ include `Service`, `Task`, `Network` and `Volume`.

### Service

The _service_ instructs the cluster on what needs to be run. It is the central
structure of the cluster system and the primary root of user interaction. The
service informs the orchestrator about how to create and manage tasks.

A _service_ is configured and updated with the `ServiceSpec`. The
central structure of the spec is a `RuntimeSpec`. This contains definitions on
how to run a container, including attachments to volumes and networks.

### Task

A _task_ represents a unit of work assigned to a node. A _task_ carries a runtime
definition that describes how to run the container.

As a task flows through the system, its state is updated accordingly. The state
of a task only increases monotonically, meaning that once the task has failed,
it must be recreated to retry.

The assignment of a _task_ to a node is immutable. Once a the task is bound to a
node, it can only run on that node or fail.

### Volume
### Network

