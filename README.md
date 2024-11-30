# engine-ci

Welcome to the **engine-ci** project, a key component of the **containifyci** organization. **engine-ci** is a robust CI/CD pipeline engine designed to operate in a containerized environment. It supports both Docker and Podman as container runtimes and is implemented in Go.

## Getting Started

To start using **engine-ci**, you need to install the binary. The installation can be done via the following command:

```bash
go install github.com/containifyci/engine-ci@latest
```

Alternatively, you can download the pre-built binary from the [releases page](https://github.com/containifyci/engine-ci/releases).

## Usage

Once the binary is installed, initialize your project by creating a `.containifyci` directory with the necessary `containifyci.go` file:

```bash
engine-ci init
```

This command generates the `.containifyci` directory with the `containifyci.go` file, which is the core configuration for your pipelines.

To execute the pipeline defined in `containifyci.go`, use:

```bash
engine-ci run
```

## Example

For a practical example of how **engine-ci** is used, check out the [containifyci.go](./.containifyci/containifyci.go) file within this repository. It demonstrates how the **engine-ci** project is self-hosted using its own pipeline.

## Roadmap

### Completed Tasks:
- [x] **Podman Support**: Integrate with Podman through the [Podman bindings](https://github.com/containers/podman/tree/main/pkg/bindings).
- [x] **Pipeline Execution**: Explore alternatives to running pipelines, such as compiling the pipeline into a binary for execution with `go run -C .containifyci/containifyci.go build`.
- [x] **Pipeline Abstraction**: Simplify pipeline code by implementing a container pipeline abstraction layer to reduce redundancy across different languages like Go, Maven, Python, etc.

### Ongoing and Upcoming Tasks:
- [ ] **REST API Endpoint**: Develop and integrate a REST API endpoint, potentially implementing the first pipeline using Python (low priority).
- [ ] **NPM Pipeline**: Add support for pipelines targeting npm-based repositories (medium priority).
- [ ] **Podman Logging**: Improve progress logging functionality within Podman (high priority).
- [ ] **Multi-Architecture Docker Image**: Add Docker build support for the `sonar-scanner-cli` multi-architecture image (low priority).
- [x] **Golang Libraries Support**: Enable builds for Go libraries that do not include a `main` package (high priority).
- [x] **Golang Submodule Support**: Allow Go submodules to be built as part of the main module build (high priority).
- [x] **Container Image Push**: Provide an option to opt out of pushing container images (enabled by default).
- [x] **Goreleaser Integration**: Provide an option to opt out of using Goreleaser (enabled by default).
- [x] **Dependabot**: Update DependaBot configuration also to run daily (high priority).
- [ ] **Github Action**: Change release process to create draft a release and to publish it after all artifacts are uploaded by goreleaser.
  - https://goreleaser.com/customization/release/#github
- [ ] **Goreleaser Signed Apple Builds**: Add signing for Apple builds (high priority).
- [x] **Concurrent Build Support**: Implement concurrent build support (medium priority).
- [ ] **Add Go Quality Build Steps**: Add extra Go quality build steps to post test coverage and other reports to the Pull Request similar to the trivy report. [go-coverage-report](github.com/fgrosse/go-coverage-report/) (low priority).

## Contribution

We welcome contributions from the community! If you're interested in contributing to **engine-ci**, please create a fork and open a pull request with your changes.

We appreciate your contributions and look forward to your involvement in improving **engine-ci**!
