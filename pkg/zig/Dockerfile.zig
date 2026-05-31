FROM --platform=$TARGETPLATFORM alpine:3.23
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG ZIG_VERSION=0.17.0-dev.263+0add2dfc4
ARG ZIG_ARCH=x86_64  # Define the architecture

RUN apk add --no-cache curl xz && \
    curl -L https://ziglang.org/builds/zig-${ZIG_ARCH}-linux-${ZIG_VERSION}.tar.xz \
    | tar -xJ -C /usr/local && \
    ln -s /usr/local/zig-${ZIG_ARCH}-linux-${ZIG_VERSION}/zig /usr/local/bin/zig

WORKDIR /app

# Verify Zig installation
RUN zig version