package proto2sysl

import (
	"bytes"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/anz-bank/sysl/pkg/printer"
	"github.com/anz-bank/sysl/pkg/sysl"
	"google.golang.org/protobuf/compiler/protogen"
)

type Converter struct {
	Module *sysl.Module
}

// GenerateFile generates the contents of a .pb.go file.
func GenerateFiles(gen *protogen.Plugin) error {
	filename := "index.sysl"
	var buf bytes.Buffer
	g := gen.NewGeneratedFile(filename, gen.Files[0].GoImportPath)
	m := syslpopulate.NewModule()
	for _, file := range gen.Files {
		if err := VisitFile(m, file); err != nil {
			return err
		}
	}
	printer.Module(&buf, m)
	g.P(buf.String())
	return nil
}

func VisitFile(module *sysl.Module, file *protogen.File) (err error) {
	for _, s := range file.Services {
		if err := VisitService(module, s); err != nil {
			return err
		}
	}
	for _, t := range file.Messages {
		if err := VisitMessage(module, t); err != nil {
			return err
		}
	}
	for _, e := range file.Enums {
		if err := VisitEnum(module, e); err != nil {
			return nil
		}
	}
	return nil
}

// VisitService converts to sysl and constructs endpoints from methods
// service myservice{...} --> myservice:
func VisitService(module *sysl.Module, s *protogen.Service) error {
	pkgName, name := descToSyslName(s.Desc)
	app := syslpopulate.NewApplication(name)

	app.Attrs["package"] = syslpopulate.NewAttribute(pkgName)
	app.Attrs["description"] = syslpopulate.NewAttribute(s.Comments.Leading.String() + s.Comments.Trailing.String())
	module.Apps[name] = app
	for _, e := range s.Methods {
		if err := VisitMethod(module, s, e); err != nil {
			return err
		}
	}
	return nil
}

// VisitMethod converts a message to a sysl endpoint and fills in calls to other functions
// rpc thisEndpoint(InputType)returns(outputType) -->
// thisEndpoint(input <: InputType):
//     return ok <: outputType
func VisitMethod(module *sysl.Module, s *protogen.Service, m *protogen.Method) error {
	appName := string(m.Desc.Parent().Name())
	endpointName := syslpopulate.SanitiseTypeName(string(m.Desc.Name()))
	endpoint := syslpopulate.NewEndpoint(endpointName)

	// Apps types are stored in a sysl app which is the same as the package name
	// Input
	packageName, Name := descToSyslName(m.Input.Desc)
	endpoint.Param = []*sysl.Param{syslpopulate.NewParameter(Name, packageName)}

	// Output
	packageName, Name = descToSyslName(m.Output.Desc)
	endpoint.Stmt = []*sysl.Statement{syslpopulate.NewReturn(packageName, Name)}

	// Attributes
	endpoint.Attrs = make(map[string]*sysl.Attribute)
	endpoint.Attrs["description"] = syslpopulate.NewAttribute(m.Comments.Leading.String() + m.Comments.Trailing.String())

	module.Apps[appName].Endpoints[endpointName] = endpoint
	return nil
}

// VisitMessage converts to sysl and constructs types from messages. All types are writen to the
// TypeApplication (as in sysl types belong to applications but not in proto
// message foo{...} --> !type foo:
func VisitMessage(module *sysl.Module, m *protogen.Message) error {
	var fieldName string
	attrs := make(map[string]*sysl.Attribute)
	attrDefs := make(map[string]*sysl.Type)
	packageName, typeName := descToSyslName(m.Desc)

	if description := m.Comments.Leading.String() + m.Comments.Trailing.String(); description != "" {
		attrs["description"] = syslpopulate.NewAttribute(description)
	}
	for _, e := range m.Fields {
		fieldName = syslpopulate.SanitiseTypeName(string(e.Desc.Name()))
		attrDefs[fieldName] = fieldGoType(packageName, e)
	}
	// If there are no fields add ~empty pattern
	if len(m.Fields) == 0 {
		attrs["patterns"] = syslpopulate.NewPattern("empty")
	}
	// in proto messages can be defined within messages
	for _, e := range m.Messages {
		if err := VisitMessage(module, e); err != nil {
			return err
		}
	}
	// same with enums
	for _, e := range m.Enums {
		if err := VisitEnum(module, e); err != nil {
			return err
		}
	}
	// If this is the first service in the package, we need to make an app to store the types
	if _, ok := module.Apps[packageName]; !ok {
		module.Apps[packageName] = syslpopulate.NewApplication(packageName)
		module.Apps[packageName].Attrs["package"] = syslpopulate.NewAttribute(packageName)
	}
	module.Apps[packageName].Types[typeName] = &sysl.Type{
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
func VisitEnum(module *sysl.Module, e *protogen.Enum) error {
	packageName, typeName := descToSyslName(e.Desc)
	if _, ok := module.Apps[packageName]; !ok {
		module.Apps[packageName] = syslpopulate.NewApplication(packageName)
	}
	values := make(map[string]int64)
	t := e.Values
	for _, val := range t {
		values[syslpopulate.SanitiseTypeName(string(val.Desc.Name()))] = int64(val.Desc.Number())
	}
	module.Apps[packageName].Types[typeName] = &sysl.Type{
		Type: &sysl.Type_Enum_{
			Enum: &sysl.Type_Enum{
				Items: values,
			},
		},
	}
	return nil
}

func descToSyslName(Desc protoreflect.Descriptor) (string, string) {
	return syslNames(string(Desc.Parent().ParentFile().Package()), string(Desc.FullName()))

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
