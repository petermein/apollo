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

// GetPendingJobs retrieves pending jobs from the API
func (c *APIClient) GetPendingJobs(ctx context.Context) ([]*Job, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/jobs/pending", c.baseURL), nil)
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

	var jobs []*Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return jobs, nil
}

// UpdateJob updates a job's status and result
func (c *APIClient) UpdateJob(ctx context.Context, jobID, status, result, errMsg string) error {
	update := struct {
		Status string `json:"status"`
		Result string `json:"result"`
		Error  string `json:"error"`
	}{
		Status: status,
		Result: result,
		Error:  errMsg,
	}

	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s/api/v1/jobs/%s", c.baseURL, jobID), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
