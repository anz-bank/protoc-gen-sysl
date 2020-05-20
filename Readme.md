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

## Example
Example workflow:

```
+---------------------+ imported  +------------------+
| generated sysl file |---------->| manual sysl file |---------> sysl toolchain (sequence diagrams/sysl-catalog)
+---------------------+           +------------------+
     ^                                     ^ imported
     |    protoc-gen-sysl                  |
     |                            +----------------------+
  +--+-----------+                |other manual/generated|
  | .proto file  |                |sysl files            |
  +--------------+                +----------------------+

```

Where the manual sysl file can "redefine" applications with the `<-` syntax and other sysl specifics before being used in the sysl toolchain.


- Given the proto file:
```
syntax = "proto3";

package grpc.testing;

message Request {
    string query = 1;
}

message Response {
    string query = 1;
}

service Foo{
    rpc thisEndpoint(Request) returns(Response);
}

service Bar{
    rpc AnotherEndpoint(Request) returns(Response);
}

```

We can convert this to sysl using:
`protoc --sysl_out=. example.proto`

we have our new sysl file:

```
Bar:
    @package="grpc_testing"
    AnotherEndpoint(input <: grpc_testing.Request):
        return Response
Foo:
    @package="grpc_testing"
    thisEndpoint(input <: grpc_testing.Request):
        return Response
grpc_testing:
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
`package  grpc.testing;`|`@package="grpc_testing"`|package attributes will be attached to any applications|
`message Request` | `grpc_testing:`<br>` !type Request:...`| types belong to an application the same name as the package|
`string query = 1;` | `query <: string:` <br>`    @rpcid="1" @json_tag="query"`||
`service Foo`| `Foo:`<br>`    @package="grpc_testing"` | The foo application
`rpc thisEndpoint(Request) returns(Request){}`|`thisEndpoint(req <: grpc_testing.Request)[~grpc, ~GRPC]:`<br>`    returns ok <: grpc_testing.Response`| Individual endpoints will have the ~grpc + ~GRPC <br>patterns to differentiate from any other endpoints in sysl|
`int64` | `int` ||
`int32` | `int` ||
`float<x>`| `float`||
`message` | `!type` ||
`enum` | `!enum`||
`repeated type`| `sequence of type`||
`message foo{`<br>`message bar`| `!type foo_bar`| Messages defined in messages will have names<br> in the format outername_innername