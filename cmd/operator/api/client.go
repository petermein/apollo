package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/petermein/apollo/cmd/operator/modules"
)

// Client represents an API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	operatorID string
}

// NewClient creates a new API client
func NewClient(baseURL, operatorID string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		operatorID: operatorID,
	}
}

// RegisterOperator registers the operator with the API
func (c *Client) RegisterOperator(ctx context.Context) error {
	req := struct {
		ID string `json:"id"`
	}{
		ID: c.operatorID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/operators/register", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to register operator: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to register operator: status %d", resp.StatusCode)
	}

	return nil
}

// RegisterServer registers a MySQL server with the API
func (c *Client) RegisterServer(ctx context.Context, server modules.ServerInfo) error {
	data, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server info: %v", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/mysql/servers/register", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to register server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to register server: status %d", resp.StatusCode)
	}

	return nil
}

// MarkServerInactive marks a MySQL server as inactive
func (c *Client) MarkServerInactive(ctx context.Context, name string) error {
	req := struct {
		Name string `json:"name"`
	}{
		Name: name,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/mysql/servers/inactive", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to mark server inactive: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to mark server inactive: status %d", resp.StatusCode)
	}

	return nil
}

// SendHealthCheck sends a health check to the API
func (c *Client) SendHealthCheck(ctx context.Context) error {
	req := struct {
		ID        string    `json:"id"`
		Timestamp time.Time `json:"timestamp"`
	}{
		ID:        c.operatorID,
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal health check: %v", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/operators/health", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send health check: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send health check: status %d", resp.StatusCode)
	}

	return nil
}
