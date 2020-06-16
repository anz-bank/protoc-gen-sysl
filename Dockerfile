FROM golang:1.14-buster as builder
ADD . /src
RUN cd /src && go build -o protoc-gen-sysl

FROM golang:1.14-buster
RUN apt-get update && apt-get  install -y \
    git \
    unzip \
    build-essential \
    autoconf \
    libtool \
    curl \
    && rm -rf /var/lib/apt/lists/*
ENV PROTOC_VERSION="3.11.2"
RUN curl -L -O https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
RUN apt-get update && apt-get install --assume-yes bash
RUN unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local/

WORKDIR /
COPY --from=builder /src/protoc-gen-sysl /bin/
RUN go get -u -v github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
RUN go get -u -v github.com/envoyproxy/protoc-gen-validate
RUN go get -u -v github.com/anz-bank/protoc-gen-sysl

ENTRYPOINT [ "protoc", "--sysl-out" ]
