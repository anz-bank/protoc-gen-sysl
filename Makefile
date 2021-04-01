all: ci install
.PHONY: install test demo update-sysl update-tests

test: update-sysl update-tests
	go test -v ./... -count=1

update: install update-sysl update-tests ## Update tests and debug

help:			## Show this help.
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

%.pb.go: %.proto	## Generates .pb.go files from .proto files with protoc.
	protoc --go_out=. $<

install:	## Installs this project as a binary in your go binary directory.
	go install github.com/anz-bank/protoc-gen-sysl

update-sysl:		## Updates the expected sysl files by compiling with the current protoc-gen-sysl installation.
	protoc --sysl_out=tests/simple/ tests/simple/simple.proto
	protoc --sysl_out=tests/test/ tests/test/test.proto
	protoc --sysl_out=tests/enum/ tests/enum/enum.proto
	protoc --sysl_out=tests/multiplefiles/ tests/multiplefiles/services.proto
	protoc --sysl_out=tests/otheroption/ tests/otheroption/otheroption.proto
	protoc --sysl_out=tests/disconnectedimport/ tests/disconnectedimport/*.proto
	protoc --sysl_out=tests/date/ tests/date/*.proto
	protoc --sysl_out=tests/externaltype/ tests/externaltype/*.proto
	protoc --sysl_out=tests/empty/ tests/empty/*.proto
	protoc --sysl_out=tests/messageinmessage/ tests/messageinmessage/*.proto
	protoc --sysl_out=tests/repeated/ tests/repeated/*.proto
	protoc --sysl_out=tests/any/ tests/any/*.proto
	protoc --sysl_out=tests/hello/ tests/hello/*.proto
	protoc --sysl_out=tests/names/ tests/names/*.proto

update-tests:		## Updates the code_generator_request.pb.bin for the go test cases.
	protoc --debug_out="tests/test:tests/." ./tests/test/*.proto
	protoc --debug_out="tests/simple:tests/" ./tests/simple/simple.proto
	protoc --debug_out="tests/multiplefiles:tests/." ./tests/multiplefiles/services.proto
	protoc --debug_out="tests/enum:tests/." ./tests/enum/enum.proto
	protoc --debug_out="tests/otheroption:tests/." ./tests/otheroption/otheroption.proto
	protoc --debug_out="tests/disconnectedimport:tests/." ./tests/disconnectedimport/*.proto
	protoc --debug_out="tests/empty:tests/." ./tests/empty/*.proto
	protoc --debug_out="tests/date:tests/." ./tests/date/*.proto
	protoc --debug_out="tests/externaltype:tests/." ./tests/externaltype/*.proto
	protoc --debug_out="tests/messageinmessage:tests/." ./tests/messageinmessage/*.proto
	protoc --debug_out="tests/repeated:tests/." ./tests/repeated/*.proto
	protoc --debug_out="tests/any:tests/." ./tests/any/*.proto
	protoc --debug_out="tests/hello:tests/." ./tests/hello/*.proto
	protoc --debug_out="tests/names:tests/." ./tests/names/*.proto

tidy:			## Tidies up Go mod and source files.
	goimports -w $$(find . -type f -name '*.go' -not -name '*.pb.go')
	gofmt -s -w .
	go mod tidy

demo:			## Makes sure the demo directory still builds and compiles
	cd demo && make

ci: test		## Runs the same ci that is on master.
	golangci-lint run

grpc: sysl grpc

sysl: *.sysl	## Build sysl into GRPC
	sysl tmpl --template grpc.sysl --app-name hello --start start --outdir tests/hello tests/hello/index.sysl
	sysl tmpl --template grpc.sysl --app-name Hello --start start --outdir tests/hello2 tests/hello/index.sysl

# grpc: *		## Executes proto to generate go code
# 	protoc -I hello/ hello/hello.proto --go_out=plugins=grpc:hello

docker:			## Builds the Docker image.
	docker build -t protoc-gen-sysl .
