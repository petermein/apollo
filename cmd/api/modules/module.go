package modules

import (
	"context"
	"strings"
	"time"
)

// ServerInfo represents information about a server
type ServerInfo struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Database string `json:"database"`
	Status   string `json:"status"`
}

// OperatorInfo represents information about an operator
type OperatorInfo struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Module represents a module that can be registered with the API
type Module interface {
	// Name returns the name of the module
	Name() string

	// Description returns a description of the module
	Description() string

	// Initialize initializes the module with the given configuration
	Initialize(config interface{}) error

	// HandlePingRequest handles a ping request for a server
	HandlePingRequest(ctx context.Context, request *PingRequest) (string, error)

	// HealthCheck performs a health check on the module
	HealthCheck(ctx context.Context) error

	// ListServers returns a list of servers managed by the module
	ListServers(ctx context.Context) ([]ServerInfo, error)

	// ListOperators returns a list of registered operators
	ListOperators(ctx context.Context) ([]OperatorInfo, error)
}

// PingRequest represents a ping request
type PingRequest struct {
	Server string `json:"server"`
}

// Registry manages a collection of modules
type Registry struct {
	modules map[string]Module
}

// NewRegistry creates a new module registry
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]Module),
	}
}

// Register registers a module with the registry
func (r *Registry) Register(module Module) {
	r.modules[module.Name()] = module
}

// GetModule returns a module by name
func (r *Registry) GetModule(name string) Module {
	return r.modules[name]
}

// GetModules returns all registered modules
func (r *Registry) GetModules() []Module {
	modules := make([]Module, 0, len(r.modules))
	for _, module := range r.modules {
		modules = append(modules, module)
	}
	return modules
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
