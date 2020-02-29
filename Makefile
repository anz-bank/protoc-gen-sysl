all: install test
.PHONY: build install test tests syslproto
build:
	go build -o=protoc-gen-example .

install:
	go install github.com/joshcarp/protoc-gen-sysl
test:
	protoc --example_out=. tests/serviceExample.proto

tests:
	protoc \
      --plugin=debug_out=. \
      --debug_out=".:." \
      ./tests/serviceExample.proto

syslproto:
	protoc --go_out=. sysloption/sysloption.proto