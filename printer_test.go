package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	plugin_go "github.com/gogo/protobuf/protoc-gen-gogo/plugin"

	"github.com/alecthomas/assert"
	"github.com/anz-bank/sysl/pkg/syslutil"

	pgs "github.com/lyft/protoc-gen-star"
	"github.com/spf13/afero"
)

var tests = []string{
	"empty/",
	"any/",
	"repeated/",
	"simple/",
	"messageinmessage/",
	"externaltype/",
	"test",
	"multiplefiles/",
	"otheroption/",
	"enum/",
	"disconnectedimport/",
	"date/",
}

const testDir = "./tests"

func TestPrinting(t *testing.T) {
	for _, test := range tests {
		test = filepath.Join(testDir, test)
		_, fs := syslutil.WriteToMemOverlayFs(test)
		GeneratorResponse, err := ConvertSyslToProto(filepath.Join(test, "code_generator_request.pb.bin"))

		t.Run(test, func(t *testing.T) {
			assert.NoError(t, err)
			golden, err := afero.ReadFile(fs, *GeneratorResponse.File[0].Name)
			assert.NoError(t, err)
			assert.Equal(t, *GeneratorResponse.File[0].Content, string(golden))
			t.Log(filepath.Join("Passed", test, *GeneratorResponse.File[0].Name))
		})
	}
}

// ConvertSyslToProto opens a sysl filename and returns the CodeGeneratorResponse for the test cases.
func ConvertSyslToProto(filename string) (*plugin_go.CodeGeneratorResponse, error) {
	req, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	res := &bytes.Buffer{}
	pgs.Init(
		pgs.ProtocInput(req),  // use the pre-generated request
		pgs.ProtocOutput(res), // capture CodeGeneratorResponse
	).RegisterModule(
		SyslPrinter(),
	).Render()
	response := plugin_go.CodeGeneratorResponse{}
	err = proto.Unmarshal(res.Bytes(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil

}
