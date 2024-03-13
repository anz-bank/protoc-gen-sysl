module github.com/anz-bank/protoc-gen-sysl

go 1.16

replace github.com/spf13/afero => github.com/anz-bank/afero v1.2.4

require (
	github.com/anz-bank/sysl v0.453.0
	github.com/golang/protobuf v1.5.2
	github.com/spf13/afero v1.4.0
	github.com/stretchr/testify v1.7.0
	google.golang.org/protobuf v1.33.0
)
