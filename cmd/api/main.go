package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
)

var (
	port         = flag.String("port", "8080", "Server port")
	slackToken   = flag.String("slack-token", "", "Slack API token")
	slackChannel = flag.String("slack-channel", "", "Slack channel for notifications")
)

func main() {
	flag.Parse()

	// Initialize Gin router
	router := gin.Default()

	// Initialize event system
	eventBus := NewEventBus()

	// Initialize Slack client if token is provided
	var slackClient *slack.Client
	if *slackToken != "" {
		slackClient = slack.New(*slackToken)
	}

	// Setup routes
	setupRoutes(router, eventBus, slackClient)

	// Create server context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down server...")
		cancel()
	}()

	// Start server
	if err := router.Run(":" + *port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(router *gin.Engine, eventBus *EventBus, slackClient *slack.Client) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Operator registration
		api.POST("/operators", registerOperator)
		api.GET("/operators", listOperators)

		// Privilege management
		api.POST("/privileges/request", requestPrivilege)
		api.POST("/privileges/:id/approve", approvePrivilege)
		api.POST("/privileges/:id/revoke", revokePrivilege)
		api.GET("/privileges/active", listActivePrivileges)
	}

	// Event subscription endpoint
	router.POST("/events/subscribe", handleEventSubscription)
}

// EventBus handles event distribution
type EventBus struct {
	subscribers []chan Event
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make([]chan Event, 0),
	}
}

func (eb *EventBus) Subscribe() chan Event {
	ch := make(chan Event, 100)
	eb.subscribers = append(eb.subscribers, ch)
	return ch
}

func (eb *EventBus) Publish(event Event) {
	for _, ch := range eb.subscribers {
		select {
		case ch <- event:
		default:
			// Skip if channel is full
		}
	}
} 