# Rust Build Support for Engine-CI

## Overview

Engine-CI now supports building Rust projects using Cargo in containerized environments. The implementation follows the same pattern as Go and Zig builds, providing a consistent experience across languages.

## Features

- **Containerized Builds**: Uses `rust:1.83-alpine` base image for consistent build environments
- **Build Profiles**: Support for release and debug builds
- **Cross-compilation**: Target specification for different platforms
- **Cargo Features**: Enable specific cargo features during build
- **Caching**: Intelligent caching of Cargo dependencies via `CARGO_HOME`
- **Testing**: Automatic test execution as part of the build process
- **Production Images**: Create optimized production images with compiled binaries

## Usage

### Basic Build Configuration

To build a Rust project with engine-ci, configure your build with `BuildType: Rust`:

```go
build := container.Build{
    BuildType: container.Rust,
    App:       "my-app",
    Image:     "my-rust-app",
    ImageTag:  "latest",
    Folder:    ".",
}
```

### Build Profiles

Specify the build profile (defaults to "release"):

```go
build.Custom.Set("profile", "release")  // Release build (optimized)
build.Custom.Set("profile", "debug")    // Debug build (with debug symbols)
```

### Target Platform

Specify a target for cross-compilation:

```go
build.Custom.Set("target", "x86_64-unknown-linux-musl")
```

### Cargo Features

Enable specific cargo features:

```go
build.Custom.Set("features", "tokio,serde")
```

### Cache Configuration

The Rust build uses Cargo's cache to speed up subsequent builds. The cache location can be configured via environment variables:

- `CARGO_HOME`: Primary location for Cargo cache
- `CONTAINIFYCI_CACHE`: Fallback cache location

Default cache location (if not set): `/tmp/.cargo`

## Build Process

The Rust build process follows these steps:

1. **Pull Base Images**: Downloads the Alpine and Rust base images
2. **Build Rust Image**: Creates an intermediate image with Rust toolchain
3. **Build Binary**: Runs `cargo build` with specified options
4. **Run Tests**: Executes `cargo test` to validate the build
5. **Create Image**: Commits the container with the built binary
6. **Tag Image**: Tags the image with the specified name and tag

## Production Build

The production build (PostBuild step) creates an optimized final image:

1. Creates a minimal Alpine-based container
2. Copies only the compiled binary from `target/{profile}/`
3. Sets appropriate CMD and WORKDIR
4. Pushes to the specified registry

## Build Script Generation

The build script generator creates shell scripts for the containerized build:

### Release Build
```bash
#!/bin/sh
set -e
export CARGO_HOME=/root/.cargo
cargo build --color never --release
cargo test --color never --release
```

### Debug Build with Target
```bash
#!/bin/sh
set -e
export CARGO_HOME=/root/.cargo
cargo build --color never --target x86_64-unknown-linux-musl
cargo test --color never --target x86_64-unknown-linux-musl
```

## Project Structure Requirements

Your Rust project should have the standard Cargo structure:

```
my-rust-project/
├── Cargo.toml
├── Cargo.lock (optional)
└── src/
    └── main.rs (or lib.rs)
```

## Examples

### Simple CLI Application

```go
build := container.Build{
    BuildType: container.Rust,
    App:       "hello-cli",
    Image:     "myorg/hello-cli",
    ImageTag:  "v1.0.0",
    Folder:    ".",
}
build.Custom.Set("profile", "release")
```

### Web Service with Features

```go
build := container.Build{
    BuildType: container.Rust,
    App:       "web-service",
    Image:     "myorg/web-service",
    ImageTag:  "latest",
    Folder:    "./services/web",
}
build.Custom.Set("profile", "release")
build.Custom.Set("features", "server,tls")
```

### Cross-compiled Binary

```go
build := container.Build{
    BuildType: container.Rust,
    App:       "my-tool",
    Image:     "myorg/my-tool",
    ImageTag:  "latest",
    Folder:    ".",
}
build.Custom.Set("profile", "release")
build.Custom.Set("target", "x86_64-unknown-linux-musl")
```

## Integration with Build Pipeline

The Rust build steps are registered in the build pipeline:

- **Build Category**: `rust.New()` - Main build step
- **PostBuild Category**: `rust.NewProd()` - Production image creation

## Troubleshooting

### Cache Issues

If you encounter cache-related issues, clear the Cargo cache:

```bash
rm -rf /tmp/.cargo
# or
rm -rf $CARGO_HOME
```

### Build Failures

Check the build logs for cargo errors. Common issues:
- Missing dependencies in Cargo.toml
- Compilation errors in Rust code
- Test failures

### Binary Not Found

Ensure your `App` name matches the binary name in Cargo.toml:

```toml
[package]
name = "my-app"  # This should match the App field
```

## Implementation Details

The Rust support is implemented in `/pkg/rust/` following the established pattern:

- `rust.go`: Main implementation with container orchestration
- `buildscript.go`: Build script generation for cargo commands
- `Dockerfile.rust`: Base image definition with Rust toolchain
- `docker_metadata_gen.go`: Generated metadata from Dockerfile

## Future Enhancements

Potential future improvements:
- Auto-discovery of Rust projects via Cargo.toml detection
- Support for workspaces
- Custom cargo commands and flags
- Build time optimization hints
- Multi-stage build improvements
