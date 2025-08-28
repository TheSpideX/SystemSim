package database

import (
	"database/sql"
	"fmt"
	"log"
)

// EnsureSchema checks if tables exist and creates them if they don't
func EnsureSchema(db *sql.DB) error {
	log.Println("Checking database schema...")

	// Check if users table exists
	if !tableExists(db, "users") {
		log.Println("Creating users table...")
		if err := createUsersTable(db); err != nil {
			return fmt.Errorf("failed to create users table: %w", err)
		}
	} else {
		log.Println("Users table already exists")
	}

	// Check if user_sessions table exists
	if !tableExists(db, "user_sessions") {
		log.Println("Creating user_sessions table...")
		if err := createUserSessionsTable(db); err != nil {
			return fmt.Errorf("failed to create user_sessions table: %w", err)
		}
	} else {
		log.Println("User_sessions table already exists")
	}

	// Check if roles table exists
	if !tableExists(db, "roles") {
		log.Println("Creating roles table...")
		if err := createRolesTable(db); err != nil {
			return fmt.Errorf("failed to create roles table: %w", err)
		}
	} else {
		log.Println("Roles table already exists")
	}

	// Check if permissions table exists
	if !tableExists(db, "permissions") {
		log.Println("Creating permissions table...")
		if err := createPermissionsTable(db); err != nil {
			return fmt.Errorf("failed to create permissions table: %w", err)
		}
	} else {
		log.Println("Permissions table already exists")
	}

	// Check if user_roles table exists
	if !tableExists(db, "user_roles") {
		log.Println("Creating user_roles table...")
		if err := createUserRolesTable(db); err != nil {
			return fmt.Errorf("failed to create user_roles table: %w", err)
		}
	} else {
		log.Println("User_roles table already exists")
	}

	// Check if role_permissions table exists
	if !tableExists(db, "role_permissions") {
		log.Println("Creating role_permissions table...")
		if err := createRolePermissionsTable(db); err != nil {
			return fmt.Errorf("failed to create role_permissions table: %w", err)
		}
	} else {
		log.Println("Role_permissions table already exists")
	}

	// Create indexes if they don't exist
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Insert initial RBAC data
	if err := insertInitialRBACData(db); err != nil {
		return fmt.Errorf("failed to insert initial RBAC data: %w", err)
	}

	log.Println("Database schema is ready!")
	return nil
}

// tableExists checks if a table exists in the database
func tableExists(db *sql.DB, tableName string) bool {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)`
	
	var exists bool
	err := db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		log.Printf("Error checking if table %s exists: %v", tableName, err)
		return false
	}
	
	return exists
}

// createUsersTable creates the users table
func createUsersTable(db *sql.DB) error {
	query := `
		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			
			-- Profile information
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			company VARCHAR(200),
			
			-- Email verification
			email_verified BOOLEAN DEFAULT FALSE,
			email_verification_token VARCHAR(255),
			email_verification_expires_at TIMESTAMP,
			email_verification_attempts INTEGER DEFAULT 0,
			
			-- Password reset
			password_reset_token VARCHAR(255),
			password_reset_expires_at TIMESTAMP,
			password_reset_attempts INTEGER DEFAULT 0,
			
			-- Account security
			failed_login_attempts INTEGER DEFAULT 0,
			locked_until TIMESTAMP,
			last_login_at TIMESTAMP,
			last_login_ip VARCHAR(45),
			
			-- Account status
			is_active BOOLEAN DEFAULT TRUE,
			is_admin BOOLEAN DEFAULT FALSE,
			
			-- Preferences (JSONB for flexible storage)
			simulation_preferences JSONB DEFAULT '{}',
			ui_preferences JSONB DEFAULT '{}',
			
			-- Audit fields
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`
	
	_, err := db.Exec(query)
	return err
}

// createUserSessionsTable creates the user_sessions table
func createUserSessionsTable(db *sql.DB) error {
	query := `
		CREATE TABLE user_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL,
			refresh_token_hash TEXT,
			
			-- Session metadata
			device_info JSONB DEFAULT '{}',
			user_agent TEXT,
			ip_address VARCHAR(45),
			
			-- Session timing
			expires_at TIMESTAMP NOT NULL,
			refresh_expires_at TIMESTAMP,
			last_used_at TIMESTAMP DEFAULT NOW(),
			
			-- Session status
			is_active BOOLEAN DEFAULT TRUE,
			revoked_at TIMESTAMP,
			revoked_reason VARCHAR(255),
			
			-- Audit fields
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`
	
	_, err := db.Exec(query)
	return err
}

// createIndexes creates necessary indexes for performance
func createIndexes(db *sql.DB) error {
	indexes := []string{
		// Users table indexes
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_verification_token ON users(email_verification_token) WHERE email_verification_token IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_password_reset_token ON users(password_reset_token) WHERE password_reset_token IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_active ON users(is_active) WHERE is_active = TRUE",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL",
		
		// User sessions table indexes
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_sessions_token_hash ON user_sessions(token_hash)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_sessions_refresh_token_hash ON user_sessions(refresh_token_hash) WHERE refresh_token_hash IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_sessions_active ON user_sessions(is_active) WHERE is_active = TRUE",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at)",
	}
	
	for _, indexQuery := range indexes {
		if _, err := db.Exec(indexQuery); err != nil {
			// Log the error but don't fail - indexes might already exist
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}
	
	return nil
}

// createRolesTable creates the roles table
func createRolesTable(db *sql.DB) error {
	query := `
		CREATE TABLE roles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(50) UNIQUE NOT NULL,
			description TEXT,
			is_system BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`

	_, err := db.Exec(query)
	return err
}

// createPermissionsTable creates the permissions table
func createPermissionsTable(db *sql.DB) error {
	query := `
		CREATE TABLE permissions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) UNIQUE NOT NULL,
			resource VARCHAR(50) NOT NULL,
			action VARCHAR(50) NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`

	_, err := db.Exec(query)
	return err
}

// createUserRolesTable creates the user_roles table
func createUserRolesTable(db *sql.DB) error {
	query := `
		CREATE TABLE user_roles (
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
			assigned_at TIMESTAMP DEFAULT NOW(),
			assigned_by UUID REFERENCES users(id),
			PRIMARY KEY (user_id, role_id)
		)`

	_, err := db.Exec(query)
	return err
}

// createRolePermissionsTable creates the role_permissions table
func createRolePermissionsTable(db *sql.DB) error {
	query := `
		CREATE TABLE role_permissions (
			role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
			permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
			PRIMARY KEY (role_id, permission_id)
		)`

	_, err := db.Exec(query)
	return err
}

// insertInitialRBACData inserts initial roles and permissions
func insertInitialRBACData(db *sql.DB) error {
	// Check if data already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM roles").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing roles: %w", err)
	}

	if count > 0 {
		log.Println("RBAC data already exists, skipping initialization")
		return nil
	}

	log.Println("Inserting initial RBAC data...")

	// Insert system roles
	roleQueries := []string{
		"INSERT INTO roles (name, description, is_system) VALUES ('admin', 'System administrator with full access', true)",
		"INSERT INTO roles (name, description, is_system) VALUES ('user', 'Standard user with basic access', true)",
	}

	for _, query := range roleQueries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to insert role: %w", err)
		}
	}

	// Insert system permissions
	permissionQueries := []string{
		"INSERT INTO permissions (name, resource, action, description) VALUES ('users.create', 'users', 'create', 'Create new users')",
		"INSERT INTO permissions (name, resource, action, description) VALUES ('users.read', 'users', 'read', 'View user information')",
		"INSERT INTO permissions (name, resource, action, description) VALUES ('users.update', 'users', 'update', 'Update user information')",
		"INSERT INTO permissions (name, resource, action, description) VALUES ('users.delete', 'users', 'delete', 'Delete users')",
		"INSERT INTO permissions (name, resource, action, description) VALUES ('sessions.read', 'sessions', 'read', 'View user sessions')",
		"INSERT INTO permissions (name, resource, action, description) VALUES ('sessions.revoke', 'sessions', 'revoke', 'Revoke user sessions')",
		"INSERT INTO permissions (name, resource, action, description) VALUES ('system.admin', 'system', 'admin', 'System administration access')",
	}

	for _, query := range permissionQueries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to insert permission: %w", err)
		}
	}

	// Assign permissions to roles
	rolePermissionQueries := []string{
		// Admin gets all permissions
		`INSERT INTO role_permissions (role_id, permission_id)
		 SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin'`,

		// User gets basic permissions
		`INSERT INTO role_permissions (role_id, permission_id)
		 SELECT r.id, p.id FROM roles r, permissions p
		 WHERE r.name = 'user' AND p.name IN ('users.read', 'sessions.read')`,
	}

	for _, query := range rolePermissionQueries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to assign role permissions: %w", err)
		}
	}

	log.Println("Initial RBAC data inserted successfully")
	return nil
}

// DropSchema drops all tables (for testing purposes)
func DropSchema(db *sql.DB) error {
	queries := []string{
		"DROP TABLE IF EXISTS user_sessions CASCADE",
		"DROP TABLE IF EXISTS users CASCADE",
	}
	
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table: %w", err)
		}
	}
	
	return nil
}
