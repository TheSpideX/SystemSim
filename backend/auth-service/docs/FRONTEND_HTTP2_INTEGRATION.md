# Frontend HTTP/2 Integration Guide

## Overview

The auth microservice now operates in **strict HTTP/2-only mode** with mandatory TLS. All frontend clients must be configured to use HTTP/2 with proper certificate handling.

## Requirements

### 1. Protocol Requirements
- **HTTP/2 Only**: No HTTP/1.1 fallback support
- **TLS Mandatory**: All connections must use HTTPS
- **Port**: 9001 (HTTPS with HTTP/2)

### 2. Certificate Requirements

**Development Environment:**
- Self-signed certificates are auto-generated
- Frontend must accept self-signed certificates
- Certificate files: `certs/server.crt`, `certs/server.key`

**Production Environment:**
- Use proper certificates from trusted CA
- Configure certificate validation properly

## Frontend Implementation Examples

### Node.js/Express Frontend

```javascript
const http2 = require('http2');
const fs = require('fs');

// Development configuration (self-signed certificates)
const authClient = http2.connect('https://localhost:9001', {
  rejectUnauthorized: false, // Accept self-signed certs in development
});

// Production configuration
const authClientProd = http2.connect('https://auth-service.yourdomain.com:9001', {
  // Use proper certificate validation in production
});

// Example: User registration
async function registerUser(userData) {
  return new Promise((resolve, reject) => {
    const req = authClient.request({
      ':method': 'POST',
      ':path': '/api/v1/auth/register',
      'content-type': 'application/json',
    });

    req.on('response', (headers) => {
      let data = '';
      req.on('data', (chunk) => data += chunk);
      req.on('end', () => {
        resolve({
          status: headers[':status'],
          data: JSON.parse(data)
        });
      });
    });

    req.on('error', reject);
    req.write(JSON.stringify(userData));
    req.end();
  });
}

// Example: User login
async function loginUser(credentials) {
  return new Promise((resolve, reject) => {
    const req = authClient.request({
      ':method': 'POST',
      ':path': '/api/v1/auth/login',
      'content-type': 'application/json',
    });

    req.on('response', (headers) => {
      let data = '';
      req.on('data', (chunk) => data += chunk);
      req.on('end', () => {
        resolve({
          status: headers[':status'],
          data: JSON.parse(data)
        });
      });
    });

    req.on('error', reject);
    req.write(JSON.stringify(credentials));
    req.end();
  });
}
```

### React/Next.js Frontend

```javascript
// utils/authClient.js
class AuthHTTP2Client {
  constructor() {
    // For browser environments, use fetch with HTTP/2 support
    this.baseURL = process.env.NODE_ENV === 'development' 
      ? 'https://localhost:9001' 
      : 'https://auth-service.yourdomain.com:9001';
  }

  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;
    
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json();
  }

  async register(userData) {
    return this.request('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  }

  async login(credentials) {
    return this.request('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
  }

  async getProfile(token) {
    return this.request('/api/v1/user/profile', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });
  }
}

export default new AuthHTTP2Client();
```

### Browser Fetch API (Modern Browsers)

```javascript
// Modern browsers automatically use HTTP/2 when available
const authService = {
  baseURL: 'https://localhost:9001', // Development
  
  async request(endpoint, options = {}) {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });
    
    return response.json();
  },

  async register(userData) {
    return this.request('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  },

  async login(credentials) {
    return this.request('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
  }
};
```

## Development Setup

### 1. Certificate Handling

**Option A: Accept Self-Signed Certificates (Development Only)**
```bash
# For development testing with curl
curl -k --http2 -X POST https://localhost:9001/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePass123!"}'
```

**Option B: Use Auth Service Certificates in Frontend**
```javascript
// Node.js: Use the same certificates
const fs = require('fs');
const http2 = require('http2');

const client = http2.connect('https://localhost:9001', {
  ca: fs.readFileSync('./backend/auth-service/certs/server.crt'),
});
```

### 2. Environment Configuration

```javascript
// .env.development
REACT_APP_AUTH_SERVICE_URL=https://localhost:9001
REACT_APP_AUTH_SERVICE_PROTOCOL=http2

// .env.production  
REACT_APP_AUTH_SERVICE_URL=https://auth-service.yourdomain.com:9001
REACT_APP_AUTH_SERVICE_PROTOCOL=http2
```

## API Gateway Integration

If using an API Gateway, it must also support HTTP/2:

```javascript
// API Gateway HTTP/2 upstream configuration
upstream auth_service {
    server localhost:9001;
}

server {
    listen 443 ssl http2;
    
    location /auth/ {
        proxy_pass https://auth_service/;
        proxy_http_version 2.0;
        proxy_ssl_verify off; # Only for development with self-signed certs
    }
}
```

## Testing Frontend Integration

### 1. Verify HTTP/2 Connection

```javascript
// Test HTTP/2 connection from frontend
async function testHTTP2Connection() {
  try {
    const response = await fetch('https://localhost:9001/health');
    console.log('Protocol:', response.headers.get('x-protocol') || 'HTTP/2');
    console.log('Status:', response.status);
    console.log('Health:', await response.json());
  } catch (error) {
    console.error('Connection failed:', error);
  }
}
```

### 2. Browser Developer Tools

- Open Network tab in browser dev tools
- Look for "h2" in the Protocol column
- Verify TLS version and cipher suite

## Production Considerations

### 1. Certificate Management

```bash
# Use proper certificates in production
# Example with Let's Encrypt
certbot certonly --standalone -d auth-service.yourdomain.com

# Update auth service configuration
export TLS_CERT_FILE=/etc/letsencrypt/live/auth-service.yourdomain.com/fullchain.pem
export TLS_KEY_FILE=/etc/letsencrypt/live/auth-service.yourdomain.com/privkey.pem
```

### 2. Load Balancer Configuration

```yaml
# Example: AWS ALB with HTTP/2
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: http2
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:...
```

## Troubleshooting

### Common Issues

1. **Certificate Errors**: Use `-k` flag with curl or `rejectUnauthorized: false` in development
2. **Protocol Mismatch**: Ensure client supports HTTP/2
3. **Port Issues**: Verify port 9001 is accessible
4. **CORS Issues**: Check CORS configuration in auth service

### Debug Commands

```bash
# Test HTTP/2 connection
curl -k --http2 -v https://localhost:9001/health

# Verify certificate
openssl s_client -connect localhost:9001 -servername localhost

# Check HTTP/2 support
curl -k --http2-prior-knowledge https://localhost:9001/health
```

## Summary

To integrate with the strict HTTP/2-only auth service:

1. **Use HTTP/2 clients** in your frontend applications
2. **Handle certificates** properly (self-signed in dev, proper certs in prod)
3. **Configure HTTPS** with the correct port (9001)
4. **Test thoroughly** to ensure HTTP/2 protocol is being used
5. **Update API Gateway** if using one to support HTTP/2 upstream
