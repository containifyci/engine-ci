# Migration Plan: Podman v5 → Podman v6

**Fork:** `franky-agent/engine-ci` (forked from `containifyci/engine-ci`)
**Branch:** `chore/migrate-podman-v6`
**Target:** Podman v6.0.2 (latest stable as of 2026-07-22), up from v5.8.5
**Date:** 2026-08-07

---

## 1. Executive Summary

engine-ci embeds Podman as a **Go library dependency** (not just a CLI runtime). The
`pkg/cri/podman` package imports `github.com/containers/podman/v5` bindings and
`github.com/containers/buildah` directly. Podman v6 introduces **breaking changes** in
the Go module import path, minimum dependency versions, the REST API bindings, and
the runtime environment requirements.

This migration has two independent tracks:

| Track | Scope | Risk |
|-------|-------|------|
| **A. Go module / library** | `go.mod`, `pkg/cri/podman/podman.go`, transitive deps | High — import path + API changes |
| **B. Runtime / CI environment** | GitHub Actions workflows, Dockerfiles, socket detection | Medium — runtime requirements |

---

## 2. Current State (v5.8.5)

### 2.1 Go Module Dependencies (`go.mod`)

```
require (
    github.com/containers/buildah v1.43.2          # ← v6 requires v1.44.0+
    github.com/containers/podman/v5 v5.8.5         # ← import path changes in v6
    go.podman.io/common v1.0.1                      # ← already on new org path
)

require (indirect)
    go.podman.io/image/v5 v5.40.0                  # ← v6 needs v5.41.0
    go.podman.io/storage v1.63.0                   # ← v6 needs v1.64.0

replace (
    github.com/opencontainers/runtime-spec => v1.2.1   # pinned for v5; may unpin
    github.com/opencontainers/runtime-tools => v0.9.1-0.20250523... # pinned for v5
    go.podman.io/common => v0.67.1                     # pinned; check v6 needs
)
```

### 2.2 Go Source Imports (`pkg/cri/podman/podman.go`)

```go
import (
    "github.com/containers/podman/v5/libpod/define"
    "github.com/containers/podman/v5/pkg/api/handlers"
    "github.com/containers/podman/v5/pkg/bindings"
    "github.com/containers/podman/v5/pkg/bindings/containers"
    "github.com/containers/podman/v5/pkg/bindings/images"
    "github.com/containers/podman/v5/pkg/bindings/manifests"
    "github.com/containers/podman/v5/pkg/bindings/secrets"
    "github.com/containers/podman/v5/pkg/specgen"
    buildahDefine "github.com/containers/buildah/define"
    nettypes "go.podman.io/common/libnetwork/types"
)
```

**API surface used:**
- `bindings.NewConnection(ctx, socket)` — connect to Podman service
- `specgen.NewSpecGenerator(image, isRootfs)` — container spec generator
- `specgen.NamedVolume`, `specgen.ContainerSecurityConfig` fields
- `containers.ExecCreate`, `containers.Inspect`, `containers.Wait`
- `containers.WaitOptions{Condition: []define.ContainerStatus{...}}`
- `define.ContainerStateStopped`, `define.ContainerStateExited`
- `handlers.ExecCreateConfig{ExecOptions: container.ExecOptions{...}}`
- `images.Build(ctx, paths, images.BuildOptions{BuildOptions: buildahDefine.BuildOptions{...}})`
- `manifests.*` (manifest list operations)
- `secrets.Create(conn, reader, opts.WithName().WithReplace())`
- `buildahDefine.BuildOptions{Output, Log, Architecture, OS, PullPolicy, Out, ContextDirectory, Platforms}`

### 2.3 Runtime / CI Surface

| File | Usage |
|------|-------|
| `pkg/cri/podman/podman.go:81-94` | CLI version detection: checks `podman version 3.` / `4.` prefix, else uses `podman machine inspect` |
| `pkg/cri/utils/socket.go:43` | `podman info -f "{{ .Host.RemoteSocket.Path }}"` for socket discovery |
| `pkg/doctor/runtime.go:172` | `podman version --format "{{.Version}}"` for doctor check |
| `pkg/golang/debian/Dockerfilego:3` | `libbtrfs-dev` install (needed by podman go module) |
| `pkg/golang/debiancgo/Dockerfilego:15` | `libbtrfs-dev` install |
| `.github/workflows/engine-ci-workflow.yml:185` | Writes `/etc/containers/registries.conf` |
| `.github/workflows/engine-ci-workflow.yml:189` | Installs `libbtrfs-dev`, `libgpgme-dev` |
| `.github/workflows/artifact.yaml:47` | Installs `libbtrfs-dev` |
| `.github/workflows/engine-ci.yml:28` | Matrix `runtime: podman` |

---

## 3. Podman v6 Breaking Changes — Impact Assessment

From the [v6.0.0 release notes](https://github.com/podman-container-tools/podman/releases/tag/v6.0.0)
and [v6.0.2](https://github.com/podman-container-tools/podman/releases/tag/v6.0.2):

### 3.1 Critical (requires code changes)

| # | Breaking Change | Impact on engine-ci | Action |
|---|----------------|---------------------|--------|
| C1 | **Import path changed** from `github.com/containers/podman/v5` → `go.podman.io/podman/v6` | `pkg/cri/podman/podman.go` — 8 import lines | Update all imports + `go.mod` |
| C2 | **Must use Buildah v1.44.0**, Skopeo v1.23, Netavark/Aardvark v2.0.0, common/v0.68.0 | `go.mod`: `github.com/containers/buildah v1.43.2` → `v1.44.1` (v6.0.2 ships Buildah v1.44.1) | Bump buildah, update common replace |
| C3 | **`artifacts.Remove()` Go binding removed `nameOrID` parameter** | Not currently used in engine-ci (no `artifacts` import) | No action — verify on upgrade |
| C4 | **Network isolation now defaults to enabled** | Container networking behavior changes — containers may not reach each other by default | Review if engine-ci relies on default network sharing between containers; may need explicit `--network` |
| C5 | **`podman volume prune` now only prunes unused anonymous volumes** (use `--all` for old behavior) | Not used in Go code; check if any CI scripts call `volume prune` | Verify CLI usage |
| C6 | **`podman volume list` multiple filters now AND instead of OR** | Not used in Go code | No action |
| C7 | **`--format='{{json .Labels}}'` now prints comma-separated key=value** | Not used in Go code | No action |
| C8 | **`MemorySwappiness` field now `nil` instead of `-1`** | `podman inspect` output parsing — check if engine-ci reads this field | Verify `containers.Inspect` result handling |
| C9 | **`podman commit` now pauses container by default** | Not used in Go code (no `commit` binding call) | No action |
| C10 | **Minimum Go version v1.25** | `go.mod` already uses `go 1.26.3` | ✅ Already satisfied |

### 3.2 Runtime Environment (CI / host requirements)

| # | Breaking Change | Impact | Action |
|---|----------------|--------|--------|
| R1 | **BoltDB dropped** — auto-migration to SQLite on first start | CI runners start fresh each run; no existing BoltDB | ✅ No action (no persistent state) |
| R2 | **Intel Mac support removed** | GitHub Actions `macos-13` runners (Intel) would break; check runner labels | Use `macos-14`/`macos-15` (ARM) if any macos jobs |
| R3 | **Windows 10 support removed** | No Windows runners in CI | ✅ No action |
| R4 | **cgroups v1 support removed** | GitHub Actions `ubuntu-22.04`/`ubuntu-24.04` use cgroups v2 | ✅ No action |
| R5 | **iptables support removed** — use nftables | Ubuntu runners default to nftables on 24.04 | Verify; likely fine |
| R6 | **CNI networking removed** — use Netavark | GitHub Actions Podman images default to netavark | Verify; likely fine |
| R7 | **slirp4netns removed** — use Pasta; `--network-cmd-path` removed | Rootless networking defaults to pasta in v6 | Remove any `--network-cmd-path` / slirp references (none found) |
| R8 | **Quadlet `.app` file tracking → subdirectories** | Not used in engine-ci | No action |
| R9 | **Linux `podman machine` volumes now use systemd** — existing VMs need recreation | Not used in CI (no podman machine in workflows) | No action |
| R10 | **Config file parsing major rewrite** | engine-ci writes `registries.conf` directly (not via podman config) | Verify `registries.conf` still valid |

### 3.3 Dependencies to bump

| Module | Current | Target (v6.0.2) | Notes |
|--------|---------|-----------------|-------|
| `github.com/containers/podman/v5` | v5.8.5 | → `go.podman.io/podman/v6` v6.0.2 | Import path change |
| `github.com/containers/buildah` | v1.43.2 | v1.44.1 | Shipped with v6.0.2 |
| `go.podman.io/common` | v1.0.1 (replace → v0.67.1) | v0.69.0 | v6.0.0 uses common v0.68.0; v6.0.2 likely v0.69.0 — verify |
| `go.podman.io/image/v5` | v5.40.0 (indirect) | v5.41.0 | Transitive |
| `go.podman.io/storage` | v1.63.0 (indirect) | v1.64.0 | Transitive |
| `github.com/opencontainers/runtime-spec` | v1.2.1 (replace pin) | **unpin?** | v6 may support `*int64` for `LinuxPids.Limit`; test removing the replace |
| `github.com/opencontainers/runtime-tools` | v0.9.1-0.20250523... (replace pin) | **unpin?** | Same — test removing |
| `go.podman.io/common` replace | → v0.67.1 | → v0.69.0 or remove | Update or remove replace |

---

## 4. Migration Steps

### Phase 1: Preparation (no code changes)

- [ ] **1.1** Verify `go.podman.io/podman/v6` v6.0.2 tag exists and is fetchable
  ```sh
  go list -m go.podman.io/podman/v6@v6.0.2
  ```
- [ ] **1.2** Verify Buildah v1.44.1 tag exists
  ```sh
  go list -m github.com/containers/buildah@v1.44.1
  ```
- [ ] **1.3** Audit all `github.com/containers/podman/v5` imports across the repo
  ```sh
  grep -rn "github.com/containers/podman/v5" --include="*.go" .
  ```
- [ ] **1.4** Check the v6 API for renamed/removed symbols in the packages we use:
  `libpod/define`, `pkg/api/handlers`, `pkg/bindings`, `pkg/bindings/containers`,
  `pkg/bindings/images`, `pkg/bindings/manifests`, `pkg/bindings/secrets`, `pkg/specgen`
- [ ] **1.5** Check if `buildahDefine.BuildOptions` struct changed between v1.43.2 and v1.44.1
  (fields: `Output`, `Log`, `Architecture`, `OS`, `PullPolicy`, `Out`, `ContextDirectory`, `Platforms`)

### Phase 2: Go Module Migration (Track A)

- [ ] **2.1** Update `go.mod` direct dependencies:
  ```diff
  - github.com/containers/buildah v1.43.2
  - github.com/containers/podman/v5 v5.8.5
  + github.com/containers/buildah v1.44.1
  + go.podman.io/podman/v6 v6.0.2
  ```
- [ ] **2.2** Update `go.mod` replace directives:
  - Update `go.podman.io/common` replace to v0.69.0 (or remove if direct dep resolves)
  - Test removing the `runtime-spec` v1.2.1 pin (v6 should use `*int64`)
  - Test removing the `runtime-tools` pin
- [ ] **2.3** Update all import paths in `pkg/cri/podman/podman.go`:
  ```diff
  - "github.com/containers/podman/v5/libpod/define"
  - "github.com/containers/podman/v5/pkg/api/handlers"
  - "github.com/containers/podman/v5/pkg/bindings"
  - "github.com/containers/podman/v5/pkg/bindings/containers"
  - "github.com/containers/podman/v5/pkg/bindings/images"
  - "github.com/containers/podman/v5/pkg/bindings/manifests"
  - "github.com/containers/podman/v5/pkg/bindings/secrets"
  - "github.com/containers/podman/v5/pkg/specgen"
  + "go.podman.io/podman/v6/libpod/define"
  + "go.podman.io/podman/v6/pkg/api/handlers"
  + "go.podman.io/podman/v6/pkg/bindings"
  + "go.podman.io/podman/v6/pkg/bindings/containers"
  + "go.podman.io/podman/v6/pkg/bindings/images"
  + "go.podman.io/podman/v6/pkg/bindings/manifests"
  + "go.podman.io/podman/v6/pkg/bindings/secrets"
  + "go.podman.io/podman/v6/pkg/specgen"
  ```
  Also check `buildahDefine` import — Buildah may have moved to `go.podman.io/buildah` in the CNCF org. Verify.
- [ ] **2.4** Run `go mod tidy` and resolve any version conflicts
- [ ] **2.5** Build: `go build -tags containers_image_openpgp,exclude_graphdriver_btrfs ./...`
- [ ] **2.6** Fix any compilation errors from API changes (check each used symbol):
  - `bindings.NewConnection` signature
  - `specgen.NewSpecGenerator` signature
  - `specgen.NamedVolume`, `specgen.ContainerSecurityConfig` fields
  - `containers.ExecCreate`, `containers.Inspect`, `containers.Wait`, `containers.WaitOptions`
  - `define.ContainerStateStopped`, `define.ContainerStateExited`
  - `handlers.ExecCreateConfig`
  - `images.Build`, `images.BuildOptions`
  - `secrets.Create`, `secrets.CreateOptions` (`.WithName()`, `.WithReplace()`)
  - `buildahDefine.BuildOptions` struct fields
- [ ] **2.7** Review network isolation default change (C4): if engine-ci containers need
  cross-container networking, explicitly set network config in `specgen`
- [ ] **2.8** Run tests: `go test -tags containers_image_openpgp,exclude_graphdriver_btrfs ./pkg/cri/...`

### Phase 3: Runtime / CI Migration (Track B)

- [ ] **3.1** Update podman version detection in `pkg/cri/podman/podman.go:81-94`:
  - Current code checks for `podman version 3.` / `4.` prefix
  - v6 output is `podman version 6.x.x` — the existing `else` branch (machine inspect) handles v5+
  - **Decision:** Either keep current logic (v6 falls into else branch, which works) OR
    simplify since v3/v4 are now very old. Recommend: keep for backwards compat but add v6 comment.
- [ ] **3.2** Verify `podman info -f "{{ .Host.RemoteSocket.Path }}"` still works in v6
  (used in `pkg/cri/utils/socket.go:43` and `pkg/cri/podman/podman.go:83`)
- [ ] **3.3** Check GitHub Actions runner OS:
  - `.github/workflows/engine-ci.yml` and `engine-ci-workflow.yml` — ensure no `macos-13` (Intel)
  - Ensure Ubuntu runners are 24.04 (nftables, netavark, cgroups v2 default)
- [ ] **3.4** Verify `/etc/containers/registries.conf` format is still valid in v6
  (config file parsing was rewritten — but `registries.conf` is a containers/common file, not podman.conf)
- [ ] **3.5** Verify `libbtrfs-dev` is still needed (the storage library v1.64.0 may have changed btrfs deps)
- [ ] **3.6** If CI installs podman from distro packages, check the distro ships v6
  (Ubuntu 24.04 ships v4.x; may need podman PPA or static binary for v6 in CI)

### Phase 4: Testing & Validation

- [ ] **4.1** Unit tests: `go test -tags containers_image_openpgp,exclude_graphdriver_btrfs ./...`
- [ ] **4.2** Integration: run engine-ci with podman v6 runtime in CI (matrix `runtime: podman`)
- [ ] **4.3** Verify container create/start/stop/wait/exec lifecycle
- [ ] **4.4** Verify image build (`BuildImage`, `BuildMultiArchImage`) with Buildah v1.44.1
- [ ] **4.5** Verify manifest list operations (`manifests.*` bindings)
- [ ] **4.6** Verify secrets creation (`secrets.Create`)
- [ ] **4.7** Run benchmarks: `go test -bench=. ./pkg/cri/...` (compare v5 vs v6 perf)
- [ ] **4.8** Run `pkg/doctor` runtime check against podman v6

### Phase 5: Cleanup

- [ ] **5.1** Remove obsolete `replace` directives in `go.mod` if v6 resolves them natively
- [ ] **5.2** Update comments in `go.mod` referencing v5 workarounds
- [ ] **5.3** Update `CLAUDE.md` / `README.md` if they reference Podman v5
- [ ] **5.4** Open PR from `franky-agent/engine-ci:chore/migrate-podman-v6` → `containifyci/engine-ci:main`

---

## 5. Risk Areas & Unknowns

| Risk | Mitigation |
|------|------------|
| `buildahDefine.BuildOptions` struct fields may have changed | Check buildah v1.44.0 changelog; the `Platforms` field type may have changed |
| `specgen` API may have restructured fields | Compare `pkg/specgen` types between v5.8.5 and v6.0.2 |
| `define.ContainerStateStopped` / `ContainerStateExited` constants may be renamed | Check `libpod/define` in v6 |
| `handlers.ExecCreateConfig` may have moved | Check `pkg/api/handlers` in v6 |
| Buildah may have moved to `go.podman.io/buildah` | Verify import path; the CNCF move may have affected buildah too |
| Transitive dependency hell (docker v28, moby, etc.) | `go mod tidy` + manual conflict resolution; may need additional replaces |
| CI Podman version: GitHub Actions runners may not have v6 | Use static binary download or Podman PPA in workflow |
| Network isolation default change breaks cross-container comms | Test thoroughly; add explicit network config if needed |

---

## 6. Rollback Plan

If migration fails or causes regressions:
1. Revert to `github.com/containers/podman/v5 v5.8.5` in `go.mod`
2. Restore original imports in `pkg/cri/podman/podman.go`
3. Restore original `replace` directives
4. The fork's `main` branch remains on v5; `chore/migrate-podman-v6` is isolated

---

## 7. References

- [Podman v6.0.0 Release Notes](https://github.com/podman-container-tools/podman/releases/tag/v6.0.0)
- [Podman v6.0.2 Release Notes](https://github.com/podman-container-tools/podman/releases/tag/v6.0.2)
- [Config file parsing design doc](https://github.com/podman-container-tools/podman/blob/main/contrib/design-docs/config-file-parsing.md)
- [Podman v6 breaking changes / upgrade guide](https://byteiota.com/podman-6-breaking-changes-upgrade-guide/)
- engine-ci fork: https://github.com/franky-agent/engine-ci
- Migration branch: `chore/migrate-podman-v6`