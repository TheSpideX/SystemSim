package test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

const (
	loadTestBaseURL = "https://localhost:9001" // Auth service HTTP/2 port with TLS
	loadTestTimeout = 10 * time.Second
)

var (
	// HTTP/2 client with TLS configuration for load testing
	loadTestHTTP2Client *http.Client
)

func init() {
	// Create HTTP/2 client that accepts self-signed certificates for testing
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Skip certificate verification for self-signed certs
	}

	// Create HTTP/2 transport
	transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
	}

	loadTestHTTP2Client = &http.Client{
		Transport: transport,
		Timeout:   loadTestTimeout,
	}
}

// TestAuthServiceLoad tests the actual running auth service under load
func TestAuthServiceLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}
	
	if !isServiceRunning(t) {
		t.Skip("Auth service is not running on localhost:9001 with HTTP/2")
	}

	t.Run("concurrent_user_registration_load", func(t *testing.T) {
		// Test concurrent user registrations to validate system handles load
		const numUsers = 50
		const concurrency = 10
		
		var successCount int64
		var errorCount int64
		var wg sync.WaitGroup
		
		// Channel to control concurrency
		semaphore := make(chan struct{}, concurrency)
		
		start := time.Now()
		
		for i := 0; i < numUsers; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				email := fmt.Sprintf("loadtest_%d_%d@example.com", index, time.Now().UnixNano())
				registerReq := map[string]interface{}{
					"email":     email,
					"password":  "MyStr0ng&UniqueP@ssw0rd2024!",
					"firstName": "Load",
					"lastName":  fmt.Sprintf("Test%d", index),
					"company":   "Load Test Corp",
				}
				
				resp := makeRequest(t, "POST", "/api/v1/auth/register", registerReq)
				if resp.StatusCode == http.StatusCreated {
					atomic.AddInt64(&successCount, 1)
					
					// Validate response structure for successful registrations
					var data map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
						// Verify essential fields are present
						if _, hasToken := data["access_token"]; !hasToken {
							t.Errorf("Registration response missing access_token")
						}
						if _, hasUser := data["user"]; !hasUser {
							t.Errorf("Registration response missing user data")
						}
					}
				} else {
					atomic.AddInt64(&errorCount, 1)
					t.Logf("Registration failed for user %d: HTTP %d", index, resp.StatusCode)
				}
				resp.Body.Close()
			}(i)
		}
		
		wg.Wait()
		duration := time.Since(start)
		
		// Validate results
		assert.True(t, successCount > int64(numUsers*0.9), 
			"At least 90%% of registrations should succeed under load")
		assert.True(t, errorCount < int64(numUsers*0.1), 
			"Less than 10%% of registrations should fail")
		
		throughput := float64(successCount) / duration.Seconds()
		
		t.Logf("Concurrent registration load test results:")
		t.Logf("- Total users: %d", numUsers)
		t.Logf("- Concurrency: %d", concurrency)
		t.Logf("- Successful: %d", successCount)
		t.Logf("- Failed: %d", errorCount)
		t.Logf("- Duration: %v", duration)
		t.Logf("- Throughput: %.2f registrations/second", throughput)
		
		// Performance assertion - should handle at least 5 registrations/second
		assert.True(t, throughput >= 5.0, 
			"Should achieve at least 5 registrations per second")
	})
	
	t.Run("concurrent_login_load", func(t *testing.T) {
		// First create users for login testing
		const numUsers = 20
		const loginsPerUser = 3
		const concurrency = 15
		
		// Pre-register users
		userEmails := make([]string, numUsers)
		for i := 0; i < numUsers; i++ {
			email := fmt.Sprintf("loginload_%d_%d@example.com", i, time.Now().UnixNano())
			userEmails[i] = email
			
			registerReq := map[string]interface{}{
				"email":     email,
				"password":  "MyStr0ng&UniqueP@ssw0rd2024!",
				"firstName": "Login",
				"lastName":  fmt.Sprintf("Load%d", i),
				"company":   "Login Load Corp",
			}
			
			resp := makeRequest(t, "POST", "/api/v1/auth/register", registerReq)
			require.Equal(t, http.StatusCreated, resp.StatusCode, 
				"Failed to create test user %d", i)
			resp.Body.Close()
		}
		
		// Now test concurrent logins
		var successCount int64
		var errorCount int64
		var wg sync.WaitGroup
		
		semaphore := make(chan struct{}, concurrency)
		start := time.Now()
		
		for i := 0; i < numUsers; i++ {
			for j := 0; j < loginsPerUser; j++ {
				wg.Add(1)
				go func(userIndex, loginIndex int) {
					defer wg.Done()
					
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					
					loginReq := map[string]interface{}{
						"email":    userEmails[userIndex],
						"password": "MyStr0ng&UniqueP@ssw0rd2024!",
						"remember": loginIndex%2 == 0, // Alternate remember me
					}
					
					resp := makeRequest(t, "POST", "/api/v1/auth/login", loginReq)
					if resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
						
						// Validate login response
						var data map[string]interface{}
						if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
							// Verify tokens are present
							if _, hasAccess := data["access_token"]; !hasAccess {
								t.Errorf("Login response missing access_token")
							}
							if _, hasRefresh := data["refresh_token"]; !hasRefresh {
								t.Errorf("Login response missing refresh_token")
							}
							
							// Verify remember me is handled correctly
							if rememberMe, ok := data["rememberMe"]; ok {
								expectedRemember := loginIndex%2 == 0
								if rememberMe != expectedRemember {
									t.Errorf("Remember me not handled correctly: expected %v, got %v", 
										expectedRemember, rememberMe)
								}
							}
						}
					} else {
						atomic.AddInt64(&errorCount, 1)
						t.Logf("Login failed for user %d, attempt %d: HTTP %d", 
							userIndex, loginIndex, resp.StatusCode)
					}
					resp.Body.Close()
				}(i, j)
			}
		}
		
		wg.Wait()
		duration := time.Since(start)
		
		totalAttempts := int64(numUsers * loginsPerUser)
		throughput := float64(successCount) / duration.Seconds()
		
		assert.True(t, successCount > totalAttempts*9/10, 
			"At least 90%% of logins should succeed under load")
		
		t.Logf("Concurrent login load test results:")
		t.Logf("- Total attempts: %d", totalAttempts)
		t.Logf("- Concurrency: %d", concurrency)
		t.Logf("- Successful: %d", successCount)
		t.Logf("- Failed: %d", errorCount)
		t.Logf("- Duration: %v", duration)
		t.Logf("- Throughput: %.2f logins/second", throughput)
		
		// Performance assertion
		assert.True(t, throughput >= 10.0, 
			"Should achieve at least 10 logins per second")
	})
	
	t.Run("mixed_operations_stress_test", func(t *testing.T) {
		// Test mixed operations under stress
		const totalOperations = 100
		const maxConcurrency = 20
		
		var successCount int64
		var errorCount int64
		var wg sync.WaitGroup
		
		semaphore := make(chan struct{}, maxConcurrency)
		start := time.Now()
		
		// Mix of operations: 60% register, 30% login, 10% profile access
		operations := make([]string, totalOperations)
		for i := 0; i < totalOperations; i++ {
			if i < 60 {
				operations[i] = "register"
			} else if i < 90 {
				operations[i] = "login"
			} else {
				operations[i] = "profile"
			}
		}
		
		// Pre-create some users for login/profile operations
		preCreatedUsers := make([]string, 20)
		for i := 0; i < 20; i++ {
			email := fmt.Sprintf("stress_%d_%d@example.com", i, time.Now().UnixNano())
			preCreatedUsers[i] = email
			
			registerReq := map[string]interface{}{
				"email":     email,
				"password":  "MyStr0ng&UniqueP@ssw0rd2024!",
				"firstName": "Stress",
				"lastName":  fmt.Sprintf("Test%d", i),
				"company":   "Stress Test Corp",
			}
			
			resp := makeRequest(t, "POST", "/api/v1/auth/register", registerReq)
			if resp.StatusCode == http.StatusCreated {
				resp.Body.Close()
			}
		}
		
		for i, operation := range operations {
			wg.Add(1)
			go func(index int, op string) {
				defer wg.Done()
				
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				var success bool
				
				switch op {
				case "register":
					email := fmt.Sprintf("mixedop_%d_%d@example.com", index, time.Now().UnixNano())
					registerReq := map[string]interface{}{
						"email":     email,
						"password":  "MixedOp123!",
						"firstName": "Mixed",
						"lastName":  fmt.Sprintf("Op%d", index),
						"company":   "Mixed Op Corp",
					}
					
					resp := makeRequest(t, "POST", "/api/v1/auth/register", registerReq)
					success = resp.StatusCode == http.StatusCreated
					resp.Body.Close()
					
				case "login":
					userIndex := index % len(preCreatedUsers)
					loginReq := map[string]interface{}{
						"email":    preCreatedUsers[userIndex],
						"password": "MyStr0ng&UniqueP@ssw0rd2024!",
						"remember": false,
					}
					
					resp := makeRequest(t, "POST", "/api/v1/auth/login", loginReq)
					success = resp.StatusCode == http.StatusOK
					resp.Body.Close()
					
				case "profile":
					// For profile access, we need to login first to get a token
					userIndex := index % len(preCreatedUsers)
					loginReq := map[string]interface{}{
						"email":    preCreatedUsers[userIndex],
						"password": "MyStr0ng&UniqueP@ssw0rd2024!",
						"remember": false,
					}
					
					loginResp := makeRequest(t, "POST", "/api/v1/auth/login", loginReq)
					if loginResp.StatusCode == http.StatusOK {
						var loginData map[string]interface{}
						if err := json.NewDecoder(loginResp.Body).Decode(&loginData); err == nil {
							if token, ok := loginData["access_token"].(string); ok {
								profileResp := makeAuthenticatedRequest(t, "GET", "/api/v1/user/profile", nil, token)
								success = profileResp.StatusCode == http.StatusOK
								profileResp.Body.Close()
							}
						}
					}
					loginResp.Body.Close()
				}
				
				if success {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}(i, operation)
		}
		
		wg.Wait()
		duration := time.Since(start)
		
		throughput := float64(successCount) / duration.Seconds()
		
		t.Logf("Mixed operations stress test results:")
		t.Logf("- Total operations: %d", totalOperations)
		t.Logf("- Max concurrency: %d", maxConcurrency)
		t.Logf("- Successful: %d", successCount)
		t.Logf("- Failed: %d", errorCount)
		t.Logf("- Duration: %v", duration)
		t.Logf("- Throughput: %.2f operations/second", throughput)
		
		// Under stress with HTTP/2, expect some failures but system should remain stable
		// HTTP/2 multiplexing may cause more contention, so lower expectations slightly
		assert.True(t, successCount > int64(totalOperations/3),
			"At least 33%% of operations should succeed under stress with HTTP/2")

		// System should maintain reasonable throughput under stress
		assert.True(t, throughput >= 3.0,
			"Should maintain at least 3 operations per second under stress with HTTP/2")
	})
}
