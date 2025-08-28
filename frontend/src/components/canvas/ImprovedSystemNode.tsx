import React, { memo, useState } from 'react';
import { Handle, Position } from '@xyflow/react';
import type { Node } from '@xyflow/react';
import { motion, AnimatePresence } from 'framer-motion';
import { getComponentIcon } from '../ui/CustomIcons';

export interface SystemNodeData {
  label: string;
  type: 'webserver' | 'database' | 'cache' | 'loadbalancer' | 'cdn' | 'queue' | 'microservice' | 'client' | 'mobile' | 'storage' | 'gateway' | 'security' | 'monitoring';
  status: 'healthy' | 'warning' | 'critical' | 'offline';
  rps?: number;
  latency?: number;
  cpu?: number;
  memory?: number;
  connections?: number;
}

const getNodeColor = (type: string) => {
  // Drawing board aesthetic - all black with subtle variations
  switch (type) {
    case 'client': return 'from-gray-800 to-gray-900';
    case 'mobile': return 'from-gray-800 to-gray-900';
    case 'webserver': return 'from-gray-800 to-gray-900';
    case 'microservice': return 'from-gray-800 to-gray-900';
    case 'loadbalancer': return 'from-gray-800 to-gray-900';
    case 'database': return 'from-gray-800 to-gray-900';
    case 'cache': return 'from-gray-800 to-gray-900';
    case 'storage': return 'from-gray-800 to-gray-900';
    case 'cdn': return 'from-gray-800 to-gray-900';
    case 'gateway': return 'from-gray-800 to-gray-900';
    case 'queue': return 'from-gray-800 to-gray-900';
    case 'security': return 'from-gray-800 to-gray-900';
    case 'monitoring': return 'from-gray-800 to-gray-900';
    default: return 'from-gray-800 to-gray-900';
  }
};

const getStatusIndicator = (status: string) => {
  switch (status) {
    case 'healthy': return 'bg-green-500';
    case 'warning': return 'bg-yellow-500 animate-pulse';
    case 'critical': return 'bg-red-500 animate-pulse';
    case 'offline': return 'bg-gray-500';
    default: return 'bg-gray-500';
  }
};

const getMetricColor = (value: number, type: 'cpu' | 'memory' | 'latency') => {
  if (type === 'latency') {
    if (value > 100) return 'bg-red-500';
    if (value > 50) return 'bg-yellow-500';
    return 'bg-green-500';
  }
  
  // For CPU and Memory
  if (value > 80) return 'bg-red-500';
  if (value > 60) return 'bg-yellow-500';
  return 'bg-green-500';
};

interface ImprovedSystemNodeProps {
  data: SystemNodeData;
  selected?: boolean;
  onDoubleClick?: (data: SystemNodeData) => void;
}

export const ImprovedSystemNode: React.FC<ImprovedSystemNodeProps> = memo(({ data, selected, onDoubleClick }) => {
  const statusIndicator = getStatusIndicator(data.status);

  const handleDoubleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDoubleClick?.(data);
  };

  return (
    <motion.div
      initial={{ scale: 0.9, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      whileHover={{ scale: 1.05 }}
      className={`relative group cursor-pointer transition-all duration-200 ${
        selected ? 'z-10' : 'z-0'
      }`}
      onDoubleClick={handleDoubleClick}
    >
      {/* Connection Handles - Hidden by default, visible on hover */}
      <Handle
        type="target"
        position={Position.Top}
        className="w-2 h-2 bg-white border border-gray-600 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        className="w-2 h-2 bg-white border border-gray-600 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
      />
      <Handle
        type="target"
        position={Position.Left}
        className="w-2 h-2 bg-white border border-gray-600 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="w-2 h-2 bg-white border border-gray-600 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
      />

      {/* Compact Node Design - Icon Only */}
      <div className={`relative w-16 h-16 bg-gray-800 border-2 rounded-lg shadow-lg transition-all duration-200 ${
        selected
          ? 'border-white shadow-white/30 shadow-lg'
          : 'border-gray-600 hover:border-gray-400 hover:shadow-md'
      }`}>

        {/* Status Indicator - Top Right Corner */}
        <div className={`absolute -top-1 -right-1 w-3 h-3 rounded-full ${statusIndicator} border border-gray-800 shadow-sm`} />

        {/* Main Icon */}
        <div className="w-full h-full flex items-center justify-center">
          {getComponentIcon(data.type, {
            className: "text-white drop-shadow-sm",
            size: 28
          })}
        </div>

        {/* Selection Indicator */}
        {selected && (
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            className="absolute inset-0 border-2 border-white rounded-lg pointer-events-none"
          />
        )}

        {/* Hover Glow Effect */}
        <motion.div
          className="absolute inset-0 rounded-lg bg-white/10 opacity-0 pointer-events-none"
          whileHover={{ opacity: 1 }}
          transition={{ duration: 0.2 }}
        />
      </div>

      {/* Hover Tooltip */}
      <motion.div
        initial={{ opacity: 0, y: 5 }}
        whileHover={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2, delay: 0.5 }}
        className="absolute top-full left-1/2 transform -translate-x-1/2 mt-2 px-3 py-2 bg-gray-900 border border-gray-600 rounded-lg text-xs text-white font-medium whitespace-nowrap shadow-lg z-20 opacity-0 group-hover:opacity-100 pointer-events-none"
      >
        <div className="text-center">
          <div className="font-semibold">{data.label}</div>
          <div className="text-gray-400 capitalize mt-1">
            {data.type.replace(/([A-Z])/g, ' $1').trim()}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            Double-click for details
          </div>
        </div>
        {/* Arrow pointing up to the node */}
        <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 w-0 h-0 border-l-3 border-r-3 border-b-3 border-transparent border-b-gray-900" />
      </motion.div>
    </motion.div>
  );
});

ImprovedSystemNode.displayName = 'ImprovedSystemNode';