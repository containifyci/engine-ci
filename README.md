# engine-ci

Tool that build containerized applications based on Pipeline implemented in golang.

## Todo:

- [x] work also with podman https://github.com/containers/podman/tree/main/pkg/bindings
- [x] to find an easier way to run the pipelines then maybe also consider to compile the pipeline into a binary and run that instead.
      go run -C .containifyci/containifyci.go build
- [ ] There is still a lot of code duplication for all the different pipelines like golang, maven, python, … . Implement a easier to use container pipeline abstraction to reduce the code for providing new pipelines.
- [ ] Working on the rest api endpoint and maybe implement a first Pipeline in
- [ ] python by using this rest api endpoint (similar to Dagger Engine)
- [ ] Implement the next Pipeline for npm based repositories.
- [ ] Fixing progress logging in podman
- [ ] Add docker build for sonar-scanner-cli multi architecture image see hack/soanrcloud folder
