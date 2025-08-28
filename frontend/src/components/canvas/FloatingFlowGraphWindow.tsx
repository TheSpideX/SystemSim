import React, { useState, useRef, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Settings, Move } from 'lucide-react';
import { FlowGraphCanvas } from './FlowGraphCanvas';

interface FloatingFlowGraphWindowProps {
  isVisible: boolean;
  onToggleVisibility: () => void;
  selectedComponent: string | null;
  isSimulationRunning: boolean;
  onToggleSimulation: () => void;
}

export const FloatingFlowGraphWindow: React.FC<FloatingFlowGraphWindowProps> = ({
  isVisible,
  onToggleVisibility,
  selectedComponent,
  isSimulationRunning,
  onToggleSimulation,
}) => {
  // Window state for dragging
  const [position, setPosition] = useState({ x: window.innerWidth - 420, y: 80 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const windowRef = useRef<HTMLDivElement>(null);

  // Drag handlers
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    setIsDragging(true);
    setDragStart({
      x: e.clientX - position.x,
      y: e.clientY - position.y,
    });
  }, [position]);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (isDragging) {
      const newX = Math.max(0, Math.min(window.innerWidth - 400, e.clientX - dragStart.x));
      const newY = Math.max(0, Math.min(window.innerHeight - 500, e.clientY - dragStart.y));
      setPosition({ x: newX, y: newY });
    }
  }, [isDragging, dragStart]);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  // Mouse event listeners
  React.useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      return () => {
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      };
    }
  }, [isDragging, handleMouseMove, handleMouseUp]);

  if (!isVisible) return null;

  return (
    <AnimatePresence>
      <motion.div
        ref={windowRef}
        initial={{ opacity: 0, scale: 0.9, x: 50, y: -50 }}
        animate={{ opacity: 1, scale: 1, x: 0, y: 0 }}
        exit={{ opacity: 0, scale: 0.9, x: 50, y: -50 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="fixed bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg shadow-2xl z-50 flex flex-col"
        style={{
          left: position.x,
          top: position.y,
          width: 400,
          height: 500,
          cursor: isDragging ? 'grabbing' : 'default',
        }}
      >
        {/* Header */}
        <div
          className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 rounded-t-lg select-none"
          onMouseDown={handleMouseDown}
          style={{ cursor: isDragging ? 'grabbing' : 'grab' }}
        >
          <div className="flex items-center space-x-3">
            <div className="w-3 h-3 bg-green-500 rounded-full"></div>
            <h2 className="text-sm font-medium text-gray-900 dark:text-white">
              Decision Graph
            </h2>
            {isDragging && (
              <div className="text-xs text-gray-500 dark:text-gray-400">
                <Move className="w-3 h-3 inline mr-1" />
                Moving...
              </div>
            )}
          </div>

          <div className="flex items-center space-x-2">
            {/* Component Selector */}
            <select
              className="px-2 py-1 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded text-xs text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500"
              onClick={(e) => e.stopPropagation()}
            >
              <option value="main">Main</option>
              <option value="auth">Auth</option>
              <option value="data">Data</option>
            </select>

            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={(e) => {
                e.stopPropagation();
                onToggleVisibility();
              }}
              className="p-1.5 text-gray-600 dark:text-gray-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-all duration-200"
              title="Close Decision Graph"
            >
              <X className="w-4 h-4" />
            </motion.button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-hidden rounded-b-lg">
          <FlowGraphCanvas
            selectedComponent={selectedComponent}
            isSimulationRunning={isSimulationRunning}
            onToggleSimulation={onToggleSimulation}
          />
        </div>
      </motion.div>
    </AnimatePresence>
  );
};
