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
// It iterates the build steps registered in InitBuildSteps() (cmd/engine.go)
// and calls step.IntermediateImages() on each. Each step is the authority on
// its own images and variants — it returns every containifyci intermediate
// image it can build, paired with its Dockerfile path, in a single call. This
// bypasses Matches() (which depends on runtime state like env vars/secrets) and
// avoids any external probing or correlation.
//
// Adding a new package with an IntermediateImagesFn declaration automatically
// includes its image in the output and the pre-build workflow.
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
InitBuildSteps() — adding a new package with an IntermediateImagesFn
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

// CollectImages enumerates all intermediate containifyci container images by
// iterating the build steps registered in InitBuildSteps() and calling
// step.IntermediateImages() on each. Each step returns all its images (with
// Dockerfile paths) in one call — no probing or correlation needed.
//
// It is exported so it can be reused by tests and other tooling.
func CollectImages() []ImageInfo {
	// Silence logs — some image functions log warnings on edge cases.
	slog.SetDefault(slog.New(discardHandler{}))

	InitBuildSteps()

	// A synthetic build with the containifyci registry. Defaults() fills in
	// the registries/platform map but does NOT require a container runtime.
	build := (&container.Build{
		App:               "images",
		ContainifyRegistry: "containifyci",
	}).Defaults()

	seen := map[string]bool{}
	var images []ImageInfo

	for _, bctx := range buildSteps.Steps {
		step := bctx.Build()
		for _, img := range step.IntermediateImages(*build) {
			uri := img.URI
			if !strings.Contains(uri, "containifyci/") {
				continue // not a containifyci intermediate image
			}
			if seen[uri] {
				continue // deduplicate across steps (e.g. build + prod)
			}
			seen[uri] = true
			info := parseImageURI(uri, img.Dockerfile, step.Name())
			images = append(images, info)
		}
	}

	// Stable ordering for reproducible output / workflow matrices.
	sort.Slice(images, func(i, j int) bool {
		return images[i].URI < images[j].URI
	})
	return images
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