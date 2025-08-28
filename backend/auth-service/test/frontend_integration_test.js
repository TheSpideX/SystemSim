#!/usr/bin/env node

/**
 * Frontend Integration Test for HTTP/2-Only Auth Service
 * 
 * This script demonstrates how frontend applications should connect
 * to the strict HTTP/2-only auth service.
 */

const http2 = require('http2');
const https = require('https');

// Configuration
const AUTH_SERVICE_URL = 'https://localhost:9001';
const TEST_USER = {
  email: `test_${Date.now()}@example.com`,
  password: 'Tr0ub4dor&3',
  firstName: 'Test',
  lastName: 'User',
  company: 'Test Corp'
};

// Create HTTP/2 client (accepts self-signed certificates for development)
const client = http2.connect(AUTH_SERVICE_URL, {
  rejectUnauthorized: false, // Accept self-signed certs in development
});

// Helper function to make HTTP/2 requests
function makeHTTP2Request(method, path, data = null, headers = {}) {
  return new Promise((resolve, reject) => {
    const requestHeaders = {
      ':method': method,
      ':path': path,
      'content-type': 'application/json',
      ...headers
    };

    const req = client.request(requestHeaders);
    let responseData = '';
    let responseHeaders = {};

    req.on('response', (headers) => {
      responseHeaders = headers;
    });

    req.on('data', (chunk) => {
      responseData += chunk;
    });

    req.on('end', () => {
      try {
        const parsedData = responseData ? JSON.parse(responseData) : {};
        resolve({
          status: responseHeaders[':status'],
          headers: responseHeaders,
          data: parsedData
        });
      } catch (error) {
        resolve({
          status: responseHeaders[':status'],
          headers: responseHeaders,
          data: responseData
        });
      }
    });

    req.on('error', reject);

    if (data) {
      req.write(JSON.stringify(data));
    }
    req.end();
  });
}

// Test functions
async function testHealthCheck() {
  console.log('🔍 Testing health check...');
  try {
    const response = await makeHTTP2Request('GET', '/health');
    console.log(`✅ Health check: ${response.status}`);
    console.log(`📊 Response:`, response.data);
    return response.status === 200;
  } catch (error) {
    console.error('❌ Health check failed:', error.message);
    return false;
  }
}

async function testUserRegistration() {
  console.log('🔍 Testing user registration...');
  try {
    const response = await makeHTTP2Request('POST', '/api/v1/auth/register', TEST_USER);
    console.log(`✅ Registration: ${response.status}`);
    
    if (response.status === 201) {
      console.log(`👤 User created:`, response.data.user);
      return response.data;
    } else {
      console.log(`⚠️  Registration response:`, response.data);
      return null;
    }
  } catch (error) {
    console.error('❌ Registration failed:', error.message);
    return null;
  }
}

async function testUserLogin() {
  console.log('🔍 Testing user login...');
  try {
    const loginData = {
      email: TEST_USER.email,
      password: TEST_USER.password
    };
    
    const response = await makeHTTP2Request('POST', '/api/v1/auth/login', loginData);
    console.log(`✅ Login: ${response.status}`);
    
    if (response.status === 200) {
      console.log(`🔑 Login response:`, response.data);
      console.log(`🔍 Available keys:`, Object.keys(response.data));
      return response.data.access_token || response.data.accessToken || response.data.token;
    } else {
      console.log(`⚠️  Login response:`, response.data);
      return null;
    }
  } catch (error) {
    console.error('❌ Login failed:', error.message);
    return null;
  }
}

async function testProtectedEndpoint(token) {
  console.log('🔍 Testing protected endpoint...');
  try {
    const response = await makeHTTP2Request('GET', '/api/v1/user/profile', null, {
      'authorization': `Bearer ${token}`
    });
    
    console.log(`✅ Profile access: ${response.status}`);
    
    if (response.status === 200) {
      console.log(`👤 Profile data:`, response.data);
      return true;
    } else {
      console.log(`⚠️  Profile response:`, response.data);
      return false;
    }
  } catch (error) {
    console.error('❌ Profile access failed:', error.message);
    return false;
  }
}

async function testHTTP2Protocol() {
  console.log('🔍 Testing HTTP/2 protocol enforcement...');
  
  // Test with HTTP/1.1 client (should fail)
  return new Promise((resolve) => {
    const options = {
      hostname: 'localhost',
      port: 9001,
      path: '/health',
      method: 'GET',
      rejectUnauthorized: false,
      // Force HTTP/1.1
      agent: new https.Agent({
        maxVersion: 'TLSv1.3',
        minVersion: 'TLSv1.2',
      })
    };

    const req = https.request(options, (res) => {
      console.log(`⚠️  HTTP/1.1 request unexpectedly succeeded: ${res.statusCode}`);
      resolve(false);
    });

    req.on('error', (error) => {
      if (error.message.includes('ECONNRESET') || 
          error.message.includes('socket hang up') ||
          error.message.includes('certificate')) {
        console.log('✅ HTTP/1.1 request correctly rejected');
        resolve(true);
      } else {
        console.log(`❌ Unexpected error: ${error.message}`);
        resolve(false);
      }
    });

    req.setTimeout(5000, () => {
      req.destroy();
      console.log('✅ HTTP/1.1 request timed out (expected behavior)');
      resolve(true);
    });

    req.end();
  });
}

// Main test runner
async function runFrontendIntegrationTests() {
  console.log('🚀 Starting Frontend HTTP/2 Integration Tests\n');
  
  const results = {
    healthCheck: false,
    registration: false,
    login: false,
    protectedAccess: false,
    http2Enforcement: false
  };

  // Test 1: Health Check
  results.healthCheck = await testHealthCheck();
  console.log('');

  if (!results.healthCheck) {
    console.log('❌ Service not available. Make sure auth service is running on port 9001 with HTTP/2.');
    process.exit(1);
  }

  // Test 2: HTTP/2 Protocol Enforcement
  results.http2Enforcement = await testHTTP2Protocol();
  console.log('');

  // Test 3: User Registration
  const registrationResult = await testUserRegistration();
  results.registration = registrationResult !== null;
  console.log('');

  // Test 4: User Login
  let accessToken = null;
  if (results.registration) {
    accessToken = await testUserLogin();
    results.login = accessToken !== null;
    console.log('');
  }

  // Test 5: Protected Endpoint Access
  if (results.login && accessToken) {
    results.protectedAccess = await testProtectedEndpoint(accessToken);
    console.log('');
  }

  // Summary
  console.log('📊 Test Results Summary:');
  console.log('========================');
  console.log(`Health Check: ${results.healthCheck ? '✅ PASS' : '❌ FAIL'}`);
  console.log(`HTTP/2 Enforcement: ${results.http2Enforcement ? '✅ PASS' : '❌ FAIL'}`);
  console.log(`User Registration: ${results.registration ? '✅ PASS' : '❌ FAIL'}`);
  console.log(`User Login: ${results.login ? '✅ PASS' : '❌ FAIL'}`);
  console.log(`Protected Access: ${results.protectedAccess ? '✅ PASS' : '❌ FAIL'}`);

  const allPassed = Object.values(results).every(result => result);
  console.log(`\n🎯 Overall Result: ${allPassed ? '✅ ALL TESTS PASSED' : '❌ SOME TESTS FAILED'}`);

  // Close HTTP/2 client
  client.close();

  process.exit(allPassed ? 0 : 1);
}

// Handle cleanup
process.on('SIGINT', () => {
  console.log('\n🛑 Tests interrupted');
  client.close();
  process.exit(1);
});

process.on('uncaughtException', (error) => {
  console.error('💥 Uncaught exception:', error.message);
  client.close();
  process.exit(1);
});

// Run tests
runFrontendIntegrationTests().catch((error) => {
  console.error('💥 Test runner failed:', error.message);
  client.close();
  process.exit(1);
});
