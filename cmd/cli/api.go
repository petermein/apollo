package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Job represents a job from the API
type Job struct {
	ID      string          `json:"id"`
	Module  string          `json:"module"`
	Type    string          `json:"type"`
	Request json.RawMessage `json:"request"`
	Status  string          `json:"status"`
	Result  string          `json:"result"`
	Error   string          `json:"error"`
}

// ServerInfo represents information about a registered MySQL server
type ServerInfo struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Database string `json:"database"`
}

// APIClient handles communication with the API server
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

// CreatePingJob creates a new ping job
func (c *APIClient) CreatePingJob(ctx context.Context, server string) (*Job, error) {
	req := struct {
		Server string `json:"server"`
	}{
		Server: server,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/v1/jobs/ping", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &job, nil
}

// GetJob retrieves a job by ID
func (c *APIClient) GetJob(ctx context.Context, jobID string) (*Job, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/jobs?id=%s", c.baseURL, jobID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &job, nil
}

// WaitForJobCompletion waits for a job to complete and returns the result
func (c *APIClient) WaitForJobCompletion(ctx context.Context, jobID string, pollInterval time.Duration) (*Job, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			job, err := c.GetJob(ctx, jobID)
			if err != nil {
				return nil, err
			}

			switch job.Status {
			case "completed":
				return job, nil
			case "failed":
				return nil, fmt.Errorf("job failed: %s", job.Error)
			}
		}
	}
}

// ListMySQLServers retrieves a list of registered MySQL servers
func (c *APIClient) ListMySQLServers(ctx context.Context) ([]ServerInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/mysql/servers", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var servers []ServerInfo
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return servers, nil
}
