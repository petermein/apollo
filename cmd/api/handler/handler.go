package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/petermein/apollo/cmd/api/modules"
)

// Handler handles API requests
type Handler struct {
	modules []modules.Module
}

// NewHandler creates a new API handler
func NewHandler(modules []modules.Module) *Handler {
	return &Handler{
		modules: modules,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/ping", h.handlePing)
	mux.HandleFunc("/api/v1/health", h.handleHealth)
	mux.HandleFunc("/api/v1/mysql/servers", h.handleListMySQLServers)
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
