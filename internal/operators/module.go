package operators

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Module represents a privilege management module
type Module interface {
	// Name returns the unique name of the module
	Name() string

	// Description returns a human-readable description of the module
	Description() string

	// Initialize sets up the module with its configuration
	Initialize(ctx context.Context, config interface{}) error

	// ValidateConfig validates the module's configuration
	ValidateConfig(config interface{}) error

	// HandlePrivilegeRequest handles a privilege escalation request
	HandlePrivilegeRequest(ctx context.Context, request *PrivilegeRequest) error

	// RevokePrivilege revokes a granted privilege
	RevokePrivilege(ctx context.Context, grantID string) error

	// HealthCheck performs a health check of the module
	HealthCheck(ctx context.Context) error
}

// PrivilegeRequest represents a request for privilege escalation
type PrivilegeRequest struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	ResourceID  string                 `json:"resource_id"`
	Level       string                 `json:"level"`
	Duration    string                 `json:"duration"`
	Reason      string                 `json:"reason"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RequestedAt string                 `json:"requested_at"`
}

// ModuleRegistry manages the registration and lookup of modules
type ModuleRegistry struct {
	mu      sync.RWMutex
	modules map[string]Module
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]Module),
	}
}

// Register registers a new module
func (r *ModuleRegistry) Register(module Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.ToLower(module.Name())
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %s is already registered", name)
	}

	r.modules[name] = module
	return nil
}

// GetModule retrieves a module by name
func (r *ModuleRegistry) GetModule(name string) (Module, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	module, exists := r.modules[strings.ToLower(name)]
	if !exists {
		return nil, fmt.Errorf("module %s not found", name)
	}

	return module, nil
}

// ListModules returns a list of all registered modules
func (r *ModuleRegistry) ListModules() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]Module, 0, len(r.modules))
	for _, module := range r.modules {
		modules = append(modules, module)
	}
	return modules
}

// GetEnabledModules returns a list of enabled modules based on the comma-separated list
func (r *ModuleRegistry) GetEnabledModules(enabledModules string) ([]Module, error) {
	if enabledModules == "" {
		return nil, fmt.Errorf("no modules enabled")
	}

	moduleNames := strings.Split(enabledModules, ",")
	modules := make([]Module, 0, len(moduleNames))

	for _, name := range moduleNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		module, err := r.GetModule(name)
		if err != nil {
			return nil, fmt.Errorf("failed to get module %s: %v", name, err)
		}

		modules = append(modules, module)
	}

	return modules, nil
} 