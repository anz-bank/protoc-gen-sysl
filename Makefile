all: ci install syslproto
.PHONY: install test tests demo update-sysl update-tests


update: install update-sysl update-tests ## Update tests and debug

help:			## Show this help.
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

install:		## Installs this project as a binary in your go binary directory.
	go install github.com/anz-bank/protoc-gen-sysl



update-sysl:		## Updates the expected sysl files by compiling with the current protoc-gen-sysl installation.
	protoc --sysl_out=tests/simple/ tests/simple/simple.proto
	protoc --sysl_out=tests/test/ tests/test/test.proto
	protoc --sysl_out=tests/enum/ tests/enum/enum.proto
	protoc --sysl_out=tests/multiplefiles/ tests/multiplefiles/services.proto
	protoc --sysl_out=tests/otheroption/ tests/otheroption/otheroption.proto
	protoc --sysl_out=tests/disconnectedimport/ tests/disconnectedimport/*.proto
	protoc --sysl_out=tests/empty/ tests/empty/*.proto

update-tests:		## Updates the code_generator_request.pb.bin for the go test cases.
	protoc --debug_out="tests/test:tests/." ./tests/test/test.proto
	protoc --debug_out="tests/simple:tests/." ./tests/simple/simple.proto
	protoc --debug_out="tests/multiplefiles:tests/." ./tests/multiplefiles/services.proto
	protoc --debug_out="tests/enum:tests/." ./tests/enum/enum.proto
	protoc --debug_out="tests/otheroption:tests/." ./tests/otheroption/otheroption.proto
	protoc --debug_out="tests/disconnectedimport:tests/." ./tests/disconnectedimport/*.proto
	protoc --debug_out="tests/empty:tests/." ./tests/empty/*.proto



validate:
	sysl validate

syslproto:		## Rebuilds the `option protos` to go and keeps the demo directory in sync
	protoc --go_out=. sysloption/sysloption.proto
	rm demo/sysloption.proto && cp sysloption/sysloption.proto demo/

demo:			## Makes sure the demo directory still builds and compiles
	cd demo && make

ci:				## Runs the same ci that is on master.
	go test -v ./... -count=1
	golangci-lint run
