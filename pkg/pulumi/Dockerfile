FROM pulumi/pulumi-go AS pulumi

ARG PULUMI_VERSION
ARG GO_RUNTIME_VERSION=1.23.1
ENV GO_RUNTIME_AMD64_SHA256 49bbb517cfa9eee677e1e7897f7cf9cfdbcf49e05f61984a2789136de359f9bd
ENV GO_RUNTIME_ARM64_SHA256 faec7f7f8ae53fda0f3d408f52182d942cc89ef5b7d3d9f23ff117437d4b2d2f

WORKDIR /golang
RUN apt-get update -y && \
    apt-get install -y \
    curl \
    tar \
    git
    
    # build-essential \
RUN case $(uname -m) in \
    aarch64) \
    ARCH="arm64" \
    GO_RUNTIME_SHA256="${GO_RUNTIME_ARM64_SHA256}" \
    ;; \
    x86_64) \
    ARCH="amd64" \
    GO_RUNTIME_SHA256="${GO_RUNTIME_AMD64_SHA256}" \
    ;; \
    esac && \
    curl -fsSLo /tmp/go.tgz https://golang.org/dl/go${GO_RUNTIME_VERSION}.linux-${ARCH}.tar.gz && \
    mkdir -p bin; \
    tar -C /golang -xzf /tmp/go.tgz; \
    export PATH="/golang/go/bin:${PATH}"; \
    go version; \
    echo "${GO_RUNTIME_SHA256} /tmp/go.tgz" | sha256sum -c -

FROM debian:11-slim

RUN apt-get update -y && \
    apt-get install -y \
    ca-certificates

COPY --from=pulumi /golang/go/bin /golang/go/bin
COPY --from=pulumi /pulumi/bin/pulumi /pulumi/bin/pulumi
COPY --from=pulumi /pulumi/bin/pulumi-language-go /pulumi/bin/pulumi-language-go
COPY --from=pulumi /pulumi/bin/pulumi-analyzer-policy /pulumi/bin/pulumi-analyzer-policy

ENV PATH="$PATH:/pulumi/bin/:/golang/go/bin"
ENV GOROOT="/golang/go"

RUN go version; \
    printf "pulumi version "; \
    pulumi version

CMD ["pulumi"]