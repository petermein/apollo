package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the API configuration structure
type Config struct {
	Server struct {
		Port           int    `yaml:"port"`
		Host           string `yaml:"host"`
		EnabledModules string `yaml:"enabled_modules"`
	} `yaml:"server"`

	Modules map[string]interface{} `yaml:"modules"`

	API struct {
		Endpoint      string `yaml:"endpoint"`
		RetryAttempts int    `yaml:"retry_attempts"`
		RetryDelay    string `yaml:"retry_delay"`
	} `yaml:"api"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		Output string `yaml:"output"`
	} `yaml:"logging"`

	Health struct {
		Interval string `yaml:"interval"`
		Timeout  string `yaml:"timeout"`
		Retries  int    `yaml:"retries"`
	} `yaml:"health"`

	Slack struct {
		Token   string `yaml:"token"`
		Channel string `yaml:"channel"`
	} `yaml:"slack"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate config
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return &cfg, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.Server.Port == 0 {
		return fmt.Errorf("server port is required")
	}
	if cfg.Server.Host == "" {
		return fmt.Errorf("server host is required")
	}
	if cfg.Server.EnabledModules == "" {
		return fmt.Errorf("enabled modules are required")
	}
	return nil
}

// GetModuleConfig returns the configuration for a specific module
func (c *Config) GetModuleConfig(name string) (interface{}, error) {
	config, exists := c.Modules[name]
	if !exists {
		return nil, fmt.Errorf("module %s not found in config", name)
	}
	return config, nil
}

// GetConfigPath returns the absolute path to the config file
func GetConfigPath() (string, error) {
	// First try the current directory
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml", nil
	}

	// Then try the configs directory
	if _, err := os.Stat("configs/config.yaml"); err == nil {
		return "configs/config.yaml", nil
	}

	// Finally try the absolute path
	absPath, err := filepath.Abs("config.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %v", err)
	}

	return absPath, nil
}
