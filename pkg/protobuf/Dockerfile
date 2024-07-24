FROM golang:alpine as builder

RUN apk add --no-cache protobuf-dev git

ARG PROTOC_GEN_GO_VERSION=v1.34.2
ARG PROTOC_GEN_GO_GRPC_VERSION=v1.4.0
ARG BUF_VERSION=v1.34.0
ARG PROTOC_GEN_GOHTTP=v1.6.0
ARG PROTOC_GEN_GOTAG=v1.0.1


RUN go install -v google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION} \
  && go install -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION} \
  && go install -v github.com/bufbuild/buf/cmd/buf@${BUF_VERSION} \
  && go install -v github.com/nametake/protoc-gen-gohttp@${PROTOC_GEN_GOHTTP} \
	&& go install -v github.com/srikrsna/protoc-gen-gotag@${PROTOC_GEN_GOTAG} \
  && cp /go/bin/protoc-gen-go /usr/bin/ \
  && cp /go/bin/protoc-gen-go-grpc /usr/bin/ \
  && cp /go/bin/buf /usr/bin/ \
  && cp /go/bin/protoc-gen-gohttp /usr/bin/ \
  && cp /go/bin/protoc-gen-gotag /usr/bin/

FROM alpine

LABEL maintainer="containifyci"
LABEL version="$PROTOC_GEN_GO_VERSION"

RUN apk add --no-cache protobuf

COPY --from=builder /usr/bin/protoc-gen-go /usr/bin/protoc-gen-go
COPY --from=builder /usr/bin/protoc-gen-go-grpc /usr/bin/protoc-gen-go-grpc
COPY --from=builder /usr/bin/buf /usr/bin/buf
COPY --from=builder /usr/include/google /usr/include/google
COPY --from=builder /usr/bin/protoc-gen-gohttp /usr/bin/protoc-gen-gohttp
COPY --from=builder /usr/bin/protoc-gen-gotag /usr/bin/protoc-gen-gotag
