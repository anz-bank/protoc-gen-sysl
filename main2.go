// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The protoc-gen-go binary is a protoc plugin to generate Go code for
// both proto2 and proto3 versions of the protocol buffer language.
//
// For more information about the usage of this plugin, see:
//	https://developers.google.com/protocol-buffers/docs/reference/go-generated
package main

import (
	"flag"

	"github.com/anz-bank/protoc-gen-sysl/gensysl"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	//if len(os.Args) == 2 && os.Args[1] == "--version" {
	//	fmt.Fprintf(os.Stderr, "%v %v\n", filepath.Base(os.Args[0]))
	//	os.Exit(1)
	//}

	var (
		flags flag.FlagSet
	)
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(gensysl.GenerateFiles)
}
