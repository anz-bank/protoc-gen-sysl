package proto2sysl

import (
	"bytes"
	"path"
	"strings"

	"github.com/anz-bank/protoc-gen-sysl/newsysl"
	"github.com/anz-bank/sysl/pkg/printer"
	"github.com/anz-bank/sysl/pkg/sysl"
	"google.golang.org/protobuf/compiler/protogen"
)

// GenContext contains top-level data relevant to the whole generation process.
type GenContext struct {
	// module is the destination Sysl module being generated.
	module *sysl.Module
	// importPrefix is a path to prefix to source locations.
	importPrefix string
	// pkgNs is a map of proto package names to Sysl namespaces declared in those files.
	// The namespace values should be used for types declared in those packages.
	pkgNs map[string][]string
}

// NewGenContext returns a new GenContext populated with the context from gen.
func NewGenContext(gen *protogen.Plugin) GenContext {
	return GenContext{
		importPrefix: getImportPrefix(gen),
		pkgNs:        extractPackageNamespaces(gen),
		module:       newsysl.Module(),
	}
}

// extractPackageNamespaces explores the set of imported files and returns a map of proto packages
// to the Sysl namespaces declared as options in those files.
func extractPackageNamespaces(gen *protogen.Plugin) map[string][]string {
	m := make(map[string][]string, len(gen.Files))
	for _, file := range gen.Files {
		m[string(file.Desc.Package())] = getFileNamespaceOption(file.Desc)
	}
	return m
}

// getImportPrefix extracts the import_prefix parameter from the gen request.
func getImportPrefix(gen *protogen.Plugin) string {
	return strings.Replace(gen.Request.GetParameter(), "import_prefix=", "", 1)
}

// includeImport returns true if the proto identified by the import path should be included in the
// generation.
func includeImport(path string) bool {
	return path != "google/protobuf/descriptor.proto"
}

// GenerateFile generates the contents of a index.sysl file.
func GenerateFiles(gen *protogen.Plugin) error {
	g := gen.NewGeneratedFile("index.sysl", gen.Files[0].GoImportPath)
	m, err := GenerateModule(gen)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	printer.Module(&buf, m)
	g.P(buf.String())
	return nil
}

func GenerateModule(gen *protogen.Plugin) (*sysl.Module, error) {
	ctx := NewGenContext(gen)

	for _, file := range gen.Files {
		if includeImport(*file.Proto.Name) {
			if err := VisitFile(ctx, file); err != nil {
				return nil, err
			}
		}
	}
	return ctx.module, nil
}

func VisitFile(ctx GenContext, file *protogen.File) (err error) {
	for _, s := range file.Services {
		if err := VisitService(ctx, s); err != nil {
			return err
		}
	}
	for _, t := range file.Messages {
		if err := VisitMessage(ctx, t); err != nil {
			return err
		}
	}
	for _, e := range file.Enums {
		if err := VisitEnum(ctx, e); err != nil {
			return nil
		}
	}
	return nil
}

// VisitService converts to sysl and constructs endpoints from methods
// service myservice{...} --> myservice:
func VisitService(ctx GenContext, s *protogen.Service) error {
	names := getNames(ctx, s.Desc)
	app := getOrCreateApp(ctx, names)

	app.Attrs["source_path"] = newsysl.Attribute(path.Join(ctx.importPrefix, s.Location.SourceFile))
	app.Attrs["description"] = newsysl.Attribute(cleanDescription(s.Comments))
	app.Attrs["patterns"] = newsysl.Pattern("gRPC")

	for _, method := range s.Methods {
		if err := VisitMethod(ctx, names.appName, method); err != nil {
			return err
		}
	}
	return nil
}

// VisitMethod converts a message to a sysl endpoint and fills in calls to other functions
// rpc thisEndpoint(InputType) returns(OutputType) -->
// thisEndpoint(input <: InputType):
//     return ok <: OutputType
func VisitMethod(ctx GenContext, appName []string, m *protogen.Method) error {
	names := getNames(ctx, m.Desc)
	appFullName := namespaceJoin(appName)
	endpoint := newsysl.Endpoint(names.name)

	endpoint.Param = []*sysl.Param{VisitParam(ctx, m.Input)}
	endpoint.Stmt = []*sysl.Statement{VisitReturn(ctx, m.Output)}

	// Attributes
	endpoint.Attrs = make(map[string]*sysl.Attribute)
	endpoint.Attrs["description"] = newsysl.Attribute(cleanDescription(m.Comments))
	endpoint.Attrs["patterns"] = newsysl.Pattern("gRPC")
	endpoint.Attrs["source_path"] = newsysl.Attribute(path.Join(ctx.importPrefix, m.Location.SourceFile))

	ctx.module.Apps[appFullName].Endpoints[endpoint.Name] = endpoint
	return nil
}

// VisitParam converts an input message to a Sysl parameter and constructs a corresponding type in
// the TypeApplication (unlike protobufs, Sysl types must be contained by applications).
// rpc thisEndpoint(InputType) returns(OutputType) -->
// Types: !type InputType: ...
func VisitParam(ctx GenContext, m *protogen.Message) *sysl.Param {
	names := getNames(ctx, m.Desc)
	app := getOrCreateApp(ctx, names)
	if _, ok := app.Types[names.name]; !ok {
		app.Types[names.name] = newsysl.Type(names.name, names.appName...)
	}
	return newsysl.Param(names.name, names.appName...)
}

// VisitReturn converts an output message to a Sysl return statement and constructs a corresponding
// type in the TypeApplication (unlike protobufs, Sysl types must be contained by applications).
// rpc thisEndpoint(InputType) returns(OutputType) -->
// Types: !type OutputType: ...
func VisitReturn(ctx GenContext, m *protogen.Message) *sysl.Statement {
	names := getNames(ctx, m.Desc)
	app := getOrCreateApp(ctx, names)
	if _, ok := app.Types[names.name]; !ok {
		app.Types[names.name] = newsysl.Type(names.name, names.appName...)
	}
	return newsysl.Return(names.fullName)
}

// VisitMessage converts to sysl and constructs types from messages. All types are writen to the
// TypeApplication (as in sysl types belong to applications but not in proto
// message foo{...} --> !type foo:
func VisitMessage(ctx GenContext, m *protogen.Message) error {
	names := getNames(ctx, m.Desc)

	attrs := make(map[string]*sysl.Attribute)
	attrs["source_path"] = newsysl.Attribute(path.Join(ctx.importPrefix, m.Location.SourceFile))
	if desc := cleanDescription(m.Comments); desc != "" {
		attrs["description"] = newsysl.Attribute(desc)
	}

	// Process nested messages and enums first, so they're available for reference.
	for _, e := range m.Messages {
		if err := VisitMessage(ctx, e); err != nil {
			return err
		}
	}
	for _, e := range m.Enums {
		if err := VisitEnum(ctx, e); err != nil {
			return err
		}
	}

	attrDefs := make(map[string]*sysl.Type)
	for _, f := range m.Fields {
		fieldName := newsysl.SanitiseTypeName(string(f.Desc.Name()))
		attrDefs[fieldName] = fieldSyslType(ctx, names, f)
	}
	// If there are no fields add ~empty pattern
	if len(m.Fields) == 0 {
		attrs["patterns"] = newsysl.Pattern("empty")
	}

	app := getOrCreateApp(ctx, names)
	app.Types[names.name] = &sysl.Type{
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
func VisitEnum(ctx GenContext, e *protogen.Enum) error {
	names := getNames(ctx, e.Desc)
	values := make(map[string]int64)
	for _, val := range e.Values {
		values[newsysl.SanitiseTypeName(string(val.Desc.Name()))] = int64(val.Desc.Number())
	}

	app := getOrCreateApp(ctx, names)
	app.Types[names.name] = &sysl.Type{
		Type: &sysl.Type_Enum_{
			Enum: &sysl.Type_Enum{
				Items: values,
			},
		},
	}
	return nil
}

// getOrCreateApp returns the app in module named n.appName, creating a new one if it doesn't
// already exist.
func getOrCreateApp(ctx GenContext, n names) *sysl.Application {
	name := namespaceJoin(n.appName)
	app, ok := ctx.module.Apps[name]
	if !ok {
		app = newsysl.Application(n.appName...)
		app.Attrs["package"] = newsysl.Attribute(strings.ReplaceAll(n.protoPackage, ".", "_"))
		ctx.module.Apps[name] = app
	}
	return app
}
