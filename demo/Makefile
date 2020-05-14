all: test diagrams

install:
	go install github.com/anz-bank/protoc-gen-sysl

test:
	protoc --sysl_out=. example.proto

diagrams:
	sysl sd -o output.png -s "Foo <- ThisEndpoint" example.sysl