package deepcopy

import (
	"github.com/docker/swarmkit/protobuf/plugin"
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

type deepCopyGen struct {
	*generator.Generator
	generator.PluginImports
	copyPkg generator.Single
}

func init() {
	generator.RegisterPlugin(new(deepCopyGen))
}

func (d *deepCopyGen) Name() string {
	return "deepcopy"
}

func (d *deepCopyGen) Init(g *generator.Generator) {
	d.Generator = g
}

func (d *deepCopyGen) genCopyFunc(dst, src string) {
	d.P(d.copyPkg.Use(), ".Copy(", dst, ", ", src, ")")
}

func (d *deepCopyGen) genCopyBytes(dst, src string) {
	d.P("if ", src, " != nil {")
	d.In()
	// allocate dst object
	d.P(dst, " = make([]byte, len(", src, "))")
	// copy bytes from src to dst
	d.P("copy(", dst, ", ", src, ")")
	d.Out()
	d.P("}")
}

func (d *deepCopyGen) genMsgDeepCopy(m *generator.Descriptor) {
	ccTypeName := generator.CamelCaseSlice(m.TypeName())

	// Generate backwards compatible, type-safe Copy() function.
	d.P("func (m *", ccTypeName, ") Copy() *", ccTypeName, "{")
	d.In()
	d.P("if m == nil {")
	d.In()
	d.P("return nil")
	d.Out()
	d.P("}")
	d.P("o := &", ccTypeName, "{}")
	d.P("o.CopyFrom(m)")
	d.P("return o")
	d.Out()
	d.P("}")
	d.P()

	if len(m.Field) == 0 {
		d.P("func (m *", ccTypeName, ") CopyFrom(src interface{})", " {}")
		return
	}

	d.P("func (m *", ccTypeName, ") CopyFrom(src interface{})", " {")
	d.P()

	d.P("o := src.(*", ccTypeName, ")")

	// shallow copy handles all scalars
	d.P("*m = *o")

	oneofByIndex := [][]*descriptor.FieldDescriptorProto{}
	for _, f := range m.Field {
		fName := generator.CamelCase(*f.Name)
		if gogoproto.IsCustomName(f) {
			fName = gogoproto.GetCustomName(f)
		}

		// Handle oneof type, we defer them to a loop below
		if f.OneofIndex != nil {
			if len(oneofByIndex) <= int(*f.OneofIndex) {
				oneofByIndex = append(oneofByIndex, []*descriptor.FieldDescriptorProto{})
			}

			oneofByIndex[*f.OneofIndex] = append(oneofByIndex[*f.OneofIndex], f)
			continue
		}

		// Handle all kinds of message type
		if f.IsMessage() {
			// Handle map type
			if d.genMap(m, f) {
				continue
			}

			// Handle any message which is not repeated or part of oneof
			if !f.IsRepeated() && f.OneofIndex == nil {
				if !gogoproto.IsNullable(f) {
					d.genCopyFunc("&m."+fName, "&o."+fName)
				} else {
					d.P("if o.", fName, " != nil {")
					d.In()
					// allocate dst object
					d.P("m.", fName, " = &", d.TypeName(d.ObjectNamed(f.GetTypeName())), "{}")
					// copy into the allocated struct
					d.genCopyFunc("m."+fName, "o."+fName)

					d.Out()
					d.P("}")
				}
				continue
			}
		}

		// Handle repeated field
		if f.IsRepeated() {
			d.genRepeated(m, f)
			continue
		}

		// Handle bytes
		if f.IsBytes() {
			d.genCopyBytes("m."+fName, "o."+fName)
			continue
		}

		// skip: field was a scalar handled by shallow copy!
	}

	for i, oo := range m.GetOneofDecl() {
		d.genOneOf(m, oo, oneofByIndex[i])
	}

	d.P("}")
	d.P()
}

func (d *deepCopyGen) genMap(m *generator.Descriptor, f *descriptor.FieldDescriptorProto) bool {
	fName := generator.CamelCase(*f.Name)
	if gogoproto.IsCustomName(f) {
		fName = gogoproto.GetCustomName(f)
	}

	dv := d.ObjectNamed(f.GetTypeName())
	desc, ok := dv.(*generator.Descriptor)
	if !ok || !desc.GetOptions().GetMapEntry() {
		return false
	}

	mt := d.GoMapType(desc, f)
	typename := mt.GoType

	d.P("if o.", fName, " != nil {")
	d.In()
	d.P("m.", fName, " = make(", typename, ", ", "len(o.", fName, "))")
	d.P("for k, v := range o.", fName, " {")
	d.In()
	if mt.ValueField.IsMessage() {
		if !gogoproto.IsNullable(f) {
			d.P("n := ", d.TypeName(d.ObjectNamed(mt.ValueField.GetTypeName())), "{}")
			d.genCopyFunc("&n", "&v")
			d.P("m.", fName, "[k] = ", "n")
		} else {
			d.P("m.", fName, "[k] = &", d.TypeName(d.ObjectNamed(mt.ValueField.GetTypeName())), "{}")
			d.genCopyFunc("m."+fName+"[k]", "v")
		}
	} else if mt.ValueField.IsBytes() {
		d.P("m.", fName, "[k] = o.", fName, "[k]")
		d.genCopyBytes("m."+fName+"[k]", "o."+fName+"[k]")
	} else {
		d.P("m.", fName, "[k] = v")
	}
	d.Out()
	d.P("}")
	d.Out()
	d.P("}")
	d.P()

	return true
}

func (d *deepCopyGen) genRepeated(m *generator.Descriptor, f *descriptor.FieldDescriptorProto) {
	fName := generator.CamelCase(*f.Name)
	if gogoproto.IsCustomName(f) {
		fName = gogoproto.GetCustomName(f)
	}

	typename, _ := d.GoType(m, f)

	d.P("if o.", fName, " != nil {")
	d.In()
	d.P("m.", fName, " = make(", typename, ", len(o.", fName, "))")
	if f.IsMessage() {
		// TODO(stevvooe): Handle custom type here?
		goType := d.TypeName(d.ObjectNamed(f.GetTypeName())) // elides [] or *

		d.P("for i := range m.", fName, " {")
		d.In()
		if !gogoproto.IsNullable(f) {
			d.genCopyFunc("&m."+fName+"[i]", "&o."+fName+"[i]")
		} else {
			d.P("m.", fName, "[i] = &", goType, "{}")
			d.genCopyFunc("m."+fName+"[i]", "o."+fName+"[i]")
		}
		d.Out()
		d.P("}")
	} else if f.IsBytes() {
		d.P("for i := range m.", fName, " {")
		d.In()
		d.genCopyBytes("m."+fName+"[i]", "o."+fName+"[i]")
		d.Out()
		d.P("}")
	} else {
		d.P("copy(m.", fName, ", ", "o.", fName, ")")
	}
	d.Out()
	d.P("}")
	d.P()
}

func (d *deepCopyGen) genOneOf(m *generator.Descriptor, oneof *descriptor.OneofDescriptorProto, fields []*descriptor.FieldDescriptorProto) {
	oneOfName := generator.CamelCase(oneof.GetName())

	d.P("if o.", oneOfName, " != nil {")
	d.In()
	d.P("switch o.", oneOfName, ".(type) {")

	for _, f := range fields {
		ccTypeName := generator.CamelCaseSlice(m.TypeName())
		fName := generator.CamelCase(*f.Name)
		if gogoproto.IsCustomName(f) {
			fName = gogoproto.GetCustomName(f)
		}

		tName := ccTypeName + "_" + fName
		d.P("case *", tName, ":")
		d.In()
		d.P("v := ", tName, " {")
		d.In()

		var rhs string
		if f.IsMessage() {
			goType := d.TypeName(d.ObjectNamed(f.GetTypeName())) // elides [] or *
			rhs = "&" + goType + "{}"
		} else if f.IsBytes() {
			rhs = "make([]byte, len(o.Get" + fName + "()))"
		} else {
			rhs = "o.Get" + fName + "()"
		}
		d.P(fName, ": ", rhs, ",")
		d.Out()
		d.P("}")

		if f.IsMessage() {
			d.genCopyFunc("v."+fName, "o.Get"+fName+"()")
		} else if f.IsBytes() {
			d.genCopyBytes("v."+fName, "o.Get"+fName+"()")
		}

		d.P("m.", oneOfName, " = &v")
		d.Out()
	}

	d.Out()
	d.P("}")
	d.Out()
	d.P("}")
	d.P()
}

func (d *deepCopyGen) Generate(file *generator.FileDescriptor) {
	d.PluginImports = generator.NewPluginImports(d.Generator)

	// TODO(stevvooe): Ideally, this could be taken as a parameter to the
	// deepcopy plugin to control the package import, but this is good enough,
	// for now.
	d.copyPkg = d.NewImport("github.com/docker/swarmkit/api/deepcopy")

	d.P()
	for _, m := range file.Messages() {
		if m.DescriptorProto.GetOptions().GetMapEntry() {
			continue
		}

		if !plugin.DeepcopyEnabled(m.Options) {
			continue
		}

		d.genMsgDeepCopy(m)
	}
	d.P()
}
