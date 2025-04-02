package rules

import (
	"errors"
	"time"

	"apollo/internal/core/models"
)

// SecurityRule defines a security rule for privilege management
type SecurityRule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RuleEngine handles the evaluation of security rules
type RuleEngine interface {
	// EvaluateRequest evaluates a privilege request against security rules
	EvaluateRequest(request *models.PrivilegeRequest) error

	// ValidateGrant validates a privilege grant against security rules
	ValidateGrant(grant *models.PrivilegeGrant) error
}

// DefaultRuleEngine implements basic security rules
type DefaultRuleEngine struct{}

// EvaluateRequest implements basic security rules for privilege requests
func (e *DefaultRuleEngine) EvaluateRequest(request *models.PrivilegeRequest) error {
	// Rule 1: Maximum privilege duration
	maxDuration := 24 * time.Hour
	if request.ExpiresAt.Sub(request.RequestedAt) > maxDuration {
		return errors.New("privilege duration exceeds maximum allowed time")
	}

	// Rule 2: Minimum privilege duration
	minDuration := 5 * time.Minute
	if request.ExpiresAt.Sub(request.RequestedAt) < minDuration {
		return errors.New("privilege duration is less than minimum allowed time")
	}

	// Rule 3: Required reason
	if request.Reason == "" {
		return errors.New("reason is required for privilege request")
	}

	return nil
}

// ValidateGrant implements basic security rules for privilege grants
func (e *DefaultRuleEngine) ValidateGrant(grant *models.PrivilegeGrant) error {
	// Rule 1: Check if grant has expired
	if time.Now().After(grant.ExpiresAt) {
		return errors.New("privilege grant has expired")
	}

	// Rule 2: Validate grant duration
	maxDuration := 24 * time.Hour
	if grant.ExpiresAt.Sub(grant.GrantedAt) > maxDuration {
		return errors.New("privilege grant duration exceeds maximum allowed time")
	}

	return nil
} 