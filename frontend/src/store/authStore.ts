import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { API_ENDPOINTS, apiCall } from '../config/api';

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  company?: string;
  email_verified: boolean;
  is_admin: boolean;
  created_at: string;
  last_login?: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
  session_id?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
  remember_me?: boolean;
}

export interface RegisterRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  company?: string;
}

interface AuthState {
  // Authentication state
  isAuthenticated: boolean;
  user: User | null;
  tokens: AuthTokens | null;
  
  // Loading states
  isLoading: boolean;
  isLoginLoading: boolean;
  isRegisterLoading: boolean;
  
  // Error states
  error: string | null;
  loginError: string | null;
  registerError: string | null;
  
  // Actions
  login: (credentials: LoginRequest) => Promise<boolean>;
  register: (userData: RegisterRequest) => Promise<boolean>;
  logout: () => void;
  refreshToken: () => Promise<boolean>;
  clearError: () => void;
  clearLoginError: () => void;
  clearRegisterError: () => void;
  
  // Token management
  getAccessToken: () => string | null;
  isTokenExpired: () => boolean;
  
  // User profile
  updateProfile: (updates: Partial<User>) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // Initial state
      isAuthenticated: false,
      user: null,
      tokens: null,
      isLoading: false,
      isLoginLoading: false,
      isRegisterLoading: false,
      error: null,
      loginError: null,
      registerError: null,

      // Login action
      login: async (credentials: LoginRequest) => {
        set({ isLoginLoading: true, loginError: null });

        try {
          // Add artificial delay to see animation (remove in production)
          await new Promise(resolve => setTimeout(resolve, 2000));

          const response = await apiCall(API_ENDPOINTS.AUTH.LOGIN, {
            method: 'POST',
            body: JSON.stringify(credentials),
          });

          const data = await response.json();

          if (!response.ok) {
            throw new Error(data.message || 'Login failed');
          }

          if (data.user && data.access_token && data.refresh_token) {
            // Transform backend response to frontend format
            const tokens = {
              access_token: data.access_token,
              refresh_token: data.refresh_token,
              expires_in: data.expires_in,
              token_type: 'Bearer', // Standard token type
              session_id: data.session_id
            };

            set({
              isAuthenticated: true,
              user: data.user,
              tokens: tokens,
              isLoginLoading: false,
              loginError: null,
            });
            return true;
          } else {
            throw new Error('Invalid response format');
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Login failed';
          set({
            isLoginLoading: false,
            loginError: errorMessage,
            isAuthenticated: false,
            user: null,
            tokens: null,
          });
          return false;
        }
      },

      // Register action
      register: async (userData: RegisterRequest) => {
        set({ isRegisterLoading: true, registerError: null });

        try {
          // Add artificial delay to see animation (remove in production)
          await new Promise(resolve => setTimeout(resolve, 2000));

          const response = await apiCall(API_ENDPOINTS.AUTH.REGISTER, {
            method: 'POST',
            body: JSON.stringify(userData),
          });

          const data = await response.json();

          if (!response.ok) {
            throw new Error(data.message || 'Registration failed');
          }

          if (data.user && data.access_token && data.refresh_token) {
            // Transform backend response to frontend format
            const tokens = {
              access_token: data.access_token,
              refresh_token: data.refresh_token,
              expires_in: data.expires_in,
              token_type: 'Bearer', // Standard token type
              session_id: data.session_id
            };

            set({
              isAuthenticated: true,
              user: data.user,
              tokens: tokens,
              isRegisterLoading: false,
              registerError: null,
            });
            return true;
          } else {
            throw new Error('Invalid response format');
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Registration failed';
          set({
            isRegisterLoading: false,
            registerError: errorMessage,
            isAuthenticated: false,
            user: null,
            tokens: null,
          });
          return false;
        }
      },

      // Logout action
      logout: async () => {
        const { tokens } = get();
        
        // Call logout endpoint if we have tokens
        if (tokens?.access_token) {
          try {
            await apiCall(API_ENDPOINTS.AUTH.LOGOUT, {
              method: 'POST',
              headers: {
                'Authorization': `Bearer ${tokens.access_token}`,
              },
            });
          } catch (error) {
            console.warn('Logout request failed:', error);
          }
        }

        set({
          isAuthenticated: false,
          user: null,
          tokens: null,
          error: null,
          loginError: null,
          registerError: null,
        });
      },

      // Refresh token action
      refreshToken: async () => {
        const { tokens } = get();
        
        if (!tokens?.refresh_token) {
          return false;
        }

        try {
          const response = await apiCall(API_ENDPOINTS.AUTH.REFRESH, {
            method: 'POST',
            body: JSON.stringify({
              refresh_token: tokens.refresh_token,
            }),
          });

          const data = await response.json();

          if (!response.ok) {
            throw new Error(data.message || 'Token refresh failed');
          }

          if (data.success && data.tokens) {
            set({
              tokens: data.tokens,
            });
            return true;
          } else {
            throw new Error('Invalid refresh response');
          }
        } catch (error) {
          console.error('Token refresh failed:', error);
          // If refresh fails, logout user
          get().logout();
          return false;
        }
      },

      // Error management
      clearError: () => set({ error: null }),
      clearLoginError: () => set({ loginError: null }),
      clearRegisterError: () => set({ registerError: null }),

      // Token utilities
      getAccessToken: () => {
        const { tokens } = get();
        return tokens?.access_token || null;
      },

      isTokenExpired: () => {
        const { tokens } = get();
        if (!tokens) return true;
        
        // Check if token expires within next 5 minutes
        const expirationTime = Date.now() + (tokens.expires_in * 1000);
        const bufferTime = 5 * 60 * 1000; // 5 minutes
        
        return Date.now() >= (expirationTime - bufferTime);
      },

      // Profile management
      updateProfile: (updates: Partial<User>) => {
        set((state) => ({
          user: state.user ? { ...state.user, ...updates } : null,
        }));
      },
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        isAuthenticated: state.isAuthenticated,
        user: state.user,
        tokens: state.tokens,
      }),
    }
  )
);
