package models

import (
	"time"
)

// PrivilegeLevel represents the level of privilege
type PrivilegeLevel string

const (
	PrivilegeLevelRead    PrivilegeLevel = "read"
	PrivilegeLevelWrite   PrivilegeLevel = "write"
	PrivilegeLevelAdmin   PrivilegeLevel = "admin"
	PrivilegeLevelRoot    PrivilegeLevel = "root"
)

// PrivilegeRequest represents a request for privilege escalation
type PrivilegeRequest struct {
	ID            string         `json:"id" gorm:"primaryKey"`
	UserID        string         `json:"user_id"`
	ResourceID    string         `json:"resource_id"`
	Level         PrivilegeLevel `json:"level"`
	Reason        string         `json:"reason"`
	RequestedAt   time.Time      `json:"requested_at"`
	ExpiresAt     time.Time      `json:"expires_at"`
	ApprovedBy    string         `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time     `json:"approved_at,omitempty"`
	Status        string         `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// PrivilegeGrant represents an active privilege grant
type PrivilegeGrant struct {
	ID          string         `json:"id" gorm:"primaryKey"`
	UserID      string         `json:"user_id"`
	ResourceID  string         `json:"resource_id"`
	Level       PrivilegeLevel `json:"level"`
	GrantedAt   time.Time      `json:"granted_at"`
	ExpiresAt   time.Time      `json:"expires_at"`
	GrantedBy   string         `json:"granted_by"`
	RequestID   string         `json:"request_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
} 