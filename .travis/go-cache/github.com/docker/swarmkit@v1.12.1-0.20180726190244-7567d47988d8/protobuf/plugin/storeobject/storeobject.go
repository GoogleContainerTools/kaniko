package storeobject

import (
	"strings"

	"github.com/docker/swarmkit/protobuf/plugin"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

// FIXME(aaronl): Look at fields inside the descriptor instead of
// special-casing based on name.
var typesWithNoSpec = map[string]struct{}{
	"Task":      {},
	"Resource":  {},
	"Extension": {},
}

type storeObjectGen struct {
	*generator.Generator
	generator.PluginImports
	eventsPkg  generator.Single
	stringsPkg generator.Single
}

func init() {
	generator.RegisterPlugin(new(storeObjectGen))
}

func (d *storeObjectGen) Name() string {
	return "storeobject"
}

func (d *storeObjectGen) Init(g *generator.Generator) {
	d.Generator = g
}

func (d *storeObjectGen) genMsgStoreObject(m *generator.Descriptor, storeObject *plugin.StoreObject) {
	ccTypeName := generator.CamelCaseSlice(m.TypeName())

	// Generate event types

	d.P("type ", ccTypeName, "CheckFunc func(t1, t2 *", ccTypeName, ") bool")
	d.P()

	// generate the event object type interface for this type
	// event types implement some empty interfaces, for ease of use, like such:
	//
	//   type EventCreate interface {
	//     IsEventCreatet() bool
	//   }
	//
	//   type EventNode interface {
	//     IsEventNode() bool
	//   }
	//
	// then, each event has the corresponding interfaces implemented for its
	// type. for example:
	//
	//   func (e EventCreateNode) IsEventCreate() bool {
	//     return true
	//   }
	//
	//   func (e EventCreateNode) IsEventNode() bool {
	//     return true
	//   }
	//
	// this lets the user filter events based on their interface type.
	// note that the event type for each object type needs to be generated for
	// each object. the event change type (Create/Update/Delete) is
	// hand-written in the storeobject.go file because they are only needed
	// once.
	d.P("type Event", ccTypeName, " interface {")
	d.In()
	d.P("IsEvent", ccTypeName, "() bool")
	d.Out()
	d.P("}")
	d.P()

	for _, event := range []string{"Create", "Update", "Delete"} {
		d.P("type Event", event, ccTypeName, " struct {")
		d.In()
		d.P(ccTypeName, " *", ccTypeName)
		if event == "Update" {
			d.P("Old", ccTypeName, " *", ccTypeName)
		}
		d.P("Checks []", ccTypeName, "CheckFunc")
		d.Out()
		d.P("}")
		d.P()
		d.P("func (e Event", event, ccTypeName, ") Matches(apiEvent ", d.eventsPkg.Use(), ".Event) bool {")
		d.In()
		d.P("typedEvent, ok := apiEvent.(Event", event, ccTypeName, ")")
		d.P("if !ok {")
		d.In()
		d.P("return false")
		d.Out()
		d.P("}")
		d.P()
		d.P("for _, check := range e.Checks {")
		d.In()
		d.P("if !check(e.", ccTypeName, ", typedEvent.", ccTypeName, ") {")
		d.In()
		d.P("return false")
		d.Out()
		d.P("}")
		d.Out()
		d.P("}")
		d.P("return true")
		d.Out()
		d.P("}")
		d.P()

		// implement event change type interface (IsEventCreate)
		d.P("func (e Event", event, ccTypeName, ") IsEvent", event, "() bool {")
		d.In()
		d.P("return true")
		d.Out()
		d.P("}")
		d.P()

		// implement event object type interface (IsEventNode)
		d.P("func (e Event", event, ccTypeName, ") IsEvent", ccTypeName, "() bool {")
		d.In()
		d.P("return true")
		d.Out()
		d.P("}")
		d.P()
	}

	// Generate methods for this type

	d.P("func (m *", ccTypeName, ") CopyStoreObject() StoreObject {")
	d.In()
	d.P("return m.Copy()")
	d.Out()
	d.P("}")
	d.P()

	d.P("func (m *", ccTypeName, ") GetMeta() Meta {")
	d.In()
	d.P("return m.Meta")
	d.Out()
	d.P("}")
	d.P()

	d.P("func (m *", ccTypeName, ") SetMeta(meta Meta) {")
	d.In()
	d.P("m.Meta = meta")
	d.Out()
	d.P("}")
	d.P()

	d.P("func (m *", ccTypeName, ") GetID() string {")
	d.In()
	d.P("return m.ID")
	d.Out()
	d.P("}")
	d.P()

	d.P("func (m *", ccTypeName, ") EventCreate() Event {")
	d.In()
	d.P("return EventCreate", ccTypeName, "{", ccTypeName, ": m}")
	d.Out()
	d.P("}")
	d.P()

	d.P("func (m *", ccTypeName, ") EventUpdate(oldObject StoreObject) Event {")
	d.In()
	d.P("if oldObject != nil {")
	d.In()
	d.P("return EventUpdate", ccTypeName, "{", ccTypeName, ": m, Old", ccTypeName, ": oldObject.(*", ccTypeName, ")}")
	d.Out()
	d.P("} else {")
	d.In()
	d.P("return EventUpdate", ccTypeName, "{", ccTypeName, ": m}")
	d.Out()
	d.P("}")
	d.Out()
	d.P("}")
	d.P()

	d.P("func (m *", ccTypeName, ") EventDelete() Event {")
	d.In()
	d.P("return EventDelete", ccTypeName, "{", ccTypeName, ": m}")
	d.Out()
	d.P("}")
	d.P()

	// Generate event check functions

	if storeObject.WatchSelectors.ID != nil && *storeObject.WatchSelectors.ID {
		d.P("func ", ccTypeName, "CheckID(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return v1.ID == v2.ID")
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.IDPrefix != nil && *storeObject.WatchSelectors.IDPrefix {
		d.P("func ", ccTypeName, "CheckIDPrefix(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return ", d.stringsPkg.Use(), ".HasPrefix(v2.ID, v1.ID)")
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.Name != nil && *storeObject.WatchSelectors.Name {
		d.P("func ", ccTypeName, "CheckName(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		// Node is a special case
		if *m.Name == "Node" {
			d.P("if v1.Description == nil || v2.Description == nil {")
			d.In()
			d.P("return false")
			d.Out()
			d.P("}")
			d.P("return v1.Description.Hostname == v2.Description.Hostname")
		} else if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
			d.P("return v1.Annotations.Name == v2.Annotations.Name")
		} else {
			d.P("return v1.Spec.Annotations.Name == v2.Spec.Annotations.Name")
		}
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.NamePrefix != nil && *storeObject.WatchSelectors.NamePrefix {
		d.P("func ", ccTypeName, "CheckNamePrefix(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		// Node is a special case
		if *m.Name == "Node" {
			d.P("if v1.Description == nil || v2.Description == nil {")
			d.In()
			d.P("return false")
			d.Out()
			d.P("}")
			d.P("return ", d.stringsPkg.Use(), ".HasPrefix(v2.Description.Hostname, v1.Description.Hostname)")
		} else if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
			d.P("return ", d.stringsPkg.Use(), ".HasPrefix(v2.Annotations.Name, v1.Annotations.Name)")
		} else {
			d.P("return ", d.stringsPkg.Use(), ".HasPrefix(v2.Spec.Annotations.Name, v1.Spec.Annotations.Name)")
		}
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.Custom != nil && *storeObject.WatchSelectors.Custom {
		d.P("func ", ccTypeName, "CheckCustom(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		// Node is a special case
		if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
			d.P("return checkCustom(v1.Annotations, v2.Annotations)")
		} else {
			d.P("return checkCustom(v1.Spec.Annotations, v2.Spec.Annotations)")
		}
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.CustomPrefix != nil && *storeObject.WatchSelectors.CustomPrefix {
		d.P("func ", ccTypeName, "CheckCustomPrefix(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		// Node is a special case
		if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
			d.P("return checkCustomPrefix(v1.Annotations, v2.Annotations)")
		} else {
			d.P("return checkCustomPrefix(v1.Spec.Annotations, v2.Spec.Annotations)")
		}
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.NodeID != nil && *storeObject.WatchSelectors.NodeID {
		d.P("func ", ccTypeName, "CheckNodeID(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return v1.NodeID == v2.NodeID")
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.ServiceID != nil && *storeObject.WatchSelectors.ServiceID {
		d.P("func ", ccTypeName, "CheckServiceID(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return v1.ServiceID == v2.ServiceID")
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.Slot != nil && *storeObject.WatchSelectors.Slot {
		d.P("func ", ccTypeName, "CheckSlot(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return v1.Slot == v2.Slot")
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.DesiredState != nil && *storeObject.WatchSelectors.DesiredState {
		d.P("func ", ccTypeName, "CheckDesiredState(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return v1.DesiredState == v2.DesiredState")
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.Role != nil && *storeObject.WatchSelectors.Role {
		d.P("func ", ccTypeName, "CheckRole(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return v1.Role == v2.Role")
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.Membership != nil && *storeObject.WatchSelectors.Membership {
		d.P("func ", ccTypeName, "CheckMembership(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return v1.Spec.Membership == v2.Spec.Membership")
		d.Out()
		d.P("}")
		d.P()
	}

	if storeObject.WatchSelectors.Kind != nil && *storeObject.WatchSelectors.Kind {
		d.P("func ", ccTypeName, "CheckKind(v1, v2 *", ccTypeName, ") bool {")
		d.In()
		d.P("return v1.Kind == v2.Kind")
		d.Out()
		d.P("}")
		d.P()
	}

	// Generate Convert*Watch function, for watch API.
	if ccTypeName == "Resource" {
		d.P("func ConvertResourceWatch(action WatchActionKind, filters []*SelectBy, kind string) ([]Event, error) {")
	} else {
		d.P("func Convert", ccTypeName, "Watch(action WatchActionKind, filters []*SelectBy) ([]Event, error) {")
	}
	d.In()
	d.P("var (")
	d.In()
	d.P("m ", ccTypeName)
	d.P("checkFuncs []", ccTypeName, "CheckFunc")
	if storeObject.WatchSelectors.DesiredState != nil && *storeObject.WatchSelectors.DesiredState {
		d.P("hasDesiredState bool")
	}
	if storeObject.WatchSelectors.Role != nil && *storeObject.WatchSelectors.Role {
		d.P("hasRole bool")
	}
	if storeObject.WatchSelectors.Membership != nil && *storeObject.WatchSelectors.Membership {
		d.P("hasMembership bool")
	}
	d.Out()
	d.P(")")
	if ccTypeName == "Resource" {
		d.P("m.Kind = kind")
		d.P("checkFuncs = append(checkFuncs, ResourceCheckKind)")
	}
	d.P()
	d.P("for _, filter := range filters {")
	d.In()
	d.P("switch v := filter.By.(type) {")

	if storeObject.WatchSelectors.ID != nil && *storeObject.WatchSelectors.ID {
		d.P("case *SelectBy_ID:")
		d.In()
		d.P(`if m.ID != "" {`)
		d.In()
		d.P("return nil, errConflictingFilters")
		d.Out()
		d.P("}")
		d.P("m.ID = v.ID")
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckID)")
		d.Out()
	}
	if storeObject.WatchSelectors.IDPrefix != nil && *storeObject.WatchSelectors.IDPrefix {
		d.P("case *SelectBy_IDPrefix:")
		d.In()
		d.P(`if m.ID != "" {`)
		d.In()
		d.P("return nil, errConflictingFilters")
		d.Out()
		d.P("}")
		d.P("m.ID = v.IDPrefix")
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckIDPrefix)")
		d.Out()
	}
	if storeObject.WatchSelectors.Name != nil && *storeObject.WatchSelectors.Name {
		d.P("case *SelectBy_Name:")
		d.In()
		if *m.Name == "Node" {
			d.P("if m.Description != nil {")
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Description = &NodeDescription{Hostname: v.Name}")

		} else if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
			d.P(`if m.Annotations.Name != "" {`)
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Annotations.Name = v.Name")
		} else {
			d.P(`if m.Spec.Annotations.Name != "" {`)
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Spec.Annotations.Name = v.Name")
		}
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckName)")
		d.Out()
	}
	if storeObject.WatchSelectors.NamePrefix != nil && *storeObject.WatchSelectors.NamePrefix {
		d.P("case *SelectBy_NamePrefix:")
		d.In()
		if *m.Name == "Node" {
			d.P("if m.Description != nil {")
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Description = &NodeDescription{Hostname: v.NamePrefix}")

		} else if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
			d.P(`if m.Annotations.Name != "" {`)
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Annotations.Name = v.NamePrefix")
		} else {
			d.P(`if m.Spec.Annotations.Name != "" {`)
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Spec.Annotations.Name = v.NamePrefix")
		}
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckNamePrefix)")
		d.Out()
	}
	if storeObject.WatchSelectors.Custom != nil && *storeObject.WatchSelectors.Custom {
		d.P("case *SelectBy_Custom:")
		d.In()
		if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
			d.P(`if len(m.Annotations.Indices) != 0 {`)
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Annotations.Indices = []IndexEntry{{Key: v.Custom.Index, Val: v.Custom.Value}}")
		} else {
			d.P(`if len(m.Spec.Annotations.Indices) != 0 {`)
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Spec.Annotations.Indices = []IndexEntry{{Key: v.Custom.Index, Val: v.Custom.Value}}")
		}
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckCustom)")
		d.Out()
	}
	if storeObject.WatchSelectors.CustomPrefix != nil && *storeObject.WatchSelectors.CustomPrefix {
		d.P("case *SelectBy_CustomPrefix:")
		d.In()
		if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
			d.P(`if len(m.Annotations.Indices) != 0 {`)
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Annotations.Indices = []IndexEntry{{Key: v.CustomPrefix.Index, Val: v.CustomPrefix.Value}}")
		} else {
			d.P(`if len(m.Spec.Annotations.Indices) != 0 {`)
			d.In()
			d.P("return nil, errConflictingFilters")
			d.Out()
			d.P("}")
			d.P("m.Spec.Annotations.Indices = []IndexEntry{{Key: v.CustomPrefix.Index, Val: v.CustomPrefix.Value}}")
		}
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckCustomPrefix)")
		d.Out()
	}
	if storeObject.WatchSelectors.ServiceID != nil && *storeObject.WatchSelectors.ServiceID {
		d.P("case *SelectBy_ServiceID:")
		d.In()
		d.P(`if m.ServiceID != "" {`)
		d.In()
		d.P("return nil, errConflictingFilters")
		d.Out()
		d.P("}")
		d.P("m.ServiceID = v.ServiceID")
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckServiceID)")
		d.Out()
	}
	if storeObject.WatchSelectors.NodeID != nil && *storeObject.WatchSelectors.NodeID {
		d.P("case *SelectBy_NodeID:")
		d.In()
		d.P(`if m.NodeID != "" {`)
		d.In()
		d.P("return nil, errConflictingFilters")
		d.Out()
		d.P("}")
		d.P("m.NodeID = v.NodeID")
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckNodeID)")
		d.Out()
	}
	if storeObject.WatchSelectors.Slot != nil && *storeObject.WatchSelectors.Slot {
		d.P("case *SelectBy_Slot:")
		d.In()
		d.P(`if m.Slot != 0 || m.ServiceID != "" {`)
		d.In()
		d.P("return nil, errConflictingFilters")
		d.Out()
		d.P("}")
		d.P("m.ServiceID = v.Slot.ServiceID")
		d.P("m.Slot = v.Slot.Slot")
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckNodeID, ", ccTypeName, "CheckSlot)")
		d.Out()
	}
	if storeObject.WatchSelectors.DesiredState != nil && *storeObject.WatchSelectors.DesiredState {
		d.P("case *SelectBy_DesiredState:")
		d.In()
		d.P(`if hasDesiredState {`)
		d.In()
		d.P("return nil, errConflictingFilters")
		d.Out()
		d.P("}")
		d.P("hasDesiredState = true")
		d.P("m.DesiredState = v.DesiredState")
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckDesiredState)")
		d.Out()
	}
	if storeObject.WatchSelectors.Role != nil && *storeObject.WatchSelectors.Role {
		d.P("case *SelectBy_Role:")
		d.In()
		d.P(`if hasRole {`)
		d.In()
		d.P("return nil, errConflictingFilters")
		d.Out()
		d.P("}")
		d.P("hasRole = true")
		d.P("m.Role = v.Role")
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckRole)")
		d.Out()
	}
	if storeObject.WatchSelectors.Membership != nil && *storeObject.WatchSelectors.Membership {
		d.P("case *SelectBy_Membership:")
		d.In()
		d.P(`if hasMembership {`)
		d.In()
		d.P("return nil, errConflictingFilters")
		d.Out()
		d.P("}")
		d.P("hasMembership = true")
		d.P("m.Spec.Membership = v.Membership")
		d.P("checkFuncs = append(checkFuncs, ", ccTypeName, "CheckMembership)")
		d.Out()
	}

	d.P("}")
	d.Out()
	d.P("}")
	d.P("var events []Event")
	d.P("if (action & WatchActionKindCreate) != 0 {")
	d.In()
	d.P("events = append(events, EventCreate", ccTypeName, "{", ccTypeName, ": &m, Checks: checkFuncs})")
	d.Out()
	d.P("}")
	d.P("if (action & WatchActionKindUpdate) != 0 {")
	d.In()
	d.P("events = append(events, EventUpdate", ccTypeName, "{", ccTypeName, ": &m, Checks: checkFuncs})")
	d.Out()
	d.P("}")
	d.P("if (action & WatchActionKindRemove) != 0 {")
	d.In()
	d.P("events = append(events, EventDelete", ccTypeName, "{", ccTypeName, ": &m, Checks: checkFuncs})")
	d.Out()
	d.P("}")
	d.P("if len(events) == 0 {")
	d.In()
	d.P("return nil, errUnrecognizedAction")
	d.Out()
	d.P("}")
	d.P("return events, nil")
	d.Out()
	d.P("}")
	d.P()

	/*                switch v := filter.By.(type) {
	default:
	        return nil, status.Errorf(codes.InvalidArgument, "selector type %T is unsupported for tasks", filter.By)
	}
	*/

	// Generate indexer by ID

	d.P("type ", ccTypeName, "IndexerByID struct{}")
	d.P()

	d.genFromArgs(ccTypeName + "IndexerByID")
	d.genPrefixFromArgs(ccTypeName + "IndexerByID")

	d.P("func (indexer ", ccTypeName, "IndexerByID) FromObject(obj interface{}) (bool, []byte, error) {")
	d.In()
	d.P("m := obj.(*", ccTypeName, ")")
	// Add the null character as a terminator
	d.P(`return true, []byte(m.ID + "\x00"), nil`)
	d.Out()
	d.P("}")

	// Generate indexer by name

	d.P("type ", ccTypeName, "IndexerByName struct{}")
	d.P()

	d.genFromArgs(ccTypeName + "IndexerByName")
	d.genPrefixFromArgs(ccTypeName + "IndexerByName")

	d.P("func (indexer ", ccTypeName, "IndexerByName) FromObject(obj interface{}) (bool, []byte, error) {")
	d.In()
	d.P("m := obj.(*", ccTypeName, ")")
	if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
		d.P(`val := m.Annotations.Name`)
	} else {
		d.P(`val := m.Spec.Annotations.Name`)
	}
	// Add the null character as a terminator
	d.P("return true, []byte(", d.stringsPkg.Use(), `.ToLower(val) + "\x00"), nil`)
	d.Out()
	d.P("}")

	// Generate custom indexer

	d.P("type ", ccTypeName, "CustomIndexer struct{}")
	d.P()

	d.genFromArgs(ccTypeName + "CustomIndexer")
	d.genPrefixFromArgs(ccTypeName + "CustomIndexer")

	d.P("func (indexer ", ccTypeName, "CustomIndexer) FromObject(obj interface{}) (bool, [][]byte, error) {")
	d.In()
	d.P("m := obj.(*", ccTypeName, ")")
	if _, hasNoSpec := typesWithNoSpec[*m.Name]; hasNoSpec {
		d.P(`return customIndexer("", &m.Annotations)`)
	} else {
		d.P(`return customIndexer("", &m.Spec.Annotations)`)
	}
	d.Out()
	d.P("}")
}

func (d *storeObjectGen) genFromArgs(indexerName string) {
	d.P("func (indexer ", indexerName, ") FromArgs(args ...interface{}) ([]byte, error) {")
	d.In()
	d.P("return fromArgs(args...)")
	d.Out()
	d.P("}")
}

func (d *storeObjectGen) genPrefixFromArgs(indexerName string) {
	d.P("func (indexer ", indexerName, ") PrefixFromArgs(args ...interface{}) ([]byte, error) {")
	d.In()
	d.P("return prefixFromArgs(args...)")
	d.Out()
	d.P("}")

}

func (d *storeObjectGen) genNewStoreAction(topLevelObjs []string) {
	// Generate NewStoreAction
	d.P("func NewStoreAction(c Event) (StoreAction, error) {")
	d.In()
	d.P("var sa StoreAction")
	d.P("switch v := c.(type) {")
	for _, ccTypeName := range topLevelObjs {
		d.P("case EventCreate", ccTypeName, ":")
		d.In()
		d.P("sa.Action = StoreActionKindCreate")
		d.P("sa.Target = &StoreAction_", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}")
		d.Out()
		d.P("case EventUpdate", ccTypeName, ":")
		d.In()
		d.P("sa.Action = StoreActionKindUpdate")
		d.P("sa.Target = &StoreAction_", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}")
		d.Out()
		d.P("case EventDelete", ccTypeName, ":")
		d.In()
		d.P("sa.Action = StoreActionKindRemove")
		d.P("sa.Target = &StoreAction_", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}")
		d.Out()
	}
	d.P("default:")
	d.In()
	d.P("return StoreAction{}, errUnknownStoreAction")
	d.Out()
	d.P("}")
	d.P("return sa, nil")
	d.Out()
	d.P("}")
	d.P()
}

func (d *storeObjectGen) genWatchMessageEvent(topLevelObjs []string) {
	// Generate WatchMessageEvent
	d.P("func WatchMessageEvent(c Event) *WatchMessage_Event {")
	d.In()
	d.P("switch v := c.(type) {")
	for _, ccTypeName := range topLevelObjs {
		d.P("case EventCreate", ccTypeName, ":")
		d.In()
		d.P("return &WatchMessage_Event{Action: WatchActionKindCreate, Object: &Object{Object: &Object_", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}}}")
		d.Out()
		d.P("case EventUpdate", ccTypeName, ":")
		d.In()
		d.P("if v.Old", ccTypeName, " != nil {")
		d.In()
		d.P("return &WatchMessage_Event{Action: WatchActionKindUpdate, Object: &Object{Object: &Object_", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}}, OldObject: &Object{Object: &Object_", ccTypeName, "{", ccTypeName, ": v.Old", ccTypeName, "}}}")
		d.Out()
		d.P("} else {")
		d.In()
		d.P("return &WatchMessage_Event{Action: WatchActionKindUpdate, Object: &Object{Object: &Object_", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}}}")
		d.Out()
		d.P("}")
		d.Out()
		d.P("case EventDelete", ccTypeName, ":")
		d.In()
		d.P("return &WatchMessage_Event{Action: WatchActionKindRemove, Object: &Object{Object: &Object_", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}}}")
		d.Out()
	}
	d.P("}")
	d.P("return nil")
	d.Out()
	d.P("}")
	d.P()
}

func (d *storeObjectGen) genEventFromStoreAction(topLevelObjs []string) {
	// Generate EventFromStoreAction
	d.P("func EventFromStoreAction(sa StoreAction, oldObject StoreObject) (Event, error) {")
	d.In()
	d.P("switch v := sa.Target.(type) {")
	for _, ccTypeName := range topLevelObjs {
		d.P("case *StoreAction_", ccTypeName, ":")
		d.In()
		d.P("switch sa.Action {")

		d.P("case StoreActionKindCreate:")
		d.In()
		d.P("return EventCreate", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}, nil")
		d.Out()

		d.P("case StoreActionKindUpdate:")
		d.In()
		d.P("if oldObject != nil {")
		d.In()
		d.P("return EventUpdate", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, ", Old", ccTypeName, ": oldObject.(*", ccTypeName, ")}, nil")
		d.Out()
		d.P("} else {")
		d.In()
		d.P("return EventUpdate", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}, nil")
		d.Out()
		d.P("}")
		d.Out()

		d.P("case StoreActionKindRemove:")
		d.In()
		d.P("return EventDelete", ccTypeName, "{", ccTypeName, ": v.", ccTypeName, "}, nil")
		d.Out()

		d.P("}")
		d.Out()
	}
	d.P("}")
	d.P("return nil, errUnknownStoreAction")
	d.Out()
	d.P("}")
	d.P()
}

func (d *storeObjectGen) genConvertWatchArgs(topLevelObjs []string) {
	// Generate ConvertWatchArgs
	d.P("func ConvertWatchArgs(entries []*WatchRequest_WatchEntry) ([]Event, error) {")
	d.In()
	d.P("var events []Event")
	d.P("for _, entry := range entries {")
	d.In()
	d.P("var newEvents []Event")
	d.P("var err error")
	d.P("switch entry.Kind {")
	d.P(`case "":`)
	d.In()
	d.P("return nil, errNoKindSpecified")
	d.Out()
	for _, ccTypeName := range topLevelObjs {
		if ccTypeName == "Resource" {
			d.P("default:")
			d.In()
			d.P("newEvents, err = ConvertResourceWatch(entry.Action, entry.Filters, entry.Kind)")
			d.Out()
		} else {
			d.P(`case "`, strings.ToLower(ccTypeName), `":`)
			d.In()
			d.P("newEvents, err = Convert", ccTypeName, "Watch(entry.Action, entry.Filters)")
			d.Out()
		}
	}
	d.P("}")
	d.P("if err != nil {")
	d.In()
	d.P("return nil, err")
	d.Out()
	d.P("}")
	d.P("events = append(events, newEvents...)")

	d.Out()
	d.P("}")
	d.P("return events, nil")
	d.Out()
	d.P("}")
	d.P()
}

func (d *storeObjectGen) Generate(file *generator.FileDescriptor) {
	d.PluginImports = generator.NewPluginImports(d.Generator)
	d.eventsPkg = d.NewImport("github.com/docker/go-events")
	d.stringsPkg = d.NewImport("strings")

	var topLevelObjs []string

	for _, m := range file.Messages() {
		if m.DescriptorProto.GetOptions().GetMapEntry() {
			continue
		}

		if m.Options == nil {
			continue
		}
		storeObjIntf, err := proto.GetExtension(m.Options, plugin.E_StoreObject)
		if err != nil {
			// no StoreObject extension
			continue
		}

		d.genMsgStoreObject(m, storeObjIntf.(*plugin.StoreObject))

		topLevelObjs = append(topLevelObjs, generator.CamelCaseSlice(m.TypeName()))
	}

	if len(topLevelObjs) != 0 {
		d.genNewStoreAction(topLevelObjs)
		d.genEventFromStoreAction(topLevelObjs)

		// for watch API
		d.genWatchMessageEvent(topLevelObjs)
		d.genConvertWatchArgs(topLevelObjs)
	}
}
