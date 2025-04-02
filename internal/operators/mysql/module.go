package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/petermein/apollo/internal/operators"
)

// Config represents the MySQL module configuration
type Config struct {
	Host             string        `json:"host"`
	Port             int           `json:"port"`
	User             string        `json:"user"`
	Password         string        `json:"password"`
	MaxConnections   int           `json:"max_connections"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	IdleTimeout      time.Duration `json:"idle_timeout"`
}

// Module implements the MySQL privilege management module
type Module struct {
	config *Config
	db     *sql.DB
}

// NewModule creates a new MySQL module
func NewModule() *Module {
	return &Module{}
}

// Name returns the module name
func (m *Module) Name() string {
	return "mysql"
}

// Description returns the module description
func (m *Module) Description() string {
	return "Manages MySQL database privileges"
}

// ValidateConfig validates the MySQL configuration
func (m *Module) ValidateConfig(config interface{}) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type: expected *Config")
	}

	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	if cfg.Port == 0 {
		return fmt.Errorf("port is required")
	}
	if cfg.User == "" {
		return fmt.Errorf("user is required")
	}
	if cfg.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}

// Initialize sets up the MySQL connection
func (m *Module) Initialize(ctx context.Context, config interface{}) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type: expected *Config")
	}

	m.config = cfg

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=%s&readTimeout=%s&writeTimeout=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port,
		cfg.ConnectionTimeout, cfg.ConnectionTimeout, cfg.ConnectionTimeout)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}

	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetConnMaxIdleTime(cfg.IdleTimeout)

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	m.db = db
	return nil
}

// HandlePrivilegeRequest handles a MySQL privilege escalation request
func (m *Module) HandlePrivilegeRequest(ctx context.Context, request *operators.PrivilegeRequest) error {
	// Parse the privilege level
	privileges, err := parsePrivileges(request.Level)
	if err != nil {
		return fmt.Errorf("invalid privilege level: %v", err)
	}

	// Create a temporary user with the requested privileges
	username := fmt.Sprintf("apollo_%s_%s", request.UserID, request.ID)
	password := generateSecurePassword()

	// Grant privileges
	for _, privilege := range privileges {
		query := fmt.Sprintf("GRANT %s ON %s TO '%s'@'%%' IDENTIFIED BY '%s'",
			privilege, request.ResourceID, username, password)
		
		if _, err := m.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to grant privileges: %v", err)
		}
	}

	// Store the grant information
	grant := struct {
		ID         string    `json:"id"`
		Username   string    `json:"username"`
		Password   string    `json:"password"`
		Privileges []string  `json:"privileges"`
		ExpiresAt  time.Time `json:"expires_at"`
	}{
		ID:         request.ID,
		Username:   username,
		Password:   password,
		Privileges: privileges,
		ExpiresAt:  time.Now().Add(parseDuration(request.Duration)),
	}

	// Store in metadata for later revocation
	metadata, err := json.Marshal(grant)
	if err != nil {
		return fmt.Errorf("failed to marshal grant metadata: %v", err)
	}
	request.Metadata = map[string]interface{}{
		"grant": grant,
	}

	return nil
}

// RevokePrivilege revokes MySQL privileges
func (m *Module) RevokePrivilege(ctx context.Context, grantID string) error {
	// In a real implementation, you would:
	// 1. Look up the grant information from persistent storage
	// 2. Revoke the privileges
	// 3. Drop the user
	return fmt.Errorf("not implemented")
}

// HealthCheck performs a MySQL health check
func (m *Module) HealthCheck(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if err := m.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %v", err)
	}

	return nil
}

// Helper functions

func parsePrivileges(level string) ([]string, error) {
	// Map privilege levels to actual MySQL privileges
	privilegeMap := map[string][]string{
		"read":    {"SELECT"},
		"write":   {"SELECT", "INSERT", "UPDATE", "DELETE"},
		"admin":   {"ALL PRIVILEGES"},
	}

	privileges, ok := privilegeMap[level]
	if !ok {
		return nil, fmt.Errorf("invalid privilege level: %s", level)
	}

	return privileges, nil
}

func parseDuration(duration string) time.Duration {
	d, err := time.ParseDuration(duration)
	if err != nil {
		// Default to 1 hour if parsing fails
		return time.Hour
	}
	return d
}

func generateSecurePassword() string {
	// In a real implementation, generate a secure random password
	return "temporary_password" // This should be replaced with proper password generation
} 