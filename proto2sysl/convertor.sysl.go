package proto2sysl

import (
	"fmt"
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

// fieldSyslType returns the Sysl type used for a field.
func fieldSyslType(ctx GenContext, parentNames names, field *protogen.Field) *sysl.Type {
	if field.Desc.IsWeak() {
		return nil
	}

	// createType returns a new type reflecting the message or enum field's descriptor.
	createType := func(d protoreflect.Descriptor, comments protogen.CommentSet) *sysl.Type {
		names := getNames(ctx, d)
		var t *sysl.Type
		if parentNames.protoPackage != names.protoPackage {
			if ns, ok := ctx.pkgNs[names.protoPackage]; ok && len(ns) > 0 {
				t = newsysl.Type(names.name, append(ns, typesAppName)...)
			} else {
				t = newsysl.Type(names.name, packageToApp(names.protoPackage))
			}
		} else if namespaceJoin(parentNames.appName) == namespaceJoin(names.appName) {
			t = newsysl.Type(names.name)
		} else {
			t = newsysl.Type(names.name, names.appName...)
		}
		if desc := cleanDescription(comments); desc != "" {
			t.Attrs = map[string]*sysl.Attribute{"description": newsysl.Attribute(desc)}
		}
		return t
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
		t = createType(field.Message.Desc, field.Message.Comments)
	case protoreflect.EnumKind:
		t = createType(field.Enum.Desc, field.Enum.Comments)
	default:
		panic(fmt.Sprintf("unknown type: %T", field.Desc.Kind()))
	}

	t.Attrs = map[string]*sysl.Attribute{
		"json_tag": newsysl.Attribute(field.Desc.JSONName()),
		"rpcId":    newsysl.Attribute(strconv.Itoa(int(field.Desc.Number()))),
	}
	if description := cleanDescription(field.Comments); description != "" {
		t.Attrs["description"] = newsysl.Attribute(description)
	}

	if field.Desc.IsList() {
		return newsysl.Sequence(t)
	}
	return t
}
