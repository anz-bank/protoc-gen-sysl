package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/joshcarp/gop/gop/retriever/retriever_local"
	"github.com/stretchr/testify/require"

	"github.com/anz-bank/sysl/pkg/parse"
	"github.com/anz-bank/sysl/pkg/syslutil"
	"github.com/spf13/afero"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/anz-bank/protoc-gen-sysl/proto2sysl"

	"google.golang.org/protobuf/proto"

	"github.com/stretchr/testify/assert"
)

var tests = []string{
	"test/",
	"multiplefiles/",
	"date/",
	"any/",
	"hello/",
	"externaltype/",
	"disconnectedimport/",
	"any/",
	"simple/",
	"empty/",
	"repeated/",
	"messageinmessage/",
	"test",
	"otheroption/",
	"enum/",
	"names/",
}

const testDir = "./tests"

func TestPrinting(t *testing.T) {
	t.Parallel()

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(testDir, test)
			_, fs := syslutil.WriteToMemOverlayFs(dir)

			GeneratorResponse, err := ConvertSyslToProto(filepath.Join(dir, "code_generator_request.pb.bin"))
			require.NoError(t, err)

			file := GeneratorResponse.File[0]
			golden, err := afero.ReadFile(fs, *file.Name)
			require.NoError(t, err)

			assert.Equal(t, *file.Content, string(golden), *file.Content)

			_, err = parse.NewParser().Parse(*file.Name, retriever_local.New(fs))
			require.NoError(t, err)
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
		_     = flags.String("tests/test", "", "prefix to prepend to import paths")
		res   bytes.Buffer
	)
	if err := run(protogen.Options{ParamFunc: flags.Set}, req, &res, proto2sysl.GenerateFiles); err != nil {
		return nil, err
	}
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
	this := ""
	req.Parameter = &this
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
