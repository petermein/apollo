package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/petermein/apollo/cmd/operator/api"
	"github.com/petermein/apollo/cmd/operator/modules"
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
	APIClient         *api.Client
}

// Module implements the MySQL module
type Module struct {
	config *Config
	db     *sql.DB
}

// NewModule creates a new MySQL module
func NewModule(apiClient *api.Client) *Module {
	return &Module{
		config: &Config{
			APIClient: apiClient,
		},
	}
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
	log.Printf("[MYSQL] Initializing MySQL module")

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

	// Set the API client from the module's config
	cfg.APIClient = m.config.APIClient
	m.config = cfg

	log.Printf("[MYSQL] Configuration loaded for server %s:%d", cfg.Host, cfg.Port)

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

	log.Printf("[MYSQL] Connecting to MySQL server at %s:%d", cfg.Host, cfg.Port)

	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MaxConnections)
	db.SetConnMaxLifetime(idleTimeout)

	log.Printf("[MYSQL] Testing connection to MySQL server")

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	log.Printf("[MYSQL] Successfully connected to MySQL server")

	m.db = db
	return nil
}

// StartMonitoring starts monitoring the MySQL server
func (m *Module) StartMonitoring(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Register this server with the API
	serverInfo := modules.ServerInfo{
		Name:     fmt.Sprintf("%s-%d", m.config.Host, m.config.Port),
		Host:     m.config.Host,
		Port:     m.config.Port,
		User:     m.config.User,
		Database: "apollo",
	}

	log.Printf("[MYSQL] Registering server %s with API", serverInfo.Name)

	// Register server with API
	if err := m.config.APIClient.RegisterServer(ctx, serverInfo); err != nil {
		return fmt.Errorf("failed to register server: %v", err)
	}

	log.Printf("[MYSQL] Successfully registered server %s", serverInfo.Name)

	// Start health check loop
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		log.Printf("[MYSQL] Starting health check loop for server %s", serverInfo.Name)

		for {
			select {
			case <-ctx.Done():
				log.Printf("[MYSQL] Stopping health check loop for server %s", serverInfo.Name)
				return
			case <-ticker.C:
				if err := m.db.PingContext(ctx); err != nil {
					log.Printf("[MYSQL] Health check failed for server %s: %v", serverInfo.Name, err)
					// Mark server as inactive in API
					if err := m.config.APIClient.MarkServerInactive(ctx, serverInfo.Name); err != nil {
						log.Printf("[MYSQL] Failed to mark server %s as inactive: %v", serverInfo.Name, err)
					} else {
						log.Printf("[MYSQL] Marked server %s as inactive", serverInfo.Name)
					}
				} else {
					log.Printf("[MYSQL] Health check passed for server %s", serverInfo.Name)
				}
			}
		}
	}()

	return nil
}

// StopMonitoring stops monitoring the MySQL server
func (m *Module) StopMonitoring(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	serverName := fmt.Sprintf("%s-%d", m.config.Host, m.config.Port)
	log.Printf("[MYSQL] Stopping monitoring for server %s", serverName)

	// Mark server as inactive in API
	if err := m.config.APIClient.MarkServerInactive(ctx, serverName); err != nil {
		log.Printf("[MYSQL] Failed to mark server %s as inactive: %v", serverName, err)
	} else {
		log.Printf("[MYSQL] Marked server %s as inactive", serverName)
	}

	if err := m.db.Close(); err != nil {
		log.Printf("[MYSQL] Failed to close database connection: %v", err)
		return err
	}

	log.Printf("[MYSQL] Successfully closed database connection")
	return nil
}
