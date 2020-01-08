package authenticatedwrapper

import (
	"strings"

	"github.com/docker/swarmkit/protobuf/plugin"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

type authenticatedWrapperGen struct {
	gen *generator.Generator
}

func init() {
	generator.RegisterPlugin(new(authenticatedWrapperGen))
}

func (g *authenticatedWrapperGen) Init(gen *generator.Generator) {
	g.gen = gen
}

func (g *authenticatedWrapperGen) Name() string {
	return "authenticatedwrapper"
}

func (g *authenticatedWrapperGen) genAuthenticatedStruct(s *descriptor.ServiceDescriptorProto) {
	g.gen.P("type " + serviceTypeName(s) + " struct {")
	g.gen.P("	local " + s.GetName() + "Server")
	g.gen.P("	authorize func(context.Context, []string) error")
	g.gen.P("}")
}

func (g *authenticatedWrapperGen) genAuthenticatedConstructor(s *descriptor.ServiceDescriptorProto) {
	g.gen.P("func NewAuthenticatedWrapper" + s.GetName() + "Server(local " + s.GetName() + "Server, authorize func(context.Context, []string) error)" + s.GetName() + "Server {")
	g.gen.P("return &" + serviceTypeName(s) + `{
		local: local,
		authorize: authorize,
	}`)
	g.gen.P("}")
}

func getInputTypeName(m *descriptor.MethodDescriptorProto) string {
	parts := strings.Split(m.GetInputType(), ".")
	return parts[len(parts)-1]
}

func getOutputTypeName(m *descriptor.MethodDescriptorProto) string {
	parts := strings.Split(m.GetOutputType(), ".")
	return parts[len(parts)-1]
}

func serviceTypeName(s *descriptor.ServiceDescriptorProto) string {
	return "authenticatedWrapper" + s.GetName() + "Server"
}

func sigPrefix(s *descriptor.ServiceDescriptorProto, m *descriptor.MethodDescriptorProto) string {
	return "func (p *" + serviceTypeName(s) + ") " + m.GetName() + "("
}

func genRoles(auth *plugin.TLSAuthorization) string {
	rolesSlice := "[]string{"
	first := true
	for _, role := range auth.Roles {
		if !first {
			rolesSlice += ","
		}
		first = false
		rolesSlice += `"` + role + `"`
	}

	rolesSlice += "}"

	return rolesSlice
}

func (g *authenticatedWrapperGen) genServerStreamingMethod(s *descriptor.ServiceDescriptorProto, m *descriptor.MethodDescriptorProto) {
	g.gen.P(sigPrefix(s, m) + "r *" + getInputTypeName(m) + ", stream " + s.GetName() + "_" + m.GetName() + "Server) error {")

	authIntf, err := proto.GetExtension(m.Options, plugin.E_TlsAuthorization)
	if err != nil {
		g.gen.P(`
	panic("no authorization information in protobuf")`)
		g.gen.P(`}`)
		return
	}

	auth := authIntf.(*plugin.TLSAuthorization)

	if auth.Insecure != nil && *auth.Insecure {
		if len(auth.Roles) != 0 {
			panic("Roles and Insecure cannot both be specified")
		}
		g.gen.P(`
	return p.local.` + m.GetName() + `(r, stream)`)
		g.gen.P(`}`)
		return
	}

	g.gen.P(`
	if err := p.authorize(stream.Context(),` + genRoles(auth) + `); err != nil {
		return err
	}
	return p.local.` + m.GetName() + `(r, stream)`)
	g.gen.P("}")
}

func (g *authenticatedWrapperGen) genClientServerStreamingMethod(s *descriptor.ServiceDescriptorProto, m *descriptor.MethodDescriptorProto) {
	g.gen.P(sigPrefix(s, m) + "stream " + s.GetName() + "_" + m.GetName() + "Server) error {")

	authIntf, err := proto.GetExtension(m.Options, plugin.E_TlsAuthorization)
	if err != nil {
		g.gen.P(`
	panic("no authorization information in protobuf")`)
		g.gen.P(`}`)
		return
	}

	auth := authIntf.(*plugin.TLSAuthorization)

	if auth.Insecure != nil && *auth.Insecure {
		if len(auth.Roles) != 0 {
			panic("Roles and Insecure cannot both be specified")
		}
		g.gen.P(`
	return p.local.` + m.GetName() + `(stream)`)
		g.gen.P(`}`)
		return
	}

	g.gen.P(`
	if err := p.authorize(stream.Context(), ` + genRoles(auth) + `); err != nil {
		return err
	}
	return p.local.` + m.GetName() + `(stream)`)
	g.gen.P("}")
}

func (g *authenticatedWrapperGen) genSimpleMethod(s *descriptor.ServiceDescriptorProto, m *descriptor.MethodDescriptorProto) {
	g.gen.P(sigPrefix(s, m) + "ctx context.Context, r *" + getInputTypeName(m) + ") (*" + getOutputTypeName(m) + ", error) {")

	authIntf, err := proto.GetExtension(m.Options, plugin.E_TlsAuthorization)
	if err != nil {
		g.gen.P(`
	panic("no authorization information in protobuf")`)
		g.gen.P(`}`)
		return
	}

	auth := authIntf.(*plugin.TLSAuthorization)

	if auth.Insecure != nil && *auth.Insecure {
		if len(auth.Roles) != 0 {
			panic("Roles and Insecure cannot both be specified")
		}
		g.gen.P(`
	return p.local.` + m.GetName() + `(ctx, r)`)
		g.gen.P(`}`)
		return
	}

	g.gen.P(`
	if err := p.authorize(ctx, ` + genRoles(auth) + `); err != nil {
		return nil, err
	}
	return p.local.` + m.GetName() + `(ctx, r)`)
	g.gen.P("}")
}

func (g *authenticatedWrapperGen) genAuthenticatedMethod(s *descriptor.ServiceDescriptorProto, m *descriptor.MethodDescriptorProto) {
	g.gen.P()
	switch {
	case m.GetClientStreaming():
		g.genClientServerStreamingMethod(s, m)
	case m.GetServerStreaming():
		g.genServerStreamingMethod(s, m)
	default:
		g.genSimpleMethod(s, m)
	}
	g.gen.P()
}

func (g *authenticatedWrapperGen) Generate(file *generator.FileDescriptor) {
	g.gen.P()
	for _, s := range file.Service {
		g.genAuthenticatedStruct(s)
		g.genAuthenticatedConstructor(s)
		for _, m := range s.Method {
			g.genAuthenticatedMethod(s, m)
		}
	}
	g.gen.P()
}

func (g *authenticatedWrapperGen) GenerateImports(file *generator.FileDescriptor) {
}
