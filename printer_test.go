package main

import (
	"bytes"
	"os"
	"testing"

	pgsgo "github.com/lyft/protoc-gen-star/lang/go"

	pgs "github.com/lyft/protoc-gen-star"
	"github.com/spf13/afero"
)

func TestModule(t *testing.T) {
	req, err := os.Open("./tests/code_generator_request.pb.bin")
	if err != nil {
		t.Fatal(err)
	}

	fs := afero.NewMemMapFs()
	res := &bytes.Buffer{}

	pgs.Init(
		pgs.ProtocInput(req),  // use the pre-generated request
		pgs.ProtocOutput(res), // capture CodeGeneratorResponse
		pgs.FileSystem(fs),    // capture any custom files written directly to disk
	).RegisterModule(
		SyslPrinter(),
	).RegisterPostProcessor(
		pgsgo.GoFmt(),
	).Render()
}

// check res and the fs for output
