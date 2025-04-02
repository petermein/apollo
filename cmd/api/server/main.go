package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/petermein/apollo/cmd/api/config"
	"github.com/petermein/apollo/cmd/api/handler"
	"github.com/petermein/apollo/cmd/api/modules"
	"github.com/petermein/apollo/cmd/api/modules/mysql"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create module registry
	registry := modules.NewRegistry()

	// Register MySQL module
	mysqlModule := mysql.NewModule()
	registry.Register(mysqlModule)

	// Get enabled modules
	enabledModules := registry.GetEnabledModules(cfg.Server.EnabledModules)
	if len(enabledModules) == 0 {
		log.Fatal("No modules enabled")
	}

	// Initialize modules
	for _, module := range enabledModules {
		moduleConfig, err := cfg.GetModuleConfig(module.Name())
		if err != nil {
			log.Fatalf("Failed to get config for module %s: %v", module.Name(), err)
		}

		if err := module.Initialize(moduleConfig); err != nil {
			log.Fatalf("Failed to initialize module %s: %v", module.Name(), err)
		}
	}

	// Create HTTP server
	mux := http.NewServeMux()
	h := handler.NewHandler(enabledModules)
	h.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
