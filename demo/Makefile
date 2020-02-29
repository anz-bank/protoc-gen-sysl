all: install test diagrams

install:
	go install github.com/joshcarp/protoc-gen-sysl

test:
	protoc --sysl_out=. example.proto

diagrams:
	sysl sd -o output.png -s "Foo <- thisEndpoint" example.sysl