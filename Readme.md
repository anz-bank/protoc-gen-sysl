# proto-gen-sysl
## Please see Issues for what isn't supported yet

## Features 
Generate sysl source code from .proto files

Supports using sysls "call" syntax through proto options


## Installation

`go get -u -v github.com/anz-bank/protoc-gen-sysl`

## Usage

`protoc --sysl_out=. input.proto`

or 
`protoc -I <import prefix> --sysl_out=import_prefix=<import prefix> input.proto` if you're using the -I flag with protoc

This will help determine where source locations are relative to where the protoc tool was run. s


## Examples

See demo directory for examples (including diagram generation)

## Intro

proto files are great; easily define your services and data structures and have them auto generated in any language

sysl tries to do this but also tries to do a couple extra things, including interactions between services.
Take the following example
  
``` 
Application:
    Endpoint:
        Foo <- thisEndpoint
        return string
```

Here we describe an Application with one Endpoint, and the `Foo <- thisEndpoint` specifies that this application calls a dependency.
This isn't supported in proto files, as proto files primarily are only for API specifications, not interactions of those APIs. 


Then once we call the proto tool:
`protoc --sysl_out=. example.proto`

we have our new sysl file:

```
Bar:
    AnotherEndpoint(input <: Types.Request):
        return Response
Foo:
    thisEndpoint(input <: Types.Request):
        return Response
Types:
    !type Request:
        query <: string
    !type Response:
        query <: string

```

## Importing manually written sysl
Once the generated sysl exists, one can write more sysl to specify the interactions:
manual.sysl:

```
import generated.sysl # Specify that we import our sysl file we just generated
Bar:
    AnotherEndpoint:
        Foo <- thisEndpoint # Here we specify that we call another service
```

Now you can generate sequence diagrams with `sysl sd -s "Bar <- AnotherEndpoint" manual.sysl` 
or use [https://github.com.anz-bank/sysl-catalog]() for building api catalogs

## Mapping from proto to sysl
proto|sysl|description|
|--|--|--|
package  grpc.testing;|@package="grpc_testing"||
message Request | grpc_testing: !type Request:...| types belong to an application the same name as the package|
string query = 1; | query <: string: <br>@rpcid="1", @json_tag="query"| |
service Foo| Foo: | The foo application
 rpc thisEndpoint(Request) returns(Request){};| thisEndpoint(req <: grpc_testing.Request)<br>returns ok <: grpc_testing.Response | 
 int64 | int | | 
 int32 | int | | 
 float<x>| float| | 