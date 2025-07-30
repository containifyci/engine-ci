// This file implements the ConfigProvider interface for centralized configuration
// management in engine-ci.
//
// The Provider type enables:
//
//   - Type-safe configuration access with automatic type conversion
//   - Dot notation for nested configuration values (e.g., "languages.python.base_image")
//   - Default value handling for missing configuration keys
//   - Environment variable override support
//   - Configuration validation and reloading
//
// The provider eliminates scattered configuration access patterns throughout
// the codebase by centralizing all configuration operations through a single,
// well-defined interface.
//
// Example usage:
//
//	provider := NewProvider(config)
//	
//	// Type-safe access with error handling
//	pythonImage, err := provider.GetString("languages.python.base_image")
//	if err != nil {
//	    log.Printf("Using default: %v", err)
//	}
//	
//	// Access with defaults
//	timeout := provider.GetDurationWithDefault("build.timeout", 30*time.Minute)
//	
//	// Validate entire configuration
//	if err := provider.Validate(); err != nil {
//	    log.Fatalf("Invalid configuration: %v", err)
//	}
package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/build"
)

// Provider implements the ConfigProvider interface for centralized configuration management
type Provider struct {
	config *Config
}

// NewProvider creates a new configuration provider with the given configuration
func NewProvider(config *Config) *Provider {
	return &Provider{
		config: config,
	}
}

// Get retrieves a configuration value by key using dot notation (e.g., "container.runtime")
func (p *Provider) Get(key string) (interface{}, error) {
	return p.getValue(key)
}

// Has checks if a configuration key exists
func (p *Provider) Has(key string) bool {
	_, err := p.getValue(key)
	return err == nil
}

// GetString retrieves a string configuration value
func (p *Provider) GetString(key string) (string, error) {
	value, err := p.getValue(key)
	if err != nil {
		return "", err
	}

	if str, ok := value.(string); ok {
		return str, nil
	}

	return "", fmt.Errorf("configuration key %s is not a string (got %T)", key, value)
}

// GetInt retrieves an integer configuration value
func (p *Provider) GetInt(key string) (int, error) {
	value, err := p.getValue(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("configuration key %s is not an integer (got %T)", key, value)
	}
}

// GetBool retrieves a boolean configuration value
func (p *Provider) GetBool(key string) (bool, error) {
	value, err := p.getValue(key)
	if err != nil {
		return false, err
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("configuration key %s is not a boolean (got %T)", key, value)
	}
}

// GetDuration retrieves a duration configuration value
func (p *Provider) GetDuration(key string) (time.Duration, error) {
	value, err := p.getValue(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case time.Duration:
		return v, nil
	case string:
		return time.ParseDuration(v)
	case int64:
		return time.Duration(v), nil
	case float64:
		return time.Duration(v), nil
	default:
		return 0, fmt.Errorf("configuration key %s is not a duration (got %T)", key, value)
	}
}

// GetStringSlice retrieves a string slice configuration value
func (p *Provider) GetStringSlice(key string) ([]string, error) {
	value, err := p.getValue(key)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case []string:
		return v, nil
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			if str, ok := item.(string); ok {
				result[i] = str
			} else {
				return nil, fmt.Errorf("configuration key %s contains non-string value at index %d", key, i)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("configuration key %s is not a string slice (got %T)", key, value)
	}
}

// GetStringMap retrieves a string map configuration value
func (p *Provider) GetStringMap(key string) (map[string]string, error) {
	value, err := p.getValue(key)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case map[string]string:
		return v, nil
	case map[string]interface{}:
		result := make(map[string]string)
		for k, val := range v {
			if str, ok := val.(string); ok {
				result[k] = str
			} else {
				return nil, fmt.Errorf("configuration key %s contains non-string value for key %s", key, k)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("configuration key %s is not a string map (got %T)", key, value)
	}
}

// GetStringWithDefault retrieves a string value with a default fallback
func (p *Provider) GetStringWithDefault(key, defaultValue string) string {
	if value, err := p.GetString(key); err == nil {
		return value
	}
	return defaultValue
}

// GetIntWithDefault retrieves an integer value with a default fallback
func (p *Provider) GetIntWithDefault(key string, defaultValue int) int {
	if value, err := p.GetInt(key); err == nil {
		return value
	}
	return defaultValue
}

// GetBoolWithDefault retrieves a boolean value with a default fallback
func (p *Provider) GetBoolWithDefault(key string, defaultValue bool) bool {
	if value, err := p.GetBool(key); err == nil {
		return value
	}
	return defaultValue
}

// GetDurationWithDefault retrieves a duration value with a default fallback
func (p *Provider) GetDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	if value, err := p.GetDuration(key); err == nil {
		return value
	}
	return defaultValue
}

// Validate validates the entire configuration
func (p *Provider) Validate() error {
	return p.config.Validate()
}

// Reload reloads the configuration from the source file
func (p *Provider) Reload() error {
	if p.config.GetConfigPath() == "" {
		return fmt.Errorf("no configuration file path set")
	}

	newConfig, err := LoadConfig(p.config.GetConfigPath())
	if err != nil {
		return err
	}

	p.config = newConfig
	return nil
}

// GetConfigPath returns the path to the configuration file
func (p *Provider) GetConfigPath() string {
	return p.config.GetConfigPath()
}

// getValue retrieves a value from the configuration using dot notation
func (p *Provider) getValue(key string) (interface{}, error) {
	parts := strings.Split(key, ".")
	current := reflect.ValueOf(p.config).Elem()

	for i, part := range parts {
		// Handle map access
		if current.Kind() == reflect.Map {
			mapValue := current.MapIndex(reflect.ValueOf(part))
			if !mapValue.IsValid() {
				return nil, fmt.Errorf("configuration key not found: %s", key)
			}
			current = mapValue
			continue
		}

		// Handle pointer dereferencing
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return nil, fmt.Errorf("configuration key not found: %s", key)
			}
			current = current.Elem()
		}

		// Handle struct field access
		if current.Kind() != reflect.Struct {
			return nil, fmt.Errorf("invalid configuration path at %s: expected struct, got %s", strings.Join(parts[:i+1], "."), current.Kind())
		}

		// Convert snake_case or kebab-case to PascalCase
		fieldName := toPascalCase(part)
		field := current.FieldByName(fieldName)
		if !field.IsValid() {
			return nil, fmt.Errorf("configuration key not found: %s", key)
		}

		current = field
	}

	return current.Interface(), nil
}

// toPascalCase converts snake_case or kebab-case strings to PascalCase
func toPascalCase(s string) string {
	// Replace underscores and hyphens with spaces
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	// Split by spaces and capitalize each word
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, "")
}

// Utility functions for environment variable parsing

func parseBoolEnv(envVar string, defaultValue bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(envVar)))
	switch value {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func parseIntEnv(value string, defaultValue int) int {
	if i, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return i
	}
	return defaultValue
}

// Ensure Provider implements ConfigProvider interface
var _ build.ConfigProvider = (*Provider)(nil)