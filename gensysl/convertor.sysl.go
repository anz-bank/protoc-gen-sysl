package gensysl

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/anz-bank/sysl/pkg/sysl"
)

// EndpointFromMethod converts a pgs Method to a sysl endpoint and fills in call and return statments
func (p *PrinterModule) endpointFromMethod(m *protogen.Method) (*sysl.Endpoint, map[string]string) {
	syslCalls := []*sysl.Statement{}
	stringCalls := make(map[string]string)
	application, Name := p.messageToSysl(m.Input)
	endpoint := syslpopulate.NewEndpoint(m.GoName)
	endpoint.Param = []*sysl.Param{syslpopulate.NewParameter(Name, application)}
	application, Name = p.messageToSysl(m.Output)
	syslCalls = append(syslCalls, syslpopulate.NewReturn(application, Name))
	endpoint.Stmt = append(endpoint.Stmt, syslCalls...)

	return endpoint, stringCalls
}

// messageToSysl converts a message to a sysl type
func (p *PrinterModule) messageToSysl(m *protogen.Message) (string, string) {
	var fieldType string
	fieldType = syslpopulate.SanitiseTypeName(m.GoIdent.GoName)
	application, _ := goPackageOptionRaw(string(m.GoIdent.GoImportPath))
	return application, fieldType
}

// enumToSysl converts an Enum to a sysl enum
func enumToSysl(e *protogen.Enum) map[string]int64 {
	values := make(map[string]int64)
	if t := e.Values; t != nil {
		for _, val := range t {
			values[syslpopulate.SanitiseTypeName(string(val.Desc.Name()))] = int64(val.Desc.Number())
		}
	}
	return values
}

// fieldGoType returns the Go type used for a field.
//
// If it returns pointer=true, the struct field is a pointer to the type.
func fieldGoType(currentApp string, field *protogen.Field) *sysl.Type {
	if field.Desc.IsWeak() {
		return nil
	}
	application, _ := goPackageOptionRaw(string(field.GoIdent.GoImportPath))
	if field.Message != nil {
		application, _ = goPackageOptionRaw(string(field.Message.GoIdent.GoImportPath))
	}
	if application == currentApp {
		application = ""
	}
	var t *sysl.Type
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		t = syslpopulate.SyslPrimitive(sysl.Type_BOOL)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		t = syslpopulate.SyslPrimitive(sysl.Type_INT)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		t = syslpopulate.SyslPrimitive(sysl.Type_INT)
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		t = syslpopulate.SyslPrimitive(sysl.Type_FLOAT)
	case protoreflect.StringKind:
		t = syslpopulate.SyslPrimitive(sysl.Type_STRING)
	case protoreflect.BytesKind:
		t = syslpopulate.SyslPrimitive(sysl.Type_BYTES)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		t = syslpopulate.NewType(syslpopulate.SanitiseTypeName(field.Message.GoIdent.GoName), application)
	case protoreflect.EnumKind:
		t = syslpopulate.NewType(syslpopulate.SanitiseTypeName(field.Enum.GoIdent.GoName), application)
	}
	switch {
	case field.Desc.IsList():
		return syslpopulate.Sequence(t)
	}
	return t
}
