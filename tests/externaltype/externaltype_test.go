package names

import (
	"testing"

	"github.com/anz-bank/protoc-gen-sysl/tests"
)

func TestGenerateModule(t *testing.T) {
	t.Parallel()

	m := tests.GenerateModule(t, "code_generator_request.pb.bin")

	names := map[string][]string{
		"Bar":                        {},
		"Car":                        {},
		"testing_externaltype":       {"date_", "this", "That", "repeatedDate", "foo", "Sibling"},
		"testing_externaltype_child": {"Child"},
		"google_protobuf":            {"Empty", "Timestamp"},
	}

	tests.AssertAppAndTypeNames(t, names, m)
}
