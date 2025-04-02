package service

import (
	"context"
	"time"

	"apollo/internal/core/models"
)

// PrivilegeService defines the interface for privilege management
type PrivilegeService interface {
	// RequestPrivilege creates a new privilege escalation request
	RequestPrivilege(ctx context.Context, userID, resourceID string, level models.PrivilegeLevel, reason string, duration time.Duration) (*models.PrivilegeRequest, error)

	// ApproveRequest approves a privilege escalation request
	ApproveRequest(ctx context.Context, requestID, approverID string) (*models.PrivilegeGrant, error)

	// RevokePrivilege revokes an active privilege grant
	RevokePrivilege(ctx context.Context, grantID string) error

	// GetActiveGrants retrieves all active privilege grants for a user
	GetActiveGrants(ctx context.Context, userID string) ([]*models.PrivilegeGrant, error)

	// GetPendingRequests retrieves all pending privilege requests
	GetPendingRequests(ctx context.Context) ([]*models.PrivilegeRequest, error)

	// ValidateAccess checks if a user has the required privilege level for a resource
	ValidateAccess(ctx context.Context, userID, resourceID string, requiredLevel models.PrivilegeLevel) (bool, error)
} 