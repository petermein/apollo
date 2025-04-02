package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/petermein/apollo/internal/operators/mysql"
	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Operator struct {
		ID             string `yaml:"id" env:"OPERATOR_ID"`
		EnabledModules string `yaml:"enabled_modules" env:"ENABLED_MODULES"`
	} `yaml:"operator"`

	Modules map[string]interface{} `yaml:"modules"`

	API struct {
		Endpoint      string `yaml:"endpoint" env:"API_ENDPOINT"`
		RetryAttempts int    `yaml:"retry_attempts" env:"API_RETRY_ATTEMPTS"`
		RetryDelay    string `yaml:"retry_delay" env:"API_RETRY_DELAY"`
	} `yaml:"api"`

	Logging struct {
		Level  string `yaml:"level" env:"LOG_LEVEL"`
		Format string `yaml:"format" env:"LOG_FORMAT"`
		Output string `yaml:"output" env:"LOG_OUTPUT"`
	} `yaml:"logging"`

	Health struct {
		Interval string `yaml:"interval" env:"HEALTH_INTERVAL"`
		Timeout  string `yaml:"timeout" env:"HEALTH_TIMEOUT"`
		Retries  int    `yaml:"retries" env:"HEALTH_RETRIES"`
	} `yaml:"health"`
}

// LoadConfig loads configuration from a YAML file and environment variables
func LoadConfig(path string) (*Config, error) {
	// Read YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Apply environment variables
	if err := applyEnvVars(&config); err != nil {
		return nil, fmt.Errorf("failed to apply environment variables: %v", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	return &config, nil
}

// applyEnvVars applies environment variables to the configuration
func applyEnvVars(config *Config) error {
	val := reflect.ValueOf(config).Elem()
	return applyEnvVarsToStruct(val)
}

// applyEnvVarsToStruct recursively applies environment variables to a struct
func applyEnvVarsToStruct(val reflect.Value) error {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := applyEnvVarsToStruct(field); err != nil {
				return err
			}
			continue
		}

		// Get environment variable name from tag
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// Get environment variable value
		envValue := os.Getenv(envTag)
		if envValue == "" {
			continue
		}

		// Set field value based on type
		switch field.Kind() {
		case reflect.String:
			field.SetString(envValue)
		case reflect.Int:
			var intVal int64
			if _, err := fmt.Sscanf(envValue, "%d", &intVal); err != nil {
				return fmt.Errorf("invalid integer value for %s: %v", envTag, err)
			}
			field.SetInt(intVal)
		default:
			return fmt.Errorf("unsupported field type for %s: %v", envTag, field.Kind())
		}
	}
	return nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Operator.ID == "" {
		return fmt.Errorf("operator.id is required")
	}
	if config.Operator.EnabledModules == "" {
		return fmt.Errorf("operator.enabled_modules is required")
	}
	if config.API.Endpoint == "" {
		return fmt.Errorf("api.endpoint is required")
	}
	if config.API.RetryAttempts <= 0 {
		return fmt.Errorf("api.retry_attempts must be positive")
	}
	if config.API.RetryDelay == "" {
		return fmt.Errorf("api.retry_delay is required")
	}
	if config.Health.Interval == "" {
		return fmt.Errorf("health.interval is required")
	}
	if config.Health.Timeout == "" {
		return fmt.Errorf("health.timeout is required")
	}
	if config.Health.Retries <= 0 {
		return fmt.Errorf("health.retries must be positive")
	}
	return nil
}

// GetModuleConfig retrieves the configuration for a specific module
func (c *Config) GetModuleConfig(moduleName string) (interface{}, error) {
	config, ok := c.Modules[moduleName]
	if !ok {
		return nil, fmt.Errorf("no configuration found for module: %s", moduleName)
	}

	// Convert to JSON and back to handle YAML types
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal module config: %v", err)
	}

	// Handle module-specific config types
	switch moduleName {
	case "mysql":
		var mysqlConfig mysql.Config
		if err := json.Unmarshal(data, &mysqlConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MySQL config: %v", err)
		}
		return &mysqlConfig, nil
	default:
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal module config: %v", err)
		}
		return result, nil
	}
}

// GetConfigPath returns the absolute path to the configuration file
func GetConfigPath(path string) (string, error) {
	if path == "" {
		// Default to config.yaml in the current directory
		path = "config.yaml"
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %v", err)
	}

	return absPath, nil
}
