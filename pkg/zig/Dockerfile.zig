FROM alpine:3.24
ARG ZIG_VERSION=0.17.0-dev.1609+11e2bb391

RUN apk add --no-cache curl xz && \
    ZIG_ARCH=$(uname -m | sed 's/arm64/aarch64/' | sed 's/amd64/x86_64/') && \
    curl -L https://ziglang.org/builds/zig-${ZIG_ARCH}-linux-${ZIG_VERSION}.tar.xz \
    | tar -xJ -C /usr/local && \
    ln -s /usr/local/zig-${ZIG_ARCH}-linux-${ZIG_VERSION}/zig /usr/local/bin/zig

WORKDIR /app

# Verify Zig installation
RUN zig version