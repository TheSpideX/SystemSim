// API Configuration that automatically detects the correct backend URL

const getBackendHost = (): string => {
  // In development, use the same host as the frontend
  const frontendHost = window.location.hostname;

  // If accessing via localhost, use localhost for backend
  if (frontendHost === 'localhost' || frontendHost === '127.0.0.1') {
    return 'localhost';
  }

  // If accessing via mDNS hostname, use it directly
  if (frontendHost === 'siked.local') {
    return 'siked.local';
  }

  // For any other access (IP, etc.), use the mDNS hostname
  // This ensures consistent access regardless of IP changes
  return 'siked.local';
};

const getApiBaseUrl = (): string => {
  const backendHost = getBackendHost();
  return `https://${backendHost}:8000`;
};

export const API_BASE_URL = getApiBaseUrl();

// API endpoints
export const API_ENDPOINTS = {
  AUTH: {
    LOGIN: `${API_BASE_URL}/api/auth/login`,
    REGISTER: `${API_BASE_URL}/api/auth/register`,
    LOGOUT: `${API_BASE_URL}/api/auth/logout`,
    REFRESH: `${API_BASE_URL}/api/auth/refresh`,
    HEALTH: `${API_BASE_URL}/api/auth/health`,
  }
};

// Common fetch options for HTTP/2 and CORS
export const DEFAULT_FETCH_OPTIONS: RequestInit = {
  mode: 'cors',
  credentials: 'omit',
  headers: {
    'Content-Type': 'application/json',
  }
};

// Helper function for API calls with timeout
export const apiCall = async (url: string, options: RequestInit = {}, timeoutMs: number = 10000): Promise<Response> => {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);
  
  try {
    const response = await fetch(url, {
      ...DEFAULT_FETCH_OPTIONS,
      ...options,
      signal: controller.signal,
    });
    
    clearTimeout(timeoutId);
    return response;
  } catch (error) {
    clearTimeout(timeoutId);
    throw error;
  }
};

// Debug function to show current configuration
export const getApiConfig = () => {
  return {
    frontendHost: window.location.hostname,
    backendHost: getBackendHost(),
    apiBaseUrl: API_BASE_URL,
    endpoints: API_ENDPOINTS
  };
};
