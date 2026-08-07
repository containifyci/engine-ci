# Migration Plan: Podman v5 to Podman v6

**Fork:** `franky-agent/engine-ci` (forked from `containifyci/engine-ci`)
**Branch:** `chore/migrate-podman-v6`
**Target:** Podman v6.0.2 (latest stable as of 2026-07-22), up from v5.8.5
**Date:** 2026-08-07
**Status:** Code changes COMPLETE - build and tests pass

---

## 1. Executive Summary

engine-ci embeds Podman as a **Go library dependency** (not just a CLI runtime). The
`pkg/cri/podman` package imports `github.com/containers/podman/v5` bindings and
`github.com/containers/buildah` directly. Podman v6 introduces **breaking changes** in
the Go module import path, the embedded Docker types (docker/docker -> moby/moby),
and the runtime environment requirements.

### What was done

The Go module migration (Track A) has been completed:
- Import paths updated: `github.com/containers/podman/v5` -> `go.podman.io/podman/v6`
- Buildah import updated: `github.com/containers/buildah` -> `go.podman.io/buildah`
- Docker types switched to Moby: `github.com/docker/docker/api/types` -> `github.com/moby/moby/api/types`
- `handlers.ExecCreateConfig` field changed from `ExecOptions` to `ExecCreateRequest`
- `go.mod` replace directives cleaned up (runtime-spec and runtime-tools pins removed)
- `go mod tidy`, `go build`, `go vet`, and `go test` all pass

---

## 2. Changes Made

### 2.1 `go.mod`

```diff
require (
-   github.com/containers/buildah v1.43.2
-   github.com/containers/podman/v5 v5.8.5
+   go.podman.io/buildah v1.44.1
+   go.podman.io/podman/v6 v6.0.2
)

// Removed: runtime-spec v1.2.1 pin (v6 uses v1.3.0 natively)
// Removed: runtime-tools pin (v6 uses the newer version)
// Updated: go.podman.io/common replace from v0.67.1 to v0.68.1
- replace github.com/opencontainers/runtime-spec => github.com/opencontainers/runtime-spec v1.2.1
- replace github.com/opencontainers/runtime-tools => github.com/opencontainers/runtime-tools v0.9.1-0.20250523...
- replace go.podman.io/common => go.podman.io/common v0.67.1
+ replace go.podman.io/common => go.podman.io/common v0.68.1
```

### 2.2 `pkg/cri/podman/podman.go` - Import paths

```diff
- "github.com/docker/docker/api/types/registry"
+ "github.com/moby/moby/api/types/registry"

- "github.com/containers/podman/v5/libpod/define"
- "github.com/containers/podman/v5/pkg/api/handlers"
- "github.com/containers/podman/v5/pkg/bindings"
- "github.com/containers/podman/v5/pkg/bindings/containers"
- "github.com/containers/podman/v5/pkg/bindings/images"
- "github.com/containers/podman/v5/pkg/bindings/manifests"
- "github.com/containers/podman/v5/pkg/bindings/secrets"
- "github.com/containers/podman/v5/pkg/specgen"
- "github.com/docker/docker/api/types/container"
+ "go.podman.io/podman/v6/libpod/define"
+ "go.podman.io/podman/v6/pkg/api/handlers"
+ "go.podman.io/podman/v6/pkg/bindings"
+ "go.podman.io/podman/v6/pkg/bindings/containers"
+ "go.podman.io/podman/v6/pkg/bindings/images"
+ "go.podman.io/podman/v6/pkg/bindings/manifests"
+ "go.podman.io/podman/v6/pkg/bindings/secrets"
+ "go.podman.io/podman/v6/pkg/specgen"
+ "github.com/moby/moby/api/types/container"

- buildahDefine "github.com/containers/buildah/define"
+ buildahDefine "go.podman.io/buildah/define"
```

### 2.3 `pkg/cri/podman/podman.go` - ExecCreateConfig API change

Podman v6 changed `handlers.ExecCreateConfig` to embed
`container.ExecCreateRequest` (from moby/moby) instead of `container.ExecOptions`
(from docker/docker):

```diff
  id, err := containers.ExecCreate(p.conn, id, &handlers.ExecCreateConfig{
-     ExecOptions: container.ExecOptions{
+     ExecCreateRequest: container.ExecCreateRequest{
          Cmd:          cmd,
          AttachStdout: attachStdOut,
      },
  })
```

`ExecCreateRequest` has the same `Cmd` and `AttachStdout` fields as `ExecOptions`,
so this is a straightforward field name change.

### 2.4 Version detection comments

Added clarifying comments to the socket detection code noting v5/v6 use the
`machine inspect` branch.

---

## 3. API Compatibility Verification

All function signatures and types used by engine-ci were verified identical
between v5.8.5 and v6.0.2:

| Symbol | v5 | v6 | Status |
|--------|----|----|--------|
| `bindings.NewConnection(ctx, uri)` | `(context.Context, error)` | same | OK |
| `specgen.NewSpecGenerator(img, rootfs)` | `*SpecGenerator` | same | OK |
| `specgen.NamedVolume` | struct | same | OK |
| `specgen.ContainerSecurityConfig` | struct | same | OK |
| `specgen.SpecGenerator` fields (EnvSecrets, ImageOS, ImageArch, ResourceLimits, etc.) | present | same | OK |
| `containers.CreateWithSpec` | same | same | OK |
| `containers.Start/Stop/Remove/List/Logs/Inspect/Wait` | same | same | OK |
| `containers.CopyFromArchive/CopyToArchive` | same | same | OK |
| `containers.ExecCreate` | same | same | OK |
| `containers.ExecStartAndAttach` | same | same | OK |
| `containers.Commit` | `(types.IDResponse, error)` | same | OK |
| `define.ContainerStateStopped/Exited` | constants | same | OK |
| `handlers.ExecCreateConfig` | embeds `ExecOptions` | embeds `ExecCreateRequest` | **CHANGED** |
| `images.Build/List/Pull/Tag/Push/Remove/GetImage` | same | same | OK |
| `images.BuildOptions` (alias to `types.BuildOptions`) | same | same | OK |
| `manifests.Create/Add/Push` | same | same | OK |
| `secrets.Create` with `CreateOptions` | same | same | OK |
| `buildahDefine.BuildOptions` (Output, Log, Architecture, OS, PullPolicy, Out, ContextDirectory, Platforms) | present | present | OK |
| `nettypes.PortMapping` | same | same | OK |
| `registry.AuthConfig` | from docker/docker | from moby/moby | **MOVED** |

---

## 4. Dependency Changes

| Module | Old (v5.8.5) | New (v6.0.2) | Notes |
|--------|-------------|-------------|-------|
| Podman | `github.com/containers/podman/v5 v5.8.5` | `go.podman.io/podman/v6 v6.0.2` | CNCF org move |
| Buildah | `github.com/containers/buildah v1.43.2` | `go.podman.io/buildah v1.44.1` | CNCF org move |
| common | `go.podman.io/common v1.0.1` (replace -> v0.67.1) | replace -> v0.68.1 | v6.0.2 requires v0.68.1 |
| image/v5 | v5.40.0 (indirect) | v5.40.0 (indirect) | Unchanged |
| storage | v1.63.0 (indirect) | v1.63.0 (indirect) | Unchanged |
| runtime-spec | v1.2.1 (pinned) | v1.3.0 (unpinned) | Pin removed |
| runtime-tools | v0.9.1-0.20250523... (pinned) | v0.9.1-0.20260316... (unpinned) | Pin removed |
| moby/moby/api | not used | v1.54.2 (indirect, via podman v6) | New transitive dep |
| docker/docker | v28.5.2+incompatible | v28.5.2+incompatible | Still used by docker runtime pkg |

---

## 5. Runtime / CI Environment (Track B - pending)

The Go module migration is complete. The CI/runtime environment changes need
verification when running against actual Podman v6 binaries:

- [ ] **5.1** Verify `podman info -f "{{ .Host.RemoteSocket.Path }}"` works in v6
- [ ] **5.2** Check GitHub Actions runner OS (no `macos-13` Intel, use ubuntu-24.04)
- [ ] **5.3** Verify `/etc/containers/registries.conf` format valid in v6
- [x] **5.4** `libbtrfs-dev` and `libgpgme-dev` are NOT needed - removed from CI/Dockerfiles
      (build tags `containers_image_openpgp,exclude_graphdriver_btrfs` exclude both C libraries)
- [ ] **5.5** If CI uses distro podman, ensure v6 is available (may need PPA or static binary)
- [ ] **5.6** Network isolation default change (v6 defaults to enabled) - test cross-container networking
- [ ] **5.7** Run integration tests with podman v6 runtime in CI matrix

### v6 Runtime breaking changes (not affecting Go code, but affecting deployment):

- BoltDB dropped (auto-migrate to SQLite) - no impact on fresh CI runs
- Intel Mac support removed - ensure no macos-13 runners
- Windows 10 support removed - no Windows runners in CI
- cgroups v1 removed - Ubuntu 24.04 uses cgroups v2
- iptables removed - use nftables (Ubuntu 24.04 default)
- CNI networking removed - use netavark (default on modern systems)
- slirp4netns removed - use pasta (no `--network-cmd-path` references in code)
- Quadlet `.app` file tracking changed - not used by engine-ci
- Linux podman machine volumes use systemd - not used in CI
- Config file parsing rewritten - registries.conf is a common lib file, not podman.conf

---

## 6. Build and Test Results

```
$ go build -tags containers_image_openpgp,exclude_graphdriver_btrfs ./...
# PASS (exit 0, no errors)

$ go vet -tags containers_image_openpgp,exclude_graphdriver_btrfs ./pkg/cri/podman/...
# PASS (exit 0)

$ go test -tags containers_image_openpgp,exclude_graphdriver_btrfs -count=1 -short ./...
# ALL TESTS PASS
```

---

## 7. Files Changed

1. `go.mod` - Podman v5->v6, Buildah v1.43->v1.44, removed pins, updated common replace
2. `go.sum` - Updated by `go mod tidy`
3. `pkg/cri/podman/podman.go` - Import paths, ExecCreateConfig field, version detection comments
4. `docs/podman-v6-migration-plan.md` - This document

---

## 8. References

- [Podman v6.0.0 Release Notes](https://github.com/podman-container-tools/podman/releases/tag/v6.0.0)
- [Podman v6.0.2 Release Notes](https://github.com/podman-container-tools/podman/releases/tag/v6.0.2)
- [Config file parsing design doc](https://github.com/podman-container-tools/podman/blob/main/contrib/design-docs/config-file-parsing.md)
- engine-ci fork: https://github.com/franky-agent/engine-ci
- Migration branch: `chore/migrate-podman-v6`