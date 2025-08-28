import React, { useState, useRef, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Move, CornerDownRight } from 'lucide-react';
import { getComponentIcon } from '../ui/CustomIcons';
import type { SystemNodeData } from './ImprovedSystemNode';
import { useGlobalMouse } from '../../hooks/useGlobalMouse';

interface NodeDetailsPanelProps {
  nodeData: SystemNodeData | null;
  onClose: () => void;
}

const getStatusIndicator = (status: string) => {
  switch (status) {
    case 'healthy': return 'bg-green-500';
    case 'warning': return 'bg-yellow-500 animate-pulse';
    case 'critical': return 'bg-red-500 animate-pulse';
    case 'offline': return 'bg-gray-500';
    default: return 'bg-gray-500';
  }
};

export const NodeDetailsPanel: React.FC<NodeDetailsPanelProps> = ({ nodeData, onClose }) => {
  const [position, setPosition] = useState({ x: Math.max(0, window.innerWidth - 320), y: 80 });
  const [size, setSize] = useState({ width: 300, height: 400 });
  const [isDragging, setIsDragging] = useState(false);
  const [isResizing, setIsResizing] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [resizeStart, setResizeStart] = useState({ x: 0, y: 0, width: 0, height: 0 });
  const panelRef = useRef<HTMLDivElement>(null);

  // Use global mouse tracking for drag and resize - preserves exact behavior
  useGlobalMouse((mousePosition) => {
    if (isDragging) {
      setPosition({
        x: Math.max(0, Math.min(window.innerWidth - size.width, mousePosition.x - dragStart.x)),
        y: Math.max(0, Math.min(window.innerHeight - size.height, mousePosition.y - dragStart.y)),
      });
    } else if (isResizing) {
      const deltaX = mousePosition.x - resizeStart.x;
      const deltaY = mousePosition.y - resizeStart.y;
      setSize({
        width: Math.max(250, Math.min(500, resizeStart.width + deltaX)),
        height: Math.max(300, Math.min(600, resizeStart.height + deltaY)),
      });
    }
  }, [isDragging, isResizing, dragStart, resizeStart, size]);

  // Handle drag start
  const handleDragStart = useCallback((e: React.MouseEvent) => {
    if (e.target === e.currentTarget || (e.target as HTMLElement).closest('.drag-handle')) {
      setIsDragging(true);
      setDragStart({
        x: e.clientX - position.x,
        y: e.clientY - position.y
      });
      e.preventDefault();
    }
  }, [position]);

  // Handle resize start
  const handleResizeStart = useCallback((e: React.MouseEvent) => {
    setIsResizing(true);
    setResizeStart({
      x: e.clientX,
      y: e.clientY,
      width: size.width,
      height: size.height
    });
    e.preventDefault();
    e.stopPropagation();
  }, [size]);

  // Handle mouse up
  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
    setIsResizing(false);
  }, []);

  // Add global mouse up listener and cursor styles
  React.useEffect(() => {
    if (isDragging || isResizing) {
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = isDragging ? 'grabbing' : 'nw-resize';
      document.body.style.userSelect = 'none';

      return () => {
        document.removeEventListener('mouseup', handleMouseUp);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      };
    }
  }, [isDragging, isResizing, handleMouseUp]);

  if (!nodeData) return null;

  const statusIndicator = getStatusIndicator(nodeData.status);

  return (
    <AnimatePresence>
      <motion.div
        ref={panelRef}
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.9 }}
        transition={{ duration: 0.3, ease: "easeOut" }}
        className="fixed z-50 bg-gray-800 border-2 border-gray-600 rounded-lg shadow-2xl select-none"
        style={{
          left: position.x,
          top: position.y,
          width: size.width,
          height: size.height,
          cursor: isDragging ? 'grabbing' : 'default'
        }}
      >
        {/* Header - Draggable */}
        <div
          className="p-3 border-b border-gray-700 bg-gray-900 rounded-t-lg drag-handle cursor-grab active:cursor-grabbing"
          onMouseDown={handleDragStart}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Move size={14} className="text-gray-500" />
              <div className="w-8 h-8 bg-gray-700 rounded flex items-center justify-center border border-gray-600">
                {getComponentIcon(nodeData.type, {
                  className: "text-white drop-shadow-sm",
                  size: 16
                })}
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="text-white font-semibold text-sm truncate">{nodeData.label}</h3>
                <p className="text-gray-400 text-xs capitalize">
                  {nodeData.type.replace(/([A-Z])/g, ' $1').trim()}
                </p>
              </div>
              <div className={`w-3 h-3 rounded-full ${statusIndicator} shadow-sm`} />
            </div>
            <button
              onClick={onClose}
              className="ml-2 p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded transition-colors"
              title="Close details panel"
            >
              <X size={16} />
            </button>
          </div>
        </div>

        {/* Content - Scrollable */}
        <div className="p-3 space-y-3 overflow-y-auto" style={{ height: size.height - 120 }}>
          {/* Status Section */}
          <div className="bg-gray-700 rounded p-2">
            <h4 className="text-white font-medium text-xs mb-2">Status</h4>
            <div className="flex items-center justify-between">
              <span className="text-gray-300 text-xs">Current:</span>
              <span className={`capitalize font-medium text-xs px-2 py-0.5 rounded ${
                nodeData.status === 'healthy' ? 'bg-green-600/30 text-green-400' :
                nodeData.status === 'warning' ? 'bg-yellow-600/30 text-yellow-400' :
                nodeData.status === 'critical' ? 'bg-red-600/30 text-red-400' :
                'bg-gray-600/30 text-gray-400'
              }`}>
                {nodeData.status}
              </span>
            </div>
          </div>

          {/* Performance Metrics */}
          {(nodeData.rps !== undefined || nodeData.latency !== undefined) && (
            <div className="bg-gray-700 rounded p-2">
              <h4 className="text-white font-medium text-xs mb-2">Performance</h4>
              <div className="space-y-1">
                {nodeData.rps !== undefined && (
                  <div className="flex justify-between items-center">
                    <span className="text-gray-300 text-xs">RPS:</span>
                    <span className="text-white font-medium text-xs">{nodeData.rps.toLocaleString()}</span>
                  </div>
                )}
                {nodeData.latency !== undefined && (
                  <div className="flex justify-between items-center">
                    <span className="text-gray-300 text-xs">Latency:</span>
                    <span className="text-white font-medium text-xs">{nodeData.latency}ms</span>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Resource Usage */}
          {(nodeData.cpu !== undefined || nodeData.memory !== undefined) && (
            <div className="bg-gray-700 rounded p-2">
              <h4 className="text-white font-medium text-xs mb-2">Resources</h4>
              <div className="space-y-2">
                {nodeData.cpu !== undefined && (
                  <div>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-gray-300 text-xs">CPU:</span>
                      <span className="text-white font-medium text-xs">{nodeData.cpu}%</span>
                    </div>
                    <div className="w-full h-1.5 bg-gray-600 rounded overflow-hidden">
                      <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${nodeData.cpu}%` }}
                        transition={{ duration: 1, ease: "easeOut" }}
                        className={`h-full rounded ${
                          nodeData.cpu > 80 ? 'bg-red-500' :
                          nodeData.cpu > 60 ? 'bg-yellow-500' :
                          'bg-green-500'
                        }`}
                      />
                    </div>
                  </div>
                )}
                {nodeData.memory !== undefined && (
                  <div>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-gray-300 text-xs">Memory:</span>
                      <span className="text-white font-medium text-xs">{nodeData.memory}%</span>
                    </div>
                    <div className="w-full h-1.5 bg-gray-600 rounded overflow-hidden">
                      <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${nodeData.memory}%` }}
                        transition={{ duration: 1, ease: "easeOut", delay: 0.2 }}
                        className={`h-full rounded ${
                          nodeData.memory > 80 ? 'bg-red-500' :
                          nodeData.memory > 60 ? 'bg-yellow-500' :
                          'bg-green-500'
                        }`}
                      />
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Connections */}
          {nodeData.connections !== undefined && (
            <div className="bg-gray-700 rounded p-2">
              <h4 className="text-white font-medium text-xs mb-1">Connections</h4>
              <div className="flex justify-between items-center">
                <span className="text-gray-300 text-xs">Active:</span>
                <span className="text-white font-medium text-sm">{nodeData.connections}</span>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-3 py-2 bg-gray-900 border-t border-gray-700 rounded-b-lg">
          <div className="text-xs text-gray-500 text-center">
            Double-click another node to switch
          </div>
        </div>

        {/* Resize Handle */}
        <div
          className="absolute bottom-0 right-0 w-4 h-4 cursor-nw-resize opacity-50 hover:opacity-100 transition-opacity"
          onMouseDown={handleResizeStart}
          title="Drag to resize"
        >
          <CornerDownRight size={16} className="text-gray-400 absolute bottom-0 right-0" />
        </div>
      </motion.div>
    </AnimatePresence>
  );
};
