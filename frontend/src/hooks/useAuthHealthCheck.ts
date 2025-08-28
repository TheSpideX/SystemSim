import { useState, useEffect, useRef } from 'react';

interface HealthStatus {
  status: 'healthy' | 'unhealthy' | 'checking' | 'disconnected';
  lastChecked: Date | null;
  error?: string;
}

// Hook for auth service health with WebSocket real-time updates (WebSocket ONLY - Port 8002)
export const useAuthHealthCheck = () => {
  const [healthStatus, setHealthStatus] = useState<HealthStatus>({
    status: 'checking',
    lastChecked: null
  });

  const isActiveRef = useRef(true);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);

  // WebSocket connection for real-time health updates (Port 8002)
  const connectWebSocket = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    if (reconnectTimeoutRef.current) {
      window.clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (!isActiveRef.current) return;

    const backendHost = window.location.hostname;
    const wsUrl = `wss://${backendHost}:8002/ws/health`;

    console.log('🔌 WebSocket connecting to:', wsUrl);

    try {
      wsRef.current = new WebSocket(wsUrl);

      wsRef.current.onopen = () => {
        console.log('✅ WebSocket connected for health updates');
        // Reset to checking state when connection is established
        setHealthStatus(prev => ({
          ...prev,
          status: prev.status === 'disconnected' ? 'checking' : prev.status,
          error: undefined
        }));

        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          wsRef.current.send(JSON.stringify({
            type: 'subscribe',
            channel: 'health:auth'
          }));
        }
      };

      wsRef.current.onmessage = (event) => {
        if (!isActiveRef.current) return;

        try {
          const data = JSON.parse(event.data);
          console.log('📨 Health update received:', data);

          if (data.type === 'health_update' && data.data && data.data.service === 'auth') {
            console.log('🏥 Auth health status:', data.data.status);
            setHealthStatus({
              status: data.data.status === 'healthy' ? 'healthy' : 'unhealthy',
              lastChecked: new Date(),
              error: undefined
            });
          }
        } catch (error) {
          console.warn('Failed to parse WebSocket message:', error);
        }
      };

      wsRef.current.onclose = (event) => {
        console.log('❌ WebSocket disconnected:', event.code, event.reason);
        wsRef.current = null;

        if (isActiveRef.current) {
          setHealthStatus(prev => ({
            ...prev,
            status: 'disconnected',
            error: 'Connection lost'
          }));

          // Auto-reconnect after 3 seconds
          reconnectTimeoutRef.current = window.setTimeout(() => {
            if (isActiveRef.current) {
              console.log('🔄 Reconnecting WebSocket...');
              connectWebSocket();
            }
          }, 3000);
        }
      };

      wsRef.current.onerror = (error) => {
        console.warn('⚠️ WebSocket error:', error);
        if (wsRef.current) {
          wsRef.current.close();
          wsRef.current = null;
        }

        if (isActiveRef.current) {
          setHealthStatus(prev => ({
            ...prev,
            status: 'disconnected',
            error: 'Connection failed'
          }));
        }
      };

    } catch (error) {
      console.error('❌ Failed to create WebSocket:', error);
      setHealthStatus(prev => ({
        ...prev,
        status: 'disconnected',
        error: 'Failed to connect'
      }));
    }
  };

  useEffect(() => {
    isActiveRef.current = true;
    connectWebSocket();

    return () => {
      isActiveRef.current = false;

      if (reconnectTimeoutRef.current) {
        window.clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }

      if (wsRef.current) {
        if (wsRef.current.readyState === WebSocket.OPEN) {
          wsRef.current.send(JSON.stringify({
            type: 'unsubscribe',
            channel: 'health:auth'
          }));
        }
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, []);

  // Cleanup function to unsubscribe and close connection
  const cleanup = () => {
    isActiveRef.current = false;

    if (reconnectTimeoutRef.current) {
      window.clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      try {
        wsRef.current.send(JSON.stringify({
          type: 'unsubscribe',
          channel: 'health:auth'
        }));
        console.log('🔌 Unsubscribed from auth health updates');
      } catch (error) {
        console.warn('Failed to send unsubscribe message:', error);
      }
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  };

  return {
    healthStatus: healthStatus.status,
    lastChecked: healthStatus.lastChecked,
    error: healthStatus.error,
    isConnected: wsRef.current?.readyState === WebSocket.OPEN,
    cleanup // Expose cleanup function
  };
};