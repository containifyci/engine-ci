package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/containifyci/engine-ci/pkg/container"
)

// Global configuration instance with thread-safe access
var (
	globalConfig *Config
	configMutex  sync.RWMutex
	configOnce   sync.Once
)

// ConfigLoader handles loading configuration from multiple sources.
type ConfigLoader struct {
	options LoadOptions
}

// NewConfigLoader creates a new configuration loader with the specified options.
func NewConfigLoader(options LoadOptions) *ConfigLoader {
	return &ConfigLoader{
		options: options,
	}
}

// LoadConfig loads configuration from all available sources in priority order:
// 1. CLI flags (handled externally via options)
// 2. Environment variables
// 3. Configuration file
// 4. Default values
func LoadConfig() (*Config, error) {
	return LoadConfigWithOptions(LoadOptions{
		ValidateConfig: true,
		MergeDefaults:  true,
	})
}

// LoadConfigWithOptions loads configuration with the specified options.
func LoadConfigWithOptions(options LoadOptions) (*Config, error) {
	loader := NewConfigLoader(options)
	return loader.Load()
}

// Load loads the configuration using the configured options.
func (l *ConfigLoader) Load() (*Config, error) {
	var config *Config
	var err error

	// Step 1: Start with defaults
	if l.options.MergeDefaults {
		if l.options.Environment != "" {
			config = GetEnvironmentDefaults(l.options.Environment)
		} else {
			config = GetDefaultConfig()
		}
	} else {
		config = &Config{}
	}

	// Step 2: Load from configuration file
	if !l.options.IgnoreConfigFile {
		fileConfig, err := l.loadFromFile()
		if err != nil {
			// Don't fail if config file is not found, just log a warning
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load configuration file: %w", err)
			}
		} else if fileConfig != nil {
			config = mergeConfigs(config, fileConfig)
		}
	}

	// Step 3: Load from environment variables
	if !l.options.IgnoreEnvVars {
		err = l.loadFromEnvironment(config)
		if err != nil {
			return nil, fmt.Errorf("failed to load environment variables: %w", err)
		}
	}

	// Step 4: Validate configuration
	if l.options.ValidateConfig {
		if err := ValidateConfig(config); err != nil {
			return nil, fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML or JSON file.
func (l *ConfigLoader) loadFromFile() (*Config, error) {
	configPath := l.getConfigFilePath()
	if configPath == "" {
		return nil, nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	config := &Config{}
	
	// Determine file format based on extension
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config file %s: %w", configPath, err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config file %s: %w", configPath, err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, config); err != nil {
			if jsonErr := json.Unmarshal(data, config); jsonErr != nil {
				return nil, fmt.Errorf("failed to parse config file %s as YAML or JSON: %w", configPath, err)
			}
		}
	}

	return config, nil
}

// getConfigFilePath returns the path to the configuration file to use.
// Checks in order: command line option, environment variable, standard locations.
func (l *ConfigLoader) getConfigFilePath() string {
	// Use explicit config file if specified
	if l.options.ConfigFile != "" {
		return l.options.ConfigFile
	}

	// Check environment variable
	if envPath := os.Getenv("ENGINE_CI_CONFIG"); envPath != "" {
		return envPath
	}

	// Check standard locations
	standardPaths := []string{
		"./engine-ci.yaml",
		"./engine-ci.yml",
		"./engine-ci.json",
		"~/.engine-ci.yaml",
		"~/.engine-ci.yml",
		"~/.engine-ci.json",
		"/etc/engine-ci/config.yaml",
		"/etc/engine-ci/config.yml",
		"/etc/engine-ci/config.json",
	}

	for _, path := range standardPaths {
		// Expand home directory
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				path = filepath.Join(home, path[2:])
			}
		}

		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// loadFromEnvironment loads configuration from environment variables.
func (l *ConfigLoader) loadFromEnvironment(config *Config) error {
	return l.loadEnvVarsIntoStruct("ENGINE_CI", reflect.ValueOf(config).Elem())
}

// loadEnvVarsIntoStruct recursively loads environment variables into a struct.
func (l *ConfigLoader) loadEnvVarsIntoStruct(prefix string, v reflect.Value) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get field name for environment variable
		envName := l.getEnvVarName(prefix, fieldType)

		// Handle different field types
		switch field.Kind() {
		case reflect.Struct:
			// Recurse into nested structs
			if err := l.loadEnvVarsIntoStruct(envName, field); err != nil {
				return err
			}
		case reflect.Ptr:
			// Handle pointer types
			if field.Type().Elem().Kind() == reflect.Struct {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				if err := l.loadEnvVarsIntoStruct(envName, field.Elem()); err != nil {
					return err
				}
			}
		default:
			// Handle primitive types
			if err := l.setFieldFromEnv(field, envName); err != nil {
				return fmt.Errorf("failed to set field %s from environment: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// getEnvVarName generates the environment variable name for a field.
func (l *ConfigLoader) getEnvVarName(prefix string, field reflect.StructField) string {
	// Check for explicit yaml/json tags
	yamlTag := field.Tag.Get("yaml")
	if yamlTag != "" && yamlTag != "-" {
		name := strings.Split(yamlTag, ",")[0]
		if name != "" {
			return prefix + "_" + strings.ToUpper(strings.ReplaceAll(name, ".", "_"))
		}
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		name := strings.Split(jsonTag, ",")[0]
		if name != "" {
			return prefix + "_" + strings.ToUpper(strings.ReplaceAll(name, ".", "_"))
		}
	}

	// Use field name as fallback
	return prefix + "_" + strings.ToUpper(field.Name)
}

// setFieldFromEnv sets a field value from an environment variable.
func (l *ConfigLoader) setFieldFromEnv(field reflect.Value, envName string) error {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return nil // Environment variable not set
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(envValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(envValue)
		if err != nil {
			return fmt.Errorf("invalid boolean value %s for %s: %w", envValue, envName, err)
		}
		field.SetBool(boolValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle time.Duration specially
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(envValue)
			if err != nil {
				return fmt.Errorf("invalid duration value %s for %s: %w", envValue, envName, err)
			}
			field.SetInt(int64(duration))
		} else {
			intValue, err := strconv.ParseInt(envValue, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer value %s for %s: %w", envValue, envName, err)
			}
			field.SetInt(intValue)
		}
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(envValue, 64)
		if err != nil {
			return fmt.Errorf("invalid float value %s for %s: %w", envValue, envName, err)
		}
		field.SetFloat(floatValue)
	case reflect.Slice:
		// Handle string slices
		if field.Type().Elem().Kind() == reflect.String {
			values := strings.Split(envValue, ",")
			for i, v := range values {
				values[i] = strings.TrimSpace(v)
			}
			field.Set(reflect.ValueOf(values))
		}
	case reflect.Map:
		// Handle string maps
		if field.Type().Key().Kind() == reflect.String && field.Type().Elem().Kind() == reflect.String {
			m := make(map[string]string)
			pairs := strings.Split(envValue, ",")
			for _, pair := range pairs {
				kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
				if len(kv) == 2 {
					m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
			field.Set(reflect.ValueOf(m))
		}
	}

	return nil
}

// GetGlobalConfig returns the global configuration instance.
// The configuration is loaded once and cached for subsequent calls.
func GetGlobalConfig() *Config {
	configOnce.Do(func() {
		config, err := LoadConfig()
		if err != nil {
			// If loading fails, use defaults
			config = GetDefaultConfig()
		}
		
		configMutex.Lock()
		globalConfig = config
		configMutex.Unlock()
	})

	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

// SetGlobalConfig sets the global configuration instance.
// This is useful for testing or when configuration is loaded externally.
func SetGlobalConfig(config *Config) {
	configMutex.Lock()
	defer configMutex.Unlock()
	globalConfig = config
}

// ReloadGlobalConfig reloads the global configuration from all sources.
func ReloadGlobalConfig() error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	configMutex.Lock()
	globalConfig = config
	configMutex.Unlock()

	return nil
}

// LoadConfigFromFile loads configuration from a specific file path.
func LoadConfigFromFile(path string) (*Config, error) {
	return LoadConfigWithOptions(LoadOptions{
		ConfigFile:     path,
		ValidateConfig: true,
		MergeDefaults:  true,
	})
}

// LoadConfigForEnvironment loads environment-specific configuration.
func LoadConfigForEnvironment(env container.EnvType) (*Config, error) {
	return LoadConfigWithOptions(LoadOptions{
		Environment:    env,
		ValidateConfig: true,
		MergeDefaults:  true,
	})
}

// WriteConfigToFile writes the configuration to a file in the specified format.
func WriteConfigToFile(config *Config, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file %s: %w", path, err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		if err := encoder.Encode(config); err != nil {
			return fmt.Errorf("failed to write YAML config: %w", err)
		}
		encoder.Close()
	case ".json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(config); err != nil {
			return fmt.Errorf("failed to write JSON config: %w", err)
		}
	default:
		// Default to YAML
		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		if err := encoder.Encode(config); err != nil {
			return fmt.Errorf("failed to write YAML config: %w", err)
		}
		encoder.Close()
	}

	return nil
}

// GetConfigValue retrieves a configuration value by path (e.g., "language.go.version").
func GetConfigValue(config *Config, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	value := reflect.ValueOf(config)

	for _, part := range parts {
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		
		if value.Kind() != reflect.Struct {
			return nil, fmt.Errorf("invalid path: %s is not a struct", path)
		}

		field := value.FieldByName(strings.Title(part))
		if !field.IsValid() {
			return nil, fmt.Errorf("field %s not found", part)
		}
		
		value = field
	}

	return value.Interface(), nil
}

// SetConfigValue sets a configuration value by path (e.g., "language.go.version").
func SetConfigValue(config *Config, path string, newValue interface{}) error {
	parts := strings.Split(path, ".")
	value := reflect.ValueOf(config)

	// Navigate to the parent struct
	for i, part := range parts[:len(parts)-1] {
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		
		if value.Kind() != reflect.Struct {
			return fmt.Errorf("invalid path: %s is not a struct", strings.Join(parts[:i+1], "."))
		}

		field := value.FieldByName(strings.Title(part))
		if !field.IsValid() {
			return fmt.Errorf("field %s not found", part)
		}
		
		value = field
	}

	// Set the final field
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	
	lastPart := parts[len(parts)-1]
	field := value.FieldByName(strings.Title(lastPart))
	if !field.IsValid() {
		return fmt.Errorf("field %s not found", lastPart)
	}
	
	if !field.CanSet() {
		return fmt.Errorf("field %s cannot be set", lastPart)
	}

	newVal := reflect.ValueOf(newValue)
	if !newVal.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("cannot assign %T to %s", newValue, field.Type())
	}

	field.Set(newVal)
	return nil
}

// PrintConfig prints the configuration in a human-readable format.
func PrintConfig(config *Config, format string) error {
	switch strings.ToLower(format) {
	case "yaml", "yml":
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		if err := encoder.Encode(config); err != nil {
			return fmt.Errorf("failed to print YAML config: %w", err)
		}
		encoder.Close()
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(config); err != nil {
			return fmt.Errorf("failed to print JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	return nil
}