FROM golang:alpine AS builder
ADD . /src
RUN cd /src && go build -o protoc-gen-sysl

FROM alpine:latest
COPY --from=builder /src/protoc-gen-sysl /bin
RUN apk add protoc
RUN apk add --no-cache --upgrade bash
ENTRYPOINT [ "protoc", "--sysl_out" ]