package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
	log.Printf("Initializing MySQL module...")

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

	log.Printf("MySQL configuration loaded: host=%s:%d, user=%s, maxConn=%d", cfg.Host, cfg.Port, cfg.User, cfg.MaxConnections)

	// Parse timeouts
	connTimeout, err := time.ParseDuration(cfg.ConnectionTimeout)
	if err != nil {
		return fmt.Errorf("invalid connection timeout: %v", err)
	}

	idleTimeout, err := time.ParseDuration(cfg.IdleTimeout)
	if err != nil {
		return fmt.Errorf("invalid idle timeout: %v", err)
	}

	// Create DSN for initial connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, connTimeout)

	log.Printf("Establishing initial database connection...")

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
	log.Printf("Testing database connection...")
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Create database if it doesn't exist
	log.Printf("Creating apollo database if it doesn't exist...")
	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS apollo"); err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}

	// Close initial connection
	db.Close()

	// Create DSN with database name
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/apollo?timeout=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, connTimeout)

	log.Printf("Connecting to apollo database...")

	// Open new connection with database selected
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MaxConnections)
	db.SetConnMaxLifetime(idleTimeout)

	// Create tables
	log.Printf("Creating required tables...")
	if err := m.createTables(db); err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	m.db = db
	log.Printf("MySQL module initialized successfully")
	return nil
}

// createTables creates the necessary tables for storing server information
func (m *Module) createTables(db *sql.DB) error {
	// Create mysql_servers table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS mysql_servers (
			name VARCHAR(255) PRIMARY KEY,
			host VARCHAR(255) NOT NULL,
			port INT NOT NULL,
			user VARCHAR(255) NOT NULL,
			db_name VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'inactive',
			last_seen TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create mysql_servers table: %v", err)
	}

	// Create operators table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS operators (
			id VARCHAR(255) PRIMARY KEY,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			last_seen TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create operators table: %v", err)
	}

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

	rows, err := m.db.QueryContext(ctx, `
		SELECT name, host, port, user, db_name, status
		FROM mysql_servers
		WHERE status = 'active'
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query servers: %v", err)
	}
	defer rows.Close()

	var servers []modules.ServerInfo
	for rows.Next() {
		var server modules.ServerInfo
		if err := rows.Scan(&server.Name, &server.Host, &server.Port, &server.User, &server.Database, &server.Status); err != nil {
			return nil, fmt.Errorf("failed to scan server: %v", err)
		}
		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating servers: %v", err)
	}

	return servers, nil
}

// RegisterServer registers a new MySQL server
func (m *Module) RegisterServer(ctx context.Context, server modules.ServerInfo) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := m.db.ExecContext(ctx, `
		INSERT INTO mysql_servers (name, host, port, user, db_name, status, last_seen)
		VALUES (?, ?, ?, ?, ?, 'active', CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE
			host = VALUES(host),
			port = VALUES(port),
			user = VALUES(user),
			db_name = VALUES(db_name),
			status = 'active',
			last_seen = CURRENT_TIMESTAMP
	`, server.Name, server.Host, server.Port, server.User, server.Database)

	return err
}

// MarkServerInactive marks a MySQL server as inactive
func (m *Module) MarkServerInactive(ctx context.Context, name string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := m.db.ExecContext(ctx, `
		UPDATE mysql_servers
		SET status = 'inactive'
		WHERE name = ?
	`, name)

	return err
}

// RegisterOperator registers a new operator
func (m *Module) RegisterOperator(ctx context.Context, id string) error {
	log.Printf("Registering operator with ID: %s", id)

	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	result, err := m.db.ExecContext(ctx, `
		INSERT INTO operators (id, status, last_seen)
		VALUES (?, 'active', CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE
			status = 'active',
			last_seen = CURRENT_TIMESTAMP
	`, id)

	if err != nil {
		log.Printf("Error registering operator %s: %v", id, err)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected for operator %s: %v", id, err)
		return err
	}

	log.Printf("Successfully registered operator %s (rows affected: %d)", id, affected)
	return nil
}

// UpdateOperatorHealth updates the health status of an operator
func (m *Module) UpdateOperatorHealth(ctx context.Context, id string, timestamp time.Time) error {
	log.Printf("Updating health for operator %s (timestamp: %s)", id, timestamp)

	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	result, err := m.db.ExecContext(ctx, `
		UPDATE operators
		SET status = 'active',
			last_seen = ?
		WHERE id = ?
	`, timestamp, id)

	if err != nil {
		log.Printf("Error updating operator health for %s: %v", id, err)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected for operator %s health update: %v", id, err)
		return err
	}

	if affected == 0 {
		log.Printf("No operator found with ID %s for health update", id)
		return fmt.Errorf("operator not found: %s", id)
	}

	log.Printf("Successfully updated health for operator %s", id)
	return nil
}

// MarkOperatorInactive marks an operator as inactive
func (m *Module) MarkOperatorInactive(ctx context.Context, id string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := m.db.ExecContext(ctx, `
		UPDATE operators
		SET status = 'inactive'
		WHERE id = ?
	`, id)

	return err
}

// GetInactiveOperators returns a list of operators that haven't sent a health check in the last timeout period
func (m *Module) GetInactiveOperators(ctx context.Context, timeout time.Duration) ([]string, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := m.db.QueryContext(ctx, `
		SELECT id
		FROM operators
		WHERE status = 'active'
		AND last_seen < DATE_SUB(NOW(), INTERVAL ? SECOND)
	`, timeout.Seconds())
	if err != nil {
		return nil, fmt.Errorf("failed to query inactive operators: %v", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan operator ID: %v", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inactive operators: %v", err)
	}

	return ids, nil
}

// ListOperators returns a list of registered operators
func (m *Module) ListOperators(ctx context.Context) ([]modules.OperatorInfo, error) {
	log.Printf("Listing operators from database...")

	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := m.db.QueryContext(ctx, `
		SELECT id, status, 
		       COALESCE(last_seen, '0001-01-01 00:00:00') as last_seen,
		       COALESCE(created_at, '0001-01-01 00:00:00') as created_at,
		       COALESCE(updated_at, '0001-01-01 00:00:00') as updated_at
		FROM operators
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("Error querying operators: %v", err)
		return nil, fmt.Errorf("failed to query operators: %v", err)
	}
	defer rows.Close()

	var operators []modules.OperatorInfo
	for rows.Next() {
		var op modules.OperatorInfo
		var lastSeen, createdAt, updatedAt string
		if err := rows.Scan(&op.ID, &op.Status, &lastSeen, &createdAt, &updatedAt); err != nil {
			log.Printf("Error scanning operator row: %v", err)
			return nil, fmt.Errorf("failed to scan operator: %v", err)
		}

		// Parse timestamps
		op.LastSeen, err = time.Parse("2006-01-02 15:04:05", lastSeen)
		if err != nil {
			log.Printf("Error parsing last_seen timestamp: %v", err)
			return nil, fmt.Errorf("failed to parse last_seen timestamp: %v", err)
		}

		op.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			log.Printf("Error parsing created_at timestamp: %v", err)
			return nil, fmt.Errorf("failed to parse created_at timestamp: %v", err)
		}

		op.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAt)
		if err != nil {
			log.Printf("Error parsing updated_at timestamp: %v", err)
			return nil, fmt.Errorf("failed to parse updated_at timestamp: %v", err)
		}

		operators = append(operators, op)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating operators: %v", err)
		return nil, fmt.Errorf("error iterating operators: %v", err)
	}

	log.Printf("Found %d operators in database", len(operators))
	return operators, nil
}
