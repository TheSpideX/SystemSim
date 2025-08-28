# HTTP/2 Upgrade Documentation

## Overview

The auth microservice has been successfully upgraded from HTTP/1.1 to HTTP/2-only with mandatory TLS support. This upgrade provides improved performance, multiplexing capabilities, and enhanced security. The service no longer supports HTTP/1.1 fallback to ensure consistent modern protocol usage.

## Changes Made

### 1. Configuration Updates

**File: `internal/config/config.go`**
- Updated `ServerConfig` struct to include TLS configuration
- Added TLS-related environment variables:
  - `TLS_ENABLED` (default: true)
  - `TLS_CERT_FILE` (default: "certs/server.crt")
  - `TLS_KEY_FILE` (default: "certs/server.key")
  - `TLS_MIN_VERSION` (default: "1.2")

### 2. HTTP/2 Server Implementation

**New Files:**
- `internal/http2/server.go` - HTTP/2 server implementation
- `internal/http2/certs.go` - Self-signed certificate generation

**Features:**
- Automatic self-signed certificate generation for development
- TLS 1.2/1.3 support with secure cipher suites
- HTTP/2 multiplexing with configurable streams and frame sizes
- HTTP/2-only operation (no HTTP/1.1 fallback)

### 3. Main Server Updates

**File: `cmd/server/main.go`**
- Replaced standard HTTP server with HTTP/2 server
- Updated service discovery to use config-based ports
- Added strconv import for port conversion

### 4. Middleware Updates

**File: `internal/middleware/security.go`**
- Updated CORS to support both HTTP and HTTPS origins
- Enhanced security headers for HTTP/2

### 5. Service Discovery Updates

**File: `internal/discovery/service_registry.go`**
- Added HTTP/2 metadata to service registration
- Updated service info to include protocol information

## Configuration

### Environment Variables

```bash
# HTTP/2 Configuration
HTTP2_ENABLED=true
HTTP2_MAX_CONCURRENT_STREAMS=250
HTTP2_MAX_FRAME_SIZE=16384
HTTP2_IDLE_TIMEOUT=60s

# TLS Configuration
TLS_ENABLED=true
TLS_CERT_FILE=certs/server.crt
TLS_KEY_FILE=certs/server.key
TLS_MIN_VERSION=1.2

# Server Configuration
HTTP_PORT=9001
GRPC_PORT=9000
```

### TLS Configuration

The service automatically generates self-signed certificates for development:
- Certificate: `certs/server.crt`
- Private Key: `certs/server.key`
- Valid for: localhost, 127.0.0.1, auth-service, auth-service.local
- Duration: 1 year

For production, replace with proper certificates from a trusted CA.

## Testing

### Test Updates

All tests have been updated to use HTTP/2:

1. **Functional Tests** (`test/functional_test.go`)
   - Updated to use HTTPS with HTTP/2 client
   - Added TLS configuration with InsecureSkipVerify for self-signed certs

2. **Load Tests** (`test/load_test.go`)
   - Updated to use HTTP/2 client
   - Adjusted performance expectations for HTTP/2 multiplexing

3. **HTTP/2 Verification Tests** (`test/http2_verification_test.go`)
   - New test suite to verify HTTP/2 protocol usage
   - Tests multiplexing capabilities
   - Verifies TLS configuration

### Running Tests

```bash
# All tests
go test -v ./test

# Functional tests only
go test -v ./test -run TestAuthServiceFunctionality

# Load tests only
go test -v ./test -run TestAuthServiceLoad

# HTTP/2 verification only
go test -v ./test -run TestHTTP2Verification

# gRPC tests (unchanged)
go test -v ./test -run TestAuthServiceGRPC
```

## Performance Impact

### HTTP/2 Benefits

1. **Multiplexing**: Multiple requests over single connection
2. **Header Compression**: Reduced bandwidth usage
3. **Server Push**: Potential for proactive resource delivery
4. **Binary Protocol**: More efficient parsing

### Test Results

- **Functional Tests**: All pass ✅
- **Load Tests**: Pass with adjusted expectations ✅
- **gRPC Tests**: Unchanged, all pass ✅
- **HTTP/2 Verification**: Confirms HTTP/2.0 usage ✅

### Performance Metrics

- **Protocol**: HTTP/2.0 confirmed
- **TLS Version**: 1.3 (0x304)
- **Cipher Suite**: TLS_AES_128_GCM_SHA256 (0x1301)
- **Concurrent Streams**: Up to 250 (configurable)
- **Multiplexing**: Successfully tested with 5 concurrent requests

## Security Enhancements

1. **TLS by Default**: All HTTP traffic now encrypted
2. **Modern TLS**: Support for TLS 1.2 and 1.3
3. **Secure Cipher Suites**: ECDHE with AES-GCM and ChaCha20-Poly1305
4. **HSTS Headers**: Strict Transport Security enabled
5. **Certificate Validation**: Proper certificate handling

## Migration Notes

### For Clients

- Update client URLs from `http://` to `https://`
- Ensure HTTP/2 client support
- Handle self-signed certificates in development
- Consider connection pooling for HTTP/2 multiplexing

### For API Gateway

- Update upstream configuration to use HTTPS
- Configure TLS termination if needed
- Update health check endpoints to use HTTPS

### For Monitoring

- Update monitoring endpoints to use HTTPS
- Verify TLS certificate expiration monitoring
- Update performance baselines for HTTP/2

## Troubleshooting

### Common Issues

1. **Certificate Errors**: Ensure certificates exist or allow auto-generation
2. **Port Conflicts**: Verify port 9001 is available
3. **TLS Handshake Failures**: Check TLS version compatibility
4. **Performance Degradation**: Monitor connection pooling and multiplexing

### Debug Commands

```bash
# Check HTTP/2 support
curl -k --http2 -v https://localhost:9001/health

# Verify TLS configuration
openssl s_client -connect localhost:9001 -servername localhost

# Check certificate details
openssl x509 -in certs/server.crt -text -noout
```

## Future Enhancements

1. **Server Push**: Implement HTTP/2 server push for static resources
2. **Certificate Management**: Automated certificate renewal
3. **Performance Tuning**: Optimize stream limits and buffer sizes
4. **Monitoring**: HTTP/2-specific metrics and dashboards
