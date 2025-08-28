import React, { useCallback, useRef, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useGlobalMouse } from '../../hooks/useGlobalMouse';

interface Point {
  x: number;
  y: number;
}

interface LassoSelectionProps {
  isActive: boolean;
  onSelection: (boundingBox: { x: number; y: number; width: number; height: number }) => void;
  children: React.ReactNode;
}

export const LassoSelection: React.FC<LassoSelectionProps> = ({
  isActive,
  onSelection,
  children,
}) => {
  const [isSelecting, setIsSelecting] = useState(false);
  const [startPoint, setStartPoint] = useState<Point>({ x: 0, y: 0 });
  const [currentPoint, setCurrentPoint] = useState<Point>({ x: 0, y: 0 });
  const containerRef = useRef<HTMLDivElement>(null);

  // Use global mouse tracking for selection - preserves exact behavior
  useGlobalMouse((mousePosition) => {
    if (isSelecting && isActive) {
      const rect = containerRef.current?.getBoundingClientRect();
      if (rect) {
        setCurrentPoint({
          x: mousePosition.x - rect.left,
          y: mousePosition.y - rect.top,
        });
      }
    }
  }, [isSelecting, isActive]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (!isActive) return;

      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect) return;

      const point = {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top,
      };

      setStartPoint(point);
      setCurrentPoint(point);
      setIsSelecting(true);
    },
    [isActive]
  );



  const handleMouseUp = useCallback(() => {
    if (!isSelecting || !isActive) return;

    const minX = Math.min(startPoint.x, currentPoint.x);
    const minY = Math.min(startPoint.y, currentPoint.y);
    const maxX = Math.max(startPoint.x, currentPoint.x);
    const maxY = Math.max(startPoint.y, currentPoint.y);

    const boundingBox = {
      x: minX,
      y: minY,
      width: maxX - minX,
      height: maxY - minY,
    };

    // Only trigger selection if the box has meaningful size
    if (boundingBox.width > 5 && boundingBox.height > 5) {
      onSelection(boundingBox);
    }

    setIsSelecting(false);
  }, [isSelecting, isActive, startPoint, currentPoint, onSelection]);

  const selectionBox = {
    left: Math.min(startPoint.x, currentPoint.x),
    top: Math.min(startPoint.y, currentPoint.y),
    width: Math.abs(currentPoint.x - startPoint.x),
    height: Math.abs(currentPoint.y - startPoint.y),
  };

  return (
    <div
      ref={containerRef}
      className="relative w-full h-full"
      onMouseDown={handleMouseDown}
      onMouseUp={handleMouseUp}
      style={{ cursor: isActive ? 'crosshair' : 'default' }}
    >
      {children}
      
      <AnimatePresence>
        {isSelecting && isActive && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute pointer-events-none border-2 border-blue-500 bg-blue-500/10 rounded"
            style={{
              left: selectionBox.left,
              top: selectionBox.top,
              width: selectionBox.width,
              height: selectionBox.height,
            }}
          />
        )}
      </AnimatePresence>
    </div>
  );
};