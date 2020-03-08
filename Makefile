all: ci install syslproto
.PHONY: install test tests syslproto

install:
	go install github.com/anz-bank/protoc-gen-sysl

update-sysl:
	protoc --sysl_out=tests/simple/ tests/simple/simple.proto
	protoc --sysl_out=tests/test/ tests/test/test.proto
	protoc --sysl_out=tests/enum/ tests/enum/enum.proto
	protoc --sysl_out=tests/multiplefiles/ tests/multiplefiles/services.proto
	protoc --sysl_out=tests/otheroption/ tests/otheroption/otheroption.proto

# This updates the code_generator_request.pb.bin for debugging
update-tests:
	protoc --debug_out="tests/test:tests/." ./tests/test/test.proto
	protoc --debug_out="tests/simple:tests/." ./tests/simple/simple.proto
	protoc --debug_out="tests/multiplefiles:tests/." ./tests/multiplefiles/services.proto
	protoc --debug_out="tests/enum:tests/." ./tests/enum/enum.proto
	protoc --debug_out="tests/otheroption:tests/." ./tests/otheroption/otheroption.proto


# This rebuilds the option protos
syslproto:
	protoc --go_out=. sysloption/sysloption.proto

ci:
	go test -v ./... -count=1
	golangci-lint run