import React, { useCallback, useRef, useState } from 'react';
import { motion } from 'framer-motion';
import { useGlobalMouse } from '../../hooks/useGlobalMouse';

interface ResizablePanelProps {
  children: React.ReactNode;
  width: number;
  minWidth?: number;
  maxWidth?: number;
  onResize: (width: number) => void;
  position: 'left' | 'right';
  isCollapsed: boolean;
  className?: string;
}

export const ResizablePanel: React.FC<ResizablePanelProps> = ({
  children,
  width,
  minWidth = 200,
  maxWidth = 600,
  onResize,
  position,
  isCollapsed,
  className = '',
}) => {
  const [isResizing, setIsResizing] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const startXRef = useRef<number>(0);
  const startWidthRef = useRef<number>(0);

  // Use global mouse tracking for resize - preserves exact behavior
  useGlobalMouse((mousePosition) => {
    if (isResizing) {
      const deltaX = position === 'left'
        ? mousePosition.x - startXRef.current
        : startXRef.current - mousePosition.x;

      const newWidth = Math.max(minWidth, Math.min(maxWidth, startWidthRef.current + deltaX));
      onResize(newWidth);
    }
  }, [isResizing, position, minWidth, maxWidth, onResize]);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
    startXRef.current = e.clientX;
    startWidthRef.current = width;

    const handleMouseUp = () => {
      setIsResizing(false);
      document.removeEventListener('mouseup', handleMouseUp);
    };

    document.addEventListener('mouseup', handleMouseUp);
  }, [width, minWidth, maxWidth, onResize, position, isResizing]);

  return (
    <motion.div
      ref={panelRef}
      initial={false}
      animate={{ 
        width: isCollapsed ? 0 : width,
        opacity: isCollapsed ? 0 : 1 
      }}
      transition={{ duration: 0.3, ease: 'easeInOut' }}
      className={`relative flex-shrink-0 ${className}`}
      style={{ width: isCollapsed ? 0 : width }}
    >
      {!isCollapsed && (
        <>
          {children}
          
          {/* Resize Handle */}
          <div
            className={`absolute top-0 bottom-0 w-1 cursor-col-resize group ${
              position === 'left' ? '-right-0.5' : '-left-0.5'
            }`}
            onMouseDown={handleMouseDown}
          >
            <div className="w-full h-full bg-transparent group-hover:bg-blue-500 transition-colors duration-200" />
            <div className={`absolute top-1/2 transform -translate-y-1/2 w-1 h-8 bg-gray-400 dark:bg-gray-600 rounded-full opacity-0 group-hover:opacity-100 transition-opacity duration-200 ${
              position === 'left' ? '-right-0.5' : '-left-0.5'
            }`} />
          </div>
        </>
      )}
    </motion.div>
  );
};