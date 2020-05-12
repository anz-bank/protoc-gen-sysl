package gensysl

import (
	"bytes"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"google.golang.org/protobuf/types/descriptorpb"

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
	pattenAttributes := make(map[string]*sysl.Attribute)
	attrDefs := make(map[string]*sysl.Type)
	packageName := syslPackageName(p.PackageName)
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
		fieldName = e.GoName
		attrDefs[fieldName] = fieldGoType(e)
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

// goPackageOption interprets a file's go_package option.
// If there is no go_package, it returns ("", "").
// If there's a simple name, it returns (pkg, "").
// If the option implies an import path, it returns (pkg, impPath).
//type GetOptions interface {
//	GetOptions() *descriptorpb.FileOptions
//}

func goPackageOption(optDesc *descriptorpb.FileOptions) (pkg string, impPath string) {
	opt := optDesc.GetGoPackage()
	if opt == "" {
		return "", ""
	}
	rawPkg, impPath := goPackageOptionRaw(opt)
	pkg = cleanPackageName(rawPkg)
	if string(pkg) != rawPkg && impPath != "" {

	}
	return pkg, impPath
}
func goPackageOptionRaw(opt string) (rawPkg string, impPath string) {
	// A semicolon-delimited suffix delimits the import path and package name.
	if i := strings.Index(opt, ";"); i >= 0 {
		return opt[i+1:], string(opt[:i])
	}
	// The presence of a slash implies there's an import path.
	if i := strings.LastIndex(opt, "/"); i >= 0 {
		return opt[i+1:], string(opt)
	}
	return opt, ""
}

// cleanPackageName converts a string to a valid Go package name.
func cleanPackageName(name string) string {
	return string(GoSanitized(name))
}

// GoSanitized converts a string to a valid Go identifier.
func GoSanitized(s string) string {
	// Sanitize the input to the set of valid characters,
	// which must be '_' or be in the Unicode L or N categories.
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, s)

	// Prepend '_' in the event of a Go keyword conflict or if
	// the identifier is invalid (does not start in the Unicode L category).
	r, _ := utf8.DecodeRuneInString(s)
	if token.Lookup(s).IsKeyword() || !unicode.IsLetter(r) {
		return "_" + s
	}
	return s
}
