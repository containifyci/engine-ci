package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/spf13/cobra"
)

// imagesCmd lists all intermediate containifyci container images as JSON.
//
// Unlike the cache save/load commands (which call buildSteps.Images() with a
// real consumer build and rely on Matches() to filter steps), this command
// enumerates ALL possible intermediate images across every build type and
// variant. It does so by iterating the steps registered in InitBuildSteps()
// (cmd/engine.go), bypassing Matches() and calling step.Images() directly with
// a set of synthetic probe builds that cover all variant-selection mechanisms
// (BuildType, go_type, from, goreleaser custom property).
//
// Each step that produces containifyci intermediate images declares its
// Dockerfile(s) via the BuildStep.Dockerfiles() method. The command correlates
// each step's Dockerfiles with the containifyci image URIs returned by
// step.Images().
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List all intermediate containifyci container images as JSON",
	Long: `List all intermediate container images that engine-ci builds and pushes
to the containifyci Docker Hub registry.

The output is a JSON array of objects, each describing one intermediate image
(image name, checksum tag, full URI, Dockerfile path, build context dir and
the build step name). Base images (e.g. golang:1.26.5, alpine:latest) are NOT
included — only the containifyci/* intermediate images that engine-ci itself
produces.

This is used by the build-intermediate-images GitHub Actions workflow to
pre-build and push every intermediate image to Docker Hub so that consumer
CI runs can pull them instead of rebuilding from scratch every time.

The image list is derived automatically from the build steps registered in
InitBuildSteps() — adding a new package with a Dockerfile and a Dockerfiles()
declaration automatically includes its intermediate image here.
`,
	RunE:       RunImagesCmd,
	Annotations: map[string]string{skipRootHooks: "true"},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
}

// ImageInfo describes a single intermediate container image.
type ImageInfo struct {
	// URI is the full image URI, e.g. docker.io/containifyci/zig-3.24:6f9120d0...
	URI string `json:"uri"`
	// LatestURI is the same image tagged :latest, e.g. docker.io/containifyci/zig-3.24:latest
	LatestURI string `json:"latest_uri"`
	// Name is the image name without registry/tag, e.g. zig-3.24
	Name string `json:"name"`
	// Tag is the image tag (the Dockerfile checksum), e.g. 6f9120d0...
	Tag string `json:"tag"`
	// Registry is the registry host + namespace, e.g. docker.io/containifyci
	Registry string `json:"registry"`
	// Dockerfile is the repository-relative path to the Dockerfile, e.g. pkg/zig/Dockerfile.zig
	Dockerfile string `json:"dockerfile"`
	// Context is the repository-relative build context directory (the directory
	// containing the Dockerfile), e.g. pkg/zig. This is needed because some
	// Dockerfiles (e.g. gcloud) ADD local files from the context.
	Context string `json:"context"`
	// BuildStep is the name of the engine-ci build step that produces this image, e.g. zig
	BuildStep string `json:"build_step"`
}

// RunImagesCmd implements the `engine-ci images` command.
func RunImagesCmd(cmd *cobra.Command, _ []string) error {
	images := CollectImages()
	out, err := json.MarshalIndent(images, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal images: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

// probeBuilds is the set of synthetic builds used to enumerate all image
// variants. Each build varies the custom properties / build type that select
// different Dockerfile variants within a single step (e.g. go_type=chromium
// selects the chromium Dockerfile in pkg/golang/alpine, from=debian selects
// the debian variant, BuildType=Zig + goreleaser=true selects the
// goreleaser-zig image).
//
// Adding a new variant mechanism to engine-ci only requires adding a probe
// build here — the rest of the discovery is automatic.
func probeBuilds() []*container.Build {
	mk := func(bt container.BuildType, custom map[string][]string) *container.Build {
		return (&container.Build{
			ContainifyRegistry: "containifyci",
			BuildType:           bt,
			Custom:              container.Custom(custom),
		}).Defaults()
	}
	return []*container.Build{
		mk(container.Generic, nil),                                       // default (zig, maven, python, generic, etc.)
		mk(container.GoLang, nil),                                        // golang alpine default
		mk(container.GoLang, map[string][]string{"go_type": {"chromium"}}), // golang alpine chromium
		mk(container.GoLang, map[string][]string{"from": {"debian"}}),    // golang debian
		mk(container.GoLang, map[string][]string{"from": {"debiancgo"}}),  // golang debiancgo
		mk(container.Zig, map[string][]string{"goreleaser": {"true"}}),   // goreleaser-zig
		mk(container.Maven, nil),                                         // maven
		mk(container.Python, nil),                                        // python
		mk(container.AI, nil),                                            // claude
	}
}

// CollectImages enumerates all intermediate containifyci container images by
// iterating the build steps registered in InitBuildSteps() and probing each
// step's Images() method with synthetic builds covering all variants.
//
// It is exported so it can be reused by tests and other tooling.
func CollectImages() []ImageInfo {
	// Silence logs — some image functions log warnings on edge cases.
	slog.SetDefault(slog.New(discardHandler{}))

	InitBuildSteps()

	// Track seen URIs to deduplicate across steps (e.g. golang build + prod
	// steps both return the same intermediate image).
	seen := map[string]bool{}
	var images []ImageInfo

	for _, bctx := range buildSteps.Steps {
		step := bctx.Build()
		dockerfiles := step.Dockerfiles()
		if len(dockerfiles) == 0 {
			continue // step doesn't build containifyci intermediate images
		}

		// Collect all containifyci image URIs this step can produce by probing
		// it with every variant build. Bypass Matches() (which depends on
		// runtime state) by calling Images() directly.
		uris := map[string]bool{}
		for _, b := range probeBuilds() {
			for _, uri := range step.Images(*b) {
				if strings.Contains(uri, "containifyci/") {
					uris[uri] = true
				}
			}
		}

		// Correlate Dockerfiles with image URIs. If the step declares exactly
		// one Dockerfile, assign it to every URI this step produces (most steps
		// produce a single intermediate image). If it declares multiple
		// Dockerfiles (e.g. golang/alpine has default + chromium), assign them
		// in sorted order to the sorted URIs.
		sortedURIs := keys(uris)
		sort.Strings(sortedURIs)
		sortedDFs := append([]string{}, dockerfiles...)
		sort.Strings(sortedDFs)

		for i, uri := range sortedURIs {
			df := sortedDFs[0]
			if i < len(sortedDFs) {
				df = sortedDFs[i]
			}
			if seen[uri] {
				continue
			}
			seen[uri] = true
			info := parseImageURI(uri, df, step.Name())
			images = append(images, info)
		}
	}

	// Stable ordering for reproducible output / workflow matrices.
	sort.Slice(images, func(i, j int) bool {
		return images[i].URI < images[j].URI
	})
	return images
}

// keys returns the keys of a map[string]bool as a slice.
func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// parseImageURI splits a full image URI into an ImageInfo, attaching the
// Dockerfile path and build step name from the producing step.
func parseImageURI(uri, dockerfile, buildStep string) ImageInfo {
	info := ImageInfo{
		URI:        uri,
		Dockerfile: dockerfile,
		Context:    filepath.Dir(dockerfile),
		BuildStep:  buildStep,
	}
	// URI format: <registry>/<image>:<tag>  e.g. containifyci/zig-3.24:6f9120d0...
	// The registry may include a host prefix (docker.io/containifyci).
	idx := strings.LastIndex(uri, ":")
	if idx >= 0 {
		info.Tag = uri[idx+1:]
		uri = uri[:idx]
	}
	// uri is now <registry>/<image>
	slash := strings.LastIndex(uri, "/")
	if slash >= 0 {
		info.Registry = uri[:slash]
		info.Name = uri[slash+1:]
	} else {
		info.Name = uri
	}
	info.LatestURI = uri + ":latest"
	return info
}

// discardHandler is a no-op slog.Handler used to keep command/test output clean.
type discardHandler struct{}

func (discardHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (discardHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (h discardHandler) WithAttrs(_ []slog.Attr) slog.Handler           { return h }
func (h discardHandler) WithGroup(_ string) slog.Handler                { return h }