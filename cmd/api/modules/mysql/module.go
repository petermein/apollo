package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/petermein/apollo/cmd/api/modules"
)

// Config represents the MySQL module configuration
type Config struct {
	Host              string `yaml:"host"`
	Port              int    `yaml:"port"`
	User              string `yaml:"user"`
	Password          string `yaml:"password"`
	MaxConnections    int    `yaml:"max_connections"`
	ConnectionTimeout string `yaml:"connection_timeout"`
	IdleTimeout       string `yaml:"idle_timeout"`
}

// Module implements the MySQL module
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
	return "MySQL database module for managing database privileges"
}

// Initialize initializes the MySQL module
func (m *Module) Initialize(config interface{}) error {
	// Convert config map to our Config struct
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid config type for MySQL module")
	}

	cfg := &Config{}

	// Extract values from the map
	if host, ok := configMap["host"].(string); ok {
		cfg.Host = host
	}
	if port, ok := configMap["port"].(int); ok {
		cfg.Port = port
	}
	if user, ok := configMap["user"].(string); ok {
		cfg.User = user
	}
	if password, ok := configMap["password"].(string); ok {
		cfg.Password = password
	}
	if maxConn, ok := configMap["max_connections"].(int); ok {
		cfg.MaxConnections = maxConn
	}
	if connTimeout, ok := configMap["connection_timeout"].(string); ok {
		cfg.ConnectionTimeout = connTimeout
	}
	if idleTimeout, ok := configMap["idle_timeout"].(string); ok {
		cfg.IdleTimeout = idleTimeout
	}

	// Validate required fields
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

	m.config = cfg

	// Parse timeouts
	connTimeout, err := time.ParseDuration(cfg.ConnectionTimeout)
	if err != nil {
		return fmt.Errorf("invalid connection timeout: %v", err)
	}

	idleTimeout, err := time.ParseDuration(cfg.IdleTimeout)
	if err != nil {
		return fmt.Errorf("invalid idle timeout: %v", err)
	}

	// Create DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, connTimeout)

	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MaxConnections)
	db.SetConnMaxLifetime(idleTimeout)

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	m.db = db
	return nil
}

// HandlePingRequest handles a MySQL ping request
func (m *Module) HandlePingRequest(ctx context.Context, request *modules.PingRequest) (string, error) {
	if m.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// Execute ping query
	var hostname string
	err := m.db.QueryRowContext(ctx, "SELECT @@hostname").Scan(&hostname)
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %v", err)
	}

	return hostname, nil
}

// HealthCheck performs a health check on the MySQL module
func (m *Module) HealthCheck(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	return m.db.PingContext(ctx)
}

// ListServers returns a list of registered MySQL servers
func (m *Module) ListServers(ctx context.Context) ([]modules.ServerInfo, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// For now, return a static list of servers for testing
	servers := []modules.ServerInfo{
		{
			Name:     "local",
			Host:     "mysql",
			Port:     3306,
			User:     "root",
			Database: "apollo",
		},
	}

	return servers, nil
}
