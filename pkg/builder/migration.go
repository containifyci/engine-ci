package builder

import (
	"embed"
	"fmt"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/builder/common"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

// Migration Example: GoBuilder shows how existing golang packages can be refactored
// to use the new unified interface while maintaining backward compatibility.
//
// This example demonstrates the transformation of pkg/golang/alpine/golang.go
// to use the new builder interface structure.

// ExampleGoBuilder demonstrates how to migrate the existing Go builder implementation.
// This serves as a template for migrating other language builders.
type ExampleGoBuilder struct {
	*BaseBuilder
	dockerFiles embed.FS
	goVersion   string
	lintImage   string
}

// NewGoBuilder creates a new Go builder instance using the new architecture.
// This replaces the New() function in existing golang packages.
func NewGoBuilder(build container.Build, dockerFiles embed.FS) *ExampleGoBuilder {
	defaults := common.GetGoDefaults()
	base := NewBaseBuilder(build, defaults)

	return &ExampleGoBuilder{
		BaseBuilder: base,
		goVersion:   defaults.LanguageVersion,
		lintImage:   defaults.LintImage,
		dockerFiles: dockerFiles,
	}
}

// CacheFolder overrides the base implementation with Go-specific logic.
func (g *ExampleGoBuilder) CacheFolder() string {
	return common.CacheFolderFromCommand("go", "env GOMODCACHE", ".tmp/go")
}

// IntermediateImage returns the Go-specific intermediate image.
func (g *ExampleGoBuilder) IntermediateImage() string {
	return common.ImageURIFromDockerfile(
		g.dockerFiles,
		"Dockerfilego",
		fmt.Sprintf("golang-%s-alpine", g.goVersion),
		g.Config.Registry,
	)
}

// BuildIntermediateImage builds the Go intermediate image.
func (g *ExampleGoBuilder) BuildIntermediateImage() error {
	platforms := make([]string, len(g.Config.Platforms))
	for i, p := range g.Config.Platforms {
		platforms[i] = p.String()
	}

	return common.BuildIntermediateImage(
		g.Container,
		g.dockerFiles,
		"Dockerfilego",
		g.IntermediateImage(),
		platforms...,
	)
}

// BuildScript generates the Go build script.
func (g *ExampleGoBuilder) BuildScript() string {
	// This would call the existing buildscript package
	// return buildscript.NewBuildScript(...).String()
	return "# Go build script placeholder"
}

// Images returns all images used by the Go builder.
func (g *ExampleGoBuilder) Images() []string {
	baseImages := g.BaseBuilder.Images()
	return append(baseImages, g.lintImage)
}

// Lint implements the LintableBuilder interface for Go.
func (g *ExampleGoBuilder) Lint() error {
	// Implementation would be similar to existing Lint() method
	// but using the new container setup methods
	opts := types.ContainerConfig{
		Image: g.lintImage,
		Env: []string{
			"GOMODCACHE=/go/pkg/",
			"GOCACHE=/go/pkg/build-cache",
			"GOLANGCI_LINT_CACHE=/go/pkg/lint-cache",
		},
		Cmd:        []string{"sh", "/tmp/script.sh"},
		WorkingDir: "/src",
	}

	g.SetupContainerVolumes(&opts)
	if err := g.SetupSSHForwarding(&opts); err != nil {
		return err
	}

	if err := g.Create(opts); err != nil {
		return err
	}

	// Copy lint script and execute
	// ... rest of lint implementation

	return g.Wait()
}

// LintImage returns the linting image name.
func (g *ExampleGoBuilder) LintImage() string {
	return g.lintImage
}

// MigrationCompatibilityLayer shows how to maintain backward compatibility
// while migrating to the new architecture.

// LegacyGoContainer provides backward compatibility for existing code.
// This wrapper allows existing code to continue working while we migrate.
type LegacyGoContainer struct {
	builder LanguageBuilder

	// Legacy fields for compatibility
	*container.Container
	App       string
	File      string
	Folder    string
	Image     string
	ImageTag  string
	Platforms []*types.PlatformSpec
	Tags      []string
}

// NewLegacyGoContainer creates a legacy-compatible Go container.
// This function maintains the same signature as the existing New() function.
func NewLegacyGoContainer(build container.Build) *LegacyGoContainer {
	// Create new builder
	builder := &ExampleGoBuilder{} // Placeholder - would use actual implementation

	// Create compatibility wrapper
	return &LegacyGoContainer{
		builder:   builder,
		Container: container.New(build),
		App:       build.App,
		File:      build.File,
		Folder:    build.Folder,
		Image:     build.Image,
		ImageTag:  build.ImageTag,
		Tags:      build.Custom.Strings("tags"),
		Platforms: common.GetDefaultPlatforms(build.Platform),
	}
}

// Legacy method implementations that delegate to the new builder
func (l *LegacyGoContainer) Name() string     { return l.builder.Name() }
func (l *LegacyGoContainer) IsAsync() bool    { return l.builder.IsAsync() }
func (l *LegacyGoContainer) Pull() error      { return l.builder.Pull() }
func (l *LegacyGoContainer) Build() error     { return l.builder.Build() }
func (l *LegacyGoContainer) Run() error       { return l.builder.Run() }
func (l *LegacyGoContainer) Images() []string { return l.builder.Images() }
func (l *LegacyGoContainer) Prod() error      { return l.builder.Prod() }

// Legacy functions that create build.Build instances
func NewLegacyLinter(build container.Build) build.Build {
	return common.NewLanguageBuild(
		func() error {
			container := NewLegacyGoContainer(build)
			if lintable, ok := container.builder.(LintableBuilder); ok {
				return lintable.Lint()
			}
			return fmt.Errorf("linting not supported")
		},
		"golangci-lint",
		[]string{"golangci/golangci-lint:v2.1.2"},
		false,
	)
}

func NewLegacyProd(build container.Build) build.Build {
	return common.NewLanguageBuild(
		func() error {
			container := NewLegacyGoContainer(build)
			return container.Prod()
		},
		"golang-prod",
		[]string{}, // Images would be populated by builder
		false,
	)
}

// Migration Steps Documentation:
//
// Phase 1.1 (Current): Create Interface and Common Code
// ✅ Created LanguageBuilder interface
// ✅ Created common utilities and types
// ✅ Created base builder implementation
// ✅ Created factory pattern for builder creation
// ✅ Created migration examples and compatibility layers
//
// Phase 1.2 (Next): Implement Language-Specific Builders
// 🔄 Create golang/builder.go that implements LanguageBuilder
// 🔄 Create maven/builder.go that implements LanguageBuilder
// 🔄 Create python/builder.go that implements LanguageBuilder
// 🔄 Register builders with the factory
// 🔄 Add configuration injection system
//
// Phase 1.3 (Future): Migrate Existing Packages
// 🔄 Update golang/alpine/golang.go to use new builder
// 🔄 Update golang/debian/golang.go to use new builder
// 🔄 Update maven/maven.go to use new builder
// 🔄 Update python/python.go to use new builder
// 🔄 Remove duplicate code from existing packages
//
// Phase 1.4 (Future): Remove Legacy Code
// 🔄 Remove compatibility wrappers
// 🔄 Update all callers to use new interface
// 🔄 Remove duplicate implementations
// 🔄 Validate no functionality regression

// RegistrationExample shows how language packages will register themselves.
func RegistrationExample() {
	// This would be called in init() functions of language packages
	err := RegisterBuilder(&BuilderRegistration{
		BuildType: container.GoLang,
		Name:      "golang",
		Constructor: func(build container.Build) (LanguageBuilder, error) {
			// Would create actual builder implementation
			return nil, fmt.Errorf("not implemented")
		},
		Features: BuilderFeatures{
			SupportsLinting:    true,
			SupportsProduction: true,
			SupportsAsync:      false,
			SupportsMultiStage: true,
			RequiredFiles:      []string{"go.mod"},
			OptionalFiles:      []string{"go.sum", "Dockerfile"},
		},
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to register golang builder: %v", err))
	}
}
