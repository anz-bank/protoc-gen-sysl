package newsysl

import (
	"fmt"
	"strings"

	"github.com/anz-bank/sysl/pkg/sysl"
)

var TypeMapping = map[string]sysl.Type_Primitive{
	"bytes":   sysl.Type_BYTES,
	"int32":   sysl.Type_INT,
	"int64":   sysl.Type_INT,
	"uint32":  sysl.Type_INT,
	"uint64":  sysl.Type_INT,
	"string":  sysl.Type_STRING,
	"bool":    sysl.Type_BOOL,
	"float64": sysl.Type_FLOAT,
	"float":   sysl.Type_FLOAT,
}

var specialMappings = map[string]string{"date": "date_", "Any": "Any_", "any": "any_"}

// Pattern returns a pattern attribute nested in a whole bunch of other types because fml
func Pattern(patterns ...string) *sysl.Attribute {
	var Elt []*sysl.Attribute
	for _, pattern := range patterns {
		Elt = append(Elt, &sysl.Attribute{Attribute: &sysl.Attribute_S{S: pattern}})
	}
	return &sysl.Attribute{Attribute: &sysl.Attribute_A{A: &sysl.Attribute_Array{Elt: Elt}}}
}

// Application Initialises a Sysl application
func Application(appName string) *sysl.Application {
	return &sysl.Application{
		Name:      AppName(appName),
		Endpoints: map[string]*sysl.Endpoint{},
		Types:     map[string]*sysl.Type{},
		Attrs:     map[string]*sysl.Attribute{},
	}
}

// Application Initialises a Sysl application
func Module() *sysl.Module {
	return &sysl.Module{Apps: map[string]*sysl.Application{}}
}

// Endpoint Initialises a Sysl Endpoint
func Endpoint(name string) *sysl.Endpoint {
	return &sysl.Endpoint{Name: name}
}

// Param Initialises a Sysl Param input
func Param(name, application string) *sysl.Param {
	return &sysl.Param{
		Name: "input",
		Type: Type(name, application),
	}
}

// Attribute Initialises a Sysl Attribute
func Attribute(value string) *sysl.Attribute {
	return &sysl.Attribute{
		Attribute: &sysl.Attribute_S{S: value},
	}
}

// Attribute Initialises a Sysl Attribute
func AttributeAny(value interface{}) *sysl.Attribute {
	return &sysl.Attribute{
		Attribute: &sysl.Attribute_S{S: fmt.Sprint(value)},
	}
}

// Type Initialises a Sysl type from string
func Type(name, application string) *sysl.Type {
	if strings.Contains(name, "sequence of") {
		return NewSequence(strings.ReplaceAll(name, "sequence of", ""), application)
	}
	if fieldType, ok := TypeMapping[name]; ok {
		return Primitive(fieldType)
	}
	return Struct(name, application)
}

// Return Initialises a return statement and wraps it in a sysl statement
// payloads will be concatenated and seperated by dots "."
func Return(payloads ...string) *sysl.Statement {
	for i := range payloads {
		payloads[i] = SanitiseTypeName(payloads[i])
	}
	return &sysl.Statement{Stmt: &sysl.Statement_Ret{Ret: &sysl.Return{
		Payload: "ok <: " + strings.Join(payloads, ".")}}}
}

// Call Initialises a call statement and wraps it in a sysl statement
func Call(app, endpoint string) *sysl.Statement {
	return &sysl.Statement{Stmt: &sysl.Statement_Call{
		Call: &sysl.Call{
			Target:   AppName(app),
			Endpoint: endpoint}}}
}

// StringStatement Initialises a call statement and wraps it in a sysl statement
func StringStatement(value string) *sysl.Statement {
	return &sysl.Statement{Stmt: &sysl.Statement_Action{
		Action: &sysl.Action{
			Action: value,
		},
	},
	}
}

// AppName returns an appname from the inputs
func AppName(name ...string) *sysl.AppName {
	return &sysl.AppName{Part: name}
}

// Primitive converts a string to a sysl primitive type (parameter must be in sysl type)
func Primitive(fieldType sysl.Type_Primitive) *sysl.Type {
	return &sysl.Type{
		Type: &sysl.Type_Primitive_{
			Primitive: fieldType,
		},
	}
}
func Sequence(t *sysl.Type) *sysl.Type {
	return &sysl.Type{
		Type: &sysl.Type_Sequence{
			Sequence: t,
		},
	}
}

// Primitive converts a string to a sysl primitive type (parameter must be in sysl type)
func NewSequence(fieldType, application string) *sysl.Type {
	return &sysl.Type{
		Type: &sysl.Type_Sequence{
			Sequence: Type(fieldType, application),
		},
	}
}

// Struct converts a string to a sysl struct type
func Struct(fieldType, application string) *sysl.Type {
	var appName *sysl.AppName
	if application != "" {
		appName = AppName(application)
	}
	return &sysl.Type{
		Type: &sysl.Type_TypeRef{
			TypeRef: &sysl.ScopedRef{
				Ref: &sysl.Scope{
					Appname: appName,
					Path:    []string{fieldType},
				},
			},
		},
	}
}

// SanitiseTypeName returns names that aren't identifiers within sysl. eg. date gets converted to date__
func SanitiseTypeName(name string) string {
	parts := strings.Split(name, ".")
	typeName := parts[len(parts)-1]
	if _, ok := specialMappings[strings.ToLower(typeName)]; ok {
		typeName = specialMappings[strings.ToLower(typeName)]
		if len(parts) > 1 {
			typeName = parts[0] + typeName
		}
		return typeName
	}
	if _, ok := TypeMapping[strings.ToLower(typeName)]; ok {
		return typeName + "_"
	}
	return name
}
