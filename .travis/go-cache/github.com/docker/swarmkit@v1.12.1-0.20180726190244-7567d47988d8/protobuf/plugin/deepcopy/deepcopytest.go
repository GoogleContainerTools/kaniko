package deepcopy

import (
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/plugin/testgen"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

type test struct {
	*generator.Generator
}

// NewTest creates a new deepcopy testgen plugin
func NewTest(g *generator.Generator) testgen.TestPlugin {
	return &test{g}
}

func (p *test) Generate(imports generator.PluginImports, file *generator.FileDescriptor) bool {
	used := false
	testingPkg := imports.NewImport("testing")
	randPkg := imports.NewImport("math/rand")
	timePkg := imports.NewImport("time")

	for _, message := range file.Messages() {
		if !gogoproto.HasTestGen(file.FileDescriptorProto, message.DescriptorProto) {
			continue
		}

		if message.DescriptorProto.GetOptions().GetMapEntry() {
			continue
		}

		used = true
		ccTypeName := generator.CamelCaseSlice(message.TypeName())
		p.P()
		p.P(`func Test`, ccTypeName, `Copy(t *`, testingPkg.Use(), `.T) {`)
		p.In()
		p.P(`popr := `, randPkg.Use(), `.New(`, randPkg.Use(), `.NewSource(`, timePkg.Use(), `.Now().UnixNano()))`)
		p.P(`in := NewPopulated`, ccTypeName, `(popr, true)`)
		p.P(`out := in.Copy()`)
		p.P(`if !in.Equal(out) {`)
		p.In()
		p.P(`t.Fatalf("%#v != %#v", in, out)`)
		p.Out()
		p.P(`}`)

		for _, f := range message.Field {
			fName := generator.CamelCase(*f.Name)
			if gogoproto.IsCustomName(f) {
				fName = gogoproto.GetCustomName(f)
			}

			if f.OneofIndex != nil {
				fName = "Get" + fName + "()"
				if f.IsMessage() {
					p.P(`if in.`, fName, ` != nil && in.`, fName, ` == out.`, fName, ` {`)
					p.In()
					p.P(`t.Fatalf("`, fName, `: %#v == %#v", in.`, fName, `, out.`, fName, `)`)
					p.Out()
					p.P(`}`)
				}
			} else {
				p.P(`if &in.`, fName, ` == &out.`, fName, ` {`)
				p.In()
				p.P(`t.Fatalf("`, fName, `: %#v == %#v", &in.`, fName, `, &out.`, fName, `)`)
				p.Out()
				p.P(`}`)
			}

			if f.IsBytes() {
				if f.IsRepeated() {
					p.P(`if len(in.`, fName, `) > 0 {`)
					p.In()
					fName += "[0]"
				}
				p.P(`if len(in.`, fName, `) > 0 {`)
				p.In()
				p.P(`in.`, fName, "[0]++")
				p.P(`if in.Equal(out) {`)
				p.In()
				p.P(`t.Fatalf("%#v == %#v", in, out)`)
				p.Out()
				p.P(`}`)
				p.Out()
				p.P(`}`)
				if f.IsRepeated() {
					p.Out()
					p.P(`}`)
				}
			}
		}

		// copying from nil should result in nil
		p.P()
		p.P(`in = nil`)
		p.P(`out = in.Copy()`)
		p.P(`if out != nil {`)
		p.In()
		p.P(`t.Fatalf("copying nil should return nil, returned: %#v", out)`)
		p.Out()
		p.P(`}`)

		p.Out()
		p.P(`}`)
	}

	return used
}

func init() {
	testgen.RegisterTestPlugin(NewTest)
}
