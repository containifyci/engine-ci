package config

import (
	"time"

	"github.com/containifyci/engine-ci/pkg/container"
)

// Config represents the complete centralized configuration for engine-ci.
// This structure replaces all hardcoded values throughout the codebase with
// configurable parameters that can be set via CLI flags, environment variables,
// configuration files, or defaults.
type Config struct {
	Version     string            `yaml:"version" json:"version"`
	Language    LanguageConfig    `yaml:"language" json:"language"`
	Container   ContainerConfig   `yaml:"container" json:"container"`
	Network     NetworkConfig     `yaml:"network" json:"network"`
	Cache       CacheConfig       `yaml:"cache" json:"cache"`
	Security    SecurityConfig    `yaml:"security" json:"security"`
	Logging     LoggingConfig     `yaml:"logging" json:"logging"`
	Environment EnvironmentConfig `yaml:"environment" json:"environment"`
}

// LanguageConfig contains configuration for all supported programming languages.
// Each language has its own sub-configuration with language-specific settings.
type LanguageConfig struct {
	Go       GoConfig       `yaml:"go" json:"go"`
	Maven    MavenConfig    `yaml:"maven" json:"maven"`
	Python   PythonConfig   `yaml:"python" json:"python"`
	Protobuf ProtobufConfig `yaml:"protobuf" json:"protobuf"`
}

// GoConfig contains Go language-specific configuration settings.
// Replaces hardcoded values from pkg/golang/ packages.
type GoConfig struct {
	Version       string            `yaml:"version" json:"version" validate:"required,semver"`
	LintImage     string            `yaml:"lint_image" json:"lint_image" validate:"required"`
	TestTimeout   time.Duration     `yaml:"test_timeout" json:"test_timeout" validate:"min=10s,max=30m"`
	BuildTimeout  time.Duration     `yaml:"build_timeout" json:"build_timeout" validate:"min=30s,max=2h"`
	CoverageMode  string            `yaml:"coverage_mode" json:"coverage_mode" validate:"oneof=binary text"`
	BuildTags     []string          `yaml:"build_tags" json:"build_tags"`
	ProjectMount  string            `yaml:"project_mount" json:"project_mount" validate:"required"`
	OutputDir     string            `yaml:"output_dir" json:"output_dir" validate:"required"`
	Variants      GoVariantConfig   `yaml:"variants" json:"variants"`
	ModCache      string            `yaml:"mod_cache" json:"mod_cache"`
	Environment   map[string]string `yaml:"environment" json:"environment"`
}

// GoVariantConfig contains configuration for different Go build variants.
type GoVariantConfig struct {
	Alpine    GoVariantSpec `yaml:"alpine" json:"alpine"`
	Debian    GoVariantSpec `yaml:"debian" json:"debian"`
	DebianCGO GoVariantSpec `yaml:"debian_cgo" json:"debian_cgo"`
}

// GoVariantSpec defines configuration for a specific Go build variant.
type GoVariantSpec struct {
	BaseImage    string `yaml:"base_image" json:"base_image" validate:"required"`
	ImageSuffix  string `yaml:"image_suffix" json:"image_suffix"`
	CGOEnabled   bool   `yaml:"cgo_enabled" json:"cgo_enabled"`
	Dependencies []string `yaml:"dependencies" json:"dependencies"`
}

// MavenConfig contains Maven/Java language-specific configuration settings.
// Replaces hardcoded values from pkg/maven/ package.
type MavenConfig struct {
	ProdImage      string            `yaml:"prod_image" json:"prod_image" validate:"required"`
	BaseImage      string            `yaml:"base_image" json:"base_image" validate:"required"`
	CacheLocation  string            `yaml:"cache_location" json:"cache_location" validate:"required"`
	TestTimeout    time.Duration     `yaml:"test_timeout" json:"test_timeout" validate:"min=30s,max=1h"`
	BuildTimeout   time.Duration     `yaml:"build_timeout" json:"build_timeout" validate:"min=1m,max=2h"`
	JavaVersion    string            `yaml:"java_version" json:"java_version" validate:"required"`
	MavenVersion   string            `yaml:"maven_version" json:"maven_version"`
	JavaOpts       string            `yaml:"java_opts" json:"java_opts"`
	MavenOpts      string            `yaml:"maven_opts" json:"maven_opts"`
	Environment    map[string]string `yaml:"environment" json:"environment"`
}

// PythonConfig contains Python language-specific configuration settings.
// Replaces hardcoded values from pkg/python/ package.
type PythonConfig struct {
	BaseImage     string            `yaml:"base_image" json:"base_image" validate:"required"`
	Version       string            `yaml:"version" json:"version" validate:"required"`
	CacheLocation string            `yaml:"cache_location" json:"cache_location" validate:"required"`
	TestTimeout   time.Duration     `yaml:"test_timeout" json:"test_timeout" validate:"min=30s,max=1h"`
	BuildTimeout  time.Duration     `yaml:"build_timeout" json:"build_timeout" validate:"min=1m,max=2h"`
	UVEnabled     bool              `yaml:"uv_enabled" json:"uv_enabled"`
	UVCacheDir    string            `yaml:"uv_cache_dir" json:"uv_cache_dir"`
	PipNoCache    bool              `yaml:"pip_no_cache" json:"pip_no_cache"`
	Requirements  []string          `yaml:"requirements" json:"requirements"`
	Environment   map[string]string `yaml:"environment" json:"environment"`
}

// ProtobufConfig contains Protocol Buffers configuration settings.
// Replaces hardcoded values from pkg/protobuf/ package.
type ProtobufConfig struct {
	BaseImage    string            `yaml:"base_image" json:"base_image" validate:"required"`
	Version      string            `yaml:"version" json:"version" validate:"required"`
	ScriptPath   string            `yaml:"script_path" json:"script_path" validate:"required"`
	OutputDir    string            `yaml:"output_dir" json:"output_dir" validate:"required"`
	SourceMount  string            `yaml:"source_mount" json:"source_mount" validate:"required"`
	Environment  map[string]string `yaml:"environment" json:"environment"`
}

// ContainerConfig contains Docker/container-related configuration settings.
// Replaces hardcoded timeout and container values throughout the codebase.
type ContainerConfig struct {
	Registry   string         `yaml:"registry" json:"registry"`
	Images     ImageConfig    `yaml:"images" json:"images"`
	Timeouts   TimeoutConfig  `yaml:"timeouts" json:"timeouts"`
	Resources  ResourceConfig `yaml:"resources" json:"resources"`
	Volumes    VolumeConfig   `yaml:"volumes" json:"volumes"`
	Runtime    RuntimeConfig  `yaml:"runtime" json:"runtime"`
}

// ImageConfig contains Docker image-related configuration.
type ImageConfig struct {
	PullPolicy string            `yaml:"pull_policy" json:"pull_policy" validate:"oneof=always never if_not_present"`
	BaseImages map[string]string `yaml:"base_images" json:"base_images"`
	TagPolicy  string            `yaml:"tag_policy" json:"tag_policy" validate:"oneof=latest semver commit"`
}

// TimeoutConfig contains timeout settings for various operations.
// Replaces hardcoded timeout values throughout the codebase.
type TimeoutConfig struct {
	Container      time.Duration `yaml:"container" json:"container" validate:"min=5s,max=1h"`
	ContainerStart time.Duration `yaml:"container_start" json:"container_start" validate:"min=5s,max=5m"`
	ContainerStop  time.Duration `yaml:"container_stop" json:"container_stop" validate:"min=1s,max=1m"`
	Build          time.Duration `yaml:"build" json:"build" validate:"min=30s,max=4h"`
	Test           time.Duration `yaml:"test" json:"test" validate:"min=10s,max=2h"`
	Pull           time.Duration `yaml:"pull" json:"pull" validate:"min=30s,max=30m"`
	Push           time.Duration `yaml:"push" json:"push" validate:"min=30s,max=30m"`
	Script         time.Duration `yaml:"script" json:"script" validate:"min=1s,max=1h"`
}

// ResourceConfig contains container resource limits and requests.
type ResourceConfig struct {
	MemoryLimit string `yaml:"memory_limit" json:"memory_limit"`
	MemoryRequest string `yaml:"memory_request" json:"memory_request"`
	CPULimit    string `yaml:"cpu_limit" json:"cpu_limit"`
	CPURequest  string `yaml:"cpu_request" json:"cpu_request"`
	DiskLimit   string `yaml:"disk_limit" json:"disk_limit"`
}

// VolumeConfig contains volume mount configuration.
// Replaces hardcoded mount paths throughout the codebase.
type VolumeConfig struct {
	SourceMount string            `yaml:"source_mount" json:"source_mount" validate:"required"`
	OutputDir   string            `yaml:"output_dir" json:"output_dir" validate:"required"`
	CacheDir    string            `yaml:"cache_dir" json:"cache_dir" validate:"required"`
	TempDir     string            `yaml:"temp_dir" json:"temp_dir" validate:"required"`
	ScriptPath  string            `yaml:"script_path" json:"script_path" validate:"required"`
	CustomMounts map[string]string `yaml:"custom_mounts" json:"custom_mounts"`
}

// RuntimeConfig contains container runtime-specific configuration.
type RuntimeConfig struct {
	Type       string            `yaml:"type" json:"type" validate:"oneof=docker podman"`
	SocketPath string            `yaml:"socket_path" json:"socket_path"`
	Options    map[string]string `yaml:"options" json:"options"`
}

// NetworkConfig contains network-related configuration settings.
type NetworkConfig struct {
	SSHForwarding bool        `yaml:"ssh_forwarding" json:"ssh_forwarding"`
	Proxy         ProxyConfig `yaml:"proxy" json:"proxy"`
	DNS           DNSConfig   `yaml:"dns" json:"dns"`
}

// ProxyConfig contains proxy configuration settings.
type ProxyConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	HTTPProxy  string `yaml:"http_proxy" json:"http_proxy"`
	HTTPSProxy string `yaml:"https_proxy" json:"https_proxy"`
	NoProxy    string `yaml:"no_proxy" json:"no_proxy"`
}

// DNSConfig contains DNS configuration settings.
type DNSConfig struct {
	Servers    []string `yaml:"servers" json:"servers"`
	SearchDomains []string `yaml:"search_domains" json:"search_domains"`
	Options    []string `yaml:"options" json:"options"`
}

// CacheConfig contains cache-related configuration settings.
// Replaces hardcoded cache paths throughout the codebase.
type CacheConfig struct {
	Enabled       bool              `yaml:"enabled" json:"enabled"`
	CleanupPolicy string            `yaml:"cleanup_policy" json:"cleanup_policy"`
	MaxSize       string            `yaml:"max_size" json:"max_size"`
	Directories   CacheDirectories  `yaml:"directories" json:"directories"`
	Permissions   CachePermissions  `yaml:"permissions" json:"permissions"`
}

// CacheDirectories contains cache directory paths for different languages.
type CacheDirectories struct {
	Go     string `yaml:"go" json:"go" validate:"required"`
	Maven  string `yaml:"maven" json:"maven" validate:"required"`
	Python string `yaml:"python" json:"python" validate:"required"`
	Trivy  string `yaml:"trivy" json:"trivy" validate:"required"`
	Docker string `yaml:"docker" json:"docker"`
	Custom map[string]string `yaml:"custom" json:"custom"`
}

// CachePermissions contains cache directory permission settings.
type CachePermissions struct {
	Mode  string `yaml:"mode" json:"mode"`
	Owner string `yaml:"owner" json:"owner"`
	Group string `yaml:"group" json:"group"`
}

// SecurityConfig contains security-related configuration settings.
type SecurityConfig struct {
	UserManagement UserManagementConfig `yaml:"user_management" json:"user_management"`
	Registries     RegistryConfig       `yaml:"registries" json:"registries"`
	Secrets        SecretsConfig        `yaml:"secrets" json:"secrets"`
	Scanning       ScanningConfig       `yaml:"scanning" json:"scanning"`
}

// UserManagementConfig contains user management settings for containers.
// Replaces hardcoded user values in production containers.
type UserManagementConfig struct {
	CreateNonRootUser bool   `yaml:"create_non_root_user" json:"create_non_root_user"`
	UID               string `yaml:"uid" json:"uid" validate:"required"`
	GID               string `yaml:"gid" json:"gid" validate:"required"`
	Username          string `yaml:"username" json:"username" validate:"required"`
	Group             string `yaml:"group" json:"group" validate:"required"`
	Home              string `yaml:"home" json:"home" validate:"required"`
	Shell             string `yaml:"shell" json:"shell"`
}

// RegistryConfig contains container registry configuration.
type RegistryConfig struct {
	VerifyTLS      bool   `yaml:"verify_tls" json:"verify_tls"`
	AuthConfigPath string `yaml:"auth_config_path" json:"auth_config_path"`
	DefaultRegistry string `yaml:"default_registry" json:"default_registry"`
	InsecureRegistries []string `yaml:"insecure_registries" json:"insecure_registries"`
}

// SecretsConfig contains secrets management configuration.
type SecretsConfig struct {
	Provider    string            `yaml:"provider" json:"provider" validate:"oneof=env file vault"`
	FileConfig  SecretsFileConfig `yaml:"file" json:"file"`
	VaultConfig SecretsVaultConfig `yaml:"vault" json:"vault"`
}

// SecretsFileConfig contains file-based secrets configuration.
type SecretsFileConfig struct {
	Path        string `yaml:"path" json:"path"`
	Permissions string `yaml:"permissions" json:"permissions"`
}

// SecretsVaultConfig contains Vault-based secrets configuration.
type SecretsVaultConfig struct {
	Address string `yaml:"address" json:"address"`
	Path    string `yaml:"path" json:"path"`
	Token   string `yaml:"token" json:"token"`
}

// ScanningConfig contains security scanning configuration.
type ScanningConfig struct {
	Trivy TrivyConfig `yaml:"trivy" json:"trivy"`
}

// TrivyConfig contains Trivy security scanner configuration.
// Replaces hardcoded values from pkg/trivy/ package.
type TrivyConfig struct {
	Image        string        `yaml:"image" json:"image" validate:"required"`
	CacheDir     string        `yaml:"cache_dir" json:"cache_dir" validate:"required"`
	Severity     []string      `yaml:"severity" json:"severity"`
	IgnoreUnfixed bool         `yaml:"ignore_unfixed" json:"ignore_unfixed"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout" validate:"min=30s,max=30m"`
	Scanners     []string      `yaml:"scanners" json:"scanners"`
	Format       string        `yaml:"format" json:"format" validate:"oneof=json table sarif"`
	OutputPath   string        `yaml:"output_path" json:"output_path"`
	Environment  map[string]string `yaml:"environment" json:"environment"`
}

// LoggingConfig contains logging configuration settings.
type LoggingConfig struct {
	Level          string            `yaml:"level" json:"level" validate:"oneof=debug info warn error"`
	Format         string            `yaml:"format" json:"format" validate:"oneof=text json structured"`
	Output         string            `yaml:"output" json:"output" validate:"oneof=stdout stderr file"`
	FilePath       string            `yaml:"file_path" json:"file_path"`
	MaxSize        string            `yaml:"max_size" json:"max_size"`
	MaxBackups     int               `yaml:"max_backups" json:"max_backups"`
	MaxAge         string            `yaml:"max_age" json:"max_age"`
	Compress       bool              `yaml:"compress" json:"compress"`
	AddSource      bool              `yaml:"add_source" json:"add_source"`
	SampleRate     float64           `yaml:"sample_rate" json:"sample_rate" validate:"min=0,max=1"`
	ProgressFormat string            `yaml:"progress_format" json:"progress_format"`
	CustomFields   map[string]string `yaml:"custom_fields" json:"custom_fields"`
}

// EnvironmentConfig contains environment-specific configuration settings.
type EnvironmentConfig struct {
	Type     container.EnvType `yaml:"type" json:"type" validate:"oneof=local build production"`
	Profiles ProfilesConfig    `yaml:"profiles" json:"profiles"`
}

// ProfilesConfig contains configuration profiles for different environments.
type ProfilesConfig struct {
	Local      EnvironmentProfile `yaml:"local" json:"local"`
	Build      EnvironmentProfile `yaml:"build" json:"build"`
	Production EnvironmentProfile `yaml:"production" json:"production"`
}

// EnvironmentProfile contains settings for a specific environment profile.
type EnvironmentProfile struct {
	Verbose                bool              `yaml:"verbose" json:"verbose"`
	PullPolicy            string            `yaml:"pull_policy" json:"pull_policy"`
	SecurityHardening     bool              `yaml:"security_hardening" json:"security_hardening"`
	ResourceLimitsEnforced bool             `yaml:"resource_limits_enforced" json:"resource_limits_enforced"`
	LogLevel              string            `yaml:"log_level" json:"log_level"`
	CacheEnabled          bool              `yaml:"cache_enabled" json:"cache_enabled"`
	ParallelBuilds        int               `yaml:"parallel_builds" json:"parallel_builds"`
	CustomSettings        map[string]interface{} `yaml:"custom_settings" json:"custom_settings"`
}

// LoadOptions contains options for configuration loading.
type LoadOptions struct {
	ConfigFile      string
	Environment     container.EnvType
	IgnoreEnvVars   bool
	IgnoreConfigFile bool
	ValidateConfig  bool
	MergeDefaults   bool
}

// ValidationResult contains the result of configuration validation.
type ValidationResult struct {
	IsValid bool
	Errors  []ValidationError
	Warnings []ValidationWarning
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
	Code    string
}

// ValidationWarning represents a configuration validation warning.
type ValidationWarning struct {
	Field   string
	Value   interface{}
	Message string
	Code    string
}

// ConfigUpdate represents a configuration update operation.
type ConfigUpdate struct {
	Path      string
	OldValue  interface{}
	NewValue  interface{}
	Source    string
	Timestamp time.Time
}

// ConfigWatch represents configuration change monitoring.
type ConfigWatch struct {
	Path     string
	Callback func(ConfigUpdate)
	Active   bool
}