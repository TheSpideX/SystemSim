import React, { useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Bell, X, Info, CheckCircle, AlertTriangle, AlertCircle, 
  Users, Activity, Zap 
} from 'lucide-react';
import { useWebSocketStore } from '../../store/websocketStore';
import type { NotificationMessage } from '../../store/websocketStore';

export const WebSocketNotifications: React.FC = () => {
  const { 
    notifications, 
    removeNotification, 
    isConnected,
    connectionError 
  } = useWebSocketStore();

  // Auto-remove notifications after 5 seconds
  useEffect(() => {
    if (notifications.length > 0) {
      const timer = setTimeout(() => {
        removeNotification(0);
      }, 5000);
      
      return () => clearTimeout(timer);
    }
  }, [notifications, removeNotification]);

  const getNotificationIcon = (type: NotificationMessage['type']) => {
    switch (type) {
      case 'success':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'warning':
        return <AlertTriangle className="w-5 h-5 text-yellow-400" />;
      case 'error':
        return <AlertCircle className="w-5 h-5 text-red-400" />;
      default:
        return <Info className="w-5 h-5 text-blue-400" />;
    }
  };

  const getNotificationColors = (type: NotificationMessage['type']) => {
    switch (type) {
      case 'success':
        return 'bg-green-500/10 border-green-500/20 text-green-400';
      case 'warning':
        return 'bg-yellow-500/10 border-yellow-500/20 text-yellow-400';
      case 'error':
        return 'bg-red-500/10 border-red-500/20 text-red-400';
      default:
        return 'bg-blue-500/10 border-blue-500/20 text-blue-400';
    }
  };

  return (
    <>
      {/* Connection Status Indicator */}
      <div className="fixed top-4 left-4 z-50">
        <motion.div
          initial={{ opacity: 0, scale: 0.8 }}
          animate={{ opacity: 1, scale: 1 }}
          className={`flex items-center px-3 py-2 rounded-lg backdrop-blur-sm border text-sm font-medium ${
            isConnected 
              ? 'bg-green-500/10 border-green-500/20 text-green-400'
              : 'bg-red-500/10 border-red-500/20 text-red-400'
          }`}
        >
          <div className={`w-2 h-2 rounded-full mr-2 ${
            isConnected ? 'bg-green-400' : 'bg-red-400'
          } ${isConnected ? 'animate-pulse' : ''}`} />
          {isConnected ? 'Connected' : 'Disconnected'}
          {connectionError && (
            <span className="ml-2 text-xs opacity-75">({connectionError})</span>
          )}
        </motion.div>
      </div>

      {/* Notifications Container */}
      <div className="fixed top-4 right-4 z-50 space-y-2 max-w-sm">
        <AnimatePresence>
          {notifications.map((notification, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, x: 300, scale: 0.8 }}
              animate={{ opacity: 1, x: 0, scale: 1 }}
              exit={{ opacity: 0, x: 300, scale: 0.8 }}
              transition={{ 
                type: "spring", 
                stiffness: 300, 
                damping: 30,
                duration: 0.3 
              }}
              className={`p-4 rounded-lg backdrop-blur-sm border shadow-lg ${getNotificationColors(notification.type)}`}
            >
              <div className="flex items-start">
                <div className="flex-shrink-0 mr-3">
                  {getNotificationIcon(notification.type)}
                </div>
                
                <div className="flex-1 min-w-0">
                  <h4 className="text-sm font-semibold mb-1">
                    {notification.title}
                  </h4>
                  <p className="text-sm opacity-90">
                    {notification.message}
                  </p>
                </div>
                
                <button
                  onClick={() => removeNotification(index)}
                  className="flex-shrink-0 ml-2 p-1 rounded-md hover:bg-white/10 transition-colors duration-200"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
              
              {/* Progress bar for auto-dismiss */}
              <motion.div
                initial={{ width: "100%" }}
                animate={{ width: "0%" }}
                transition={{ duration: 5, ease: "linear" }}
                className="absolute bottom-0 left-0 h-0.5 bg-current opacity-30 rounded-b-lg"
              />
            </motion.div>
          ))}
        </AnimatePresence>
      </div>
    </>
  );
};

// WebSocket Connection Manager Component
export const WebSocketManager: React.FC = () => {
  const { connect, disconnect, isConnected } = useWebSocketStore();

  useEffect(() => {
    // Auto-connect when component mounts
    if (!isConnected) {
      connect();
    }

    // Cleanup on unmount
    return () => {
      disconnect();
    };
  }, [connect, disconnect, isConnected]);

  return null; // This component doesn't render anything
};

// Real-time Activity Feed Component
export const ActivityFeed: React.FC = () => {
  const { messages } = useWebSocketStore();
  
  // Get last 5 messages for activity feed
  const recentMessages = messages.slice(-5).reverse();

  if (recentMessages.length === 0) {
    return null;
  }

  const getActivityIcon = (type: string) => {
    switch (type) {
      case 'user_activity':
        return <Users className="w-4 h-4 text-blue-400" />;
      case 'project_update':
      case 'component_updated':
      case 'component_added':
      case 'component_deleted':
        return <Activity className="w-4 h-4 text-green-400" />;
      case 'simulation_metrics':
      case 'simulation_progress':
        return <Zap className="w-4 h-4 text-yellow-400" />;
      default:
        return <Bell className="w-4 h-4 text-gray-400" />;
    }
  };

  const formatActivityMessage = (message: any) => {
    switch (message.type) {
      case 'component_updated':
        return `${message.data.user?.name || 'Someone'} updated a component`;
      case 'component_added':
        return `${message.data.user?.name || 'Someone'} added a component`;
      case 'component_deleted':
        return `${message.data.user?.name || 'Someone'} deleted a component`;
      case 'simulation_metrics':
        return `Simulation metrics updated`;
      case 'user_joined':
        return `${message.data.user?.name || 'Someone'} joined the project`;
      case 'user_left':
        return `${message.data.user?.name || 'Someone'} left the project`;
      default:
        return message.type.replace(/_/g, ' ');
    }
  };

  return (
    <div className="fixed bottom-4 left-4 z-40 max-w-xs">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="bg-app-secondary/80 backdrop-blur-sm border border-app-primary/20 rounded-lg p-4 shadow-lg"
      >
        <div className="flex items-center mb-3">
          <Activity className="w-4 h-4 text-app-tertiary mr-2" />
          <h3 className="text-sm font-semibold text-app-primary">Recent Activity</h3>
        </div>
        
        <div className="space-y-2">
          <AnimatePresence>
            {recentMessages.map((message, index) => (
              <motion.div
                key={`${message.timestamp}-${index}`}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                transition={{ delay: index * 0.1 }}
                className="flex items-center text-xs text-app-tertiary"
              >
                <div className="flex-shrink-0 mr-2">
                  {getActivityIcon(message.type)}
                </div>
                <span className="flex-1 truncate">
                  {formatActivityMessage(message)}
                </span>
                <span className="flex-shrink-0 ml-2 opacity-60">
                  {new Date(message.timestamp).toLocaleTimeString([], { 
                    hour: '2-digit', 
                    minute: '2-digit' 
                  })}
                </span>
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      </motion.div>
    </div>
  );
};
