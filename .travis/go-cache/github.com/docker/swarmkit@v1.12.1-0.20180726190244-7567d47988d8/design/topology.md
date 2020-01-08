# Topology aware scheduling

## Background

There is often interest in making the scheduler aware of factors such as
availability zones. This document specifies a generic way to customize scheduler
behavior based on labels attached to nodes.

## Approach

The scheduler consults a repeated field named `Preferences` under `Placement`
when it places tasks.  These "placement preferences" are be listed in
decreasing order of precedence, and have higher precedence than the default
scheduler logic.

These placement preferences are be interpreted based on their types, but the
initially supported "spread over" message tells the scheduler to spread tasks
evenly between nodes which have each distinct value of the referenced node or
engine label.

## Protobuf definitions

In the `Placement` message under `TaskSpec`, we define a repeated field called
`Preferences`.

```
repeated PlacementPreference preferences = 2;
```

`PlacementPreference` is a message that specifies how to act on a label.
The initially supported preference would is "spread".

```
message SpreadOver {
    string spread_descriptor = 1; // label descriptor, such as engine.labels.az
    // TODO: support node information beyond engine and node labels

    // TODO: in the future, add a map that provides weights for weighted
    // spreading.
}

message PlacementPreference {
    oneof Preference {
        SpreadOver spread = 1;
    }

    Preference pref = 1;
}
```

## Behavior

A simple use of this feature would be to spread tasks evenly between multiple
availability zones. The way to do this would be to create an engine label on
each node indicating its availability zone, and then create a
`PlacementPreference` with type `SpreadOver` which references the engine label.
The scheduler would prioritize balance between the availability zones, and if
it ever has a choice between multiple nodes in the preferred availability zone
(or a tie between AZs), it would choose the node based on its built-in logic.
As of Docker 1.13, this logic will prefer to schedule a task on the node which
has the fewest tasks associated with the particular service.

A slightly more complicated use case involves hierarchical topology. Say there
are two datacenters, which each have four rows, each row having 20 racks. To
spread tasks evenly at each of these levels, there could be three `SpreadOver`
messages in `Preferences`. The first would spread over datacenters, the second
would spread over rows, and the third would spread over racks. This ensures that
the highest precedence goes to spreading tasks between datacenters, but after
that, tasks are evenly distributed between rows and then racks.

Nodes that are missing the label used by `SpreadOver` will still receive task
assignments. As a group, they will receive tasks in equal proportion to any of
the other groups identified by a specific label value. In a sense, a missing
label is the same as having the label with a null value attached to it. If the
service should only run on nodes with the label being used for the `SpreadOver`
preference, the preference should be combined with a constraint.

## Future enhancements

- In addition to SpreadOver, we could add a PackInto with opposite behavior. It
  would try to locate tasks on nodes that share the same label value as other
  tasks, subject to constraints. By combining multiple SpreadOver and PackInto
  preferences, it would be possible to do things like spread over datacenters
  and then pack into racks within those datacenters.

- Support weighted spreading, i.e. for situations where one datacenter has more
  servers than another. This could be done by adding a map to SpreadOver
  containing weights for each label value.

- Support acting on items other than node labels and engine labels. For example,
  acting on node IDs to spread or pack over individual nodes, or on resource
  specifications to implement soft resource constraints.
