// Copyright 2020 lack
// MIT License
//
// Copyright (c) 2020 Lack
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package plugin

import (
	"fmt"
	"strings"
	"sync"

	"github.com/vine-io/vine/cmd/generator"
)

// flow is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for flow support.
type flow struct {
	generator.PluginImports
	gen *generator.Generator
	m   sync.Map

	contextPkg generator.Single
	reflectPkg generator.Single
	flowPkg    generator.Single
	clientPkg  generator.Single
}

func New() *flow {
	return &flow{}
}

// Name returns the name of this plugin, "flow".
func (g *flow) Name() string {
	return "flow"
}

// Init initializes the plugin.
func (g *flow) Init(gen *generator.Generator) {
	g.gen = gen
	g.PluginImports = generator.NewPluginImports(g.gen)
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *flow) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *flow) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *flow) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *flow) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.contextPkg = g.NewImport("context", "context")
	g.reflectPkg = g.NewImport("reflect", "reflect")
	g.flowPkg = g.NewImport("github.com/vine-io/flow", "flow")
	g.clientPkg = g.NewImport("github.com/vine-io/vine/core/client", "client")

	for i := range file.TagServices() {
		g.generateService(file, file.TagServices()[i], i)
	}
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
	// TODO: do we need any in vine?
}

func unexport(s string) string {
	if len(s) == 0 {
		return ""
	}
	name := strings.ToLower(s[:1]) + s[1:]
	return name
}

// generateService generates all the code for the named service.
func (g *flow) generateService(file *generator.FileDescriptor, service *generator.ServiceDescriptor, index int) {
	path := fmt.Sprintf("6,%d", index) // 6 means service.

	origServName := service.Proto.GetName()
	//serviceName := strings.ToLower(service.Proto.GetName())
	//if pkg := file.GetPackage(); pkg != "" {
	//	serviceName = pkg
	//}
	servName := generator.CamelCase(origServName)
	servAlias := servName + "FlowClient"

	// strip suffix
	if strings.HasSuffix(servAlias, "FlowClientFlowClient") {
		servAlias = strings.TrimSuffix(servAlias, "FlowClient")
	}

	g.P()
	g.P("// Client API for ", servName, " service")
	// Client interface.
	g.gen.PrintComments(fmt.Sprintf("6,%d", index))
	g.P("type ", servAlias, " interface {")
	for i, method := range service.Methods {
		g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		g.P(g.generateClientSignature(servName, method))
	}
	g.P("}")
	g.P()

	// Client structure.
	g.P("type ", unexport(servAlias), " struct {")
	g.P("target string")
	g.P("c *", g.flowPkg.Use(), ".Client")
	g.P("}")
	g.P()

	// Entity method implementations.
	tags := extractTags(g.gen, service.Comments)
	entity, ok := tags[_entity]
	if !ok {
		g.gen.Fail("missing entity")
	}

	// NewClient factory.
	m, mp, ok := g.extractEntity(entity.Value)
	if !ok {
		g.gen.Fail("Entity:", entity.Value, "not found")
	}

	g.P(fmt.Sprintf(`var _ %s.Entity = (*%s)(nil)`, g.flowPkg.Use(), mp))
	g.P()

	g.P("// ", servAlias, " for ", m.Proto.GetName())
	g.P("func New", servAlias, " (target string, c *", g.flowPkg.Use(), ".Client) ", servAlias, " {")
	g.P("return &", unexport(servAlias), "{")
	g.P("target: target,")
	g.P("c: c,")
	g.P("}")
	g.P("}")
	g.P()

	for _, method := range service.Methods {
		g.generateEntityEcho(mp, servName, method)
	}

	// Client method implementations.
	for _, method := range service.Methods {
		g.generateClientMethod(servName, method)
	}

	/*
		func RegisterEchoHandler(s *flow.ClientStore, hdlr EchoHandler) {
			s.Load(NewPingEcho(hdlr))
		}
	*/

	g.P(fmt.Sprintf(`func Register%sFlowHandler(s *%s.ClientStore, hdlr %sFlowHandler) {`, servName, g.flowPkg.Use(), servName))
	for _, method := range service.Methods {
		mname := method.Proto.GetName()
		ename := strings.Title(fmt.Sprintf("%s%s", servName, mname))
		g.P(fmt.Sprintf(`s.Load(New%s(hdlr))`, ename))
	}
	g.P("}")
	g.P()

	g.P("// Server API for ", servName, " service")
	// Server interface.
	serverType := servName + "FlowHandler"
	g.gen.PrintComments(fmt.Sprintf("6,%d", index))
	g.P("type ", serverType, " interface {")
	for i, method := range service.Methods {
		g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		g.P(g.generateServerSignature(servName, method))
	}
	g.P("}")
	g.P()

	// Handler structure.
	//g.P("type ", unexport(serverType), " struct {")
	//g.P(serverType)
	//g.P("}")
	//g.P()
	//
	//// Server handler implementations.
	//var handlerNames []string
	//for _, method := range service.Methods {
	//	hname := g.generateServerMethod(servName, method)
	//	handlerNames = append(handlerNames, hname)
	//}
}

// generateClientSignature returns the client-side signature for a method.
func (g *flow) generateClientSignature(servName string, method *generator.MethodDescriptor) string {
	origMethName := method.Proto.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	reqArg := ", in *" + g.typeName(method.Proto.GetInputType())
	respName := "*" + g.typeName(method.Proto.GetOutputType())

	return fmt.Sprintf("%s(ctx %s.Context%s, opts ...%s.CallOption) (%s, error)", methName, g.contextPkg.Use(), reqArg, g.clientPkg.Use(), respName)
}

func (g *flow) generateEntityEcho(mType, servName string, method *generator.MethodDescriptor) {
	comments := extractDesc(method.Comments)
	serveType := servName + "FlowHandler"
	mname := method.Proto.GetName()
	ename := strings.Title(fmt.Sprintf("%s%s", servName, mname))
	inType := g.typeName(method.Proto.GetInputType())
	outType := g.typeName(method.Proto.GetOutputType())

	g.P("type ", ename, " struct {")
	g.P("h ", serveType)
	g.P("}")
	g.P()

	g.P(fmt.Sprintf(`func New%s(h %s) *%s {`, ename, serveType, ename))
	g.P(fmt.Sprintf(`return &%s{h: h}`, ename))
	g.P("}")
	g.P()

	g.P(fmt.Sprintf(`func (e *%s) Owner() %s.Type {`, ename, g.reflectPkg.Use()))
	g.P(fmt.Sprintf(`return %s.TypeOf(new(%s))`, g.reflectPkg.Use(), mType))
	g.P("}")
	g.P()

	g.P(fmt.Sprintf(`func (p *%s) Call(ctx %s.Context, data []byte) ([]byte, error) {`, ename, g.contextPkg.Use()))
	g.P(fmt.Sprintf(`in := &%s{}`, inType))
	g.P("err := in.Unmarshal(data)")
	g.P("if err != nil { return nil, err }")
	g.P("")
	g.P(fmt.Sprintf("out := &%s{}", outType))
	g.P(fmt.Sprintf(`if err = p.h.%s(ctx, in, out); err != nil { return nil, err }`, mname))
	g.P("return out.Marshal()")
	g.P("}")
	g.P()

	desc := strings.Join(comments, "")
	if desc == "" {
		desc = ename
	}
	g.P(fmt.Sprintf(`func (p *%s) String() string {`, ename))
	g.P(fmt.Sprintf(`return "%s"`, desc))
	g.P("}")
	g.P()
}

func (g *flow) generateClientMethod(servName string, method *generator.MethodDescriptor) {
	outType := g.typeName(method.Proto.GetOutputType())
	reqMethod := strings.Title(fmt.Sprintf("%s%s", servName, method.Proto.GetName()))
	servAlias := servName + "FlowClient"

	// strip suffix
	if strings.HasSuffix(servAlias, "FlowClientFlowClient") {
		servAlias = strings.TrimSuffix(servAlias, "FlowClient")
	}

	g.P("func (c *", unexport(servAlias), ") ", g.generateClientSignature(servName, method), "{")
	if !method.Proto.GetServerStreaming() && !method.Proto.GetClientStreaming() {
		g.P(`data, err := in.Marshal()`)
		g.P(`if err != nil { return nil, err }`)
		g.P(fmt.Sprintf(`result, err := c.c.Call(ctx, c.target, %s.GetTypePkgName(%s.TypeOf(new(%s))), data, opts...)`, g.flowPkg.Use(), g.reflectPkg.Use(), reqMethod))
		g.P(`if err != nil { return nil, err }`)
		g.P("out := new(", outType, ")")
		g.P(fmt.Sprintf(`if err = out.Unmarshal(result); err != nil { return nil, err }`))
		g.P("return out, nil")
		g.P("}")
		g.P()
		return
	}
}

// generateServerSignature returns the server-side signature for a method.
func (g *flow) generateServerSignature(servName string, method *generator.MethodDescriptor) string {
	origMethName := method.Proto.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	ret := "error"
	reqArgs = append(reqArgs, g.contextPkg.Use()+".Context")

	if !method.Proto.GetClientStreaming() {
		reqArgs = append(reqArgs, "*"+g.typeName(method.Proto.GetInputType()))
	}
	if method.Proto.GetServerStreaming() || method.Proto.GetClientStreaming() {
		reqArgs = append(reqArgs, servName+"_"+generator.CamelCase(origMethName)+"Stream")
	}
	if !method.Proto.GetClientStreaming() && !method.Proto.GetServerStreaming() {
		reqArgs = append(reqArgs, "*"+g.typeName(method.Proto.GetOutputType()))
	}
	return methName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}

func (g *flow) generateServerMethod(servName string, method *generator.MethodDescriptor) string {
	methName := generator.CamelCase(method.Proto.GetName())
	hname := fmt.Sprintf("_%s_%s_Handler", servName, methName)
	serveType := servName + "Handler"
	inType := g.typeName(method.Proto.GetInputType())
	outType := g.typeName(method.Proto.GetOutputType())

	if !method.Proto.GetServerStreaming() && !method.Proto.GetClientStreaming() {
		g.P("func (h *", unexport(servName), "Handler) ", methName, "(ctx ", g.contextPkg.Use(), ".Context, in *", inType, ", out *", outType, ") error {")
		g.P(fmt.Sprintf(`return h.%s.%s(ctx, in, out)`, serveType, methName))
		g.P("}")
		g.P()
		return hname
	}
	streamType := unexport(servName) + methName + "Stream"
	g.P("func (h *", unexport(servName), "Handler) ", methName, "(ctx ", g.contextPkg.Use(), ".Context, stream server.Stream) error {")
	if !method.Proto.GetClientStreaming() {
		g.P("m := new(", inType, ")")
		g.P("if err := stream.Recv(m); err != nil { return err }")
		g.P(fmt.Sprintf(`return h.%s.%s(ctx, m, &%s{stream})`, serveType, methName, streamType))
	} else {
		g.P(fmt.Sprintf(`return h.%s.%s(ctx, &%s{stream})`, serveType, methName, streamType))
	}
	g.P("}")
	g.P()

	genSend := method.Proto.GetServerStreaming()
	genRecv := method.Proto.GetClientStreaming()

	// Stream auxiliary types and methods.
	g.P("type ", servName, "_", methName, "Stream interface {")
	g.P("Context() context.Context")
	g.P("SendMsg(interface{}) error")
	g.P("RecvMsg(interface{}) error")

	if !genSend {
		// client streaming, the server will send a response upon close
		g.P("SendAndClose(*", outType, ")  error")
	} else {
		g.P("Close() error")
	}

	if genSend {
		g.P("Send(*", outType, ") error")
	}

	if genRecv {
		g.P("Recv() (*", inType, ", error)")
	}

	g.P("}")
	g.P()

	g.P("type ", streamType, " struct {")
	g.P("stream ", g.flowPkg.Use(), ".Stream")
	g.P("}")
	g.P()

	if !genSend {
		// client streaming, the server will send a response upon close
		g.P("func (x *", streamType, ") SendAndClose(in *", outType, ") error {")
		g.P("if err := x.SendMsg(in); err != nil {")
		g.P("return err")
		g.P("}")
		g.P("return x.stream.Close()")
		g.P("}")
		g.P()
	} else {
		// other types of rpc don't send a response when the stream closes
		g.P("func (x *", streamType, ") Close() error {")
		g.P("return x.stream.Close()")
		g.P("}")
		g.P()
	}

	g.P("func (x *", streamType, ") Context() context.Context {")
	g.P("return x.stream.Context()")
	g.P("}")
	g.P()

	g.P("func (x *", streamType, ") SendMsg(m interface{}) error {")
	g.P("return x.stream.Send(m)")
	g.P("}")
	g.P()

	g.P("func (x *", streamType, ") RecvMsg(m interface{}) error {")
	g.P("return x.stream.Recv(m)")
	g.P("}")
	g.P()

	if genSend {
		g.P("func (x *", streamType, ") Send(m *", outType, ") error {")
		g.P("return x.stream.Send(m)")
		g.P("}")
		g.P()
	}

	if genRecv {
		g.P("func (x *", streamType, ") Recv() (*", inType, ", error) {")
		g.P("m := new(", inType, ")")
		g.P("if err := x.stream.Recv(m); err != nil { return nil, err }")
		g.P("return m, nil")
		g.P("}")
		g.P()
	}

	return hname
}

func (g *flow) extractEntity(name string) (*generator.MessageDescriptor, string, bool) {
	var pkg, ename string
	idx := strings.LastIndex(name, ".")
	if idx > 0 {
		pkg = name[:idx]
		ename = name[idx+1:]
	} else {
		ename = name
	}

	if pkg == "" {
		m := g.GetMessage(g.gen.File(), ename)
		return m, ename, m != nil
	}

	for _, file := range g.gen.AllFiles() {
		if file.GetPackage() == pkg {
			m := g.GetMessage(file, ename)
			if m != nil {
				v, ok := g.gen.ImportMap[m.Proto.GoImportPath().String()]
				if !ok {
					v = string(g.gen.AddImport(m.Proto.GoImportPath()))
				}
				return m, v + "." + ename, true
			}
			return nil, "", false
		}
	}

	return nil, "", false
}

func (g *flow) GetMessage(file *generator.FileDescriptor, mname string) *generator.MessageDescriptor {
	for _, m := range file.Messages() {
		if m.Proto.GetName() == mname {
			return m
		}
	}
	return nil
}
