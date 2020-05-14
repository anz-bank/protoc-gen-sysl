package proto2sysl

import (
	"strconv"

	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/anz-bank/sysl/pkg/sysl"
)

// fieldGoType returns the Go type used for a field.
func fieldGoType(currentApp string, field *protogen.Field) *sysl.Type {
	if field.Desc.IsWeak() {
		return nil
	}
	application, _ := descToSyslName(field.Desc)
	if field.Message != nil {
		application, _ = descToSyslName(field.Message.Desc)
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
		_, typeName := descToSyslName(field.Message.Desc)
		if application == currentApp {
			application = ""
		}
		t = syslpopulate.NewType(typeName, application)
	case protoreflect.EnumKind:
		_, typeName := descToSyslName(field.Enum.Desc)
		if application == currentApp {
			application = ""
		}
		t = syslpopulate.NewType(typeName, application)
	}
	t.Attrs = map[string]*sysl.Attribute{
		"json_tag": syslpopulate.NewAttribute(field.Desc.JSONName()),
		"rpcId":    syslpopulate.NewAttribute(strconv.Itoa(int(field.Desc.Number()))),
	}
	switch {
	case field.Desc.IsList():
		return syslpopulate.Sequence(t)
	}
	return t
}
