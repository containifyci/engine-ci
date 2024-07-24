# sonarcloud

There is a problem with podman to run [sonarsource/sonar-scanner-cli](https://hub.docker.com/r/sonarsource/sonar-scanner-cli) image. The image only provide for `amd64` architecture. This is a workaround to build the image for `arm64` and `amd64` architecture.

## build

This builds the sonar image for `amd64` and `arm64` architecture with podman.
Either run the commands below of if you like you can use [mask](https://github.com/jacobdeichert/mask) to do it for you.

```bash
mask build
```

> Build and Push sonar image with podman

```bash
set -xe
podman manifest create -a containifyci/sonar
podman build --platform linux/amd64,linux/arm64  --manifest containifyci/sonar .
podman manifest push containifyci/sonar
```
