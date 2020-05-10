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
	Log         *logrus.Logger
	Module      *sysl.Module
	PackageName string
}

func syslPackageName(m string) string {
	return strings.ReplaceAll(strings.ReplaceAll(m, ".", " "), " ", "")
}

// GenerateFile generates the contents of a .pb.go file.
func GenerateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	filename := "index.sysl"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	p := &PrinterModule{
		Log:    logrus.New(),
		Module: &sysl.Module{Apps: make(map[string]*sysl.Application)},
	}
	p.VisitFile(file)
	var buf bytes.Buffer
	printer.Module(&buf, p.Module)
	g.P(buf.String())
	return g
}

func (p *PrinterModule) VisitFile(file *protogen.File) (err error) {
	p.PackageName = string(file.GoPackageName)
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
	name := s.GoName
	p.Module.Apps[name] = syslpopulate.NewApplication(name)
	p.Module.Apps[name].Attrs["package"] = syslpopulate.NewAttribute(p.PackageName)
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
func (p *PrinterModule) VisitMethod(s *protogen.Service, m *protogen.Method) (err error) {
	var Calls map[string]string
	appName := s.GoName
	endpointName := m.GoName
	p.Module.Apps[appName].Endpoints[endpointName], Calls = p.endpointFromMethod(m)
	p.Module.Apps[appName].Endpoints[endpointName].Attrs = make(map[string]*sysl.Attribute)
	p.Module.Apps[appName].Endpoints[endpointName].Attrs["description"] = syslpopulate.NewAttribute(m.Comments.Leading.String() + m.Comments.Trailing.String())
	for app, endpoint := range Calls {
		if _, ok := p.Module.Apps[app]; !ok {
			p.Module.Apps[app] = syslpopulate.NewApplication(app)
		}
		if _, ok := p.Module.Apps[app].Endpoints[endpoint]; !ok {
			p.Module.Apps[app].Endpoints[endpoint] = syslpopulate.NewEndpoint(endpoint)
		}
	}
	return nil
}

// VisitMessage converts to sysl and constructs types from messages. All types are writen to the
// TypeApplication (as in sysl types belong to applications but not in proto
// message foo{...} --> !type foo:
func (p *PrinterModule) VisitMessage(m *protogen.Message) error {
	var fieldName string
	var syslType *sysl.Type
	pattenAttributes := make(map[string]*sysl.Attribute)
	attrDefs := make(map[string]*sysl.Type)
	packageName := syslPackageName(string(m.Desc.Parent().ParentFile().Package().Name()))
	if len(m.Fields) == 0 {
		pattenAttributes["patterns"] = &sysl.Attribute{Attribute: &sysl.Attribute_A{A: &sysl.Attribute_Array{
			Elt: []*sysl.Attribute{&sysl.Attribute{
				Attribute: &sysl.Attribute_S{S: "empty"},
			},
			},
		},
		},
		}
	}
	if description := m.Comments.Leading.String() + m.Comments.Trailing.String(); description != "" {
		pattenAttributes["description"] = syslpopulate.NewAttribute(description)
	}
	for _, e := range m.Fields {
		fieldName, syslType = p.fieldToSysl(e)
		fieldName = syslpopulate.SanitiseTypeName(fieldName)
		attrDefs[fieldName] = syslType
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
		p.Module.Apps[packageName] = syslpopulate.NewApplication(packageName)
		p.Module.Apps[packageName].Attrs["package"] = syslpopulate.NewAttribute(packageName)
	}

	typeName := syslpopulate.SanitiseTypeName(m.GoIdent.GoName)
	p.Module.Apps[packageName].Types[typeName] = &sysl.Type{
		Attrs: pattenAttributes,
		Type: &sysl.Type_Tuple_{
			Tuple: &sysl.Type_Tuple{
				AttrDefs: attrDefs,
			},
		},
	}
	return nil
}

// fieldToString converts a field type to a string and returns name and type respectively
func (p *PrinterModule) fieldToSysl(f *protogen.Field) (string, *sysl.Type) {
	var fieldName, fieldType string
	application := syslPackageName(string(f.Desc.Parent().ParentFile().Package().Name()))
	fieldName = f.GoName
	if t := f.Desc; t != nil && t.Name() != "" {
		if arr := NoEmptyStrings(strings.Split(string(t.Name()), ".")); len(arr) > 1 {

			// This is some wack logic to process messages and enums that are defined in other messages
			fieldType = arr[len(arr)-1]
			remove := strings.ReplaceAll(string(f.Message.Desc.FullName()), string(f.Message.Desc.Parent().Name()), "")
			remove = strings.ReplaceAll(remove, ".", "")
			application = strings.ReplaceAll(strings.Join(arr[:len(arr)-1], ""), remove, "")
			if enum := f.Enum; enum != nil {
				remove = strings.ReplaceAll(string(enum.Desc.Name()), string(f.Message.Desc.Parent().Name()), "")
				remove = strings.ReplaceAll(remove, fieldType, "")
				remove = strings.ReplaceAll(remove, ".", "")

				application = strings.ReplaceAll(application, remove, "")

			}
		} else {
			fieldType = strings.ReplaceAll(string(t.Name()), application, "")
			fieldType = strings.ReplaceAll(fieldType, ".", "")
		}
	} else {
		//fieldType = f.

	}
	fieldType = syslpopulate.SanitiseTypeName(fieldType)
	if f.Desc.IsList() {
		return fieldName, &sysl.Type{
			Type: &sysl.Type_Sequence{
				Sequence: syslpopulate.NewType(fieldType, application),
			},
		}
	}
	return fieldName, syslpopulate.NewType(fieldType, application)
}

func NoEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, element := range in {
		if element != "" {
			out = append(out, element)
		}
	}
	return out
}

// VisitEnumValue converts to sysl enums. All types are writen to the
// Currently this sysl syntax is unsupported, but enums exist within the sysl data object
// enum foo{...} --> !enum foo:
func (p *PrinterModule) VisitEnum(e *protogen.Enum) error {
	packageName := syslPackageName(string(e.Desc.Parent().ParentFile().Package().Name()))
	typeName := e.GoIdent.GoName
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
