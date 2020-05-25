package proto2sysl

import (
	"strconv"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/anz-bank/protoc-gen-sysl/newsysl"
	"github.com/anz-bank/sysl/pkg/sysl"
)

func cleanDescription(comment protogen.CommentSet) string {
	var ret string
	s := []string{comment.Leading.String(), comment.Trailing.String()}
	for _, e := range s {
		ret += strings.ReplaceAll(e, "//", "\n")
	}
	return ret
}

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
		t = newsysl.Primitive(sysl.Type_BOOL)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		t = newsysl.Primitive(sysl.Type_INT)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		t = newsysl.Primitive(sysl.Type_INT)
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		t = newsysl.Primitive(sysl.Type_FLOAT)
	case protoreflect.StringKind:
		t = newsysl.Primitive(sysl.Type_STRING)
	case protoreflect.BytesKind:
		t = newsysl.Primitive(sysl.Type_BYTES)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		_, typeName := descToSyslName(field.Message.Desc)
		if application == currentApp {
			application = ""
		}
		t = newsysl.Type(typeName, application)
	case protoreflect.EnumKind:
		_, typeName := descToSyslName(field.Enum.Desc)
		if application == currentApp {
			application = ""
		}
		t = newsysl.Type(typeName, application)
	}

	t.Attrs = map[string]*sysl.Attribute{
		"json_tag": newsysl.Attribute(field.Desc.JSONName()),
		"rpcId":    newsysl.Attribute(strconv.Itoa(int(field.Desc.Number()))),
	}
	if description := cleanDescription(field.Comments); description != "" {
		t.Attrs["description"] = newsysl.Attribute(description)

	}
	switch {
	case field.Desc.IsList():
		return newsysl.Sequence(t)
	}
	return t
}
