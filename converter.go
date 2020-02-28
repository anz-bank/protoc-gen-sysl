package main

import (
	"strings"

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
				//Context: &sysl.Scope{
				//	Appname: &sysl.AppName{
				//		Part: []string{typeApplication},
				//	},
				//	Path: []string{fieldType},
				//},
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
