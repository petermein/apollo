package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/petermein/apollo/cmd/api/modules"
	"github.com/petermein/apollo/cmd/api/modules/mysql"
)

// Handler handles API requests
type Handler struct {
	modules []modules.Module
}

// NewHandler creates a new API handler
func NewHandler(modules []modules.Module) *Handler {
	log.Printf("Initializing API handler with %d modules", len(modules))
	for _, m := range modules {
		log.Printf("- Module enabled: %s (%s)", m.Name(), m.Description())
	}
	return &Handler{
		modules: modules,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	log.Println("Registering API routes...")
	mux.HandleFunc("/api/v1/ping", h.handlePing)
	mux.HandleFunc("/api/v1/health", h.handleHealth)
	mux.HandleFunc("/api/v1/mysql/servers", h.handleListMySQLServers)
	mux.HandleFunc("/api/v1/mysql/servers/register", h.handleRegisterMySQLServer)
	mux.HandleFunc("/api/v1/mysql/servers/inactive", h.handleMarkMySQLServerInactive)
	mux.HandleFunc("/api/v1/operators/register", h.handleRegisterOperator)
	mux.HandleFunc("/api/v1/operators/health", h.handleOperatorHealth)
	mux.HandleFunc("/api/v1/operators", h.handleListOperators)
	log.Println("API routes registered successfully")
}

// handlePing handles ping requests
func (h *Handler) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Module string `json:"module"`
		Server string `json:"server"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find the appropriate module
	var module modules.Module
	for _, m := range h.modules {
		if m.Name() == req.Module {
			module = m
			break
		}
	}

	if module == nil {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	// Handle the ping request
	result, err := module.HandlePingRequest(r.Context(), &modules.PingRequest{
		Server: req.Server,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"result": result,
	})
}

// handleHealth handles health check requests
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check health of all modules
	health := make(map[string]string)
	for _, module := range h.modules {
		err := module.HealthCheck(r.Context())
		if err != nil {
			health[module.Name()] = "unhealthy"
		} else {
			health[module.Name()] = "healthy"
		}
	}

	// Return health status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"time":    time.Now().UTC(),
		"modules": health,
	})
}

// handleListMySQLServers handles requests to list MySQL servers
func (h *Handler) handleListMySQLServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Find MySQL module
	var mysqlModule modules.Module
	for _, m := range h.modules {
		if m.Name() == "mysql" {
			mysqlModule = m
			break
		}
	}

	if mysqlModule == nil {
		http.Error(w, "MySQL module not found", http.StatusNotFound)
		return
	}

	// Get list of servers
	servers, err := mysqlModule.ListServers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the servers list
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

// handleRegisterMySQLServer handles requests to register a new MySQL server
func (h *Handler) handleRegisterMySQLServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var server modules.ServerInfo
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find MySQL module
	var mysqlModule modules.Module
	for _, m := range h.modules {
		if m.Name() == "mysql" {
			mysqlModule = m
			break
		}
	}

	if mysqlModule == nil {
		http.Error(w, "MySQL module not found", http.StatusNotFound)
		return
	}

	// Register the server
	if err := mysqlModule.(*mysql.Module).RegisterServer(r.Context(), server); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// handleMarkMySQLServerInactive handles requests to mark a MySQL server as inactive
func (h *Handler) handleMarkMySQLServerInactive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Server name is required", http.StatusBadRequest)
		return
	}

	// Find MySQL module
	var mysqlModule modules.Module
	for _, m := range h.modules {
		if m.Name() == "mysql" {
			mysqlModule = m
			break
		}
	}

	if mysqlModule == nil {
		http.Error(w, "MySQL module not found", http.StatusNotFound)
		return
	}

	// Mark the server as inactive
	if err := mysqlModule.(*mysql.Module).MarkServerInactive(r.Context(), req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleRegisterOperator handles requests to register a new operator
func (h *Handler) handleRegisterOperator(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received operator registration request from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		log.Printf("Operator ID is required")
		http.Error(w, "Operator ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("Processing registration for operator: %s", req.ID)

	// Find MySQL module
	var mysqlModule modules.Module
	for _, m := range h.modules {
		if m.Name() == "mysql" {
			mysqlModule = m
			break
		}
	}

	if mysqlModule == nil {
		log.Printf("MySQL module not found in enabled modules")
		http.Error(w, "MySQL module not found", http.StatusNotFound)
		return
	}

	// Register the operator
	if err := mysqlModule.(*mysql.Module).RegisterOperator(r.Context(), req.ID); err != nil {
		log.Printf("Error registering operator %s: %v", req.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully registered operator: %s", req.ID)
	w.WriteHeader(http.StatusCreated)
}

// handleOperatorHealth handles operator health check requests
func (h *Handler) handleOperatorHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received operator health check from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID        string    `json:"id"`
		Timestamp time.Time `json:"timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		log.Printf("Operator ID is required")
		http.Error(w, "Operator ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("Processing health check for operator: %s (timestamp: %s)", req.ID, req.Timestamp)

	// Find MySQL module
	var mysqlModule modules.Module
	for _, m := range h.modules {
		if m.Name() == "mysql" {
			mysqlModule = m
			break
		}
	}

	if mysqlModule == nil {
		log.Printf("MySQL module not found in enabled modules")
		http.Error(w, "MySQL module not found", http.StatusNotFound)
		return
	}

	// Update operator health
	if err := mysqlModule.(*mysql.Module).UpdateOperatorHealth(r.Context(), req.ID, req.Timestamp); err != nil {
		log.Printf("Error updating operator health for %s: %v", req.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated health for operator: %s", req.ID)
	w.WriteHeader(http.StatusOK)
}

// handleListOperators handles requests to list operators
func (h *Handler) handleListOperators(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to list operators from %s", r.RemoteAddr)

	if r.Method != http.MethodGet {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Find MySQL module
	var mysqlModule modules.Module
	for _, m := range h.modules {
		if m.Name() == "mysql" {
			mysqlModule = m
			break
		}
	}

	if mysqlModule == nil {
		log.Printf("MySQL module not found in enabled modules")
		http.Error(w, "MySQL module not found", http.StatusNotFound)
		return
	}

	// Get list of operators
	log.Printf("Fetching operators list from MySQL module")
	operators, err := mysqlModule.(*mysql.Module).ListOperators(r.Context())
	if err != nil {
		log.Printf("Error listing operators: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully retrieved %d operators", len(operators))
	for i, op := range operators {
		log.Printf("Operator %d: ID=%s, Status=%s, LastSeen=%s", i+1, op.ID, op.Status, op.LastSeen)
	}

	// Return the operators list
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(operators); err != nil {
		log.Printf("Error encoding operators response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully sent response to client")
}
