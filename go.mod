module github.com/anz-bank/protoc-gen-sysl

go 1.13

replace github.com/spf13/afero => github.com/anz-bank/afero v1.2.4

require (
	github.com/anz-bank/sysl v0.342.0
	github.com/golang/protobuf v1.5.1
	github.com/joshcarp/gop v0.0.0-20200922043230-a225272c1746
	github.com/spf13/afero v1.4.0
	github.com/stretchr/testify v1.6.1
	google.golang.org/protobuf v1.26.0
)
