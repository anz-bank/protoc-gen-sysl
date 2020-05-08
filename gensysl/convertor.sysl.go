package gensysl

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/anz-bank/protoc-gen-sysl/syslpopulate"
	"github.com/anz-bank/sysl/pkg/sysl"
)

// EndpointFromMethod converts a pgs Method to a sysl endpoint and fills in call and return statments
func (p *PrinterModule) endpointFromMethod(m *protogen.Method) (*sysl.Endpoint, map[string]string) {
	syslCalls := []*sysl.Statement{}
	stringCalls := make(map[string]string)
	application := string(m.Desc.Parent().ParentFile().Package().Name())
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
	m.Desc.Parent().ParentFile().Package().Name()
	if t := m.Desc; t != nil && t.Name != nil {
		//fieldType = strings.ReplaceAll(string(t.Name()), p.PackageName, "")
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
