package models

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a system role
type Role struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	IsSystem    bool      `json:"is_system" db:"is_system"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Permission represents a system permission
type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// UserRole represents a user's role assignment
type UserRole struct {
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	RoleID     uuid.UUID  `json:"role_id" db:"role_id"`
	AssignedAt time.Time  `json:"assigned_at" db:"assigned_at"`
	AssignedBy *uuid.UUID `json:"assigned_by" db:"assigned_by"`
}

// RolePermission represents a role's permission assignment
type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id" db:"role_id"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
}

// RoleResponse represents role data returned to clients
type RoleResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PermissionResponse represents permission data returned to clients
type PermissionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserRoleResponse represents user role assignment data
type UserRoleResponse struct {
	UserID     uuid.UUID  `json:"user_id"`
	RoleID     uuid.UUID  `json:"role_id"`
	RoleName   string     `json:"role_name"`
	AssignedAt time.Time  `json:"assigned_at"`
	AssignedBy *uuid.UUID `json:"assigned_by"`
}

// ToResponse converts Role to RoleResponse
func (r *Role) ToResponse() *RoleResponse {
	return &RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// ToResponse converts Permission to PermissionResponse
func (p *Permission) ToResponse() *PermissionResponse {
	return &PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
	}
}

// System role constants
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// System permission constants
const (
	PermissionUsersCreate   = "users.create"
	PermissionUsersRead     = "users.read"
	PermissionUsersUpdate   = "users.update"
	PermissionUsersDelete   = "users.delete"
	PermissionSessionsRead  = "sessions.read"
	PermissionSessionsRevoke = "sessions.revoke"
	PermissionSystemAdmin   = "system.admin"
)
