package main

import (
	"strings"

	"github.com/anz-bank/protoc-gen-sysl/sysloption"
	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/golang/protobuf/proto"

	"github.com/anz-bank/sysl/pkg/sysl"
	pgs "github.com/lyft/protoc-gen-star"
)

func syslPackageName(m pgs.Entity) string {
	return strings.ReplaceAll(strings.ReplaceAll(m.Package().ProtoName().String(), ".", " "), " ", "")
}

// fieldToString converts a field type to a string and returns name and type respectively
func (p *PrinterModule) fieldToSysl(e pgs.Field) (string, *sysl.Type) {
	var fieldName, fieldType string
	application := syslPackageName(e)
	fieldName = e.Name().String()
	if t := e.Descriptor(); t != nil && t.TypeName != nil {
		if arr := NoEmptyStrings(strings.Split(*t.TypeName, ".")); len(arr) > 1 {

			// This is some wack logic to process messages and enums that are defined in other messages
			fieldType = arr[len(arr)-1]
			remove := strings.ReplaceAll(e.Message().FullyQualifiedName(), e.Message().Parent().FullyQualifiedName(), "")
			remove = strings.ReplaceAll(remove, ".", "")
			application = strings.ReplaceAll(strings.Join(arr[:len(arr)-1], ""), remove, "")
			if enum := e.Type().Enum(); enum != nil {
				remove = strings.ReplaceAll(enum.FullyQualifiedName(), e.Message().Parent().FullyQualifiedName(), "")
				remove = strings.ReplaceAll(remove, fieldType, "")
				remove = strings.ReplaceAll(remove, ".", "")

				application = strings.ReplaceAll(application, remove, "")

			}
		} else {
			fieldType = strings.ReplaceAll(*t.TypeName, e.Package().ProtoName().String(), "")
			fieldType = strings.ReplaceAll(fieldType, ".", "")
		}
	} else {
		fieldType = e.Type().ProtoType().String()
	}
	return fieldName, syslpopulate.NewType(fieldType, application)
}
func NoEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, element := range in {
		if element != "" {
			out = append(out, element)
		}
	}
	return out
}

// messageToSysl converts a message to a sysl type
func messageToSysl(e pgs.Message) string {
	var fieldType string
	if t := e.Descriptor(); t != nil && t.Name != nil {
		fieldType = strings.ReplaceAll(*t.Name, syslPackageName(e), "")
		fieldType = strings.ReplaceAll(fieldType, ".", "")
		fieldType = syslpopulate.SanitiseTypeName(fieldType)
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
func endpointFromMethod(method pgs.Method) (*sysl.Endpoint, map[string]string) {
	callOption := customOption(method)
	syslCalls := []*sysl.Statement{}
	stringCalls := make(map[string]string)
	application := syslPackageName(method)
	for _, calls := range callOption {
		for _, call := range calls.Call {
			syslCallSplit := strings.Split(call, ".")
			syslCalls = append(syslCalls, syslpopulate.NewCall(syslCallSplit[0], syslCallSplit[1]))
			stringCalls[syslCallSplit[0]] = syslCallSplit[1]
		}
	}
	endpoint := syslpopulate.NewEndpoint(method.Name().String())
	endpoint.Param = []*sysl.Param{syslpopulate.NewParameter(messageToSysl(method.Input()), application)}
	endpoint.Stmt = append(syslCalls, syslpopulate.NewReturn(syslPackageName(method.Output()), method.Output().Name().String()))
	return endpoint, stringCalls
}

// syslFilename converts replaces a .proto filename to .sysl, removing any paths
//func syslFilename(s string) string {
//	return strings.Replace(regexp.MustCompile(`(?m)\w*\.proto`).FindString(s), ".proto", "", -1)
//}

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
