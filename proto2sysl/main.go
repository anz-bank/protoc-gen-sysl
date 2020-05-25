package proto2sysl

import (
	"bytes"
	"path"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/anz-bank/protoc-gen-sysl/newsysl"
	"github.com/anz-bank/sysl/pkg/printer"
	"github.com/anz-bank/sysl/pkg/sysl"
	"google.golang.org/protobuf/compiler/protogen"
)

// GenerateFile generates the contents of a index.sysl file.
func GenerateFiles(gen *protogen.Plugin) error {
	g := gen.NewGeneratedFile("index.sysl", gen.Files[0].GoImportPath)
	m := newsysl.Module() // destination sysl module
	import_prefix := strings.Replace(gen.Request.GetParameter(), "import_prefix=", "", 1)
	for _, file := range gen.Files {
		if err := VisitFile(import_prefix, m, file); err != nil {
			return err
		}
	}
	var buf bytes.Buffer
	printer.Module(&buf, m)
	g.P(buf.String())
	return nil
}

func VisitFile(importPrefix string, module *sysl.Module, file *protogen.File) (err error) {
	for _, s := range file.Services {
		if err := VisitService(importPrefix, module, s); err != nil {
			return err
		}
	}
	for _, t := range file.Messages {
		if err := VisitMessage(importPrefix, module, t); err != nil {
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
func VisitService(importPrefix string, module *sysl.Module, s *protogen.Service) error {
	pkgName, name := descToSyslName(s.Desc)
	app := newsysl.Application(name)

	app.Attrs["package"] = newsysl.Attribute(pkgName)
	app.Attrs["source_path"] = newsysl.Attribute(path.Join(importPrefix, s.Location.SourceFile))
	app.Attrs["description"] = newsysl.Attribute(cleanDescription(s.Comments))
	module.Apps[name] = app
	for _, e := range s.Methods {
		if err := VisitMethod(importPrefix, module, e); err != nil {
			return err
		}
	}
	return nil
}

// VisitMethod converts a message to a sysl endpoint and fills in calls to other functions
// rpc thisEndpoint(InputType)returns(outputType) -->
// thisEndpoint(input <: InputType):
//     return ok <: outputType
func VisitMethod(importPrefix string, module *sysl.Module, m *protogen.Method) error {
	appName := string(m.Desc.Parent().Name())
	endpointName := newsysl.SanitiseTypeName(string(m.Desc.Name()))
	endpoint := newsysl.Endpoint(endpointName)

	// Apps types are stored in a sysl app which is the same as the package name
	// Input
	packageName, Name := descToSyslName(m.Input.Desc)
	endpoint.Param = []*sysl.Param{newsysl.Param(Name, packageName)}

	// Output
	packageName, Name = descToSyslName(m.Output.Desc)
	endpoint.Stmt = []*sysl.Statement{newsysl.Return(packageName, Name)}

	// Attributes
	endpoint.Attrs = make(map[string]*sysl.Attribute)
	endpoint.Attrs["description"] = newsysl.Attribute(cleanDescription(m.Comments))
	endpoint.Attrs["patterns"] = newsysl.Pattern("grpc", "GRPC")
	endpoint.Attrs["source_path"] = newsysl.AttributeAny(path.Join(importPrefix, m.Location.SourceFile))

	module.Apps[appName].Endpoints[endpointName] = endpoint
	return nil
}

// VisitMessage converts to sysl and constructs types from messages. All types are writen to the
// TypeApplication (as in sysl types belong to applications but not in proto
// message foo{...} --> !type foo:
func VisitMessage(importPrefix string, module *sysl.Module, m *protogen.Message) error {
	var fieldName string
	attrs := make(map[string]*sysl.Attribute)
	attrDefs := make(map[string]*sysl.Type)
	packageName, typeName := descToSyslName(m.Desc)
	attrs["source_path"] = newsysl.AttributeAny(path.Join(importPrefix, m.Location.SourceFile))
	attrs["description"] = newsysl.Attribute(cleanDescription(m.Comments))
	for _, e := range m.Fields {
		fieldName = newsysl.SanitiseTypeName(string(e.Desc.Name()))
		attrDefs[fieldName] = fieldGoType(packageName, e)
	}
	// If there are no fields add ~empty pattern
	if len(m.Fields) == 0 {
		attrs["patterns"] = newsysl.Pattern("empty")
	}
	// in proto messages can be defined within messages
	for _, e := range m.Messages {
		if err := VisitMessage(importPrefix, module, e); err != nil {
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
		module.Apps[packageName] = newsysl.Application(packageName)
		module.Apps[packageName].Attrs["package"] = newsysl.Attribute(packageName)
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
		module.Apps[packageName] = newsysl.Application(packageName)
	}
	values := make(map[string]int64)
	t := e.Values
	for _, val := range t {
		values[newsysl.SanitiseTypeName(string(val.Desc.Name()))] = int64(val.Desc.Number())
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
	return pkg, newsysl.SanitiseTypeName(name)
}
