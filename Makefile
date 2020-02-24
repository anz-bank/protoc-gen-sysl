all: install test
.PHONY: build install test
build:
	go build -o=protoc-gen-example .

install:
	go install github.com/joshcarp/protoc-gen-example
test:
	protoc --example_out=. tests/serviceExample.proto