package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/systemsim/auth-service/internal/testutils"
)

func TestSessionRepository_RedisOperations(t *testing.T) {
	// Focus on Redis operations that we can test without PostgreSQL
	
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	t.Run("redis_connectivity", func(t *testing.T) {
		ctx := context.Background()
		
		// Test basic Redis operations
		err := redisClient.Set(ctx, "test_key", "test_value", time.Minute).Err()
		assert.NoError(t, err, "Should be able to set value in Redis")
		
		val, err := redisClient.Get(ctx, "test_key").Result()
		assert.NoError(t, err, "Should be able to get value from Redis")
		assert.Equal(t, "test_value", val, "Retrieved value should match set value")
		
		err = redisClient.Del(ctx, "test_key").Err()
		assert.NoError(t, err, "Should be able to delete key from Redis")
	})

	t.Run("redis_session_caching", func(t *testing.T) {
		ctx := context.Background()
		
		// Test session-like data caching
		sessionID := uuid.New().String()
		sessionData := map[string]interface{}{
			"user_id":    uuid.New().String(),
			"token_hash": "hashed_token_123",
			"expires_at": time.Now().Add(15 * time.Minute).Unix(),
			"is_active":  true,
		}
		
		// Store session data
		for key, value := range sessionData {
			redisKey := fmt.Sprintf("session:%s:%s", sessionID, key)
			err := redisClient.Set(ctx, redisKey, value, 15*time.Minute).Err()
			assert.NoError(t, err, "Should be able to store session data in Redis")
		}
		
		// Retrieve session data
		for key, expectedValue := range sessionData {
			redisKey := fmt.Sprintf("session:%s:%s", sessionID, key)
			val, err := redisClient.Get(ctx, redisKey).Result()
			assert.NoError(t, err, "Should be able to retrieve session data from Redis")
			
			// Convert back and compare (simplified for test)
			switch key {
			case "user_id", "token_hash":
				assert.Equal(t, fmt.Sprintf("%v", expectedValue), val)
			case "expires_at":
				assert.Equal(t, fmt.Sprintf("%v", expectedValue), val)
			case "is_active":
				// Redis stores boolean as "1" for true
				if expectedValue == true {
					assert.Equal(t, "1", val)
				} else {
					assert.Equal(t, "0", val)
				}
			}
		}
		
		// Clean up session data
		pattern := fmt.Sprintf("session:%s:*", sessionID)
		keys, err := redisClient.Keys(ctx, pattern).Result()
		assert.NoError(t, err, "Should be able to find session keys")
		
		if len(keys) > 0 {
			err = redisClient.Del(ctx, keys...).Err()
			assert.NoError(t, err, "Should be able to delete session keys")
		}
	})

	t.Run("redis_session_expiration", func(t *testing.T) {
		ctx := context.Background()
		
		// Test session expiration handling
		sessionID := uuid.New().String()
		shortTTL := 100 * time.Millisecond
		
		// Store session with short TTL
		redisKey := fmt.Sprintf("session:%s:token", sessionID)
		err := redisClient.Set(ctx, redisKey, "test_token", shortTTL).Err()
		assert.NoError(t, err, "Should be able to store session with TTL")
		
		// Verify session exists
		val, err := redisClient.Get(ctx, redisKey).Result()
		assert.NoError(t, err, "Session should exist before expiration")
		assert.Equal(t, "test_token", val)
		
		// Wait for expiration
		time.Sleep(150 * time.Millisecond)
		
		// Verify session is expired
		_, err = redisClient.Get(ctx, redisKey).Result()
		assert.Error(t, err, "Session should be expired")
		assert.Contains(t, err.Error(), "redis: nil", "Should get nil error for expired key")
	})

	t.Run("redis_concurrent_operations", func(t *testing.T) {
		ctx := context.Background()
		
		// Test concurrent Redis operations
		const numGoroutines = 10
		results := make(chan error, numGoroutines)
		
		// Perform concurrent operations
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				sessionID := fmt.Sprintf("concurrent_session_%d", index)
				redisKey := fmt.Sprintf("session:%s:data", sessionID)
				
				// Set value
				err := redisClient.Set(ctx, redisKey, fmt.Sprintf("data_%d", index), time.Minute).Err()
				if err != nil {
					results <- err
					return
				}
				
				// Get value
				val, err := redisClient.Get(ctx, redisKey).Result()
				if err != nil {
					results <- err
					return
				}
				
				// Verify value
				expected := fmt.Sprintf("data_%d", index)
				if val != expected {
					results <- fmt.Errorf("expected %s, got %s", expected, val)
					return
				}
				
				results <- nil
			}(i)
		}
		
		// Collect results
		var errors []error
		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				errors = append(errors, err)
			}
		}
		
		assert.Empty(t, errors, "No errors should occur during concurrent operations")
	})

	t.Run("redis_hash_operations", func(t *testing.T) {
		ctx := context.Background()
		
		// Test Redis hash operations for session data
		sessionID := uuid.New().String()
		hashKey := fmt.Sprintf("session:%s", sessionID)
		
		sessionData := map[string]interface{}{
			"user_id":     uuid.New().String(),
			"token_hash":  "hashed_token_123",
			"expires_at":  time.Now().Add(15 * time.Minute).Unix(),
			"is_active":   "true",
			"ip_address":  "192.168.1.100",
			"user_agent":  "Test Agent",
		}
		
		// Store session data as hash
		err := redisClient.HMSet(ctx, hashKey, sessionData).Err()
		assert.NoError(t, err, "Should be able to store session hash")
		
		// Set expiration on the hash
		err = redisClient.Expire(ctx, hashKey, 15*time.Minute).Err()
		assert.NoError(t, err, "Should be able to set expiration on hash")
		
		// Retrieve all session data
		retrievedData, err := redisClient.HGetAll(ctx, hashKey).Result()
		assert.NoError(t, err, "Should be able to retrieve session hash")
		
		// Verify data
		for key, expectedValue := range sessionData {
			actualValue, exists := retrievedData[key]
			assert.True(t, exists, "Key %s should exist in retrieved data", key)
			assert.Equal(t, fmt.Sprintf("%v", expectedValue), actualValue, 
				"Value for key %s should match", key)
		}
		
		// Test individual field retrieval
		userID, err := redisClient.HGet(ctx, hashKey, "user_id").Result()
		assert.NoError(t, err, "Should be able to get individual field")
		assert.Equal(t, sessionData["user_id"], userID)
		
		// Test field update
		newTokenHash := "updated_token_hash_456"
		err = redisClient.HSet(ctx, hashKey, "token_hash", newTokenHash).Err()
		assert.NoError(t, err, "Should be able to update field")
		
		updatedTokenHash, err := redisClient.HGet(ctx, hashKey, "token_hash").Result()
		assert.NoError(t, err, "Should be able to get updated field")
		assert.Equal(t, newTokenHash, updatedTokenHash)
		
		// Clean up
		err = redisClient.Del(ctx, hashKey).Err()
		assert.NoError(t, err, "Should be able to delete session hash")
	})

	t.Run("redis_pattern_operations", func(t *testing.T) {
		ctx := context.Background()
		
		// Test pattern-based operations for session management
		userID := uuid.New().String()
		
		// Create multiple sessions for the same user
		sessionIDs := make([]string, 3)
		for i := 0; i < 3; i++ {
			sessionIDs[i] = uuid.New().String()
			sessionKey := fmt.Sprintf("user:%s:session:%s", userID, sessionIDs[i])
			
			err := redisClient.Set(ctx, sessionKey, fmt.Sprintf("session_data_%d", i), time.Hour).Err()
			assert.NoError(t, err, "Should be able to create session %d", i)
		}
		
		// Find all sessions for the user
		pattern := fmt.Sprintf("user:%s:session:*", userID)
		keys, err := redisClient.Keys(ctx, pattern).Result()
		assert.NoError(t, err, "Should be able to find sessions by pattern")
		assert.Len(t, keys, 3, "Should find all 3 sessions")
		
		// Verify each session exists
		for _, key := range keys {
			exists, err := redisClient.Exists(ctx, key).Result()
			assert.NoError(t, err, "Should be able to check key existence")
			assert.Equal(t, int64(1), exists, "Session should exist")
		}
		
		// Delete all sessions for the user
		if len(keys) > 0 {
			err = redisClient.Del(ctx, keys...).Err()
			assert.NoError(t, err, "Should be able to delete all user sessions")
		}
		
		// Verify sessions are deleted
		remainingKeys, err := redisClient.Keys(ctx, pattern).Result()
		assert.NoError(t, err, "Should be able to search for remaining keys")
		assert.Empty(t, remainingKeys, "No sessions should remain after deletion")
	})

	t.Run("redis_pipeline_operations", func(t *testing.T) {
		ctx := context.Background()
		
		// Test Redis pipeline for batch operations
		sessionID := uuid.New().String()
		
		// Use pipeline for multiple operations
		pipe := redisClient.Pipeline()
		
		// Queue multiple operations
		sessionKey := fmt.Sprintf("session:%s", sessionID)
		pipe.HSet(ctx, sessionKey, "user_id", uuid.New().String())
		pipe.HSet(ctx, sessionKey, "token_hash", "batch_token_hash")
		pipe.HSet(ctx, sessionKey, "is_active", "true")
		pipe.Expire(ctx, sessionKey, time.Hour)
		
		// Execute pipeline
		cmds, err := pipe.Exec(ctx)
		assert.NoError(t, err, "Pipeline should execute successfully")
		assert.Len(t, cmds, 4, "Should have 4 commands in pipeline")
		
		// Verify all operations succeeded
		for i, cmd := range cmds {
			assert.NoError(t, cmd.Err(), "Command %d should succeed", i)
		}
		
		// Verify data was stored
		exists, err := redisClient.Exists(ctx, sessionKey).Result()
		assert.NoError(t, err, "Should be able to check existence")
		assert.Equal(t, int64(1), exists, "Session should exist")
		
		// Clean up
		err = redisClient.Del(ctx, sessionKey).Err()
		assert.NoError(t, err, "Should be able to clean up")
	})
}
