import { create } from 'zustand';
import { useAuthStore } from './authStore';

export interface WebSocketMessage {
  type: string;
  channel: string;
  data: any;
  timestamp: string;
}

export interface NotificationMessage {
  title: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error';
}

export interface ProjectCollaborationMessage {
  component_id?: string;
  changes?: any;
  user: {
    id: string;
    name: string;
  };
}

export interface SimulationMessage {
  current_rps?: number;
  avg_latency_ms?: number;
  error_rate?: number;
  progress?: number;
  estimated_completion?: string;
}

interface WebSocketState {
  // Connection state
  isConnected: boolean;
  isConnecting: boolean;
  connectionError: string | null;
  reconnectAttempts: number;
  maxReconnectAttempts: number;
  
  // WebSocket instances
  genericWs: WebSocket | null;
  projectWs: WebSocket | null;
  simulationWs: WebSocket | null;
  
  // Message handling
  messages: WebSocketMessage[];
  notifications: NotificationMessage[];
  
  // Subscriptions
  subscribedChannels: Set<string>;
  
  // Actions
  connect: () => void;
  disconnect: () => void;
  connectToProject: (projectId: string) => void;
  connectToSimulation: (simulationId: string) => void;
  disconnectFromProject: () => void;
  disconnectFromSimulation: () => void;
  sendMessage: (type: string, data: any, channel?: string) => void;
  clearMessages: () => void;
  clearNotifications: () => void;
  addNotification: (notification: NotificationMessage) => void;
  removeNotification: (index: number) => void;
}

export const useWebSocketStore = create<WebSocketState>((set, get) => ({
  // Initial state
  isConnected: false,
  isConnecting: false,
  connectionError: null,
  reconnectAttempts: 0,
  maxReconnectAttempts: 5,
  
  genericWs: null,
  projectWs: null,
  simulationWs: null,
  
  messages: [],
  notifications: [],
  subscribedChannels: new Set(),

  // Connect to generic WebSocket
  connect: () => {
    const { isConnected, isConnecting, reconnectAttempts, maxReconnectAttempts } = get();
    
    if (isConnected || isConnecting) {
      return;
    }

    if (reconnectAttempts >= maxReconnectAttempts) {
      set({ connectionError: 'Max reconnection attempts reached' });
      return;
    }

    const token = useAuthStore.getState().getAccessToken();
    if (!token) {
      set({ connectionError: 'No authentication token available' });
      return;
    }

    set({ isConnecting: true, connectionError: null });

    try {
      const ws = new WebSocket(`wss://localhost:8000/ws?token=${token}`);
      
      ws.onopen = () => {
        console.log('Generic WebSocket connected');
        set({
          isConnected: true,
          isConnecting: false,
          genericWs: ws,
          reconnectAttempts: 0,
          connectionError: null,
        });
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          
          // Add to messages
          set((state) => ({
            messages: [...state.messages.slice(-99), message], // Keep last 100 messages
          }));

          // Handle specific message types
          if (message.type === 'notification') {
            const notificationData = message.data as NotificationMessage;
            get().addNotification(notificationData);
          }

          // Handle system announcements
          if (message.type === 'system_announcement') {
            get().addNotification({
              title: 'System Announcement',
              message: message.data.message || 'System update',
              type: 'info',
            });
          }

        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      ws.onclose = (event) => {
        console.log('Generic WebSocket disconnected:', event.code, event.reason);
        set({
          isConnected: false,
          isConnecting: false,
          genericWs: null,
        });

        // Auto-reconnect if not intentionally closed
        if (event.code !== 1000 && get().reconnectAttempts < maxReconnectAttempts) {
          setTimeout(() => {
            set((state) => ({ reconnectAttempts: state.reconnectAttempts + 1 }));
            get().connect();
          }, Math.pow(2, get().reconnectAttempts) * 1000); // Exponential backoff
        }
      };

      ws.onerror = (error) => {
        console.error('Generic WebSocket error:', error);
        set({
          connectionError: 'WebSocket connection failed',
          isConnecting: false,
        });
      };

    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      set({
        isConnecting: false,
        connectionError: 'Failed to create WebSocket connection',
      });
    }
  },

  // Disconnect generic WebSocket
  disconnect: () => {
    const { genericWs } = get();
    
    if (genericWs) {
      genericWs.close(1000, 'User disconnected');
    }
    
    set({
      isConnected: false,
      isConnecting: false,
      genericWs: null,
      reconnectAttempts: 0,
    });
  },

  // Connect to project-specific WebSocket
  connectToProject: (projectId: string) => {
    const token = useAuthStore.getState().getAccessToken();
    if (!token) {
      console.error('No authentication token for project WebSocket');
      return;
    }

    try {
      const ws = new WebSocket(`wss://localhost:8000/ws/project/${projectId}?token=${token}`);
      
      ws.onopen = () => {
        console.log(`Project WebSocket connected for project: ${projectId}`);
        set({ projectWs: ws });
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          
          // Add to messages
          set((state) => ({
            messages: [...state.messages.slice(-99), message],
          }));

          // Handle project collaboration messages
          if (message.type === 'component_updated' || message.type === 'component_added' || message.type === 'component_deleted') {
            const collaborationData = message.data as ProjectCollaborationMessage;
            
            // Show notification for other users' changes
            if (collaborationData.user.id !== useAuthStore.getState().user?.id) {
              get().addNotification({
                title: 'Project Update',
                message: `${collaborationData.user.name} ${message.type.replace('component_', '')} a component`,
                type: 'info',
              });
            }
          }

        } catch (error) {
          console.error('Failed to parse project WebSocket message:', error);
        }
      };

      ws.onclose = () => {
        console.log('Project WebSocket disconnected');
        set({ projectWs: null });
      };

      ws.onerror = (error) => {
        console.error('Project WebSocket error:', error);
      };

    } catch (error) {
      console.error('Failed to create project WebSocket connection:', error);
    }
  },

  // Connect to simulation-specific WebSocket
  connectToSimulation: (simulationId: string) => {
    const token = useAuthStore.getState().getAccessToken();
    if (!token) {
      console.error('No authentication token for simulation WebSocket');
      return;
    }

    try {
      const ws = new WebSocket(`wss://localhost:8000/ws/simulation/${simulationId}?token=${token}`);
      
      ws.onopen = () => {
        console.log(`Simulation WebSocket connected for simulation: ${simulationId}`);
        set({ simulationWs: ws });
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          
          // Add to messages
          set((state) => ({
            messages: [...state.messages.slice(-99), message],
          }));

          // Handle simulation updates
          if (message.type === 'simulation_metrics' || message.type === 'simulation_progress') {
            // Real-time simulation data - can be handled by simulation components
            console.log('Simulation update:', message.data);
          }

          if (message.type === 'simulation_completed') {
            get().addNotification({
              title: 'Simulation Complete',
              message: 'Your simulation has finished running',
              type: 'success',
            });
          }

          if (message.type === 'simulation_error') {
            get().addNotification({
              title: 'Simulation Error',
              message: message.data.error || 'Simulation encountered an error',
              type: 'error',
            });
          }

        } catch (error) {
          console.error('Failed to parse simulation WebSocket message:', error);
        }
      };

      ws.onclose = () => {
        console.log('Simulation WebSocket disconnected');
        set({ simulationWs: null });
      };

      ws.onerror = (error) => {
        console.error('Simulation WebSocket error:', error);
      };

    } catch (error) {
      console.error('Failed to create simulation WebSocket connection:', error);
    }
  },

  // Disconnect from project WebSocket
  disconnectFromProject: () => {
    const { projectWs } = get();
    if (projectWs) {
      projectWs.close(1000, 'User left project');
      set({ projectWs: null });
    }
  },

  // Disconnect from simulation WebSocket
  disconnectFromSimulation: () => {
    const { simulationWs } = get();
    if (simulationWs) {
      simulationWs.close(1000, 'User left simulation');
      set({ simulationWs: null });
    }
  },

  // Send message through appropriate WebSocket
  sendMessage: (type: string, data: any, channel?: string) => {
    const { genericWs, projectWs, simulationWs } = get();
    
    const message = {
      type,
      data,
      channel: channel || 'general',
      timestamp: new Date().toISOString(),
    };

    const messageStr = JSON.stringify(message);

    // Determine which WebSocket to use based on channel
    if (channel?.startsWith('project:') && projectWs) {
      projectWs.send(messageStr);
    } else if (channel?.startsWith('simulation:') && simulationWs) {
      simulationWs.send(messageStr);
    } else if (genericWs) {
      genericWs.send(messageStr);
    } else {
      console.warn('No WebSocket connection available for message:', message);
    }
  },

  // Utility actions
  clearMessages: () => set({ messages: [] }),
  clearNotifications: () => set({ notifications: [] }),
  
  addNotification: (notification: NotificationMessage) => {
    set((state) => ({
      notifications: [...state.notifications, notification].slice(-10), // Keep last 10 notifications
    }));
  },
  
  removeNotification: (index: number) => {
    set((state) => ({
      notifications: state.notifications.filter((_, i) => i !== index),
    }));
  },
}));
