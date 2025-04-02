package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/petermein/apollo/internal/config"
	"github.com/petermein/apollo/internal/operators"
	"github.com/petermein/apollo/internal/operators/kubernetes"
	"github.com/petermein/apollo/internal/operators/mysql"
)

var (
	configPath = flag.String("config", "configs/operator.yaml", "Path to operator configuration file")
	port       = flag.Int("port", 8081, "Port to listen on")
)

func main() {
	flag.Parse()

	// Get absolute path to config file
	absConfigPath, err := config.GetConfigPath(*configPath)
	if err != nil {
		log.Fatalf("Failed to get config path: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(absConfigPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create module registry
	registry := operators.NewModuleRegistry()

	// Register available modules
	if err := registry.Register(mysql.NewModule()); err != nil {
		log.Fatalf("Failed to register MySQL module: %v", err)
	}
	if err := registry.Register(kubernetes.NewModule()); err != nil {
		log.Fatalf("Failed to register Kubernetes module: %v", err)
	}

	// Get enabled modules
	enabledModules, err := registry.GetEnabledModules(cfg.Operator.EnabledModules)
	if err != nil {
		log.Fatalf("Failed to get enabled modules: %v", err)
	}

	// Initialize modules
	ctx := context.Background()
	for _, module := range enabledModules {
		moduleConfig, err := cfg.GetModuleConfig(module.Name())
		if err != nil {
			log.Fatalf("Failed to get config for module %s: %v", module.Name(), err)
		}

		if err := module.Initialize(ctx, moduleConfig); err != nil {
			log.Fatalf("Failed to initialize module %s: %v", module.Name(), err)
		}
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth(enabledModules))
	mux.HandleFunc("/privilege/request", handlePrivilegeRequest(enabledModules))
	mux.HandleFunc("/privilege/revoke", handlePrivilegeRevoke(enabledModules))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	// Start server
	go func() {
		log.Printf("Starting operator on port %d", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

func handleHealth(modules []operators.Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		status := make(map[string]string)

		for _, module := range modules {
			if err := module.HealthCheck(ctx); err != nil {
				status[module.Name()] = fmt.Sprintf("error: %v", err)
			} else {
				status[module.Name()] = "healthy"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"modules": status,
		})
	}
}

func handlePrivilegeRequest(modules []operators.Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request operators.PrivilegeRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// Find appropriate module
		var module operators.Module
		for _, m := range modules {
			if m.Name() == request.ResourceID {
				module = m
				break
			}
		}

		if module == nil {
			http.Error(w, fmt.Sprintf("No module found for resource: %s", request.ResourceID), http.StatusNotFound)
			return
		}

		// Handle request
		if err := module.HandlePrivilegeRequest(r.Context(), &request); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(request)
	}
}

func handlePrivilegeRevoke(modules []operators.Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			GrantID string `json:"grant_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// Try to revoke from all modules
		var errors []string
		for _, module := range modules {
			if err := module.RevokePrivilege(r.Context(), request.GrantID); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", module.Name(), err))
			}
		}

		if len(errors) > 0 {
			http.Error(w, fmt.Sprintf("Failed to revoke privileges: %v", errors), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
} 