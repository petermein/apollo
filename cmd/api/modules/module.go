package modules

import (
	"context"
	"fmt"
	"strings"
)

// ServerInfo represents information about a registered server
type ServerInfo struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Database string `json:"database"`
}

// Module defines the interface for all modules
type Module interface {
	// Name returns the name of the module
	Name() string

	// Description returns a description of the module
	Description() string

	// Initialize initializes the module with its configuration
	Initialize(config interface{}) error

	// HandlePingRequest handles a ping request
	HandlePingRequest(ctx context.Context, request *PingRequest) (string, error)

	// HealthCheck performs a health check on the module
	HealthCheck(ctx context.Context) error

	// ListServers returns a list of registered servers
	ListServers(ctx context.Context) ([]ServerInfo, error)
}

// PingRequest represents a ping request
type PingRequest struct {
	Server string `json:"server"`
}

// Registry manages module registration and lookup
type Registry struct {
	modules map[string]Module
}

// NewRegistry creates a new module registry
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]Module),
	}
}

// Register registers a new module
func (r *Registry) Register(module Module) error {
	if _, exists := r.modules[module.Name()]; exists {
		return fmt.Errorf("module %s already registered", module.Name())
	}
	r.modules[module.Name()] = module
	return nil
}

// GetModule returns a module by name
func (r *Registry) GetModule(name string) (Module, error) {
	module, exists := r.modules[name]
	if !exists {
		return nil, fmt.Errorf("module %s not found", name)
	}
	return module, nil
}

// GetEnabledModules returns a list of enabled modules
func (r *Registry) GetEnabledModules(names string) []Module {
	if names == "" {
		return nil
	}

	var enabled []Module
	for _, name := range strings.Split(names, ",") {
		name = strings.TrimSpace(name)
		if module, exists := r.modules[name]; exists {
			enabled = append(enabled, module)
		}
	}
	return enabled
}
