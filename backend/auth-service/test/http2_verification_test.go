package test

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

// TestHTTP2Verification verifies that the auth service is actually using HTTP/2
func TestHTTP2Verification(t *testing.T) {
	if !isServiceRunning(t) {
		t.Skip("Auth service is not running on localhost:9001 with HTTP/2")
	}

	t.Run("verify_http2_protocol", func(t *testing.T) {
		// Create HTTP/2 client
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Skip certificate verification for self-signed certs
		}
		
		transport := &http2.Transport{
			TLSClientConfig: tlsConfig,
		}
		
		client := &http.Client{
			Transport: transport,
		}

		// Make request to health endpoint
		resp, err := client.Get("https://localhost:9001/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify HTTP/2 is being used
		assert.Equal(t, "HTTP/2.0", resp.Proto, "Expected HTTP/2.0 protocol")
		assert.Equal(t, 2, resp.ProtoMajor, "Expected HTTP major version 2")
		assert.Equal(t, 0, resp.ProtoMinor, "Expected HTTP minor version 0")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health check should return 200")

		t.Logf("Successfully verified HTTP/2 connection:")
		t.Logf("- Protocol: %s", resp.Proto)
		t.Logf("- Version: %d.%d", resp.ProtoMajor, resp.ProtoMinor)
		t.Logf("- Status: %d", resp.StatusCode)
		t.Logf("- TLS: %v", resp.TLS != nil)
		if resp.TLS != nil {
			t.Logf("- TLS Version: %x", resp.TLS.Version)
			t.Logf("- Cipher Suite: %x", resp.TLS.CipherSuite)
		}
	})

	t.Run("verify_http2_features", func(t *testing.T) {
		// Test HTTP/2 specific features like server push (if implemented)
		// and multiplexing by making concurrent requests
		
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		
		transport := &http2.Transport{
			TLSClientConfig: tlsConfig,
		}
		
		client := &http.Client{
			Transport: transport,
		}

		// Make multiple concurrent requests to test multiplexing
		const numRequests = 5
		responses := make([]*http.Response, numRequests)
		errors := make([]error, numRequests)
		
		// Use channels to coordinate concurrent requests
		done := make(chan int, numRequests)
		
		for i := 0; i < numRequests; i++ {
			go func(index int) {
				resp, err := client.Get("https://localhost:9001/health")
				responses[index] = resp
				errors[index] = err
				done <- index
			}(i)
		}
		
		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			<-done
		}
		
		// Verify all requests succeeded and used HTTP/2
		successCount := 0
		for i := 0; i < numRequests; i++ {
			if errors[i] == nil && responses[i] != nil {
				assert.Equal(t, "HTTP/2.0", responses[i].Proto, 
					"Request %d should use HTTP/2.0", i)
				assert.Equal(t, http.StatusOK, responses[i].StatusCode,
					"Request %d should return 200", i)
				responses[i].Body.Close()
				successCount++
			}
		}
		
		assert.Equal(t, numRequests, successCount, 
			"All concurrent requests should succeed with HTTP/2")
		
		t.Logf("Successfully tested HTTP/2 multiplexing with %d concurrent requests", numRequests)
	})

	t.Run("verify_strict_http2_only_enforcement", func(t *testing.T) {
		// Test that our server strictly enforces HTTP/2-only and rejects HTTP/1.1

		// First, verify HTTP/2 works
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}

		transport := &http2.Transport{
			TLSClientConfig: tlsConfig,
		}

		client := &http.Client{
			Transport: transport,
		}

		resp, err := client.Get("https://localhost:9001/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify HTTP/2 is working
		assert.Equal(t, "HTTP/2.0", resp.Proto, "Should use HTTP/2.0")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health check should return 200")

		t.Logf("HTTP/2 strict enforcement verified:")
		t.Logf("- Protocol: %s", resp.Proto)
		t.Logf("- Version: %d.%d", resp.ProtoMajor, resp.ProtoMinor)
		t.Logf("- Status: %d", resp.StatusCode)

		// Now test that HTTP/1.1 is strictly rejected
		http1TlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"http/1.1"}, // Force HTTP/1.1
		}

		http1Client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: http1TlsConfig,
			},
		}

		// This should fail due to strict HTTP/2-only enforcement
		_, err = http1Client.Get("https://localhost:9001/health")
		assert.Error(t, err, "HTTP/1.1 client should be rejected by strict HTTP/2-only server")

		if err != nil {
			t.Logf("HTTP/1.1 correctly rejected: %v", err)
		}
	})
}
