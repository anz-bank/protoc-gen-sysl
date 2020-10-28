package tests

import (
	"io/ioutil"
	"testing"

	"github.com/anz-bank/protoc-gen-sysl/proto2sysl"
	"github.com/anz-bank/sysl/pkg/sysl"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
)

func AssertAppAndTypeNames(t *testing.T, names map[string][]string, m *sysl.Module) {
	assert.Len(t, m.Apps, len(names))
	for name, types := range names {
		app, ok := m.Apps[name]
		if !ok {
			var exist []string
			for e := range m.Apps {
				exist = append(exist, e)
			}
			assert.True(t, ok, "no app named %s in %v", name, exist)
		}
		AssertTypesHaveNames(t, types, app.Types)
	}
}

func AssertTypesHaveNames(t *testing.T, names []string, types map[string]*sysl.Type) {
	assert.Len(t, types, len(names))
	for _, expected := range names {
		found := false
		for name := range types {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			assert.Fail(t, "app does not have expected types",
				"missing expected type %s in %v", expected, types)
		}
	}
}

// generateModule returns a Sysl Module generated from the given code_generator_request.pb.bin.
func GenerateModule(t *testing.T, requestPath string) *sysl.Module {
	bs, err := ioutil.ReadFile(requestPath)
	require.NoError(t, err)

	var req plugin.CodeGeneratorRequest
	err = proto.Unmarshal(bs, &req)
	require.NoError(t, err)

	pi, err := protogen.Options{}.New(&req)
	require.NoError(t, err)

	m, err := proto2sysl.GenerateModule(pi)
	require.NoError(t, err)

	return m
}
