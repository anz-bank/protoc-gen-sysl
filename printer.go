package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anz-bank/sysl/pkg/sysl"
	printer "github.com/joshcarp/sysl-printer"
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

const typeApplication = "Types"

var TypeMapping = map[string]sysl.Type_Primitive{"TYPE_BYTES": sysl.Type_BYTES, "TYPE_INT32": sysl.Type_INT, "TYPE_STRING": sysl.Type_STRING, "TYPE_BOOL": sysl.Type_BOOL}

func ASTPrinter() *PrinterModule { return &PrinterModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *PrinterModule) Name() string { return "printer" }

func (p *PrinterModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	buf := &bytes.Buffer{}
	if p.Log == nil {
		p.Log = logrus.New()
	}
	p.Module = &sysl.Module{
		Apps:                 make(map[string]*sysl.Application, 0),
		SourceContext:        nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	p.Module.Apps[typeApplication] = &sysl.Application{
		Name: &sysl.AppName{
			Part: []string{typeApplication},
		},
		Attrs:     nil,
		Endpoints: nil,
		Types:     nil,
		Views:     nil,
	}
	p.Module.Apps[typeApplication].Types = map[string]*sysl.Type{}

	for _, f := range targets {
		p.populateModule(f, buf)
	}
	prin := printer.NewPrinter(os.Stdout)
	prin.PrintModule(p.Module)
	return p.Artifacts()
}

func (p *PrinterModule) populateModule(f pgs.File, buf *bytes.Buffer) {
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
	subNodePrefix  = "┃"
	leafNodePrefix = "┣"
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
	return v.initPrintVisitor(v.w, fmt.Sprintf("%s%v", v.prefix, subNodePrefix))
}

func (v PrinterModule) writeLeaf(str string) {
}

func (v PrinterModule) VisitFile(f pgs.File) (pgs.Visitor, error) {
	return v.writeSubNode("File: " + f.Name().String()), nil
}

func (v PrinterModule) VisitMessage(m pgs.Message) (pgs.Visitor, error) {
	attrDefs := make(map[string]*sysl.Type)

	var fieldName string
	var syslType *sysl.Type
	for _, e := range m.Fields() {
		fieldName, syslType = fieldToSysl(e)
		attrDefs[fieldName] = syslType
	}
	v.Module.Apps[typeApplication].Types[m.Name().String()] = &sysl.Type{
		Type: &sysl.Type_Tuple_{
			Tuple: &sysl.Type_Tuple{
				AttrDefs: attrDefs,
			},
		},
	}
	return v.writeSubNode(""), nil
}

func (v PrinterModule) VisitEnum(e pgs.Enum) (pgs.Visitor, error) {
	return v.writeSubNode("Enum: " + e.Name().String()), nil
}

func (v PrinterModule) VisitService(s pgs.Service) (pgs.Visitor, error) {
	v.Module.Apps[s.Name().String()] = &sysl.Application{
		Name:      &sysl.AppName{Part: []string{s.Name().String()}},
		Endpoints: v.fillEndpoints(s.Methods()),
	}
	return v.writeSubNode("Service: " + s.Name().String()), nil
}

func (v PrinterModule) fillEndpoints(methods []pgs.Method) map[string]*sysl.Endpoint {
	ep := make(map[string]*sysl.Endpoint, len(methods))
	for _, method := range methods {
		ep[method.Name().String()] = &sysl.Endpoint{
			Name:     method.Name().String(),
			LongName: method.FullyQualifiedName(),
			Param: []*sysl.Param{{
				Name: "input",
				Type: messageToSysl(method.Input()),
			}},
			Stmt: []*sysl.Statement{{Stmt: &sysl.Statement_Ret{Ret: &sysl.Return{
				Payload: method.Output().Name().String(),
			}}}},
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
