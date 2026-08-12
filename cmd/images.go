package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	claudealpine "github.com/containifyci/engine-ci/pkg/ai/claude/alpine"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/gcloud"
	"github.com/containifyci/engine-ci/pkg/github"
	golangalpine "github.com/containifyci/engine-ci/pkg/golang/alpine"
	golangdebian "github.com/containifyci/engine-ci/pkg/golang/debian"
	golangdebiancgo "github.com/containifyci/engine-ci/pkg/golang/debiancgo"
	"github.com/containifyci/engine-ci/pkg/goreleaser"
	"github.com/containifyci/engine-ci/pkg/maven"
	"github.com/containifyci/engine-ci/pkg/packer"
	"github.com/containifyci/engine-ci/pkg/protobuf"
	"github.com/containifyci/engine-ci/pkg/pulumi"
	"github.com/containifyci/engine-ci/pkg/python"
	"github.com/containifyci/engine-ci/pkg/zig"

	"github.com/spf13/cobra"
)

// imagesCmd lists all intermediate containifyci container images as JSON.
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List all intermediate containifyci container images as JSON",
	Long: `List all intermediate container images that engine-ci builds and pushes
to the containifyci Docker Hub registry.

The output is a JSON array of objects, each describing one intermediate image
(image name, checksum tag, full URI, Dockerfile path and the build step name).
Base images (e.g. golang:1.26.5, alpine:latest) are NOT included — only the
containifyci/* intermediate images that engine-ci itself produces.

This is used by the build-intermediate-images GitHub Actions workflow to
pre-build and push every intermediate image to Docker Hub so that consumer
CI runs can pull them instead of rebuilding from scratch every time.
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

// imageProducer describes a single source of intermediate images.
type imageProducer struct {
	// produce returns the containifyci intermediate image URI(s) for this step.
	// Base images returned by a step's Images() method are filtered out.
	produce func(build container.Build) []string
	// buildStep is the engine-ci build step name (for documentation/debugging).
	buildStep string
	// dockerfile is the repository-relative path to the Dockerfile.
	dockerfile string
}

// allImageProducers returns the list of every intermediate image producer in
// engine-ci. Adding a new package here automatically includes its image in the
// `engine-ci images` output and the pre-build workflow.
//
// Each producer calls the package's image function directly (e.g. zig.ZigImage,
// alpine.GoImage) rather than going through buildSteps.Images(), because the
// latter requires the step's Matches() predicate to pass — and many Matches()
// functions depend on runtime state (env vars, secrets, custom properties) that
// is not available when simply enumerating images.
func allImageProducers() []imageProducer {
	return []imageProducer{
		{
			buildStep:  "zig",
			dockerfile: "pkg/zig/Dockerfile.zig",
			produce: func(b container.Build) []string {
				return []string{zig.ZigImage(b)}
			},
		},
		{
			buildStep:  "goreleaser",
			dockerfile: "pkg/goreleaser/Dockerfile.goreleaser-zig",
			produce: func(b container.Build) []string {
				return []string{goreleaser.ZigGoreleaserImage(b)}
			},
		},
		{
			buildStep:  "golang (alpine)",
			dockerfile: "pkg/golang/alpine/Dockerfile_go",
			produce: func(b container.Build) []string {
				return []string{golangalpine.GoImage(b)}
			},
		},
		{
			buildStep:  "golang (alpine, chromium)",
			dockerfile: "pkg/golang/alpine/Dockerfile_chromium_go",
			produce: func(b container.Build) []string {
				// The chromium variant is selected via the go_type custom property.
				b.Custom = container.Custom{"go_type": []string{"chromium"}}
				return []string{golangalpine.GoImage(b)}
			},
		},
		{
			buildStep:  "golang (debian)",
			dockerfile: "pkg/golang/debian/Dockerfilego",
			produce: func(b container.Build) []string {
				return []string{golangdebian.GoImage(b)}
			},
		},
		{
			buildStep:  "golang (debian cgo)",
			dockerfile: "pkg/golang/debiancgo/Dockerfilego",
			produce: func(b container.Build) []string {
				return []string{golangdebiancgo.GoImage(b)}
			},
		},
		{
			buildStep:  "claude",
			dockerfile: "pkg/ai/claude/alpine/Dockerfile",
			produce: func(b container.Build) []string {
				return []string{claudealpine.ClaudeImage(b)}
			},
		},
		{
			buildStep:  "gcloud",
			dockerfile: "pkg/gcloud/Dockerfile",
			produce: func(b container.Build) []string {
				return []string{gcloud.Image(&b)}
			},
		},
		{
			buildStep:  "github",
			dockerfile: "pkg/github/Dockerfile",
			produce: func(b container.Build) []string {
				return []string{github.Image(b)}
			},
		},
		{
			buildStep:  "maven",
			dockerfile: "pkg/maven/Dockerfile.maven",
			produce: func(b container.Build) []string {
				return []string{maven.MavenImage(b)}
			},
		},
		{
			buildStep:  "packer",
			dockerfile: "pkg/packer/Dockerfile",
			produce: func(b container.Build) []string {
				return []string{packer.Image(b)}
			},
		},
		{
			buildStep:  "protobuf",
			dockerfile: "pkg/protobuf/Dockerfile",
			produce: func(b container.Build) []string {
				return []string{protobuf.Image(b)}
			},
		},
		{
			buildStep:  "pulumi",
			dockerfile: "pkg/pulumi/Dockerfile",
			produce: func(b container.Build) []string {
				return []string{pulumi.Image(b)}
			},
		},
		{
			buildStep:  "python",
			dockerfile: "pkg/python/Dockerfile.python",
			produce: func(b container.Build) []string {
				return []string{python.PythonImage(b)}
			},
		},
	}
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

// CollectImages enumerates all intermediate containifyci container images.
// It is exported so it can be reused by tests and other tooling.
func CollectImages() []ImageInfo {
	// A synthetic build with the containifyci registry. Defaults() fills in
	// the registries/platform map but does NOT require a container runtime.
	build := (&container.Build{
		App:                "images",
		ContainifyRegistry: "containifyci",
	}).Defaults()

	var images []ImageInfo
	seen := make(map[string]bool)

	for _, p := range allImageProducers() {
		for _, uri := range p.produce(*build) {
			// Only include containifyci intermediate images, not base images.
			if !strings.Contains(uri, "containifyci/") {
				continue
			}
			if seen[uri] {
				continue
			}
			seen[uri] = true
			images = append(images, parseImageURI(uri, p.dockerfile, p.buildStep))
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
