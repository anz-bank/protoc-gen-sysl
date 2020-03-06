package main

import (
	"bytes"

	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/anz-bank/sysl/pkg/sysl"
	printer "github.com/joshcarp/sysl-printer"
	pgs "github.com/lyft/protoc-gen-star"
	"github.com/sirupsen/logrus"
)

// PrinterModule holds the sysl module that prints the sysl source code
type PrinterModule struct {
	*pgs.ModuleBase
	pgs.Visitor
	Log    *logrus.Logger
	Module *sysl.Module
}

func SyslPrinter() *PrinterModule { return &PrinterModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *PrinterModule) Name() string { return "printer" }

func (p *PrinterModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	buf := &bytes.Buffer{}

	if p.Log == nil {
		p.Log = logrus.New()
	}
	for _, f := range targets {
		buf.Reset()
		p.Module = &sysl.Module{Apps: make(map[string]*sysl.Application)}
		fileName := syslFilename(f.Name().String())
		p.CheckErr(pgs.Walk(p, f), "unable to print AST tree")
		printer.NewPrinter(buf).PrintModule(p.Module)
		p.AddGeneratorFile(fileName+".sysl", buf.String())
	}
	return p.Artifacts()
}

func (p *PrinterModule) VisitFile(file pgs.File) (v pgs.Visitor, err error) {
	for _, s := range file.Services() {
		if _, err := p.VisitService(s); err != nil {
			return nil, err
		}
	}

	// Initialise the "Type" application which will store all the types
	p.Module.Apps[syslPackageName(file)] = syslpopulate.NewApplication(syslPackageName(file))
	for _, t := range file.Messages() {
		if _, err := p.VisitMessage(t); err != nil {
			return nil, err
		}
	}
	for _, e := range file.Enums() {
		if _, err := p.VisitEnum(e); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// VisitService converts to sysl and constructs endpoints from methods
// service myservice{...} --> myservice:
func (p *PrinterModule) VisitService(s pgs.Service) (pgs.Visitor, error) {
	name := s.Name().String()
	p.Module.Apps[name] = syslpopulate.NewApplication(name)
	for _, e := range s.Methods() {
		if _, err := p.VisitMethod(e); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// VisitMessage converts to sysl and constructs types from messages. All types are writen to the
// TypeApplication (as in sysl types belong to applications but not in proto
// message foo{...} --> !type foo:
func (p *PrinterModule) VisitMessage(m pgs.Message) (pgs.Visitor, error) {
	attrDefs := make(map[string]*sysl.Type)
	var packageName = syslPackageName(m)
	var fieldName string
	var syslType *sysl.Type
	for _, e := range m.Fields() {
		fieldName, syslType = fieldToSysl(e)
		attrDefs[fieldName] = syslType
	}
	p.Module.Apps[packageName].Types[m.Name().String()] = &sysl.Type{
		Type: &sysl.Type_Tuple_{
			Tuple: &sysl.Type_Tuple{
				AttrDefs: attrDefs,
			},
		},
	}
	return p, nil
}

// VisitMethod converts a message to a sysl endpoint and fills in calls to other functions
// rpc thisEndpoint(InputType)returns(outputType) -->
// thisEndpoint(input <: InputType):
//     return ok <: outputType
func (p *PrinterModule) VisitMethod(m pgs.Method) (v pgs.Visitor, err error) {
	p.Module.Apps[m.Service().Name().String()].Endpoints[m.Name().String()] = endpointFromMethod(m)
	return p, nil
}

// VisitEnumValue converts to sysl enums. All types are writen to the
// Currently this sysl syntax is unsupported, but enums exist within the sysl data object
// enum foo{...} --> !enum foo:
func (p *PrinterModule) VisitEnum(e pgs.Enum) (v pgs.Visitor, err error) {
	var packageName = syslPackageName(e)
	p.Module.Apps[packageName].Types[e.Name().String()] = &sysl.Type{
		Type: &sysl.Type_Enum_{
			Enum: &sysl.Type_Enum{
				Items: enumToSysl(e),
			},
		},
	}
	return v, nil
}
