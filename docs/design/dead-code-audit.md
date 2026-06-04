# Dead Code Audit (initial deep pass)

Date: 2026-05-10
Scope: static repository scan of Go code

## Method

This report is based on grep-style static analysis:
- searched for exported symbols and likely call sites
- flagged items with no obvious internal references
- excluded generated protobuf / metadata code from dead-code claims unless clearly unused
- treated plugin/reflection/Cobra wiring as potentially live even when direct call sites are sparse

## High-confidence dead code

### `cmd/root.go` commented-out `All`
- Status: dead
- Evidence: function is fully commented out in source
- Recommendation: delete unless you plan to restore it soon

### `pkg/kv/api.go` / `pkg/kv/client.go` unused helpers? 
- Status: not yet confirmed dead
- Evidence: package is actively imported by higher-level command code, so do not remove blindly
- Recommendation: inspect per-symbol usage before pruning

## Probable dead code candidates

### `cmd/update.go`
- Symbols: `ConfigureUpdate`, `SetUpdateConfig`, `GetUpdateConfig`, `UpdateConfig`
- Evidence: repo-wide grep only found definitions in `cmd/update.go`
- Confidence: high
- Caveat: these are exported helpers and may be intended for external consumers
- Suggested action: if this repo is only consumed as a binary, consider removing or moving behind a narrow internal API

### `pkg/dummy`
- Symbols: `Matches`, `New`
- Evidence: package is referenced from `cmd/engine.go` via `dummy.New()` for a Goreleaser step
- Confidence: low for deadness; likely live
- Note: this package is a placeholder/adapter rather than dead code

### `pkg/pulumi`
- Symbols: `Matches`, `New`, `new`, `CacheFolder`, `ComputeChecksum`, `(*PulumiContainer).PulumiImage`, `BuildPulumiImage`, `CopyScript`, `ApplyEnvs`, `Release`, `Pull`, `Run`
- Evidence: `cmd/engine.go` imports `pkg/pulumi` and registers `pulumi.New()` at build-step setup
- Confidence: low for deadness; live
- Note: the entire package appears intentionally wired into the build graph
- Next step: only prune inside this package if a symbol-level search shows no internal calls

### `pkg/packer`
- Symbols: `Matches`, `New`, `new`, `ComputeChecksum`, `(*packerContainer).packerImage`, `BuildpackerImage`, `CopyScript`, `Release`, `Pull`, `Run`
- Evidence: `cmd/engine.go` imports `pkg/packer` and registers `packer.New()` in build-step setup
- Confidence: low for deadness; live
- Note: the package is intentionally wired into the build graph
- Next step: only prune inside this package if a symbol-level search shows no internal calls

### `pkg/logger/altscreen.go`
- Symbols: `AltScreen`, `NewAlt`, `Enter`, `Exit`, `Render`, `isTTY`, `enableWindowsVT`
- Evidence: `pkg/logger/terminal.go` references `AltScreen` and `NewAlt(os.Stdout)`
- Confidence: low for deadness; likely live
- Note: platform-specific helper, not dead by inspection

### `client/pkg/build/plugin.go`
- Symbols: `Plugin.GetBuild`, `Plugin.GetBuilds`, `Serve`, `Build`, `BuildAsync`, `BuildGroups`
- Evidence: `client/client.go` uses `Build`, `BuildAsync`, `BuildGroups`
- Confidence: low for deadness; live
- Note: plugin entry points can be invoked indirectly

## Package wiring discovered during inspection

- `cmd/engine.go` imports and registers `pkg/pulumi`
- `cmd/engine.go` imports and registers `pkg/packer`
- `cmd/engine.go` imports and registers `pkg/dummy`
- `client/client.go` uses `client/pkg/build` helper entry points

This means the earlier suspicion that `pkg/pulumi` and `pkg/packer` were unused was incorrect; they are part of the runtime build-step registry.

## Generated / not-dead by default

### `protos2/*.pb.go`
- Status: generated support code, should not be pruned just because direct call sites are sparse

### `pkg/*/docker_metadata_gen.go`
- Status: generated metadata used by Dockerfile helpers

## False positives to avoid

- Cobra command functions may be invoked via command registration, not direct calls
- plugin/service interfaces may be used reflectively or by gRPC-generated code
- build-step packages often register themselves indirectly through factory functions
- tests often exercise symbols that are otherwise not referenced in production code

## Recommended next verification pass

1. Run a full import graph check for these packages:
   - `pkg/pulumi`
   - `pkg/packer`
   - `cmd/update.go`
2. Search for package registration paths:
   - build step registries
   - `init()` functions
   - `New()` factories
3. Use `go list -deps` / `go test ./...` to ensure no build-tag-only code is missed
4. If you want actual deletion candidates, focus on unexported helpers inside `pkg/pulumi` and `pkg/packer` first

## Current summary

At this stage, the only unequivocally dead code found is the commented-out `All` function in `cmd/root.go`. Most other suspects are either clearly live or require a deeper call-graph pass before removal.

## Deeper inspection: `pkg/maven`

### Findings
- `cmd/engine.go` imports `pkg/maven` and registers both `maven.New()` and `maven.NewProd()`.
- The package provides the expected build-step lifecycle: match, image build, container release, and run.
- Buildscript helpers are used by the Maven container implementation.

### Conclusion
- No obvious dead code identified in `pkg/maven`.

## Deeper inspection: `pkg/python`

### Findings
- `cmd/engine.go` imports `pkg/python` and registers both `python.New()` and `python.NewProd()`.
- The package includes its own builder helpers and buildscript support, all of which feed the runtime pipeline.

### Conclusion
- No obvious dead code identified in `pkg/python`.

## Deeper inspection: `pkg/zig`

### Findings
- `cmd/engine.go` imports `pkg/zig` and registers both `zig.New()` and `zig.NewProd()`.
- The buildscript helpers are used by the Zig container implementation and are covered by tests.

### Conclusion
- No obvious dead code identified in `pkg/zig`.

## Deeper inspection: `pkg/goreleaser`

### Findings
- `cmd/engine.go` imports `pkg/goreleaser` and registers the goreleaser build step.
- The package has additional injection points (`cacheFolderFn`, `zigCacheFolderFn`, `newWithManager`) that are exercised by tests and support dependency injection.

### Conclusion
- No obvious dead code identified in `pkg/goreleaser`.

## Deeper inspection: `pkg/sonarcloud`

### Findings
- `cmd/engine.go` imports `pkg/sonarcloud` and registers the build step.
- The `sonarqube.go` support code is used internally by the Sonarcloud container logic.

### Conclusion
- No obvious dead code identified in `pkg/sonarcloud`.

## Deeper inspection: `pkg/trivy`

### Findings
- `cmd/engine.go` imports `pkg/trivy` and registers the build step.
- The `report.go` parser and formatter are used by the Trivy workflow and unit tests.

### Conclusion
- No obvious dead code identified in `pkg/trivy`.

## Deeper inspection: `pkg/gcloud`

### Findings
- `cmd/engine.go` imports `pkg/gcloud` and registers the build step.
- `pkg/gcloud/src/main.go` is a standalone helper entry point compiled only with the `submodule` build tag.
- That means it is intentionally excluded from normal builds and should not be treated as dead code.

### Conclusion
- No obvious dead code identified in `pkg/gcloud`.

## Build-tag / auxiliary entry point check

### `pkg/gcloud/src/main.go`
- Status: intentionally build-tagged auxiliary binary
- Evidence: `//go:build submodule`
- Conclusion: not dead; it is opt-in code for a specific build mode.

### General rule added to the audit
- Files protected by build tags, especially standalone `package main` helpers, must not be marked dead unless the build mode they serve is itself removed.

## Final sweep: `pkg/logger`

### Findings
- `altscreen.go` is referenced by `terminal.go` and is part of the terminal rendering path.
- `slog_handler.go` is the public logging handler implementation used by higher-level logger setup.
- Benchmarks are test-only and not relevant to dead-code removal.

### Conclusion
- No obvious dead code identified in `pkg/logger`.

## Heuristic sweep: TODO / compatibility / wrapper markers

### Findings
- Many TODOs represent incomplete features, not dead code.
- Backward-compatibility helpers and wrappers are especially present in `pkg/autodiscovery`, `protos2`, and the build-step packages.
- These should be treated as live unless both local and downstream usage are disproven.

### Conclusion
- No additional dead code found from TODO/wrapper heuristics.

## Audit status

### Confirmed dead code
- Commented-out `All` function in `cmd/root.go`

### Cleanup proposal: safe removal candidates
1. **Delete the commented-out `All` block** in `cmd/root.go`
   - This is the only confirmed dead code.
   - Safe because it is commented out and unreachable.

### Cleanup proposal: keep, but document as supported API
2. **Retain `cmd/update.go` exported config helpers**
   - `ConfigureUpdate`
   - `SetUpdateConfig`
   - `GetUpdateConfig`
   - These are not used internally, but they are exported extension points.
   - Because this repo is used as a dependency by downstream projects, deleting them could be a breaking change.
   - Recommendation: document them as supported extension API rather than pruning them now.

### Explicitly not dead due to wiring / generated / build-tag / downstream usage
- build-step packages registered from `cmd/engine.go`
- generated protobuf and Docker metadata code
- `pkg/gcloud/src/main.go` (`submodule` build tag)
- logger, doctor, autodiscovery, and client build helpers

### Overall conclusion
- The codebase is largely cohesive and intentionally modular.
- The only safe code cleanup identified is removal of the commented-out `All` block.
- Everything else should be preserved unless downstream compatibility is explicitly audited.

## Deeper inspection: `cmd/update.go`

### Findings
- `ConfigureUpdate`, `SetUpdateConfig`, and `GetUpdateConfig` are exported but currently only referenced inside `cmd/update.go`.
- `updateCmd` and `runUpdate` are live because they are attached to Cobra via `init()`.
- The package is reachable through the CLI, but the exported config helpers are not currently used anywhere else in this repository.

### Interpretation
- These helpers are **not dead in the strict binary sense** if external consumers import `cmd` as a library.
- However, for this repository’s current binary-oriented usage, they are **unreferenced internal API surface** and are the strongest deletion candidates after the commented-out code.

### Recommendation
- Keep them if you expect downstream projects to override update metadata.
- Otherwise, consider moving them to a dedicated extension package or marking them as supported integration API in documentation rather than deleting immediately.

## Dependency-aware audit notes

Because this repository is used as a dependency by other projects such as `engine-java`, symbols that appear unused locally may still be relied on externally through:
- exported builder constructors
- build-step registration helpers
- Cobra command extension points
- package-level configuration setters/getters

That means any deletion candidate should be checked against:
1. local imports/call sites,
2. generated code or command registration,
3. downstream repositories that compile against this module.

## Deeper inspection: `pkg/autodiscovery`

### Findings
- The autodiscovery packages are heavily used by command entry points such as `cmd/engine.go` and `cmd/init.go`.
- Functions like `DiscoverProjects`, `DiscoverAndGenerateBuildGroups`, `DiscoverPythonProjects`, and the build conversion helpers are part of the main project detection pipeline.
- Test files also exercise many helper methods directly, so the package has broad live coverage.

### Conclusion
- No obvious dead code was identified in `pkg/autodiscovery` during this pass.
- Some helpers are only used via downstream selection logic and should not be pruned without a full call-graph / integration sweep.

## Deeper inspection: `client/pkg/build`

### Findings
- Constructors like `NewGoServiceBuild`, `NewGoLibraryBuild`, `NewMavenServiceBuild`, `NewMavenLibraryBuild`, `NewPythonServiceBuild`, and `NewPythonLibraryBuild` are used both locally and in autodiscovery tests.
- `Build`, `BuildAsync`, and `BuildGroups` are used by the CLI client and the generated/extension entry points.
- The AI build constructors are also referenced by the top-level config and tests.

### Conclusion
- No obvious dead code identified here.
- These are public helper APIs and should be treated as part of the supported extension surface.

## Deeper inspection: `pkg/golang`

### Findings
- The top-level `pkg/golang/golang.go` selectors are live and route into the alpine/debian/cgo implementations.
- `NewGoLibraryBuild` / `NewMavenLibraryBuild` / `NewPythonLibraryBuild` equivalents are used by autodiscovery to map project types.
- The alpine/debian/cgo implementation packages are actively wired through `cmd/engine.go`.

### Conclusion
- No dead code identified in the Go build packages during this pass.
- Many helpers are public dispatch points and should be retained unless downstream usage is disproven.

## Deeper inspection: `pkg/doctor`

### Findings
- `cmd/doctor.go` constructs `doctor.NewDoctor(...)` and runs checks via `RunChecks`.
- `doctor.NewDoctor` registers all built-in checks:
  - runtime detection
  - runtime connectivity
  - runtime version
  - volume config
  - volume write test
- The individual check constructors are therefore live, even though some return unexported concrete types.

### Conclusion
- No obvious dead code identified in `pkg/doctor`.
- The package is command-driven and should be treated as runtime code, not library deadwood.
