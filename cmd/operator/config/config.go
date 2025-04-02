package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the operator configuration
type Config struct {
	OperatorID     string                 `yaml:"operator_id"`
	API            APIConfig              `yaml:"api"`
	EnabledModules string                 `yaml:"enabled_modules"`
	Modules        map[string]interface{} `yaml:"modules"`
}

// APIConfig represents the API configuration
type APIConfig struct {
	Endpoint string `yaml:"endpoint"`
}

// expandEnvVars expands environment variables in the format ${VAR:-default}
func expandEnvVars(input string) string {
	// Split on ${ to find all potential env vars
	parts := strings.Split(input, "${")
	if len(parts) == 1 {
		return input
	}

	var result strings.Builder
	result.WriteString(parts[0])

	for _, part := range parts[1:] {
		// Find the closing brace
		closeBrace := strings.Index(part, "}")
		if closeBrace == -1 {
			result.WriteString("${")
			result.WriteString(part)
			continue
		}

		// Extract the env var expression and the rest of the string
		envVar := part[:closeBrace]
		rest := part[closeBrace+1:]

		// Check if there's a default value
		var defaultVal string
		if idx := strings.Index(envVar, ":-"); idx != -1 {
			defaultVal = envVar[idx+2:]
			envVar = envVar[:idx]
		}

		// Get the environment variable value
		val := os.Getenv(envVar)
		if val == "" {
			val = defaultVal
		}

		result.WriteString(val)
		result.WriteString(rest)
	}

	return result.String()
}

// Load loads the configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Expand environment variables in the config file
	configStr := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(configStr), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate required fields
	if cfg.OperatorID == "" {
		return nil, fmt.Errorf("operator_id is required")
	}
	if cfg.API.Endpoint == "" {
		return nil, fmt.Errorf("api.endpoint is required")
	}

	return &cfg, nil
}
