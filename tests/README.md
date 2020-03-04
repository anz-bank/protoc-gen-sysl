## Test Directory

Each folder holds 3 files:
- A proto file that is used to generate the `CodeGeneratorRequest`
- The expected correct "golden" sysl file
- The code_generator_request.pb.bin that's generated from `make generator` [using lyfts proto-gen-debug tool](https://github.com/lyft/protoc-gen-star/blob/master/protoc-gen-debug)