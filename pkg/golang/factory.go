package golang

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/builder"
	"github.com/containifyci/engine-ci/pkg/builder/common"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
)

// GoBuilderFactory implements BuildFactory for creating Go language builders.
type GoBuilderFactory struct {
	config *config.Config
}

// NewGoBuilderFactory creates a new factory for Go builders.
func NewGoBuilderFactory() (*GoBuilderFactory, error) {
	// For now, just use default config since LoadDefaultConfig doesn't exist yet
	cfg := config.GetDefaultConfig()

	return &GoBuilderFactory{
		config: cfg,
	}, nil
}

// CreateBuilder creates a new Go LanguageBuilder for the specified build configuration.
func (f *GoBuilderFactory) CreateBuilder(build container.Build) (builder.LanguageBuilder, error) {
	// Determine variant from custom configuration
	variant := VariantAlpine // default

	if from, ok := build.Custom["from"]; ok && len(from) > 0 {
		switch from[0] {
		case "debian":
			variant = VariantDebian
		case "debiancgo":
			variant = VariantDebianCGO
		case "alpine":
			variant = VariantAlpine
		default:
			slog.Warn("Unknown Go variant, using alpine", "variant", from[0])
			variant = VariantAlpine
		}
	}

	return NewGoBuilder(build, variant)
}

// CreateLinter creates a build.Build instance for Go linting operations.
func (f *GoBuilderFactory) CreateLinter(build container.Build) (build.Build, error) {
	return common.NewLanguageBuild(
		func() error {
			// Determine variant from custom configuration
			variant := VariantAlpine // default for linting
			if from, ok := build.Custom["from"]; ok && len(from) > 0 {
				switch from[0] {
				case "debian":
					variant = VariantDebian
				case "debiancgo":
					variant = VariantDebianCGO
				case "alpine":
					variant = VariantAlpine
				}
			}

			// Create Go builder and run lint
			goBuilder, err := NewGoBuilder(build, variant)
			if err != nil {
				return fmt.Errorf("failed to create Go builder: %w", err)
			}

			// Pull lint image first
			err = goBuilder.Container.Pull(goBuilder.LintImage())
			if err != nil {
				slog.Error("Failed to pull lint image", "error", err, "image", goBuilder.LintImage())
				return fmt.Errorf("failed to pull lint image: %w", err)
			}

			return goBuilder.Lint()
		},
		"golangci-lint",
		[]string{"golangci/golangci-lint:v2.1.2"}, // TODO: Use f.config.Language.Go.LintImage once config is fixed
		false, // Linting is synchronous
	), nil
}

// CreateProd creates a build.Build instance for Go production image creation.
func (f *GoBuilderFactory) CreateProd(build container.Build) (build.Build, error) {
	return common.NewLanguageBuild(
		func() error {
			// Determine variant from custom configuration
			variant := VariantAlpine // default
			if from, ok := build.Custom["from"]; ok && len(from) > 0 {
				switch from[0] {
				case "debian":
					variant = VariantDebian
				case "debiancgo":
					variant = VariantDebianCGO
				case "alpine":
					variant = VariantAlpine
				}
			}

			// Create Go builder and run production build
			goBuilder, err := NewGoBuilder(build, variant)
			if err != nil {
				return fmt.Errorf("failed to create Go builder: %w", err)
			}

			return goBuilder.Prod()
		},
		"golang-prod",
		[]string{"alpine:latest"}, // Production builds use alpine base
		false,                     // Production builds are synchronous
	), nil
}

// SupportedTypes returns the list of container.BuildType values this factory supports.
func (f *GoBuilderFactory) SupportedTypes() []container.BuildType {
	return []container.BuildType{container.GoLang}
}

// RegisterGoBuilder registers the Go builder with the global builder registry.
func RegisterGoBuilder() error {
	factory, err := NewGoBuilderFactory()
	if err != nil {
		return fmt.Errorf("failed to create Go builder factory: %w", err)
	}

	registration := &builder.BuilderRegistration{
		BuildType: container.GoLang,
		Name:      "golang",
		Constructor: func(build container.Build) (builder.LanguageBuilder, error) {
			return factory.CreateBuilder(build)
		},
		Features: builder.BuilderFeatures{
			SupportsLinting:    true,
			SupportsProduction: true,
			SupportsAsync:      false,
			SupportsMultiStage: true,
			RequiredFiles:      []string{"go.mod"},
			OptionalFiles:      []string{"go.sum", ".golangci.yml", ".custom-gcl.yml"},
		},
	}

	return builder.RegisterBuilder(registration)
}

// Backward compatibility functions that maintain the existing API

// New creates a new alpine Go container (backward compatibility).
func New(build container.Build) (*GoBuilder, error) {
	return NewGoBuilder(build, VariantAlpine)
}

// NewDebian creates a new debian Go container (backward compatibility).
func NewDebian(build container.Build) (*GoBuilder, error) {
	return NewGoBuilder(build, VariantDebian)
}

// NewCGO creates a new debian CGO Go container (backward compatibility).
func NewCGO(build container.Build) (*GoBuilder, error) {
	return NewGoBuilder(build, VariantDebianCGO)
}

// NewProd creates a production build for alpine variant (backward compatibility).
func NewProd(build container.Build) build.Build {
	factory, err := NewGoBuilderFactory()
	if err != nil {
		slog.Error("Failed to create Go factory", "error", err)
		// Return a build that will fail
		return common.NewLanguageBuild(
			func() error { return err },
			"golang-prod-error",
			[]string{},
			false,
		)
	}

	// Override the variant to alpine for backward compatibility
	build.Custom["from"] = []string{"alpine"}

	prodBuild, err := factory.CreateProd(build)
	if err != nil {
		slog.Error("Failed to create production build", "error", err)
		// Return a build that will fail
		return common.NewLanguageBuild(
			func() error { return err },
			"golang-prod-error",
			[]string{},
			false,
		)
	}

	return prodBuild
}

// NewProdDebian creates a production build for debian variant (backward compatibility).
func NewProdDebian(build container.Build) build.Build {
	factory, err := NewGoBuilderFactory()
	if err != nil {
		slog.Error("Failed to create Go factory", "error", err)
		// Return a build that will fail
		return common.NewLanguageBuild(
			func() error { return err },
			"golang-prod-error",
			[]string{},
			false,
		)
	}

	// Override the variant to debian for backward compatibility
	build.Custom["from"] = []string{"debian"}

	prodBuild, err := factory.CreateProd(build)
	if err != nil {
		slog.Error("Failed to create production build", "error", err)
		// Return a build that will fail
		return common.NewLanguageBuild(
			func() error { return err },
			"golang-prod-error",
			[]string{},
			false,
		)
	}

	return prodBuild
}

// NewLinter creates a linter build (backward compatibility).
func NewLinter(build container.Build) build.Build {
	factory, err := NewGoBuilderFactory()
	if err != nil {
		slog.Error("Failed to create Go factory", "error", err)
		// Return a build that will fail
		return common.NewLanguageBuild(
			func() error { return err },
			"golang-lint-error",
			[]string{},
			false,
		)
	}

	linterBuild, err := factory.CreateLinter(build)
	if err != nil {
		slog.Error("Failed to create linter build", "error", err)
		// Return a build that will fail
		return common.NewLanguageBuild(
			func() error { return err },
			"golang-lint-error",
			[]string{},
			false,
		)
	}

	return linterBuild
}

// LintImage returns the lint image name (backward compatibility).
func LintImage() string {
	// Return hardcoded value for now since config isn't ready
	return "golangci/golangci-lint:v2.1.2"
}

// CacheFolder returns the Go cache folder (backward compatibility).
func CacheFolder() string {
	// Fallback to original logic since config isn't ready
	cmd := exec.Command("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err != nil {
		slog.Error("Failed to execute command", "error", err)
		// Use fallback cache location
		fallbackCache := filepath.Join(os.TempDir(), "go-cache")
		return fallbackCache
	}
	
	return strings.TrimSpace(string(output))
}

// init automatically registers the Go builder when the package is imported.
func init() {
	if err := RegisterGoBuilder(); err != nil {
		slog.Error("Failed to register Go builder", "error", err)
	}
}
