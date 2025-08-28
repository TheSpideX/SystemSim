package services

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/repository"
)

// RBACService handles RBAC business logic
type RBACService struct {
	rbacRepo *repository.RBACRepository
	userRepo *repository.UserRepository
}

// NewRBACService creates a new RBAC service
func NewRBACService(rbacRepo *repository.RBACRepository, userRepo *repository.UserRepository) *RBACService {
	return &RBACService{
		rbacRepo: rbacRepo,
		userRepo: userRepo,
	}
}

// GetUserRoles retrieves all roles for a user
func (s *RBACService) GetUserRoles(userID uuid.UUID) ([]*models.RoleResponse, error) {
	roles, err := s.rbacRepo.GetUserRoles(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	responses := make([]*models.RoleResponse, len(roles))
	for i, role := range roles {
		responses[i] = role.ToResponse()
	}

	return responses, nil
}

// GetUserPermissions retrieves all permissions for a user
func (s *RBACService) GetUserPermissions(userID uuid.UUID) ([]*models.PermissionResponse, error) {
	permissions, err := s.rbacRepo.GetUserPermissions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	responses := make([]*models.PermissionResponse, len(permissions))
	for i, permission := range permissions {
		responses[i] = permission.ToResponse()
	}

	return responses, nil
}

// HasPermission checks if a user has a specific permission
func (s *RBACService) HasPermission(userID uuid.UUID, permissionName string) (bool, error) {
	return s.rbacRepo.HasPermission(userID, permissionName)
}

// AssignRoleToUser assigns a role to a user (admin only)
func (s *RBACService) AssignRoleToUser(adminUserID, targetUserID uuid.UUID, roleName string) error {
	// Check if admin has permission
	hasPermission, err := s.rbacRepo.HasPermission(adminUserID, models.PermissionUsersUpdate)
	if err != nil {
		return fmt.Errorf("failed to check admin permission: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to assign roles")
	}

	// Check if target user exists
	_, err = s.userRepo.GetByID(targetUserID)
	if err != nil {
		return fmt.Errorf("target user not found: %w", err)
	}

	// Get role by name
	role, err := s.rbacRepo.GetRoleByName(roleName)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Assign role
	if err := s.rbacRepo.AssignRoleToUser(targetUserID, role.ID, adminUserID); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// RemoveRoleFromUser removes a role from a user (admin only)
func (s *RBACService) RemoveRoleFromUser(adminUserID, targetUserID uuid.UUID, roleName string) error {
	// Check if admin has permission
	hasPermission, err := s.rbacRepo.HasPermission(adminUserID, models.PermissionUsersUpdate)
	if err != nil {
		return fmt.Errorf("failed to check admin permission: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to remove roles")
	}

	// Get role by name
	role, err := s.rbacRepo.GetRoleByName(roleName)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Don't allow removing system roles from system users
	if role.IsSystem {
		isAdmin, err := s.rbacRepo.IsUserAdmin(targetUserID)
		if err != nil {
			return fmt.Errorf("failed to check user admin status: %w", err)
		}
		if isAdmin && role.Name == models.RoleAdmin {
			return fmt.Errorf("cannot remove admin role from admin user")
		}
	}

	// Remove role
	if err := s.rbacRepo.RemoveRoleFromUser(targetUserID, role.ID); err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	return nil
}

// GetAllRoles retrieves all system roles (admin only)
func (s *RBACService) GetAllRoles(userID uuid.UUID) ([]*models.RoleResponse, error) {
	// Check if user has permission
	hasPermission, err := s.rbacRepo.HasPermission(userID, models.PermissionSystemAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}
	if !hasPermission {
		return nil, fmt.Errorf("insufficient permissions to view all roles")
	}

	roles, err := s.rbacRepo.GetAllRoles()
	if err != nil {
		return nil, fmt.Errorf("failed to get all roles: %w", err)
	}

	responses := make([]*models.RoleResponse, len(roles))
	for i, role := range roles {
		responses[i] = role.ToResponse()
	}

	return responses, nil
}

// GetAllPermissions retrieves all system permissions (admin only)
func (s *RBACService) GetAllPermissions(userID uuid.UUID) ([]*models.PermissionResponse, error) {
	// Check if user has permission
	hasPermission, err := s.rbacRepo.HasPermission(userID, models.PermissionSystemAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}
	if !hasPermission {
		return nil, fmt.Errorf("insufficient permissions to view all permissions")
	}

	permissions, err := s.rbacRepo.GetAllPermissions()
	if err != nil {
		return nil, fmt.Errorf("failed to get all permissions: %w", err)
	}

	responses := make([]*models.PermissionResponse, len(permissions))
	for i, permission := range permissions {
		responses[i] = permission.ToResponse()
	}

	return responses, nil
}

// IsUserAdmin checks if a user has admin role
func (s *RBACService) IsUserAdmin(userID uuid.UUID) (bool, error) {
	return s.rbacRepo.IsUserAdmin(userID)
}

// EnsureUserHasDefaultRole ensures a new user has the default 'user' role
func (s *RBACService) EnsureUserHasDefaultRole(userID uuid.UUID) error {
	// Get default user role
	userRole, err := s.rbacRepo.GetRoleByName(models.RoleUser)
	if err != nil {
		return fmt.Errorf("failed to get default user role: %w", err)
	}

	// Assign default role (use system UUID for system assignment)
	systemUserID := uuid.Nil // System assignment
	if err := s.rbacRepo.AssignRoleToUser(userID, userRole.ID, systemUserID); err != nil {
		return fmt.Errorf("failed to assign default role: %w", err)
	}

	return nil
}

// ValidatePermission is a helper function to validate permissions in middleware
func (s *RBACService) ValidatePermission(userID uuid.UUID, permissionName string) error {
	hasPermission, err := s.rbacRepo.HasPermission(userID, permissionName)
	if err != nil {
		return fmt.Errorf("failed to check permission: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions: %s", permissionName)
	}
	return nil
}

// GetUserPermissionsForGRPC returns user permissions as string slice (for gRPC)
func (s *RBACService) GetUserPermissionsForGRPC(userID string) ([]string, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	permissions, err := s.rbacRepo.GetUserPermissions(userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	permissionNames := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionNames[i] = permission.Name
	}

	return permissionNames, nil
}

// CheckUserPermission checks if a user has a specific permission (for gRPC)
func (s *RBACService) CheckUserPermission(userID, permissionName string) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.rbacRepo.HasPermission(userUUID, permissionName)
}

// GetUserRolesForGRPC returns user roles as string slice (for gRPC)
func (s *RBACService) GetUserRolesForGRPC(userID string) ([]string, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	roles, err := s.rbacRepo.GetUserRoles(userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	return roleNames, nil
}

// GetUserRolesWithDetails returns detailed role information (for gRPC)
func (s *RBACService) GetUserRolesWithDetails(userID string) ([]*models.Role, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.rbacRepo.GetUserRoles(userUUID)
}

// GetRolePermissions returns permissions for a specific role (for gRPC)
func (s *RBACService) GetRolePermissions(roleID string) ([]string, error) {
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	permissions, err := s.rbacRepo.GetRolePermissions(roleUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	permissionNames := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionNames[i] = permission.Name
	}

	return permissionNames, nil
}
