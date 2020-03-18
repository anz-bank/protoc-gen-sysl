package main

import (
	"bytes"
	"fmt"

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
	indexFile := bytes.Buffer{}
	generatedFileSet := make(map[string]struct{})
	if p.Log == nil {
		p.Log = logrus.New()
	}

	for _, targetFile := range targets {
		for _, f := range packages[targetFile.Package().ProtoName().String()].Files() {
			fileName := syslFilename(f.Name().String()) + ".sysl"
			if _, ok := generatedFileSet[fileName]; !ok {
				generatedFileSet[fileName] = struct{}{}
				buf.Reset()
				buf.Write([]byte("import index.sysl\n\n"))
				indexFile.Write([]byte(fmt.Sprintf("import %s\n", fileName)))
				p.Module = &sysl.Module{Apps: make(map[string]*sysl.Application)}
				p.CheckErr(pgs.Walk(p, f), "unable to print AST tree")
				printer.NewPrinter(buf).PrintModule(p.Module)
				p.AddGeneratorFile(fileName, buf.String())
			}
		}
	}
	indexFile.Write([]byte("\n_[~ignore]:\n    ...\n"))
	p.AddGeneratorFile("index.sysl", indexFile.String())
	return p.Artifacts()
}

func (p *PrinterModule) VisitFile(file pgs.File) (v pgs.Visitor, err error) {
	for _, s := range file.Services() {
		if _, err := p.VisitService(s); err != nil {
			return nil, err
		}
	}
	for _, t := range file.Messages() {
		if _, ok := p.Module.Apps[syslPackageName(file)]; !ok {
			p.Module.Apps[syslPackageName(file)] = syslpopulate.NewApplication(syslPackageName(file))
			p.Module.Apps[syslPackageName(file)].Attrs["package"] = syslpopulate.NewAttribute(syslPackageName(file))
		}
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
	p.Module.Apps[name].Attrs["package"] = syslpopulate.NewAttribute(syslPackageName(s))
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
	var fieldName string
	var syslType *sysl.Type
	attrDefs := make(map[string]*sysl.Type)
	packageName := syslPackageName(m)
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
	var Calls map[string]string
	appName := m.Service().Name().String()
	endpointName := m.Name().String()
	p.Module.Apps[appName].Endpoints[endpointName], Calls = endpointFromMethod(m)

	for app, endpoint := range Calls {
		if _, ok := p.Module.Apps[app]; !ok {
			p.Module.Apps[app] = syslpopulate.NewApplication(app)
		}
		if _, ok := p.Module.Apps[app].Endpoints[endpoint]; !ok {
			p.Module.Apps[app].Endpoints[endpoint] = syslpopulate.NewEndpoint(endpoint)
		}
	}
	return p, nil
}

// VisitEnumValue converts to sysl enums. All types are writen to the
// Currently this sysl syntax is unsupported, but enums exist within the sysl data object
// enum foo{...} --> !enum foo:
func (p *PrinterModule) VisitEnum(e pgs.Enum) (v pgs.Visitor, err error) {
	packageName := syslPackageName(e)
	typeName := e.Name().String()
	if _, ok := p.Module.Apps[packageName]; !ok {
		p.Module.Apps[packageName] = syslpopulate.NewApplication(packageName)
	}
	p.Module.Apps[packageName].Types[typeName] = &sysl.Type{
		Type: &sysl.Type_Enum_{
			Enum: &sysl.Type_Enum{
				Items: enumToSysl(e),
			},
		},
	}
	return v, nil
}
