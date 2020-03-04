all: install tests syslproto ci
.PHONY: install test tests syslproto

install:
	go install github.com/joshcarp/protoc-gen-sysl

test:
	protoc --sysl_out=tests/ tests/test.proto

# This updates the code_generator_request.pb.bin for debugging
generator:
	protoc --debug_out="tests/test:tests/." ./tests/test/test.proto
	protoc --debug_out="tests/simple:tests/." ./tests/simple/simple.proto

# This rebuilds the option protos
syslproto:
	protoc --go_out=. sysloption/sysloption.proto

ci:
	go test -v ./... -count=1
	golangci-lint run