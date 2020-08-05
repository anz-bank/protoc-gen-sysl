FROM golang:1.14-buster as builder
ADD . /src
RUN cd /src && go build -o protoc-gen-sysl

FROM golang:1.14-buster
RUN apt-get update && apt-get install -y \
    git \
    unzip \
    build-essential \
    autoconf \
    libtool \
    curl \
    bash \
    make \
    && rm -rf /var/lib/apt/lists/*
ENV PROTOC_VERSION="3.12.4"
RUN curl -L -O https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
RUN unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local/

WORKDIR /
COPY --from=builder /src/protoc-gen-sysl /bin/
RUN go get -u -v \
    github.com/envoyproxy/protoc-gen-validate \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
    github.com/golang/protobuf/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc

ENTRYPOINT [ "protoc" ]
