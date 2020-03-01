package main

import (
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/joshcarp/protoc-gen-sysl/sysloption"

	"github.com/anz-bank/sysl/pkg/sysl"
	pgs "github.com/lyft/protoc-gen-star"
)

// syslStruct converts a string to a sysl struct type
func syslStruct(fieldType string) *sysl.Type {
	//Path :=
	return &sysl.Type{
		Type: &sysl.Type_TypeRef{
			TypeRef: &sysl.ScopedRef{
				Ref: &sysl.Scope{
					Appname: &sysl.AppName{
						Part: []string{typeApplication},
					},
					Path: []string{fieldType},
				},
			},
		},
	}
}

// syslPrimitive converts a string to a sysl primitive type
func syslPrimitive(fieldType string) *sysl.Type {
	return &sysl.Type{
		Type: &sysl.Type_Primitive_{
			Primitive: TypeMapping[fieldType],
		},
	}
}

// fieldToString converts a field type to a string and returns name and type respectively
func fieldToSysl(e pgs.Field) (string, *sysl.Type) {
	var fieldName, fieldType string
	var syslType *sysl.Type
	fieldName = e.Name().String()
	if t := e.Descriptor(); t != nil && t.TypeName != nil {
		fieldType = strings.ReplaceAll(*t.TypeName, e.Package().ProtoName().String(), "")
		fieldType = strings.ReplaceAll(fieldType, ".", "")
		syslType = syslStruct(fieldType)
	} else {
		fieldType = e.Type().ProtoType().String()
		syslType = syslPrimitive(fieldType)
	}
	return fieldName, syslType
}

// fieldToString converts a field type to a string and returns name and type respectively
func messageToSysl(e pgs.Message) *sysl.Type {
	var fieldType string
	var syslType *sysl.Type
	if t := e.Descriptor(); t != nil && t.Name != nil {
		fieldType = strings.ReplaceAll(*t.Name, e.Package().ProtoName().String(), "")
		fieldType = strings.ReplaceAll(fieldType, ".", "")
		syslType = syslStruct(fieldType)
	} else {
		syslType = syslPrimitive(fieldType)
	}
	return syslType
}

// customOption converts a pgs method to a slice of the sysl CallRule
func customOption(meth pgs.Method) []*sysloption.CallRule {
	var call []*sysloption.CallRule
	if proto.HasExtension(meth.Descriptor().Options, sysloption.E_Calls) {
		this, _ := proto.GetExtension(meth.Descriptor().Options, sysloption.E_Calls)
		call = this.([]*sysloption.CallRule)
	}
	return call
}

// EndpointFromMethod converts a pgs Method to a sysl endpoint and fills in call and return statments
func endpointFromMethod(method pgs.Method) *sysl.Endpoint {
	calls := customOption(method)
	syslCalls := []*sysl.Statement{}
	for _, call := range calls {
		syslCallSplit := strings.Split(call.Call, ".")
		syslCalls = append(syslCalls, &sysl.Statement{
			Stmt: &sysl.Statement_Call{
				Call: &sysl.Call{
					Target: &sysl.AppName{
						Part: []string{syslCallSplit[0]},
					},
					Endpoint: syslCallSplit[1],
				},
			},
		},
		)
	}
	return &sysl.Endpoint{
		Name:     method.Name().String(),
		LongName: method.FullyQualifiedName(),
		Param: []*sysl.Param{{
			Name: "input",
			Type: messageToSysl(method.Input()),
		}},
		Stmt: append(syslCalls, &sysl.Statement{Stmt: &sysl.Statement_Ret{Ret: &sysl.Return{
			Payload: method.Output().Name().String(),
		}}}),
	}
}

// syslFilename converts replaces a .proto filename to .sysl, removing any paths
func syslFilename(s string) string {
	return strings.Replace(regexp.MustCompile(`(?m)\w*\.proto`).FindString(s), ".proto", "", -1)
}
