package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/anz-bank/sysl/pkg/sysl"
	"github.com/sirupsen/logrus"

	"bytes"

	pgs "github.com/lyft/protoc-gen-star"
)

type PrinterModule struct {
	*pgs.ModuleBase
	pgs.Visitor
	prefix string
	w      io.Writer
	Log    *logrus.Logger
	Module *sysl.Module
}

func ASTPrinter() *PrinterModule { return &PrinterModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *PrinterModule) Name() string { return "printer" }

func (p *PrinterModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	buf := &bytes.Buffer{}
	println("wefg")
	p.Module = &sysl.Module{
		Apps: make(map[string]*sysl.Application, 0),
	}
	for _, f := range targets {
		p.printFile(f, buf)
	}

	return p.Artifacts()
}

func (p *PrinterModule) printFile(f pgs.File, buf *bytes.Buffer) {
	p.Push(f.Name().String())
	defer p.Pop()

	buf.Reset()
	v := p.initPrintVisitor(buf, "")
	p.CheckErr(pgs.Walk(v, f), "unable to print AST tree")
	out := buf.String()

	if ok, _ := p.Parameters().Bool("log_tree"); ok {
		p.Logf("Proto Tree:\n%s", out)
	}

	p.AddGeneratorFile(
		f.InputPath().SetExt(".tree.txt").String(),
		out,
	)

}

const (
	startNodePrefix = "┳ "
	subNodePrefix   = "┃"
	leafNodePrefix  = "┣"
	leafNodeSpacer  = "━ "
)

func (p *PrinterModule) initPrintVisitor(w io.Writer, prefix string) pgs.Visitor {
	p.prefix = prefix
	p.Visitor = pgs.PassThroughVisitor(p)
	p.w = w
	return p
}

func (v PrinterModule) leafPrefix() string {
	if strings.HasSuffix(v.prefix, subNodePrefix) {
		return strings.TrimSuffix(v.prefix, subNodePrefix) + leafNodePrefix
	}
	return v.prefix
}

func (v PrinterModule) writeSubNode(str string) pgs.Visitor {
	fmt.Fprintf(v.w, "%s%s%s\n", v.leafPrefix(), startNodePrefix, str)
	return v.initPrintVisitor(v.w, fmt.Sprintf("%s%v", v.prefix, subNodePrefix))
}

func (v PrinterModule) writeLeaf(str string) {
	fmt.Fprintf(v.w, "%s%s%s\n", v.leafPrefix(), leafNodeSpacer, str)
}

func (v PrinterModule) VisitFile(f pgs.File) (pgs.Visitor, error) {
	return v.writeSubNode("File: " + f.Name().String()), nil
}

func (v PrinterModule) VisitMessage(m pgs.Message) (pgs.Visitor, error) {
	return v.writeSubNode("sysltemplate.Execute(sysltemplate.Type, m)"), nil
}

func (v PrinterModule) VisitEnum(e pgs.Enum) (pgs.Visitor, error) {
	return v.writeSubNode("Enum: " + e.Name().String()), nil
}

func (v PrinterModule) VisitService(s pgs.Service) (pgs.Visitor, error) {
	v.Module.Apps[s.Name().String()] = &sysl.Application{
		Name:          &sysl.AppName{Part: []string{s.Name().String()}},
		LongName:      "",
		Docstring:     "",
		Attrs:         nil,
		Endpoints:     fillEndpoints(s.Methods()),
		Types:         nil,
		Views:         nil,
		Mixin2:        nil,
		Wrapped:       nil,
		SourceContext: nil,
	}
	return v.writeSubNode("Service: " + s.Name().String()), nil
}

func fillEndpoints(methods []pgs.Method) map[string]*sysl.Endpoint {
	ep := make(map[string]*sysl.Endpoint, len(methods))
	for _, method := range methods {
		ep[method.Name().String()] = &sysl.Endpoint{
			Name:      method.Name().String(),
			LongName:  method.FullyQualifiedName(),
			Docstring: "",
			Attrs:     nil,
			Flag:      nil,
			Source:    nil,
			IsPubsub:  false,
			Param: []*sysl.Param{&sysl.Param{
				Name: "var" + method.Input().Name().String(),
				Type: typeFromMessage(method.Input())},
			},
			Stmt:                 nil,
			RestParams:           nil,
			SourceContext:        nil,
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		}

	}
	return ep
}
func typeFromMessage(message pgs.Message) *sysl.Type {
	return &sysl.Type{
		Type:                 &sysl.Type_TypeRef{},
		Attrs:                nil,
		Constraint:           nil,
		Docstring:            "",
		Opt:                  false,
		SourceContext:        nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
}

func (v PrinterModule) VisitEnumValue(ev pgs.EnumValue) (pgs.Visitor, error) {
	v.writeLeaf(ev.Name().String())
	return nil, nil
}

func (v PrinterModule) VisitField(f pgs.Field) (pgs.Visitor, error) {
	v.writeLeaf(f.Name().String())
	return nil, nil
}

func (v PrinterModule) VisitMethod(m pgs.Method) (pgs.Visitor, error) {
	v.writeLeaf(m.Name().String())
	return nil, nil
}
