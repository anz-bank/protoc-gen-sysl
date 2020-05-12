package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/anz-bank/sysl/pkg/syslutil"
	"github.com/spf13/afero"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/anz-bank/protoc-gen-sysl/gensysl"

	"google.golang.org/protobuf/proto"

	"github.com/alecthomas/assert"
)

var tests = []string{
	"simple/",
	"empty/",
	"any/",
	"repeated/",
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
			if *GeneratorResponse.File[0].Content != string(golden) {
				fmt.Println(*GeneratorResponse.File[0].Content)
			}
			assert.Equal(t, *GeneratorResponse.File[0].Content, string(golden))
		})
	}
}

// ConvertSyslToProto opens a sysl filename and returns the CodeGeneratorResponse for the test cases.
func ConvertSyslToProto(filename string) (*pluginpb.CodeGeneratorResponse, error) {
	req, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	var (
		flags flag.FlagSet
	)
	res := &bytes.Buffer{}
	run(protogen.Options{ParamFunc: flags.Set}, req, res, gensysl.GenerateFiles)
	response := &pluginpb.CodeGeneratorResponse{}
	err = proto.Unmarshal(res.Bytes(), response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func run(opts protogen.Options, input io.Reader, output io.Writer, f func(*protogen.Plugin) error) error {
	in, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(in, req); err != nil {
		return err
	}
	replace := ""
	req.Parameter = &replace
	gen, err := opts.New(req)
	if err != nil {
		return err
	}
	if err := f(gen); err != nil {
		// Errors from the plugin function are reported by setting the
		// error field in the CodeGeneratorResponse.
		//
		// In contrast, errors that indicate a problem in protoc
		// itself (unparsable input, I/O errors, etc.) are reported
		// to stderr.
		gen.Error(err)
	}
	resp := gen.Response()
	out, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := output.Write(out); err != nil {
		return err
	}
	return nil
}
