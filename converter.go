package main

import (
	"regexp"
	"strings"

	"github.com/anz-bank/protoc-gen-sysl/sysloption"
	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/golang/protobuf/proto"

	"github.com/anz-bank/sysl/pkg/sysl"
	pgs "github.com/lyft/protoc-gen-star"
)

func syslPackageName(m pgs.Entity) string {
	return m.Package().ProtoName().UpperCamelCase().String()
}

// fieldToString converts a field type to a string and returns name and type respectively
func fieldToSysl(e pgs.Field) (string, *sysl.Type) {
	var fieldName, fieldType string
	application := syslPackageName(e)
	fieldName = e.Name().String()
	if t := e.Descriptor(); t != nil && t.TypeName != nil {
		fieldType = strings.ReplaceAll(*t.TypeName, e.Package().ProtoName().String(), "")
		fieldType = strings.ReplaceAll(fieldType, ".", "")
	} else {
		fieldType = e.Type().ProtoType().String()
	}
	return fieldName, syslpopulate.NewType(fieldType, application)
}

// messageToSysl converts a message to a sysl type
func messageToSysl(e pgs.Message) string {
	var fieldType string
	if t := e.Descriptor(); t != nil && t.Name != nil {
		fieldType = strings.ReplaceAll(*t.Name, syslPackageName(e), "")
		fieldType = strings.ReplaceAll(fieldType, ".", "")
	}
	return fieldType
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
	application := syslPackageName(method)
	for _, call := range calls {
		syslCallSplit := strings.Split(call.Call, ".")
		syslCalls = append(syslCalls, syslpopulate.NewCall(syslCallSplit[0], syslCallSplit[1]))
	}
	endpoint := syslpopulate.NewEndpoint(method.Name().String())
	endpoint.Param = []*sysl.Param{syslpopulate.NewParameter(messageToSysl(method.Input()), application)}
	endpoint.Stmt = append(syslCalls, syslpopulate.NewReturn(syslPackageName(method.Output())+"."+method.Output().Name().String()))
	return endpoint
}

// syslFilename converts replaces a .proto filename to .sysl, removing any paths
func syslFilename(s string) string {
	return strings.Replace(regexp.MustCompile(`(?m)\w*\.proto`).FindString(s), ".proto", "", -1)
}

// enumToSysl converts an Enum to a sysl enum
func enumToSysl(e pgs.Enum) map[string]int64 {
	values := make(map[string]int64)
	if t := e.Descriptor(); t != nil && t.Name != nil {
		for _, val := range t.Value {
			values[*val.Name] = int64(*val.Number)
		}
	}
	return values
}
