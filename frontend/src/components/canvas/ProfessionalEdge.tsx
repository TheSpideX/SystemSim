import React from 'react';
import { getBezierPath, EdgeLabelRenderer, Position } from '@xyflow/react';
import { motion } from 'framer-motion';

interface ProfessionalEdgeData {
  throughput?: number;
  label?: string;
  status?: 'healthy' | 'warning' | 'critical';
}

interface ProfessionalEdgeProps {
  id: string;
  sourceX: number;
  sourceY: number;
  targetX: number;
  targetY: number;
  sourcePosition: Position;
  targetPosition: Position;
  style?: React.CSSProperties;
  data?: ProfessionalEdgeData;
  selected?: boolean;
  animated?: boolean;
}

export const ProfessionalEdge: React.FC<ProfessionalEdgeProps> = ({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  data,
  selected,
  animated,
}) => {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  // Determine edge color based on throughput and status
  const getEdgeColor = () => {
    if (data?.status === 'critical') return '#ef4444';
    if (data?.status === 'warning') return '#f59e0b';
    if (selected) return '#fff';
    return '#6b7280'; // Gray for very low throughput
  };

  // Determine edge width based on throughput
  const getEdgeWidth = () => {
    const throughput = data?.throughput || 0;
    if (throughput > 20000) return 4;
    if (throughput > 10000) return 3;
    if (throughput > 5000) return 2;
    return 1.5;
  };

  const edgeColor = getEdgeColor();
  const edgeWidth = getEdgeWidth();

  return (
    <>
      {/* Main Edge Path */}
      <path
        id={id}
        style={{
          ...style,
          stroke: edgeColor,
          strokeWidth: edgeWidth,
          fill: 'none',
          strokeLinecap: 'round',
          strokeLinejoin: 'round',
          opacity: selected ? 1 : 0.8,
          filter: selected ? 'drop-shadow(0 0 6px rgba(59, 130, 246, 0.4))' : 'none',
        }}
        className={`react-flow__edge-path ${animated ? 'animate-pulse' : ''}`}
        d={edgePath}
      />

      {/* Animated Flow Indicator */}
      {animated && (
        <motion.circle
          r="3"
          fill={edgeColor}
          className="opacity-80"
          initial={{ offsetDistance: '0%' }}
          animate={{ offsetDistance: '100%' }}
          transition={{
            duration: 2,
            repeat: Infinity,
            ease: 'linear',
          }}
          style={{
            offsetPath: `path('${edgePath}')`,
            offsetRotate: 'auto',
          }}
        />
      )}

      {/* Edge Label */}
      <EdgeLabelRenderer>
        <div
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            fontSize: 11,
            pointerEvents: 'all',
          }}
          className="nodrag nopan"
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            className={`bg-white/95 backdrop-blur-sm border border-gray-200/80 rounded-md px-2 py-1 shadow-sm ${
              selected ? 'border-blue-300 shadow-blue-100' : ''
            }`}
          >
            <div className="flex flex-col items-center space-y-0.5">
              {data?.label && (
                <div className="text-xs font-medium text-gray-700">
                  {data.label}
                </div>
              )}
              {data?.throughput && (
                <div className="text-xs text-gray-600 font-mono">
                  {data.throughput.toLocaleString()} RPS
                </div>
              )}
            </div>
          </motion.div>
        </div>
      </EdgeLabelRenderer>

      {/* Selection Indicator */}
      {selected && (
        <path
          style={{
            stroke: '#3b82f6',
            strokeWidth: edgeWidth + 4,
            fill: 'none',
            strokeLinecap: 'round',
            strokeLinejoin: 'round',
            opacity: 0.2,
          }}
          d={edgePath}
        />
      )}
    </>
  );
};