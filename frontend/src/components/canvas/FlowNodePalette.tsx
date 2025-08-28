import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  ArrowRight, Database, Filter, Shuffle, GitBranch, Zap, Clock, CheckCircle,
  Search, X
} from 'lucide-react';

// Flow node types for internal component logic
const flowNodeTypes = [
  { 
    type: 'input', 
    label: 'Input', 
    icon: ArrowRight, 
    color: 'text-green-400',
    description: 'Data input point'
  },
  { 
    type: 'process', 
    label: 'Process', 
    icon: Zap, 
    color: 'text-blue-400',
    description: 'Processing logic'
  },
  { 
    type: 'decision', 
    label: 'Decision', 
    icon: GitBranch, 
    color: 'text-yellow-400',
    description: 'Conditional branching'
  },
  { 
    type: 'data', 
    label: 'Data Store', 
    icon: Database, 
    color: 'text-purple-400',
    description: 'Data storage'
  },
  { 
    type: 'filter', 
    label: 'Filter', 
    icon: Filter, 
    color: 'text-orange-400',
    description: 'Data filtering'
  },
  { 
    type: 'transform', 
    label: 'Transform', 
    icon: Shuffle, 
    color: 'text-pink-400',
    description: 'Data transformation'
  },
  { 
    type: 'timer', 
    label: 'Timer', 
    icon: Clock, 
    color: 'text-indigo-400',
    description: 'Time-based operations'
  },
  { 
    type: 'output', 
    label: 'Output', 
    icon: CheckCircle, 
    color: 'text-red-400',
    description: 'Data output point'
  },
];

interface FlowNodePaletteProps {
  onDragStart: (event: React.DragEvent, nodeType: any) => void;
}

export const FlowNodePalette: React.FC<FlowNodePaletteProps> = ({
  onDragStart,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [isExpanded, setIsExpanded] = useState(true);

  // Filter nodes based on search
  const filteredNodes = flowNodeTypes.filter(node =>
    node.label.toLowerCase().includes(searchTerm.toLowerCase()) ||
    node.description.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="h-full flex flex-col bg-gray-800 border-t border-gray-700">
      {/* Palette Header */}
      <div className="p-3 border-b border-gray-700">
        <div className="flex items-center justify-between mb-3">
          <h4 className="text-sm font-medium text-white">Flow Nodes</h4>
          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={() => setIsExpanded(!isExpanded)}
            className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded transition-all duration-200"
          >
            {isExpanded ? <X className="w-4 h-4" /> : <ArrowRight className="w-4 h-4" />}
          </motion.button>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search flow nodes..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
      </div>

      {/* Node List */}
      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="flex-1 overflow-y-auto"
          >
            <div className="p-2 space-y-1">
              {filteredNodes.map((node) => {
                const Icon = node.icon;
                return (
                  <motion.div
                    key={node.type}
                    draggable
                    onDragStart={(event) => onDragStart(event, node)}
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    className="flex items-center space-x-3 p-3 bg-gray-700 hover:bg-gray-600 rounded-lg cursor-grab active:cursor-grabbing transition-all duration-200 group"
                  >
                    <div className={`p-2 rounded-md bg-gray-800 group-hover:bg-gray-900 transition-colors`}>
                      <Icon className={`w-4 h-4 ${node.color}`} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-white">
                        {node.label}
                      </div>
                      <div className="text-xs text-gray-400 truncate">
                        {node.description}
                      </div>
                    </div>
                  </motion.div>
                );
              })}
            </div>

            {filteredNodes.length === 0 && (
              <div className="p-4 text-center">
                <div className="text-gray-400 text-sm">
                  No flow nodes found
                </div>
                <div className="text-gray-500 text-xs mt-1">
                  Try adjusting your search terms
                </div>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>

      {/* Node Categories */}
      {isExpanded && (
        <div className="p-3 border-t border-gray-700">
          <div className="text-xs font-medium text-gray-400 mb-2">Categories</div>
          <div className="flex flex-wrap gap-1">
            {['Input/Output', 'Processing', 'Logic', 'Data'].map((category) => (
              <motion.button
                key={category}
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                className="px-2 py-1 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 rounded transition-all duration-200"
              >
                {category}
              </motion.button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
