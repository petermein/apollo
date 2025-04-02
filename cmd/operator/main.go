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
	"github.com/petermein/apollo/internal/operators/mysql"
)

var (
	cfgFile = flag.String("config", "", "Path to config file")
	port    = flag.Int("port", 8081, "Port to listen on")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*cfgFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize module registry
	registry := operators.NewModuleRegistry()

	// Register MySQL module
	mysqlModule := mysql.NewModule()
	if err := registry.Register(mysqlModule); err != nil {
		log.Fatalf("Failed to register MySQL module: %v", err)
	}

	// Get enabled modules
	modules, err := registry.GetEnabledModules(cfg.Operator.EnabledModules)
	if err != nil {
		log.Fatalf("Failed to get enabled modules: %v", err)
	}

	// Initialize modules
	ctx := context.Background()
	for _, module := range modules {
		moduleConfig, err := cfg.GetModuleConfig(module.Name())
		if err != nil {
			log.Fatalf("Failed to get config for module %s: %v", module.Name(), err)
		}

		if err := module.Initialize(ctx, moduleConfig); err != nil {
			log.Fatalf("Failed to initialize module %s: %v", module.Name(), err)
		}
	}

	// Create API client
	client := NewAPIClient(cfg.API.Endpoint)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start job processing loop
	go func() {
		for {
			select {
			case <-sigChan:
				log.Println("Shutting down operator...")
				return
			default:
				if err := processJobs(ctx, client, modules); err != nil {
					log.Printf("Error processing jobs: %v", err)
				}
			}
		}
	}()

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth(modules))
	mux.HandleFunc("/privilege/request", handlePrivilegeRequest(modules))
	mux.HandleFunc("/privilege/revoke", handlePrivilegeRevoke(modules))

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

	<-sigChan

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

func processJobs(ctx context.Context, client *APIClient, modules []operators.Module) error {
	// Get pending jobs
	jobs, err := client.GetPendingJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending jobs: %v", err)
	}

	for _, job := range jobs {
		// Find matching module
		var module operators.Module
		for _, m := range modules {
			if m.Name() == job.Module {
				module = m
				break
			}
		}

		if module == nil {
			log.Printf("No matching module found for job %s", job.ID)
			continue
		}

		// Process job based on type
		switch job.Type {
		case "ping":
			if err := processPingJob(ctx, client, job, module); err != nil {
				log.Printf("Failed to process ping job %s: %v", job.ID, err)
			}
		default:
			log.Printf("Unknown job type: %s", job.Type)
		}
	}

	return nil
}

func processPingJob(ctx context.Context, client *APIClient, job *Job, module operators.Module) error {
	// Parse request
	var req mysql.PingRequest
	if err := json.Unmarshal(job.Request, &req); err != nil {
		return fmt.Errorf("failed to parse ping request: %v", err)
	}

	// Handle ping request
	result, err := module.(*mysql.Module).HandlePingRequest(ctx, &req)
	if err != nil {
		return client.UpdateJob(ctx, job.ID, "failed", "", err.Error())
	}

	return client.UpdateJob(ctx, job.ID, "completed", result, "")
}
