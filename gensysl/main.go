package gensysl

import (
	"bytes"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

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
		if err := p.VisitService(file, s); err != nil {
			return err
		}
	}
	for _, t := range file.Messages {
		if err := p.VisitMessage(file, t); err != nil {
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
func (p *PrinterModule) VisitService(file *protogen.File, s *protogen.Service) error {
	name := syslpopulate.SanitiseTypeName(s.GoName)
	p.Module.Apps[name] = syslpopulate.NewApplication(name)
	pkgName, _ := goPackageOptionRaw(string(file.Desc.FullName()))
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
func (p *PrinterModule) VisitMethod(s *protogen.Service, m *protogen.Method) (err error) {
	appName := syslpopulate.SanitiseTypeName(s.GoName)
	endpointName := syslpopulate.SanitiseTypeName(m.GoName)
	application, Name := p.messageToSysl(m.Input)
	endpoint := syslpopulate.NewEndpoint(m.GoName)
	endpoint.Param = []*sysl.Param{syslpopulate.NewParameter(Name, application)}
	application, Name = p.messageToSysl(m.Output)
	endpoint.Stmt = []*sysl.Statement{syslpopulate.NewReturn(application, Name)}
	endpoint.Attrs = make(map[string]*sysl.Attribute)
	endpoint.Attrs["description"] = syslpopulate.NewAttribute(m.Comments.Leading.String() + m.Comments.Trailing.String())
	p.Module.Apps[appName].Endpoints[endpointName] = endpoint
	return nil
}

// VisitMessage converts to sysl and constructs types from messages. All types are writen to the
// TypeApplication (as in sysl types belong to applications but not in proto
// message foo{...} --> !type foo:
func (p *PrinterModule) VisitMessage(file *protogen.File, m *protogen.Message) error {
	typeName := syslpopulate.SanitiseTypeName(m.GoIdent.GoName)
	var fieldName string
	pattenAttributes := make(map[string]*sysl.Attribute)
	attrDefs := make(map[string]*sysl.Type)
	packageName, _ := goPackageOptionRaw(string(m.Desc.FullName()), string(m.Desc.Name()))
	packageName = syslpopulate.SanitiseTypeName(packageName)
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
		fieldName = syslpopulate.SanitiseTypeName(e.GoName)
		attrDefs[fieldName] = fieldGoType(packageName, e)
	}
	for _, e := range m.Messages {
		if err := p.VisitMessage(file, e); err != nil {
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
	packageName, _ := goPackageOptionRaw(string(e.Desc.FullName()), string(e.Desc.Name()))
	typeName := syslpopulate.SanitiseTypeName(string(e.Desc.Name()))
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

//func goPackageOption(optDesc protoreflect.ProtoMessage) (pkg string, impPath string) {
//	fileOpt, ok := optDesc.(*descriptorpb.FileOptions)
//	if !ok || fileOpt == nil {
//		return
//	}
//	opt := *fileOpt.GoPackage
//	if opt == "" {
//		return "", ""
//	}
//	rawPkg, impPath := goPackageOptionRaw(opt)
//	pkg = cleanPackageName(rawPkg)
//	if string(pkg) != rawPkg && impPath != "" {
//
//	}
//	return pkg, impPath
//}
func goPackageOptionRaw(opt string, t ...string) (rawPkg string, impPath string) {
	// A semicolon-delimited suffix delimits the import path and package name.
	if i := strings.Index(opt, ";"); i >= 0 {
		//return syslpopulate.SanitiseTypeName(opt[i+1:]), string(opt[:i])
		rawPkg = opt[i+1:]
		// The presence of a slash implies there's an import path.
	} else if i := strings.LastIndex(opt, "/"); i >= 0 {
		//return syslpopulate.SanitiseTypeName(opt[i+1:]), string(opt)
		rawPkg = opt[i+1:]
	} else {
		rawPkg = opt
	}
	rawPkg = syslpopulate.SanitiseTypeName(rawPkg)
	rawPkg = strings.ReplaceAll(rawPkg, ".", "_")
	for _, e := range t {
		rawPkg = strings.ReplaceAll(rawPkg, e, "")
	}
	for i := len(rawPkg) - 1; i >= 0; i-- {
		if rawPkg[i] == '_' {
			rawPkg = rawPkg[0:i]
		} else {
			break
		}
	}
	return rawPkg, ""
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
