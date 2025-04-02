package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/petermein/apollo/cmd/operator/api"
	"github.com/petermein/apollo/cmd/operator/config"
	"github.com/petermein/apollo/cmd/operator/modules"
	"github.com/petermein/apollo/cmd/operator/modules/mysql"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[OPERATOR] ")

	// Parse command line flags
	configPath := flag.String("config", "configs/operator.yaml", "Path to config file")
	flag.Parse()

	log.Printf("Starting operator with config file: %s", *configPath)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Loaded configuration for operator: %s", cfg.OperatorID)

	// Create API client
	apiClient := api.NewClient(cfg.API.Endpoint, cfg.OperatorID)
	log.Printf("Created API client with endpoint: %s", cfg.API.Endpoint)

	// Register operator with API
	if err := apiClient.RegisterOperator(context.Background()); err != nil {
		log.Fatalf("Failed to register operator: %v", err)
	}
	log.Printf("Successfully registered operator with API")

	// Create module registry
	registry := modules.NewRegistry()
	log.Printf("Created module registry")

	// Register MySQL module
	mysqlModule := mysql.NewModule(apiClient)
	registry.Register(mysqlModule)
	log.Printf("Registered MySQL module")

	// Initialize enabled modules
	enabledModules := registry.GetEnabledModules(cfg.EnabledModules)
	log.Printf("Enabled modules: %s", cfg.EnabledModules)

	for _, module := range enabledModules {
		if err := module.Initialize(cfg.Modules[module.Name()]); err != nil {
			log.Fatalf("Failed to initialize module %s: %v", module.Name(), err)
		}
		log.Printf("Initialized module: %s", module.Name())
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start monitoring for enabled modules
	for _, module := range enabledModules {
		if err := module.StartMonitoring(ctx); err != nil {
			log.Fatalf("Failed to start monitoring for module %s: %v", module.Name(), err)
		}
		log.Printf("Started monitoring for module: %s", module.Name())
	}

	// Start health check loop
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := apiClient.SendHealthCheck(ctx); err != nil {
					log.Printf("Failed to send health check: %v", err)
				} else {
					log.Printf("Health check sent successfully")
				}
			}
		}
	}()

	log.Printf("Operator is running. Press Ctrl+C to stop.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down...", sig)

	// Stop monitoring for enabled modules
	for _, module := range enabledModules {
		if err := module.StopMonitoring(ctx); err != nil {
			log.Printf("Failed to stop monitoring for module %s: %v", module.Name(), err)
		} else {
			log.Printf("Stopped monitoring for module: %s", module.Name())
		}
	}

	log.Printf("Operator shutdown complete")
}
