all: install test
.PHONY: build install test tests
build:
	go build -o=protoc-gen-example .

install:
	go install github.com/joshcarp/protoc-gen-example
test:
	protoc --example_out=. tests/serviceExample.proto

tests:
	protoc \
      --plugin=debug_out=. \
      --debug_out=".:." \
      ./tests/serviceExample.proto
