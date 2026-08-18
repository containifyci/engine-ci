FROM pulumi/pulumi-go:3.258.0 AS pulumi

ARG PULUMI_VERSION
ARG GO_RUNTIME_VERSION=1.26.6
ENV GO_RUNTIME_AMD64_SHA256 708effb774be8237570d0add163225abbdfaf4fca28b2611df167beba4feef89
ENV GO_RUNTIME_ARM64_SHA256 d0507e9e9d7fe012aae570108cbd76c15de879e17130ab8cb90d4d7445cb1f2e

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

FROM debian:12-slim

RUN apt-get update -y && \
    apt-get install -y \
    ca-certificates

COPY --from=pulumi /golang/go/bin /golang/go/bin
COPY --from=pulumi /pulumi/bin/pulumi /pulumi/bin/pulumi
COPY --from=pulumi /pulumi/bin/pulumi-language-go /pulumi/bin/pulumi-language-go

ENV PATH="$PATH:/pulumi/bin/:/golang/go/bin"
ENV GOROOT="/golang/go"

RUN go version; \
    printf "pulumi version "; \
    pulumi version

CMD ["pulumi"]