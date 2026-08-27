package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectImagesCount(t *testing.T) {
	images := CollectImages()
	assert.Len(t, images, 13, "expected 13 intermediate images, got %d", len(images))
}

func TestCollectImagesUnique(t *testing.T) {
	images := CollectImages()
	seen := map[string]bool{}
	for _, img := range images {
		require.Falsef(t, seen[img.URI], "duplicate image URI: %s", img.URI)
		seen[img.URI] = true
	}
}

func TestCollectImagesContainifyOnly(t *testing.T) {
	images := CollectImages()
	require.NotEmpty(t, images, "expected at least one image")
	for _, img := range images {
		assert.Containsf(t, img.URI, "containifyci/", "image URI should be a containifyci intermediate: %s", img.URI)
		assert.NotEmptyf(t, img.Name, "image name should not be empty: %s", img.URI)
		assert.NotEmptyf(t, img.Tag, "image tag should not be empty: %s", img.URI)
		assert.NotEmptyf(t, img.Dockerfile, "dockerfile path should not be empty: %s", img.URI)
		assert.NotEmptyf(t, img.Context, "context path should not be empty: %s", img.URI)
		assert.NotEmptyf(t, img.BuildStep, "build step should not be empty: %s", img.URI)
	}
}

func TestCollectImagesKnownImages(t *testing.T) {
	images := CollectImages()
	names := map[string]ImageInfo{}
	for _, img := range images {
		names[img.Name] = img
	}

	expected := []string{
		"zig-3.24",
		"goreleaser-zig-v2.17.1",
		"golang-1.27.0-alpine",
		"golang-1.27.0-alpine-chromium",
		"golang-1.27.0",
		"golang-1.27.0-cgo",

		"gcloud",
		"gh",
		"maven-3-eclipse-temurin-17-alpine",
		"packer",
		"protobuf",
		"pulumi-go",
		"python-3.14-slim-bookworm",
	}
	for _, name := range expected {
		img, ok := names[name]
		require.Truef(t, ok, "expected image %q in output", name)
		assert.Containsf(t, img.Dockerfile, "Dockerfile",
			"dockerfile path should reference a Dockerfile: %s", img.Dockerfile)
		assert.NotEmptyf(t, img.Context, "context should be the dockerfile dir: %s", img.Dockerfile)
		assert.Truef(t, strings.HasSuffix(img.Dockerfile, "/"+img.Context) || strings.Contains(img.Dockerfile, img.Context),
			"context should be the directory of the dockerfile: ctx=%s df=%s", img.Context, img.Dockerfile)
	}
}

func TestCollectImagesSorted(t *testing.T) {
	images := CollectImages()
	for i := 1; i < len(images); i++ {
		assert.LessOrEqualf(t, images[i-1].URI, images[i].URI,
			"images should be sorted by URI: %s > %s", images[i-1].URI, images[i].URI)
	}
}

func TestCollectImagesJSONSerializable(t *testing.T) {
	images := CollectImages()
	out, err := json.Marshal(images)
	require.NoError(t, err)
	require.NotEmpty(t, out)

	// And can be unmarshalled back.
	var back []ImageInfo
	require.NoError(t, json.Unmarshal(out, &back))
	assert.Len(t, back, len(images))
}

// TestCollectImagesNoSonar verifies that the sonarcloud image (containifyci/sonar)
// is intentionally excluded — its Dockerfile lives in hack/sonarcloud/, not a
// pkg/ package, and it is not part of the pre-build workflow.
func TestCollectImagesNoSonar(t *testing.T) {
	for _, img := range CollectImages() {
		assert.NotContainsf(t, img.Name, "sonar", "sonar image should not be included: %s", img.URI)
	}
}

func TestParseImageURI(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		dockerfile string
		buildStep  string
		wantName   string
		wantTag    string
		wantReg    string
	}{
		{
			name:       "simple containifyci image",
			uri:        "containifyci/zig-3.24:6f9120d0aabb",
			dockerfile: "pkg/zig/Dockerfile.zig",
			buildStep:  "zig",
			wantName:   "zig-3.24",
			wantTag:    "6f9120d0aabb",
			wantReg:    "containifyci",
		},
		{
			name:       "image with hyphens in name",
			uri:        "containifyci/maven-3-eclipse-temurin-17-alpine:214f702cd3",
			dockerfile: "pkg/maven/Dockerfile.maven",
			buildStep:  "maven",
			wantName:   "maven-3-eclipse-temurin-17-alpine",
			wantTag:    "214f702cd3",
			wantReg:    "containifyci",
		},
		{
			name:       "image with docker.io host prefix",
			uri:        "docker.io/containifyci/gh:035c2a5d",
			dockerfile: "pkg/github/Dockerfile",
			buildStep:  "github",
			wantName:   "gh",
			wantTag:    "035c2a5d",
			wantReg:    "docker.io/containifyci",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := parseImageURI(tt.uri, tt.dockerfile, tt.buildStep)
			assert.Equal(t, tt.uri, info.URI)
			assert.Equal(t, tt.wantName, info.Name)
			assert.Equal(t, tt.wantTag, info.Tag)
			assert.Equal(t, tt.wantReg, info.Registry)
			assert.Equal(t, tt.dockerfile, info.Dockerfile)
			assert.Equal(t, tt.buildStep, info.BuildStep)
			assert.NotEmpty(t, info.Context, "context should be the dockerfile directory")
		})
	}
}
