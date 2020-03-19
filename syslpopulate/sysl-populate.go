package syslpopulate

import (
	"strings"

	"github.com/anz-bank/sysl/pkg/sysl"
)

var TypeMapping = map[string]sysl.Type_Primitive{
	"TYPE_BYTES":  sysl.Type_BYTES,
	"TYPE_INT32":  sysl.Type_INT,
	"TYPE_STRING": sysl.Type_STRING,
	"TYPE_BOOL":   sysl.Type_BOOL,
}

var specialMappings = map[string]string{"date": "date__"}

// NewApplication Initialises a Sysl application
func NewApplication(appName string) *sysl.Application {
	return &sysl.Application{
		Name:      NewAppName(appName),
		Endpoints: map[string]*sysl.Endpoint{},
		Types:     map[string]*sysl.Type{},
		Attrs:     map[string]*sysl.Attribute{},
	}
}

// NewEndpoint Initialises a Sysl Endpoint
func NewEndpoint(name string) *sysl.Endpoint {
	return &sysl.Endpoint{Name: name}
}

// NewParameter Initialises a Sysl Parameter input
func NewParameter(name, application string) *sysl.Param {
	return &sysl.Param{
		Name: "input",
		Type: NewType(name, application),
	}
}

// NewAttribute Initialises a Sysl Attribute
func NewAttribute(value string) *sysl.Attribute {
	return &sysl.Attribute{
		Attribute: &sysl.Attribute_S{S: value},
	}
}

// NewType Initialises a Sysl type from string
func NewType(name, application string) *sysl.Type {
	if fieldType, ok := TypeMapping[name]; ok {
		return SyslPrimitive(fieldType)
	}
	return SyslStruct(name, application)
}

// NewReturn Initialises a return statement and wraps it in a sysl statement
// payloads will be concatenated and seperated by dots "."
func NewReturn(payloads ...string) *sysl.Statement {
	for i := range payloads {
		payloads[i] = SanitiseTypeName(payloads[i])
	}
	return &sysl.Statement{Stmt: &sysl.Statement_Ret{Ret: &sysl.Return{
		Payload: "ok <: " + strings.Join(payloads, ".")}}}
}

// NewCall Initialises a call statement and wraps it in a sysl statement
func NewCall(app, endpoint string) *sysl.Statement {
	return &sysl.Statement{Stmt: &sysl.Statement_Call{
		Call: &sysl.Call{
			Target:   NewAppName(app),
			Endpoint: endpoint}}}
}

// NewStringStatement Initialises a call statement and wraps it in a sysl statement
func NewStringStatement(value string) *sysl.Statement {
	return &sysl.Statement{Stmt: &sysl.Statement_Action{
		Action: &sysl.Action{
			Action: value,
		},
	},
	}
}

// AppName returns an appname from the inputs
func NewAppName(name ...string) *sysl.AppName {
	return &sysl.AppName{Part: name}
}

// SyslPrimitive converts a string to a sysl primitive type (parameter must be in sysl type)
func SyslPrimitive(fieldType sysl.Type_Primitive) *sysl.Type {
	return &sysl.Type{
		Type: &sysl.Type_Primitive_{
			Primitive: fieldType,
		},
	}
}

// SyslStruct converts a string to a sysl struct type
func SyslStruct(fieldType, application string) *sysl.Type {
	return &sysl.Type{
		Type: &sysl.Type_TypeRef{
			TypeRef: &sysl.ScopedRef{
				Ref: &sysl.Scope{
					Appname: NewAppName(application),
					Path:    []string{fieldType},
				},
			},
		},
	}
}

// SanitiseTypeName returns names that aren't identifiers within sysl. eg. date gets converted to date__
func SanitiseTypeName(name string) string {
	if _, ok := specialMappings[name]; ok {
		return specialMappings[name]
	}
	return name
}
