all: install test tests syslproto
.PHONY: install test tests syslproto

install:
	go install github.com/joshcarp/protoc-gen-sysl

test:
	protoc --sysl_out=. tests/serviceExample.proto

# This updates the code_generator_request.pb.bin for debugging
tests:
	protoc --debug_out="tests/.:tests/." \
      ./tests/serviceExample.proto

# This rebuilds the option protos
syslproto:
	protoc --go_out=. sysloption/sysloption.proto