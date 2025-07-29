package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/container"
)

// EnvironmentMapper handles mapping between configuration fields and environment variables.
type EnvironmentMapper struct {
	prefix string
}

// NewEnvironmentMapper creates a new environment variable mapper with the specified prefix.
func NewEnvironmentMapper(prefix string) *EnvironmentMapper {
	return &EnvironmentMapper{
		prefix: prefix,
	}
}

// GetEnvironmentVariables returns a map of all engine-ci environment variables and their descriptions.
func GetEnvironmentVariables() map[string]string {
	return map[string]string{
		// Configuration file and general settings
		"ENGINE_CI_CONFIG":         "Path to configuration file (YAML/JSON)",
		"ENGINE_CI_ENVIRONMENT":    "Environment type (local, build, production)",
		"ENGINE_CI_LOG_LEVEL":      "Logging level (debug, info, warn, error)",
		"ENGINE_CI_VERBOSE":        "Enable verbose logging (true/false)",

		// Go language configuration
		"ENGINE_CI_LANGUAGE_GO_VERSION":       "Go language version (e.g., 1.24.2)",
		"ENGINE_CI_LANGUAGE_GO_LINT_IMAGE":    "Go linting Docker image (e.g., golangci/golangci-lint:v2.1.2)",
		"ENGINE_CI_LANGUAGE_GO_TEST_TIMEOUT":  "Go test timeout (e.g., 120s, 2m)",
		"ENGINE_CI_LANGUAGE_GO_BUILD_TIMEOUT": "Go build timeout (e.g., 10m, 1h)",
		"ENGINE_CI_LANGUAGE_GO_COVERAGE_MODE": "Go coverage mode (binary, text)",
		"ENGINE_CI_LANGUAGE_GO_BUILD_TAGS":    "Go build tags (comma-separated)",
		"ENGINE_CI_LANGUAGE_GO_PROJECT_MOUNT": "Go project mount path (e.g., /src)",
		"ENGINE_CI_LANGUAGE_GO_OUTPUT_DIR":    "Go output directory (e.g., /out)",

		// Go variant configurations
		"ENGINE_CI_LANGUAGE_GO_VARIANTS_ALPINE_BASE_IMAGE":    "Go Alpine variant base image",
		"ENGINE_CI_LANGUAGE_GO_VARIANTS_DEBIAN_BASE_IMAGE":    "Go Debian variant base image",
		"ENGINE_CI_LANGUAGE_GO_VARIANTS_DEBIAN_CGO_BASE_IMAGE": "Go Debian CGO variant base image",

		// Maven/Java language configuration
		"ENGINE_CI_LANGUAGE_MAVEN_PROD_IMAGE":     "Maven production Docker image",
		"ENGINE_CI_LANGUAGE_MAVEN_BASE_IMAGE":     "Maven base Docker image",
		"ENGINE_CI_LANGUAGE_MAVEN_CACHE_LOCATION": "Maven cache directory (e.g., /root/.m2/)",
		"ENGINE_CI_LANGUAGE_MAVEN_TEST_TIMEOUT":   "Maven test timeout",
		"ENGINE_CI_LANGUAGE_MAVEN_BUILD_TIMEOUT":  "Maven build timeout",
		"ENGINE_CI_LANGUAGE_MAVEN_JAVA_VERSION":   "Java version for Maven builds",
		"ENGINE_CI_LANGUAGE_MAVEN_MAVEN_VERSION":  "Maven version",
		"ENGINE_CI_LANGUAGE_MAVEN_JAVA_OPTS":      "Java JVM options",
		"ENGINE_CI_LANGUAGE_MAVEN_MAVEN_OPTS":     "Maven options",

		// Python language configuration
		"ENGINE_CI_LANGUAGE_PYTHON_BASE_IMAGE":     "Python base Docker image",
		"ENGINE_CI_LANGUAGE_PYTHON_VERSION":        "Python version (e.g., 3.11)",
		"ENGINE_CI_LANGUAGE_PYTHON_CACHE_LOCATION": "Python cache directory (e.g., /root/.cache/pip)",
		"ENGINE_CI_LANGUAGE_PYTHON_TEST_TIMEOUT":   "Python test timeout",
		"ENGINE_CI_LANGUAGE_PYTHON_BUILD_TIMEOUT":  "Python build timeout",
		"ENGINE_CI_LANGUAGE_PYTHON_UV_ENABLED":     "Enable UV package manager (true/false)",
		"ENGINE_CI_LANGUAGE_PYTHON_UV_CACHE_DIR":   "UV cache directory",
		"ENGINE_CI_LANGUAGE_PYTHON_PIP_NO_CACHE":   "Disable pip cache (true/false)",

		// Protobuf configuration
		"ENGINE_CI_LANGUAGE_PROTOBUF_BASE_IMAGE":   "Protobuf base Docker image",
		"ENGINE_CI_LANGUAGE_PROTOBUF_VERSION":      "Protobuf version",
		"ENGINE_CI_LANGUAGE_PROTOBUF_SCRIPT_PATH":  "Protobuf script path",
		"ENGINE_CI_LANGUAGE_PROTOBUF_OUTPUT_DIR":   "Protobuf output directory",
		"ENGINE_CI_LANGUAGE_PROTOBUF_SOURCE_MOUNT": "Protobuf source mount path",

		// Container configuration
		"ENGINE_CI_CONTAINER_REGISTRY":              "Default container registry",
		"ENGINE_CI_CONTAINER_IMAGES_PULL_POLICY":    "Image pull policy (always, never, if_not_present)",
		"ENGINE_CI_CONTAINER_IMAGES_TAG_POLICY":     "Image tag policy (latest, semver, commit)",
		"ENGINE_CI_CONTAINER_RUNTIME_TYPE":          "Container runtime type (docker, podman)",
		"ENGINE_CI_CONTAINER_RUNTIME_SOCKET_PATH":   "Container runtime socket path",

		// Container timeouts
		"ENGINE_CI_CONTAINER_TIMEOUTS_CONTAINER":       "General container operation timeout",
		"ENGINE_CI_CONTAINER_TIMEOUTS_CONTAINER_START": "Container start timeout",
		"ENGINE_CI_CONTAINER_TIMEOUTS_CONTAINER_STOP":  "Container stop timeout",
		"ENGINE_CI_CONTAINER_TIMEOUTS_BUILD":           "Build operation timeout",
		"ENGINE_CI_CONTAINER_TIMEOUTS_TEST":            "Test operation timeout",
		"ENGINE_CI_CONTAINER_TIMEOUTS_PULL":            "Image pull timeout",
		"ENGINE_CI_CONTAINER_TIMEOUTS_PUSH":            "Image push timeout",
		"ENGINE_CI_CONTAINER_TIMEOUTS_SCRIPT":          "Script execution timeout",

		// Container resources
		"ENGINE_CI_CONTAINER_RESOURCES_MEMORY_LIMIT":   "Container memory limit (e.g., 2GB, 1Gi)",
		"ENGINE_CI_CONTAINER_RESOURCES_MEMORY_REQUEST": "Container memory request",
		"ENGINE_CI_CONTAINER_RESOURCES_CPU_LIMIT":      "Container CPU limit (e.g., 2, 1.5, 1000m)",
		"ENGINE_CI_CONTAINER_RESOURCES_CPU_REQUEST":    "Container CPU request",
		"ENGINE_CI_CONTAINER_RESOURCES_DISK_LIMIT":     "Container disk limit",

		// Container volumes
		"ENGINE_CI_CONTAINER_VOLUMES_SOURCE_MOUNT": "Source code mount path",
		"ENGINE_CI_CONTAINER_VOLUMES_OUTPUT_DIR":   "Build output directory",
		"ENGINE_CI_CONTAINER_VOLUMES_CACHE_DIR":    "Cache directory",
		"ENGINE_CI_CONTAINER_VOLUMES_TEMP_DIR":     "Temporary directory",
		"ENGINE_CI_CONTAINER_VOLUMES_SCRIPT_PATH":  "Script file path",

		// Network configuration
		"ENGINE_CI_NETWORK_SSH_FORWARDING":   "Enable SSH agent forwarding (true/false)",
		"ENGINE_CI_NETWORK_PROXY_ENABLED":    "Enable proxy support (true/false)",
		"ENGINE_CI_NETWORK_PROXY_HTTP_PROXY": "HTTP proxy URL",
		"ENGINE_CI_NETWORK_PROXY_HTTPS_PROXY": "HTTPS proxy URL",
		"ENGINE_CI_NETWORK_PROXY_NO_PROXY":   "No proxy hosts (comma-separated)",

		// Cache configuration
		"ENGINE_CI_CACHE_ENABLED":        "Enable caching (true/false)",
		"ENGINE_CI_CACHE_CLEANUP_POLICY": "Cache cleanup policy (e.g., 30d, 1w)",
		"ENGINE_CI_CACHE_MAX_SIZE":       "Maximum cache size (e.g., 10GB, 5Gi)",

		// Cache directories
		"ENGINE_CI_CACHE_DIRECTORIES_GO":     "Go cache directory",
		"ENGINE_CI_CACHE_DIRECTORIES_MAVEN":  "Maven cache directory",
		"ENGINE_CI_CACHE_DIRECTORIES_PYTHON": "Python cache directory",
		"ENGINE_CI_CACHE_DIRECTORIES_TRIVY":  "Trivy cache directory",
		"ENGINE_CI_CACHE_DIRECTORIES_DOCKER": "Docker cache directory",

		// Security configuration
		"ENGINE_CI_SECURITY_USER_MANAGEMENT_CREATE_NON_ROOT_USER": "Create non-root user in containers (true/false)",
		"ENGINE_CI_SECURITY_USER_MANAGEMENT_UID":                  "User ID for non-root user",
		"ENGINE_CI_SECURITY_USER_MANAGEMENT_GID":                  "Group ID for non-root user",
		"ENGINE_CI_SECURITY_USER_MANAGEMENT_USERNAME":             "Username for non-root user",
		"ENGINE_CI_SECURITY_USER_MANAGEMENT_GROUP":                "Group name for non-root user",
		"ENGINE_CI_SECURITY_USER_MANAGEMENT_HOME":                 "Home directory for non-root user",
		"ENGINE_CI_SECURITY_USER_MANAGEMENT_SHELL":                "Shell for non-root user",

		// Registry security
		"ENGINE_CI_SECURITY_REGISTRIES_VERIFY_TLS":       "Verify TLS for container registries (true/false)",
		"ENGINE_CI_SECURITY_REGISTRIES_AUTH_CONFIG_PATH": "Path to registry authentication config",
		"ENGINE_CI_SECURITY_REGISTRIES_DEFAULT_REGISTRY": "Default container registry",

		// Trivy security scanning
		"ENGINE_CI_SECURITY_SCANNING_TRIVY_IMAGE":         "Trivy scanner Docker image",
		"ENGINE_CI_SECURITY_SCANNING_TRIVY_CACHE_DIR":     "Trivy cache directory",
		"ENGINE_CI_SECURITY_SCANNING_TRIVY_SEVERITY":      "Trivy severity levels (comma-separated)",
		"ENGINE_CI_SECURITY_SCANNING_TRIVY_IGNORE_UNFIXED": "Ignore unfixed vulnerabilities (true/false)",
		"ENGINE_CI_SECURITY_SCANNING_TRIVY_TIMEOUT":       "Trivy scan timeout",
		"ENGINE_CI_SECURITY_SCANNING_TRIVY_FORMAT":        "Trivy output format (json, table, sarif)",
		"ENGINE_CI_SECURITY_SCANNING_TRIVY_OUTPUT_PATH":   "Trivy output file path",

		// Logging configuration
		"ENGINE_CI_LOGGING_LEVEL":           "Log level (debug, info, warn, error)",
		"ENGINE_CI_LOGGING_FORMAT":          "Log format (text, json, structured)",
		"ENGINE_CI_LOGGING_OUTPUT":          "Log output (stdout, stderr, file)",
		"ENGINE_CI_LOGGING_FILE_PATH":       "Log file path (when output=file)",
		"ENGINE_CI_LOGGING_MAX_SIZE":        "Maximum log file size",
		"ENGINE_CI_LOGGING_MAX_BACKUPS":     "Maximum number of log file backups",
		"ENGINE_CI_LOGGING_MAX_AGE":         "Maximum age of log files",
		"ENGINE_CI_LOGGING_COMPRESS":        "Compress old log files (true/false)",
		"ENGINE_CI_LOGGING_ADD_SOURCE":      "Add source file info to logs (true/false)",
		"ENGINE_CI_LOGGING_SAMPLE_RATE":     "Log sampling rate (0.0-1.0)",
		"ENGINE_CI_LOGGING_PROGRESS_FORMAT": "Progress logging format",

		// Environment profiles
		"ENGINE_CI_ENVIRONMENT_PROFILES_LOCAL_VERBOSE":                     "Verbose mode in local environment",
		"ENGINE_CI_ENVIRONMENT_PROFILES_LOCAL_PULL_POLICY":                 "Pull policy in local environment",
		"ENGINE_CI_ENVIRONMENT_PROFILES_BUILD_SECURITY_HARDENING":          "Security hardening in build environment",
		"ENGINE_CI_ENVIRONMENT_PROFILES_PRODUCTION_RESOURCE_LIMITS_ENFORCED": "Enforce resource limits in production",
	}
}

// LoadFromEnvironmentVariables loads configuration from environment variables.
func LoadFromEnvironmentVariables(config *Config) error {
	mapper := NewEnvironmentMapper("ENGINE_CI")
	return mapper.LoadIntoConfig(config)
}

// LoadIntoConfig loads environment variables into the provided configuration.
func (em *EnvironmentMapper) LoadIntoConfig(config *Config) error {
	// Load general configuration
	if err := em.loadGeneralConfig(config); err != nil {
		return fmt.Errorf("failed to load general config from environment: %w", err)
	}

	// Load language configuration
	if err := em.loadLanguageConfig(&config.Language); err != nil {
		return fmt.Errorf("failed to load language config from environment: %w", err)
	}

	// Load container configuration
	if err := em.loadContainerConfig(&config.Container); err != nil {
		return fmt.Errorf("failed to load container config from environment: %w", err)
	}

	// Load network configuration
	if err := em.loadNetworkConfig(&config.Network); err != nil {
		return fmt.Errorf("failed to load network config from environment: %w", err)
	}

	// Load cache configuration
	if err := em.loadCacheConfig(&config.Cache); err != nil {
		return fmt.Errorf("failed to load cache config from environment: %w", err)
	}

	// Load security configuration
	if err := em.loadSecurityConfig(&config.Security); err != nil {
		return fmt.Errorf("failed to load security config from environment: %w", err)
	}

	// Load logging configuration
	if err := em.loadLoggingConfig(&config.Logging); err != nil {
		return fmt.Errorf("failed to load logging config from environment: %w", err)
	}

	// Load environment configuration
	if err := em.loadEnvironmentConfig(&config.Environment); err != nil {
		return fmt.Errorf("failed to load environment config from environment: %w", err)
	}

	return nil
}

// loadGeneralConfig loads general configuration from environment variables.
func (em *EnvironmentMapper) loadGeneralConfig(config *Config) error {
	if version := os.Getenv(em.prefix + "_VERSION"); version != "" {
		config.Version = version
	}

	if env := os.Getenv(em.prefix + "_ENVIRONMENT"); env != "" {
		switch env {
		case "local":
			config.Environment.Type = container.LocalEnv
		case "build":
			config.Environment.Type = container.BuildEnv
		case "production":
			config.Environment.Type = container.ProdEnv
		default:
			return fmt.Errorf("invalid environment type: %s", env)
		}
	}

	return nil
}

// loadLanguageConfig loads language configuration from environment variables.
func (em *EnvironmentMapper) loadLanguageConfig(config *LanguageConfig) error {
	// Load Go configuration
	goPrefix := em.prefix + "_LANGUAGE_GO"
	if version := os.Getenv(goPrefix + "_VERSION"); version != "" {
		config.Go.Version = version
	}
	if lintImage := os.Getenv(goPrefix + "_LINT_IMAGE"); lintImage != "" {
		config.Go.LintImage = lintImage
	}
	if timeout := os.Getenv(goPrefix + "_TEST_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.Go.TestTimeout = duration
		}
	}
	if timeout := os.Getenv(goPrefix + "_BUILD_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.Go.BuildTimeout = duration
		}
	}
	if coverageMode := os.Getenv(goPrefix + "_COVERAGE_MODE"); coverageMode != "" {
		config.Go.CoverageMode = coverageMode
	}
	if buildTags := os.Getenv(goPrefix + "_BUILD_TAGS"); buildTags != "" {
		config.Go.BuildTags = strings.Split(buildTags, ",")
	}
	if projectMount := os.Getenv(goPrefix + "_PROJECT_MOUNT"); projectMount != "" {
		config.Go.ProjectMount = projectMount
	}
	if outputDir := os.Getenv(goPrefix + "_OUTPUT_DIR"); outputDir != "" {
		config.Go.OutputDir = outputDir
	}

	// Load Go variant configurations
	variantPrefix := goPrefix + "_VARIANTS"
	if alpineImage := os.Getenv(variantPrefix + "_ALPINE_BASE_IMAGE"); alpineImage != "" {
		config.Go.Variants.Alpine.BaseImage = alpineImage
	}
	if debianImage := os.Getenv(variantPrefix + "_DEBIAN_BASE_IMAGE"); debianImage != "" {
		config.Go.Variants.Debian.BaseImage = debianImage
	}
	if debianCGOImage := os.Getenv(variantPrefix + "_DEBIAN_CGO_BASE_IMAGE"); debianCGOImage != "" {
		config.Go.Variants.DebianCGO.BaseImage = debianCGOImage
	}

	// Load Maven configuration
	mavenPrefix := em.prefix + "_LANGUAGE_MAVEN"
	if prodImage := os.Getenv(mavenPrefix + "_PROD_IMAGE"); prodImage != "" {
		config.Maven.ProdImage = prodImage
	}
	if baseImage := os.Getenv(mavenPrefix + "_BASE_IMAGE"); baseImage != "" {
		config.Maven.BaseImage = baseImage
	}
	if cacheLocation := os.Getenv(mavenPrefix + "_CACHE_LOCATION"); cacheLocation != "" {
		config.Maven.CacheLocation = cacheLocation
	}
	if javaVersion := os.Getenv(mavenPrefix + "_JAVA_VERSION"); javaVersion != "" {
		config.Maven.JavaVersion = javaVersion
	}
	if javaOpts := os.Getenv(mavenPrefix + "_JAVA_OPTS"); javaOpts != "" {
		config.Maven.JavaOpts = javaOpts
	}

	// Load Python configuration
	pythonPrefix := em.prefix + "_LANGUAGE_PYTHON"
	if baseImage := os.Getenv(pythonPrefix + "_BASE_IMAGE"); baseImage != "" {
		config.Python.BaseImage = baseImage
	}
	if version := os.Getenv(pythonPrefix + "_VERSION"); version != "" {
		config.Python.Version = version
	}
	if cacheLocation := os.Getenv(pythonPrefix + "_CACHE_LOCATION"); cacheLocation != "" {
		config.Python.CacheLocation = cacheLocation
	}
	if uvEnabled := os.Getenv(pythonPrefix + "_UV_ENABLED"); uvEnabled != "" {
		if enabled, err := strconv.ParseBool(uvEnabled); err == nil {
			config.Python.UVEnabled = enabled
		}
	}

	// Load Protobuf configuration
	protobufPrefix := em.prefix + "_LANGUAGE_PROTOBUF"
	if baseImage := os.Getenv(protobufPrefix + "_BASE_IMAGE"); baseImage != "" {
		config.Protobuf.BaseImage = baseImage
	}
	if version := os.Getenv(protobufPrefix + "_VERSION"); version != "" {
		config.Protobuf.Version = version
	}

	return nil
}

// loadContainerConfig loads container configuration from environment variables.
func (em *EnvironmentMapper) loadContainerConfig(config *ContainerConfig) error {
	containerPrefix := em.prefix + "_CONTAINER"

	// Load general container settings
	if registry := os.Getenv(containerPrefix + "_REGISTRY"); registry != "" {
		config.Registry = registry
	}

	// Load image configuration
	imagePrefix := containerPrefix + "_IMAGES"
	if pullPolicy := os.Getenv(imagePrefix + "_PULL_POLICY"); pullPolicy != "" {
		config.Images.PullPolicy = pullPolicy
	}
	if tagPolicy := os.Getenv(imagePrefix + "_TAG_POLICY"); tagPolicy != "" {
		config.Images.TagPolicy = tagPolicy
	}

	// Load timeout configuration
	timeoutPrefix := containerPrefix + "_TIMEOUTS"
	timeoutMap := map[string]*time.Duration{
		"CONTAINER":       &config.Timeouts.Container,
		"CONTAINER_START": &config.Timeouts.ContainerStart,
		"CONTAINER_STOP":  &config.Timeouts.ContainerStop,
		"BUILD":           &config.Timeouts.Build,
		"TEST":            &config.Timeouts.Test,
		"PULL":            &config.Timeouts.Pull,
		"PUSH":            &config.Timeouts.Push,
		"SCRIPT":          &config.Timeouts.Script,
	}

	for envSuffix, timeoutField := range timeoutMap {
		if timeout := os.Getenv(timeoutPrefix + "_" + envSuffix); timeout != "" {
			if duration, err := time.ParseDuration(timeout); err == nil {
				*timeoutField = duration
			}
		}
	}

	// Load resource configuration
	resourcePrefix := containerPrefix + "_RESOURCES"
	if memoryLimit := os.Getenv(resourcePrefix + "_MEMORY_LIMIT"); memoryLimit != "" {
		config.Resources.MemoryLimit = memoryLimit
	}
	if memoryRequest := os.Getenv(resourcePrefix + "_MEMORY_REQUEST"); memoryRequest != "" {
		config.Resources.MemoryRequest = memoryRequest
	}
	if cpuLimit := os.Getenv(resourcePrefix + "_CPU_LIMIT"); cpuLimit != "" {
		config.Resources.CPULimit = cpuLimit
	}
	if cpuRequest := os.Getenv(resourcePrefix + "_CPU_REQUEST"); cpuRequest != "" {
		config.Resources.CPURequest = cpuRequest
	}

	// Load volume configuration
	volumePrefix := containerPrefix + "_VOLUMES"
	if sourceMount := os.Getenv(volumePrefix + "_SOURCE_MOUNT"); sourceMount != "" {
		config.Volumes.SourceMount = sourceMount
	}
	if outputDir := os.Getenv(volumePrefix + "_OUTPUT_DIR"); outputDir != "" {
		config.Volumes.OutputDir = outputDir
	}
	if cacheDir := os.Getenv(volumePrefix + "_CACHE_DIR"); cacheDir != "" {
		config.Volumes.CacheDir = cacheDir
	}

	// Load runtime configuration
	runtimePrefix := containerPrefix + "_RUNTIME"
	if runtimeType := os.Getenv(runtimePrefix + "_TYPE"); runtimeType != "" {
		config.Runtime.Type = runtimeType
	}
	if socketPath := os.Getenv(runtimePrefix + "_SOCKET_PATH"); socketPath != "" {
		config.Runtime.SocketPath = socketPath
	}

	return nil
}

// loadNetworkConfig loads network configuration from environment variables.
func (em *EnvironmentMapper) loadNetworkConfig(config *NetworkConfig) error {
	networkPrefix := em.prefix + "_NETWORK"

	if sshForwarding := os.Getenv(networkPrefix + "_SSH_FORWARDING"); sshForwarding != "" {
		if enabled, err := strconv.ParseBool(sshForwarding); err == nil {
			config.SSHForwarding = enabled
		}
	}

	// Load proxy configuration
	proxyPrefix := networkPrefix + "_PROXY"
	if proxyEnabled := os.Getenv(proxyPrefix + "_ENABLED"); proxyEnabled != "" {
		if enabled, err := strconv.ParseBool(proxyEnabled); err == nil {
			config.Proxy.Enabled = enabled
		}
	}
	if httpProxy := os.Getenv(proxyPrefix + "_HTTP_PROXY"); httpProxy != "" {
		config.Proxy.HTTPProxy = httpProxy
	}
	if httpsProxy := os.Getenv(proxyPrefix + "_HTTPS_PROXY"); httpsProxy != "" {
		config.Proxy.HTTPSProxy = httpsProxy
	}
	if noProxy := os.Getenv(proxyPrefix + "_NO_PROXY"); noProxy != "" {
		config.Proxy.NoProxy = noProxy
	}

	return nil
}

// loadCacheConfig loads cache configuration from environment variables.
func (em *EnvironmentMapper) loadCacheConfig(config *CacheConfig) error {
	cachePrefix := em.prefix + "_CACHE"

	if enabled := os.Getenv(cachePrefix + "_ENABLED"); enabled != "" {
		if cacheEnabled, err := strconv.ParseBool(enabled); err == nil {
			config.Enabled = cacheEnabled
		}
	}
	if cleanupPolicy := os.Getenv(cachePrefix + "_CLEANUP_POLICY"); cleanupPolicy != "" {
		config.CleanupPolicy = cleanupPolicy
	}
	if maxSize := os.Getenv(cachePrefix + "_MAX_SIZE"); maxSize != "" {
		config.MaxSize = maxSize
	}

	// Load cache directories
	dirPrefix := cachePrefix + "_DIRECTORIES"
	if goDir := os.Getenv(dirPrefix + "_GO"); goDir != "" {
		config.Directories.Go = goDir
	}
	if mavenDir := os.Getenv(dirPrefix + "_MAVEN"); mavenDir != "" {
		config.Directories.Maven = mavenDir
	}
	if pythonDir := os.Getenv(dirPrefix + "_PYTHON"); pythonDir != "" {
		config.Directories.Python = pythonDir
	}
	if trivyDir := os.Getenv(dirPrefix + "_TRIVY"); trivyDir != "" {
		config.Directories.Trivy = trivyDir
	}

	return nil
}

// loadSecurityConfig loads security configuration from environment variables.
func (em *EnvironmentMapper) loadSecurityConfig(config *SecurityConfig) error {
	securityPrefix := em.prefix + "_SECURITY"

	// Load user management configuration
	userPrefix := securityPrefix + "_USER_MANAGEMENT"
	if createNonRoot := os.Getenv(userPrefix + "_CREATE_NON_ROOT_USER"); createNonRoot != "" {
		if create, err := strconv.ParseBool(createNonRoot); err == nil {
			config.UserManagement.CreateNonRootUser = create
		}
	}
	if uid := os.Getenv(userPrefix + "_UID"); uid != "" {
		config.UserManagement.UID = uid
	}
	if gid := os.Getenv(userPrefix + "_GID"); gid != "" {
		config.UserManagement.GID = gid
	}
	if username := os.Getenv(userPrefix + "_USERNAME"); username != "" {
		config.UserManagement.Username = username
	}

	// Load registry security configuration
	registryPrefix := securityPrefix + "_REGISTRIES"
	if verifyTLS := os.Getenv(registryPrefix + "_VERIFY_TLS"); verifyTLS != "" {
		if verify, err := strconv.ParseBool(verifyTLS); err == nil {
			config.Registries.VerifyTLS = verify
		}
	}
	if authConfigPath := os.Getenv(registryPrefix + "_AUTH_CONFIG_PATH"); authConfigPath != "" {
		config.Registries.AuthConfigPath = authConfigPath
	}

	// Load Trivy scanning configuration
	trivyPrefix := securityPrefix + "_SCANNING_TRIVY"
	if image := os.Getenv(trivyPrefix + "_IMAGE"); image != "" {
		config.Scanning.Trivy.Image = image
	}
	if cacheDir := os.Getenv(trivyPrefix + "_CACHE_DIR"); cacheDir != "" {
		config.Scanning.Trivy.CacheDir = cacheDir
	}
	if severity := os.Getenv(trivyPrefix + "_SEVERITY"); severity != "" {
		config.Scanning.Trivy.Severity = strings.Split(severity, ",")
	}
	if ignoreUnfixed := os.Getenv(trivyPrefix + "_IGNORE_UNFIXED"); ignoreUnfixed != "" {
		if ignore, err := strconv.ParseBool(ignoreUnfixed); err == nil {
			config.Scanning.Trivy.IgnoreUnfixed = ignore
		}
	}

	return nil
}

// loadLoggingConfig loads logging configuration from environment variables.
func (em *EnvironmentMapper) loadLoggingConfig(config *LoggingConfig) error {
	loggingPrefix := em.prefix + "_LOGGING"

	if level := os.Getenv(loggingPrefix + "_LEVEL"); level != "" {
		config.Level = level
	}
	if format := os.Getenv(loggingPrefix + "_FORMAT"); format != "" {
		config.Format = format
	}
	if output := os.Getenv(loggingPrefix + "_OUTPUT"); output != "" {
		config.Output = output
	}
	if filePath := os.Getenv(loggingPrefix + "_FILE_PATH"); filePath != "" {
		config.FilePath = filePath
	}

	return nil
}

// loadEnvironmentConfig loads environment-specific configuration from environment variables.
func (em *EnvironmentMapper) loadEnvironmentConfig(config *EnvironmentConfig) error {
	envPrefix := em.prefix + "_ENVIRONMENT"

	// Environment profiles are typically loaded from configuration files
	// rather than environment variables to avoid excessive variable proliferation
	
	// Load basic environment type override
	if envType := os.Getenv(envPrefix + "_TYPE"); envType != "" {
		switch envType {
		case "local":
			config.Type = container.LocalEnv
		case "build":
			config.Type = container.BuildEnv
		case "production":
			config.Type = container.ProdEnv
		}
	}

	return nil
}

// ExportEnvironmentVariables exports the current configuration as environment variables.
// This is useful for passing configuration to child processes or containers.
func ExportEnvironmentVariables(config *Config) []string {
	var envVars []string
	mapper := NewEnvironmentMapper("ENGINE_CI")

	// Export general configuration
	if config.Version != "" {
		envVars = append(envVars, fmt.Sprintf("%s_VERSION=%s", mapper.prefix, config.Version))
	}

	// Export language configuration
	envVars = append(envVars, mapper.exportLanguageConfig(config.Language)...)
	
	// Export container configuration
	envVars = append(envVars, mapper.exportContainerConfig(config.Container)...)

	// Export other configuration sections as needed
	// ... (implement additional export functions)

	return envVars
}

// exportLanguageConfig exports language configuration as environment variables.
func (em *EnvironmentMapper) exportLanguageConfig(config LanguageConfig) []string {
	var envVars []string

	// Export Go configuration
	goPrefix := em.prefix + "_LANGUAGE_GO"
	if config.Go.Version != "" {
		envVars = append(envVars, fmt.Sprintf("%s_VERSION=%s", goPrefix, config.Go.Version))
	}
	if config.Go.LintImage != "" {
		envVars = append(envVars, fmt.Sprintf("%s_LINT_IMAGE=%s", goPrefix, config.Go.LintImage))
	}
	if config.Go.TestTimeout > 0 {
		envVars = append(envVars, fmt.Sprintf("%s_TEST_TIMEOUT=%s", goPrefix, config.Go.TestTimeout.String()))
	}

	return envVars
}

// exportContainerConfig exports container configuration as environment variables.
func (em *EnvironmentMapper) exportContainerConfig(config ContainerConfig) []string {
	var envVars []string

	containerPrefix := em.prefix + "_CONTAINER"
	if config.Registry != "" {
		envVars = append(envVars, fmt.Sprintf("%s_REGISTRY=%s", containerPrefix, config.Registry))
	}

	// Export timeouts
	timeoutPrefix := containerPrefix + "_TIMEOUTS"
	if config.Timeouts.Container > 0 {
		envVars = append(envVars, fmt.Sprintf("%s_CONTAINER=%s", timeoutPrefix, config.Timeouts.Container.String()))
	}

	return envVars
}

// ValidateEnvironmentVariables validates environment variables without loading them into config.
func ValidateEnvironmentVariables() []string {
	var issues []string
	envVars := GetEnvironmentVariables()

	for envVar := range envVars {
		value := os.Getenv(envVar)
		if value == "" {
			continue // Not set, which is fine
		}

		// Validate specific patterns
		switch {
		case strings.Contains(envVar, "TIMEOUT"):
			if _, err := time.ParseDuration(value); err != nil {
				issues = append(issues, fmt.Sprintf("%s has invalid duration format: %s", envVar, value))
			}
		case strings.Contains(envVar, "ENABLED") || strings.Contains(envVar, "_BOOL"):
			if _, err := strconv.ParseBool(value); err != nil {
				issues = append(issues, fmt.Sprintf("%s has invalid boolean format: %s", envVar, value))
			}
		case strings.Contains(envVar, "VERSION"):
			if !isValidSemanticVersion(value) && !strings.Contains(envVar, "JAVA") {
				issues = append(issues, fmt.Sprintf("%s has invalid version format: %s", envVar, value))
			}
		}
	}

	return issues
}

// PrintEnvironmentVariables prints all available environment variables and their descriptions.
func PrintEnvironmentVariables() {
	envVars := GetEnvironmentVariables()
	
	fmt.Println("Engine-CI Environment Variables:")
	fmt.Println("=================================")
	
	categories := map[string][]string{
		"General":   {},
		"Language":  {},
		"Container": {},
		"Network":   {},
		"Cache":     {},
		"Security":  {},
		"Logging":   {},
	}
	
	for envVar, description := range envVars {
		category := "General"
		switch {
		case strings.Contains(envVar, "_LANGUAGE_"):
			category = "Language"
		case strings.Contains(envVar, "_CONTAINER_"):
			category = "Container"
		case strings.Contains(envVar, "_NETWORK_"):
			category = "Network"
		case strings.Contains(envVar, "_CACHE_"):
			category = "Cache"
		case strings.Contains(envVar, "_SECURITY_"):
			category = "Security"
		case strings.Contains(envVar, "_LOGGING_"):
			category = "Logging"
		}
		
		categories[category] = append(categories[category], fmt.Sprintf("  %s\n    %s", envVar, description))
	}
	
	for category, vars := range categories {
		if len(vars) > 0 {
			fmt.Printf("\n%s:\n", category)
			for _, varDesc := range vars {
				fmt.Println(varDesc)
			}
		}
	}
}