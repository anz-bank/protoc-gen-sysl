package names

import (
	"testing"

	"github.com/anz-bank/protoc-gen-sysl/tests"
)

func TestGenerateModule(t *testing.T) {
	t.Parallel()

	m := tests.GenerateModule(t, "code_generator_request.pb.bin")

	names := map[string][]string{
		"Org :: Team :: Project :: Foo":            {},
		"Org :: Team :: Project :: Types":          {"Request", "Response", "Nested", "Bar"},
		"Org :: Team :: Project :: Child :: Types": {"Child"},
		"google_protobuf":                          {"Timestamp"},
	}

	tests.AssertAppAndTypeNames(t, names, m)
}
