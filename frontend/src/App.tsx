import React, { useEffect, useState } from 'react';
import { EnhancedSimulationPage } from './components/EnhancedSimulationPage';
import { ThemeProvider } from './components/ui/ThemeProvider';
import { ToastProvider } from './components/ui/Toast';
import { LoginPage } from './components/auth/LoginPage';
import { SignupPage } from './components/auth/SignupPage';
import { WebSocketNotifications, WebSocketManager, ActivityFeed } from './components/ui/WebSocketNotifications';
import { useAuthStore } from './store/authStore';
import { useWebSocketStore } from './store/websocketStore';

type AppPage = 'login' | 'signup' | 'simulation';

function App() {
  const { isAuthenticated, user, logout, refreshToken, isTokenExpired } = useAuthStore();
  const { connect, disconnect } = useWebSocketStore();
  const [currentPage, setCurrentPage] = useState<AppPage>('login');

  // Check authentication status on app load
  useEffect(() => {
    if (isAuthenticated && user) {
      setCurrentPage('simulation');
      // Connect to WebSocket when authenticated
      connect();
    } else {
      setCurrentPage('login');
      // Disconnect WebSocket when not authenticated
      disconnect();
    }
  }, [isAuthenticated, user, connect, disconnect]);

  // Auto-refresh token when it's about to expire
  useEffect(() => {
    if (isAuthenticated && isTokenExpired()) {
      refreshToken();
    }
  }, [isAuthenticated, isTokenExpired, refreshToken]);

  // Handle successful authentication
  const handleAuthSuccess = () => {
    setCurrentPage('simulation');
  };

  // Handle logout
  const handleLogout = () => {
    logout();
    setCurrentPage('login');
  };

  // Handle demo mode (no authentication required)
  const handleTryDemo = () => {
    setCurrentPage('simulation');
  };

  // Navigation handlers
  const goToLogin = () => setCurrentPage('login');
  const goToSignup = () => setCurrentPage('signup');

  // Render current page
  const renderCurrentPage = () => {
    switch (currentPage) {
      case 'signup':
        return (
          <SignupPage
            onSuccess={handleAuthSuccess}
            onSwitchToLogin={goToLogin}
          />
        );

      case 'simulation':
        return (
          <>
            <EnhancedSimulationPage
              onBack={isAuthenticated ? handleLogout : goToLogin}
            />
            {/* WebSocket components for real-time features */}
            {isAuthenticated && (
              <>
                <WebSocketManager />
                <WebSocketNotifications />
                <ActivityFeed />
              </>
            )}
          </>
        );

      case 'login':
      default:
        return (
          <LoginPage
            onSuccess={handleAuthSuccess}
            onSwitchToSignup={goToSignup}
            onTryDemo={handleTryDemo}
          />
        );
    }
  };

  return (
    <ThemeProvider>
      <ToastProvider />
      {renderCurrentPage()}
    </ThemeProvider>
  );
}

export default App;
