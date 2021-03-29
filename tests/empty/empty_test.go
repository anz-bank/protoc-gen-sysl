package empty

import (
	"testing"

	"github.com/anz-bank/protoc-gen-sysl/tests"
)

func TestGenerateModule(t *testing.T) {
	t.Parallel()

	m := tests.GenerateModule(t, "code_generator_request.pb.bin")

	names := map[string][]string{
		"Bar":          {},
		"grpc_testing": {"Response"},
	}

	tests.AssertAppAndTypeNames(t, names, m)
}
