package gensysl

import (
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"

	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/anz-bank/sysl/pkg/sysl"
)

// EndpointFromMethod converts a pgs Method to a sysl endpoint and fills in call and return statments
func (p *PrinterModule) endpointFromMethod(m *protogen.Method) (*sysl.Endpoint, map[string]string) {
	syslCalls := []*sysl.Statement{}
	stringCalls := make(map[string]string)
	this, ok := m.Desc.ParentFile().Options().(*descriptorpb.FileOptions)
	if !ok {
		panic(this)
	}
	application, _ := goPackageOption(this)
	endpoint := syslpopulate.NewEndpoint(m.GoName)
	endpoint.Param = []*sysl.Param{syslpopulate.NewParameter(p.messageToSysl(m.Input), application)}
	for _, out := range m.Output.Messages {
		syslCalls = append(syslCalls, syslpopulate.NewReturn(application, p.messageToSysl(out)))
	}
	endpoint.Stmt = append(endpoint.Stmt, syslCalls...)
	return endpoint, stringCalls
}

// messageToSysl converts a message to a sysl type
func (p *PrinterModule) messageToSysl(m *protogen.Message) string {
	var fieldType string

	if t := m.Desc; t != nil {
		fieldType = m.GoIdent.GoName
		fieldType = strings.ReplaceAll(fieldType, ".", "")
		fieldType = syslpopulate.SanitiseTypeName(fieldType)
	}
	return fieldType
}

// enumToSysl converts an Enum to a sysl enum
func enumToSysl(e *protogen.Enum) map[string]int64 {
	values := make(map[string]int64)
	if t := e.Values; t != nil {
		for _, val := range t {
			values[string(val.Desc.Name())] = int64(val.Desc.Number())
		}
	}
	return values
}

// fieldGoType returns the Go type used for a field.
//
// If it returns pointer=true, the struct field is a pointer to the type.
func fieldGoType(field *protogen.Field) *sysl.Type {
	if field.Desc.IsWeak() {
		return nil
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
		t = syslpopulate.NewType(field.Message.GoIdent.GoName, "")
	case protoreflect.EnumKind:
		t = syslpopulate.NewType(field.Enum.GoIdent.GoName, "")
	}
	switch {
	case field.Desc.IsList():
		return syslpopulate.Sequence(t)
	}
	return t
}
