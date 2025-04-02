package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/petermein/apollo/internal/operators"
	"github.com/petermein/apollo/internal/operators/mysql"
)

// Job represents a job in the system
type Job struct {
	ID      string          `json:"id"`
	Module  string          `json:"module"`
	Type    string          `json:"type"`
	Request json.RawMessage `json:"request"`
	Status  string          `json:"status"`
	Result  string          `json:"result"`
	Error   string          `json:"error"`
}

// JobStore manages jobs in memory
type JobStore struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// NewJobStore creates a new job store
func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*Job),
	}
}

// CreateJob creates a new job
func (s *JobStore) CreateJob(module, jobType string, request json.RawMessage) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := &Job{
		ID:      generateJobID(),
		Module:  module,
		Type:    jobType,
		Request: request,
		Status:  "pending",
	}

	s.jobs[job.ID] = job
	return job
}

// GetJob retrieves a job by ID
func (s *JobStore) GetJob(id string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jobs[id]
}

// GetPendingJobs retrieves all pending jobs
func (s *JobStore) GetPendingJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []*Job
	for _, job := range s.jobs {
		if job.Status == "pending" {
			pending = append(pending, job)
		}
	}
	return pending
}

// UpdateJob updates a job's status and result
func (s *JobStore) UpdateJob(id, status, result, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Status = status
	job.Result = result
	job.Error = errMsg
	return nil
}

// Handler handles API requests
type Handler struct {
	modules  []operators.Module
	jobStore *JobStore
}

// NewHandler creates a new API handler
func NewHandler(modules []operators.Module) *Handler {
	return &Handler{
		modules:  modules,
		jobStore: NewJobStore(),
	}
}

// HandleCreatePingJob handles creating a new ping job
func (h *Handler) HandleCreatePingJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Server string `json:"server"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Server == "" {
		http.Error(w, "Server name is required", http.StatusBadRequest)
		return
	}

	// Create ping request
	pingReq := mysql.PingRequest{
		Server: req.Server,
	}

	requestJSON, err := json.Marshal(pingReq)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	// Create job
	job := h.jobStore.CreateJob("mysql", "ping", requestJSON)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// HandleGetJob handles retrieving a job by ID
func (h *Handler) HandleGetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	job := h.jobStore.GetJob(jobID)
	if job == nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// HandleGetPendingJobs handles retrieving pending jobs
func (h *Handler) HandleGetPendingJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobs := h.jobStore.GetPendingJobs()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// HandleUpdateJob handles updating a job's status
func (h *Handler) HandleUpdateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	var update struct {
		Status string `json:"status"`
		Result string `json:"result"`
		Error  string `json:"error"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.jobStore.UpdateJob(jobID, update.Status, update.Result, update.Error); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleListMySQLServers handles listing registered MySQL servers
func (h *Handler) HandleListMySQLServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Find MySQL module
	var mysqlModule *mysql.Module
	for _, module := range h.modules {
		if m, ok := module.(*mysql.Module); ok {
			mysqlModule = m
			break
		}
	}

	if mysqlModule == nil {
		http.Error(w, "MySQL module not found", http.StatusNotFound)
		return
	}

	// Get registered servers
	servers, err := mysqlModule.ListServers(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list servers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string            `json:"status"`
	Modules map[string]string `json:"modules"`
}

// HandleHealthCheck handles the health check request
func (h *Handler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:  "healthy",
		Modules: make(map[string]string),
	}

	// Check health of each module
	for _, module := range h.modules {
		err := module.HealthCheck(r.Context())
		if err != nil {
			response.Status = "degraded"
			response.Modules[module.Name()] = fmt.Sprintf("unhealthy: %v", err)
		} else {
			response.Modules[module.Name()] = "healthy"
		}
	}

	// If any module is unhealthy, return 503 Service Unavailable
	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
