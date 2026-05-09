FROM --platform=$TARGETPLATFORM alpine:3.23
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG ZIG_VERSION=0.17.0-dev.263+0add2dfc4

##    curl -L https://ziglang.org/download/${ZIG_VERSION}/zig-${ZIG_ARCH}-linux-${ZIG_VERSION}.tar.xz \
RUN apk add --no-cache curl xz && \
    curl -L https://ziglang.org/builds/zig-x86_64-linux-${ZIG_VERSION}.tar.xz \
    | tar -xJ -C /usr/local && \
    ln -s /usr/local/zig-${ZIG_ARCH}-linux-${ZIG_VERSION}/zig /usr/local/bin/zig && \
    apk del curl xz && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# Verify Zig installation
RUN zig version
