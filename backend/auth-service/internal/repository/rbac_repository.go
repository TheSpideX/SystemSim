package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/systemsim/auth-service/internal/models"
)

// RBACRepository handles RBAC database operations
type RBACRepository struct {
	db *sql.DB
}

// NewRBACRepository creates a new RBAC repository
func NewRBACRepository(db *sql.DB) *RBACRepository {
	return &RBACRepository{db: db}
}

// GetUserRoles retrieves all roles for a user
func (r *RBACRepository) GetUserRoles(userID uuid.UUID) ([]*models.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.is_system, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user roles: %w", err)
	}
	defer rows.Close()

	var roles []*models.Role
	for rows.Next() {
		role := &models.Role{}
		err := rows.Scan(
			&role.ID, &role.Name, &role.Description,
			&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetUserPermissions retrieves all permissions for a user
func (r *RBACRepository) GetUserPermissions(userID uuid.UUID) ([]*models.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles ro ON rp.role_id = ro.id
		JOIN user_roles ur ON ro.id = ur.role_id
		WHERE ur.user_id = $1
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*models.Permission
	for rows.Next() {
		permission := &models.Permission{}
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.Resource,
			&permission.Action, &permission.Description, &permission.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// HasPermission checks if a user has a specific permission
func (r *RBACRepository) HasPermission(userID uuid.UUID, permissionName string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM permissions p
			JOIN role_permissions rp ON p.id = rp.permission_id
			JOIN roles ro ON rp.role_id = ro.id
			JOIN user_roles ur ON ro.id = ur.role_id
			WHERE ur.user_id = $1 AND p.name = $2
		)
	`

	var exists bool
	err := r.db.QueryRow(query, userID, permissionName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return exists, nil
}

// AssignRoleToUser assigns a role to a user
func (r *RBACRepository) AssignRoleToUser(userID, roleID, assignedBy uuid.UUID) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	_, err := r.db.Exec(query, userID, roleID, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	return nil
}

// RemoveRoleFromUser removes a role from a user
func (r *RBACRepository) RemoveRoleFromUser(userID, roleID uuid.UUID) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`

	_, err := r.db.Exec(query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	return nil
}

// GetRoleByName retrieves a role by name
func (r *RBACRepository) GetRoleByName(name string) (*models.Role, error) {
	query := `
		SELECT id, name, description, is_system, created_at, updated_at
		FROM roles
		WHERE name = $1
	`

	role := &models.Role{}
	err := r.db.QueryRow(query, name).Scan(
		&role.ID, &role.Name, &role.Description,
		&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return role, nil
}

// GetAllRoles retrieves all system roles
func (r *RBACRepository) GetAllRoles() ([]*models.Role, error) {
	query := `
		SELECT id, name, description, is_system, created_at, updated_at
		FROM roles
		ORDER BY name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []*models.Role
	for rows.Next() {
		role := &models.Role{}
		err := rows.Scan(
			&role.ID, &role.Name, &role.Description,
			&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetAllPermissions retrieves all system permissions
func (r *RBACRepository) GetAllPermissions() ([]*models.Permission, error) {
	query := `
		SELECT id, name, resource, action, description, created_at
		FROM permissions
		ORDER BY resource, action
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*models.Permission
	for rows.Next() {
		permission := &models.Permission{}
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.Resource,
			&permission.Action, &permission.Description, &permission.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// IsUserAdmin checks if a user has admin role
func (r *RBACRepository) IsUserAdmin(userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM roles ro
			JOIN user_roles ur ON ro.id = ur.role_id
			WHERE ur.user_id = $1 AND ro.name = 'admin'
		)
	`

	var isAdmin bool
	err := r.db.QueryRow(query, userID).Scan(&isAdmin)
	if err != nil {
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}

	return isAdmin, nil
}

// GetRolePermissions retrieves all permissions for a specific role
func (r *RBACRepository) GetRolePermissions(roleID uuid.UUID) ([]*models.Permission, error) {
	query := `
		SELECT p.id, p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.resource, p.action
	`

	rows, err := r.db.Query(query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*models.Permission
	for rows.Next() {
		permission := &models.Permission{}
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.Resource,
			&permission.Action, &permission.Description, &permission.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}
