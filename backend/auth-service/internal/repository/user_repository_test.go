package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/testutils"
	"golang.org/x/crypto/bcrypt"
)

// createTestUser creates a test user in the database for repository tests
func createTestUser(t *testing.T, repo *UserRepository, email string) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		ID:            uuid.New(),
		Email:         email,
		PasswordHash:  string(hashedPassword),
		FirstName:     "Test",
		LastName:      "User",
		Company:       "Test Company",
		EmailVerified: false,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = repo.Create(user)
	require.NoError(t, err)

	return user
}

func TestUserRepository_Create(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	tests := []struct {
		name        string
		user        *models.User
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_user_creation",
			user: &models.User{
				ID:            uuid.New(),
				Email:         "test@example.com",
				PasswordHash:  "$2a$12$hashedpassword",
				FirstName:     "John",
				LastName:      "Doe",
				Company:       "Test Company",
				EmailVerified: false,
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			expectError: false,
		},
		{
			name: "duplicate_email",
			user: &models.User{
				ID:            uuid.New(),
				Email:         "test@example.com", // Same email as above
				PasswordHash:  "$2a$12$anotherhash",
				FirstName:     "Jane",
				LastName:      "Smith",
				Company:       "Another Company",
				EmailVerified: false,
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			expectError: true,
			errorMsg:    "duplicate",
		},
		{
			name: "empty_email",
			user: &models.User{
				ID:            uuid.New(),
				Email:         "",
				PasswordHash:  "$2a$12$hashedpassword",
				FirstName:     "Test",
				LastName:      "User",
				Company:       "Test Company",
				EmailVerified: false,
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			expectError: true,
			errorMsg:    "email",
		},
		{
			name: "empty_password_hash",
			user: &models.User{
				ID:            uuid.New(),
				Email:         "nopassword@example.com",
				PasswordHash:  "",
				FirstName:     "No",
				LastName:      "Password",
				Company:       "Test Company",
				EmailVerified: false,
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			expectError: true,
			errorMsg:    "password",
		},
		{
			name: "very_long_email",
			user: &models.User{
				ID:            uuid.New(),
				Email:         "verylongemailaddressthatexceedsthenormallimitandmightcauseissues@verylongdomainnamethatisunreasonablylongandmightcauseproblems.com",
				PasswordHash:  "$2a$12$hashedpassword",
				FirstName:     "Long",
				LastName:      "Email",
				Company:       "Test Company",
				EmailVerified: false,
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			expectError: true,
			errorMsg:    "email",
		},
		{
			name: "unicode_names",
			user: &models.User{
				ID:            uuid.New(),
				Email:         "unicode@example.com",
				PasswordHash:  "$2a$12$hashedpassword",
				FirstName:     "José",
				LastName:      "García",
				Company:       "Compañía de Pruebas",
				EmailVerified: false,
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			expectError: false,
		},
		{
			name: "minimal_user_data",
			user: &models.User{
				ID:            uuid.New(),
				Email:         "minimal@example.com",
				PasswordHash:  "$2a$12$hashedpassword",
				FirstName:     "",
				LastName:      "",
				Company:       "",
				EmailVerified: false,
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.user)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, 
						"Error message should contain '%s' for test case: %s", tt.errorMsg, tt.name)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)

				// Verify user was created by retrieving it
				retrievedUser, err := repo.GetByEmail(tt.user.Email)
				require.NoError(t, err, "Should be able to retrieve created user")
				assert.Equal(t, tt.user.ID, retrievedUser.ID)
				assert.Equal(t, tt.user.Email, retrievedUser.Email)
				assert.Equal(t, tt.user.FirstName, retrievedUser.FirstName)
				assert.Equal(t, tt.user.LastName, retrievedUser.LastName)
				assert.Equal(t, tt.user.Company, retrievedUser.Company)
				assert.Equal(t, tt.user.EmailVerified, retrievedUser.EmailVerified)
				assert.Equal(t, tt.user.IsActive, retrievedUser.IsActive)
			}
		})
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	// Create test user
	testUser := createTestUser(t, repo, "getbyid@example.com")

	tests := []struct {
		name        string
		userID      uuid.UUID
		expectError bool
		errorMsg    string
	}{
		{
			name:        "existing_user",
			userID:      testUser.ID,
			expectError: false,
		},
		{
			name:        "non_existent_user",
			userID:      uuid.New(),
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "nil_uuid",
			userID:      uuid.Nil,
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByID(tt.userID)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Nil(t, user, "User should be nil when error occurs")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"Error message should contain '%s' for test case: %s", tt.errorMsg, tt.name)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
				assert.NotNil(t, user, "User should not be nil")
				assert.Equal(t, tt.userID, user.ID)
				assert.Equal(t, testUser.Email, user.Email)
				assert.Equal(t, testUser.FirstName, user.FirstName)
				assert.Equal(t, testUser.LastName, user.LastName)
			}
		})
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	// Create test user
	testUser := createTestUser(t, repo, "getbyemail@example.com")

	tests := []struct {
		name        string
		email       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "existing_user",
			email:       testUser.Email,
			expectError: false,
		},
		{
			name:        "non_existent_email",
			email:       "nonexistent@example.com",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "empty_email",
			email:       "",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "case_sensitive_email",
			email:       "GETBYEMAIL@EXAMPLE.COM",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "email_with_spaces",
			email:       " getbyemail@example.com ",
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByEmail(tt.email)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Nil(t, user, "User should be nil when error occurs")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"Error message should contain '%s' for test case: %s", tt.errorMsg, tt.name)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
				assert.NotNil(t, user, "User should not be nil")
				assert.Equal(t, tt.email, user.Email)
				assert.Equal(t, testUser.ID, user.ID)
				assert.Equal(t, testUser.FirstName, user.FirstName)
				assert.Equal(t, testUser.LastName, user.LastName)
			}
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	// Create test user
	testUser := createTestUser(t, repo, "update@example.com")

	tests := []struct {
		name        string
		user        *models.User
		expectError bool
		errorMsg    string
	}{
		{
			name: "update_first_name",
			user: &models.User{
				ID:        testUser.ID,
				Email:     testUser.Email,
				FirstName: "UpdatedFirst",
				LastName:  testUser.LastName,
				Company:   testUser.Company,
			},
			expectError: false,
		},
		{
			name: "update_last_name",
			user: &models.User{
				ID:        testUser.ID,
				Email:     testUser.Email,
				FirstName: testUser.FirstName,
				LastName:  "UpdatedLast",
				Company:   testUser.Company,
			},
			expectError: false,
		},
		{
			name: "update_company",
			user: &models.User{
				ID:        testUser.ID,
				Email:     testUser.Email,
				FirstName: testUser.FirstName,
				LastName:  testUser.LastName,
				Company:   "Updated Company",
			},
			expectError: false,
		},
		{
			name: "update_multiple_fields",
			user: &models.User{
				ID:        testUser.ID,
				Email:     testUser.Email,
				FirstName: "MultiFirst",
				LastName:  "MultiLast",
				Company:   "Multi Company",
			},
			expectError: false,
		},
		{
			name: "update_non_existent_user",
			user: &models.User{
				ID:        uuid.New(),
				Email:     "nonexistent@example.com",
				FirstName: "NonExistent",
				LastName:  "User",
				Company:   "Test Company",
			},
			expectError: true,
			errorMsg:    "no rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Update(tt.user)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"Error message should contain '%s' for test case: %s", tt.errorMsg, tt.name)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)

				// Verify updates were applied
				updatedUser, err := repo.GetByID(tt.user.ID)
				require.NoError(t, err, "Should be able to retrieve updated user")

				assert.Equal(t, tt.user.FirstName, updatedUser.FirstName)
				assert.Equal(t, tt.user.LastName, updatedUser.LastName)
				assert.Equal(t, tt.user.Company, updatedUser.Company)

				// Verify updated_at was changed
				assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt),
					"UpdatedAt should be more recent after update")
			}
		})
	}
}

func TestUserRepository_SoftDelete(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	tests := []struct {
		name        string
		setupUser   bool
		userID      uuid.UUID
		expectError bool
		errorMsg    string
	}{
		{
			name:        "soft_delete_existing_user",
			setupUser:   true,
			expectError: false,
		},
		{
			name:        "soft_delete_non_existent_user",
			setupUser:   false,
			userID:      uuid.New(),
			expectError: true,
			errorMsg:    "no rows",
		},
		{
			name:        "soft_delete_with_nil_uuid",
			setupUser:   false,
			userID:      uuid.Nil,
			expectError: true,
			errorMsg:    "no rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testUser *models.User
			var userID uuid.UUID

			if tt.setupUser {
				testUser = createTestUser(t, repo, "delete"+tt.name+"@example.com")
				userID = testUser.ID
			} else {
				userID = tt.userID
			}

			err := repo.SoftDelete(userID)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"Error message should contain '%s' for test case: %s", tt.errorMsg, tt.name)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)

				// Verify user was soft deleted (should not be retrievable by normal methods)
				_, err := repo.GetByID(userID)
				assert.Error(t, err, "Should not be able to retrieve soft deleted user")
				assert.Contains(t, err.Error(), "not found", "Error should indicate user not found")

				// Verify user is marked as inactive and has deleted_at set
				// We would need a special method to retrieve soft-deleted users to verify this
				// For now, we just verify the operation succeeded
			}
		})
	}
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	// Create test user
	testUser := createTestUser(t, repo, "updatepassword@example.com")
	originalHash := testUser.PasswordHash

	tests := []struct {
		name        string
		userID      uuid.UUID
		newHash     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_password_update",
			userID:      testUser.ID,
			newHash:     "$2a$12$newhashedpassword",
			expectError: false,
		},
		{
			name:        "empty_password_hash",
			userID:      testUser.ID,
			newHash:     "",
			expectError: true,
			errorMsg:    "password hash cannot be empty",
		},
		{
			name:        "non_existent_user",
			userID:      uuid.New(),
			newHash:     "$2a$12$anotherhashedpassword",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "nil_uuid",
			userID:      uuid.Nil,
			newHash:     "$2a$12$hashedpassword",
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdatePassword(tt.userID, tt.newHash)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"Error message should contain '%s' for test case: %s", tt.errorMsg, tt.name)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)

				// Verify password was updated
				updatedUser, err := repo.GetByID(tt.userID)
				require.NoError(t, err, "Should be able to retrieve user after password update")
				assert.Equal(t, tt.newHash, updatedUser.PasswordHash)
				assert.NotEqual(t, originalHash, updatedUser.PasswordHash)
				assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt),
					"UpdatedAt should be more recent after password update")
			}
		})
	}
}

func TestUserRepository_EmailExists(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	// Create test user
	testUser := createTestUser(t, repo, "emailexists@example.com")

	tests := []struct {
		name        string
		email       string
		expected    bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "existing_email",
			email:       testUser.Email,
			expected:    true,
			expectError: false,
		},
		{
			name:        "non_existing_email",
			email:       "nonexistent@example.com",
			expected:    false,
			expectError: false,
		},
		{
			name:        "empty_email",
			email:       "",
			expected:    false,
			expectError: false,
		},
		{
			name:        "case_sensitive_email",
			email:       "EMAILEXISTS@EXAMPLE.COM",
			expected:    false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := repo.EmailExists(tt.email)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"Error message should contain '%s' for test case: %s", tt.errorMsg, tt.name)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
				assert.Equal(t, tt.expected, exists,
					"Expected exists=%v for test case: %s", tt.expected, tt.name)
			}
		})
	}
}

func TestUserRepository_ConcurrentOperations(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	t.Run("concurrent_user_creation", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		// Create users concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				user := &models.User{
					ID:            uuid.New(),
					Email:         fmt.Sprintf("concurrent%d@example.com", index),
					PasswordHash:  "$2a$12$hashedpassword",
					FirstName:     fmt.Sprintf("User%d", index),
					LastName:      "Concurrent",
					Company:       "Test Company",
					EmailVerified: false,
					IsActive:      true,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				results <- repo.Create(user)
			}(i)
		}

		// Collect results
		var errors []error
		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				errors = append(errors, err)
			}
		}

		// All operations should succeed
		assert.Empty(t, errors, "No errors should occur during concurrent user creation")

		// Verify all users were created by checking a few of them exist
		for i := 0; i < 3; i++ {
			exists, err := repo.EmailExists(fmt.Sprintf("concurrent%d@example.com", i))
			require.NoError(t, err)
			assert.True(t, exists, "User %d should exist", i)
		}
	})

	t.Run("concurrent_updates_same_user", func(t *testing.T) {
		// Create a test user
		testUser := createTestUser(t, repo, "concurrentupdate@example.com")

		const numGoroutines = 5
		results := make(chan error, numGoroutines)

		// Update the same user concurrently with different fields
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				updatedUser := &models.User{
					ID:        testUser.ID,
					Email:     testUser.Email,
					FirstName: fmt.Sprintf("Updated%d", index),
					LastName:  testUser.LastName,
					Company:   testUser.Company,
				}
				results <- repo.Update(updatedUser)
			}(i)
		}

		// Collect results
		var errors []error
		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				errors = append(errors, err)
			}
		}

		// All operations should succeed (last writer wins)
		assert.Empty(t, errors, "No errors should occur during concurrent updates")

		// Verify user still exists and has been updated
		updatedUser, err := repo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.NotEqual(t, testUser.FirstName, updatedUser.FirstName,
			"User should have been updated")
		assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt),
			"UpdatedAt should be more recent")
	})
}

func TestUserRepository_DatabaseConstraints(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	t.Run("email_uniqueness_constraint", func(t *testing.T) {
		// Create first user
		user1 := &models.User{
			ID:            uuid.New(),
			Email:         "unique@example.com",
			PasswordHash:  "$2a$12$hashedpassword1",
			FirstName:     "First",
			LastName:      "User",
			Company:       "Test Company",
			EmailVerified: false,
			IsActive:      true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		err := repo.Create(user1)
		require.NoError(t, err)

		// Try to create second user with same email
		user2 := &models.User{
			ID:            uuid.New(),
			Email:         "unique@example.com", // Same email
			PasswordHash:  "$2a$12$hashedpassword2",
			FirstName:     "Second",
			LastName:      "User",
			Company:       "Test Company",
			EmailVerified: false,
			IsActive:      true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		err = repo.Create(user2)
		assert.Error(t, err, "Should not be able to create user with duplicate email")
		assert.Contains(t, err.Error(), "duplicate", "Error should indicate duplicate constraint violation")
	})

	t.Run("primary_key_constraint", func(t *testing.T) {
		userID := uuid.New()

		// Create first user
		user1 := &models.User{
			ID:            userID,
			Email:         "pk1@example.com",
			PasswordHash:  "$2a$12$hashedpassword1",
			FirstName:     "First",
			LastName:      "User",
			Company:       "Test Company",
			EmailVerified: false,
			IsActive:      true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		err := repo.Create(user1)
		require.NoError(t, err)

		// Try to create second user with same ID
		user2 := &models.User{
			ID:            userID, // Same ID
			Email:         "pk2@example.com",
			PasswordHash:  "$2a$12$hashedpassword2",
			FirstName:     "Second",
			LastName:      "User",
			Company:       "Test Company",
			EmailVerified: false,
			IsActive:      true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		err = repo.Create(user2)
		assert.Error(t, err, "Should not be able to create user with duplicate ID")
		assert.Contains(t, err.Error(), "duplicate", "Error should indicate primary key constraint violation")
	})
}
