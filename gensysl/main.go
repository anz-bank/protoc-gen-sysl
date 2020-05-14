package gensysl

import (
	"bytes"
	"strings"

	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/anz-bank/sysl/pkg/printer"
	"github.com/anz-bank/sysl/pkg/sysl"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/compiler/protogen"
)

type PrinterModule struct {
	Log    *logrus.Logger
	Module *sysl.Module
}

// GenerateFile generates the contents of a .pb.go file.
func GenerateFiles(gen *protogen.Plugin) error {
	filename := "index.sysl"
	var buf bytes.Buffer
	g := gen.NewGeneratedFile(filename, gen.Files[0].GoImportPath)
	p := &PrinterModule{
		Log:    logrus.New(),
		Module: &sysl.Module{Apps: make(map[string]*sysl.Application)},
	}
	for _, file := range gen.Files {
		if err := p.VisitFile(file); err != nil {
			return err
		}
	}
	printer.Module(&buf, p.Module)
	g.P(buf.String())
	return nil
}

func (p *PrinterModule) VisitFile(file *protogen.File) (err error) {
	for _, s := range file.Services {
		if err := p.VisitService(s); err != nil {
			return err
		}
	}
	for _, t := range file.Messages {
		if err := p.VisitMessage(t); err != nil {
			return err
		}
	}
	for _, e := range file.Enums {
		if err := p.VisitEnum(e); err != nil {
			return nil
		}
	}
	return nil
}

// VisitService converts to sysl and constructs endpoints from methods
// service myservice{...} --> myservice:
func (p *PrinterModule) VisitService(s *protogen.Service) error {
	pkgName, name := syslNames(string(s.Desc.Parent().ParentFile().Package()), string(s.Desc.FullName()))
	p.Module.Apps[name] = syslpopulate.NewApplication(name)
	p.Module.Apps[name].Attrs["package"] = syslpopulate.NewAttribute(pkgName)
	p.Module.Apps[name].Attrs["description"] = syslpopulate.NewAttribute(s.Comments.Leading.String() + s.Comments.Trailing.String())
	for _, e := range s.Methods {
		if err := p.VisitMethod(s, e); err != nil {
			return err
		}
	}
	return nil
}

// VisitMethod converts a message to a sysl endpoint and fills in calls to other functions
// rpc thisEndpoint(InputType)returns(outputType) -->
// thisEndpoint(input <: InputType):
//     return ok <: outputType
func (p *PrinterModule) VisitMethod(s *protogen.Service, m *protogen.Method) error {
	appName := syslpopulate.SanitiseTypeName(s.GoName)
	endpointName := syslpopulate.SanitiseTypeName(m.GoName)
	application, Name := syslNames(string(m.Input.Desc.Parent().ParentFile().Package()), string(m.Input.Desc.FullName()))
	endpoint := syslpopulate.NewEndpoint(m.GoName)
	endpoint.Param = []*sysl.Param{syslpopulate.NewParameter(Name, application)}
	application, Name = syslNames(string(m.Output.Desc.Parent().ParentFile().Package()), string(m.Output.Desc.FullName()))
	endpoint.Stmt = []*sysl.Statement{syslpopulate.NewReturn(application, Name)}
	endpoint.Attrs = make(map[string]*sysl.Attribute)
	endpoint.Attrs["description"] = syslpopulate.NewAttribute(m.Comments.Leading.String() + m.Comments.Trailing.String())
	p.Module.Apps[appName].Endpoints[endpointName] = endpoint
	return nil
}

// VisitMessage converts to sysl and constructs types from messages. All types are writen to the
// TypeApplication (as in sysl types belong to applications but not in proto
// message foo{...} --> !type foo:
func (p *PrinterModule) VisitMessage(m *protogen.Message) error {
	typeName := syslpopulate.SanitiseTypeName(string(m.Desc.Name()))
	var fieldName string
	attrs := make(map[string]*sysl.Attribute)
	attrDefs := make(map[string]*sysl.Type)
	packageName, typeName := syslNames(string(m.Desc.Parent().ParentFile().Package()), string(m.Desc.FullName()))
	if len(m.Fields) == 0 {
		attrs["patterns"] = syslpopulate.NewPattern("empty")
	}

	if description := m.Comments.Leading.String() + m.Comments.Trailing.String(); description != "" {
		attrs["description"] = syslpopulate.NewAttribute(description)
	}
	for _, e := range m.Fields {
		fieldName = syslpopulate.SanitiseTypeName(string(e.Desc.Name()))
		attrDefs[fieldName] = fieldGoType(packageName, e)
	}
	for _, e := range m.Messages {
		if err := p.VisitMessage(e); err != nil {
			return err
		}
	}
	for _, e := range m.Enums {
		if err := p.VisitEnum(e); err != nil {
			return err
		}
	}
	if _, ok := p.Module.Apps[packageName]; !ok {
		packageName = syslpopulate.SanitiseTypeName(packageName)
		p.Module.Apps[packageName] = syslpopulate.NewApplication(packageName)
		p.Module.Apps[packageName].Attrs["package"] = syslpopulate.NewAttribute(packageName)
	}
	p.Module.Apps[packageName].Types[typeName] = &sysl.Type{
		Attrs: attrs,
		Type: &sysl.Type_Tuple_{
			Tuple: &sysl.Type_Tuple{
				AttrDefs: attrDefs,
			},
		},
	}
	return nil
}

// VisitEnumValue converts to sysl enums. All types are writen to the
// Currently this sysl syntax is unsupported, but enums exist within the sysl data object
// enum foo{...} --> !enum foo:
func (p *PrinterModule) VisitEnum(e *protogen.Enum) error {
	packageName, typeName := syslNames(string(e.Desc.Parent().ParentFile().Package()), string(e.Desc.FullName()))
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
	return nil
}

func syslNames(pkg, fullName string) (string, string) {
	// A semicolon-delimited suffix delimits the import path and package name.
	name := strings.ReplaceAll(fullName, pkg, "")
	pkg = strings.ReplaceAll(pkg, ".", "_")
	name = strings.ReplaceAll(name, ".", "_")
	for i := 0; i < len(name); i++ {
		if name[i] == '_' && len(name) > i-1 {
			name = name[i+1:]
		} else {
			break
		}
	}
	return pkg, syslpopulate.SanitiseTypeName(name)
}
