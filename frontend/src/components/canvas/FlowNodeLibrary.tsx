import React, { useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  ArrowRight, Database, Filter, Shuffle, GitBranch, Zap, Clock, CheckCircle,
  Search, Layers, X
} from 'lucide-react';

// Flow node types for internal component logic
const flowNodeTypes = [
  { 
    type: 'input', 
    name: 'Input', 
    icon: ArrowRight, 
    color: 'text-green-400',
    description: 'Data input point'
  },
  { 
    type: 'process', 
    name: 'Process', 
    icon: Zap, 
    color: 'text-blue-400',
    description: 'Processing logic'
  },
  { 
    type: 'decision', 
    name: 'Decision', 
    icon: GitBranch, 
    color: 'text-yellow-400',
    description: 'Conditional branching'
  },
  { 
    type: 'data', 
    name: 'Data Store', 
    icon: Database, 
    color: 'text-purple-400',
    description: 'Data storage'
  },
  { 
    type: 'filter', 
    name: 'Filter', 
    icon: Filter, 
    color: 'text-orange-400',
    description: 'Data filtering'
  },
  { 
    type: 'transform', 
    name: 'Transform', 
    icon: Shuffle, 
    color: 'text-pink-400',
    description: 'Data transformation'
  },
  { 
    type: 'timer', 
    name: 'Timer', 
    icon: Clock, 
    color: 'text-indigo-400',
    description: 'Time-based operations'
  },
  { 
    type: 'output', 
    name: 'Output', 
    icon: CheckCircle, 
    color: 'text-red-400',
    description: 'Data output point'
  },
];

interface FlowNodeLibraryProps {
  isVisible: boolean;
  onToggleVisibility: () => void;
  onDragStart: (event: React.DragEvent, flowNode: any) => void;
}

export const FlowNodeLibrary: React.FC<FlowNodeLibraryProps> = ({
  isVisible,
  onToggleVisibility,
  onDragStart,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [isButtonVisible, setIsButtonVisible] = useState(false);

  // Handle mouse enter/leave for auto-reveal
  const handleMouseEnter = useCallback(() => {
    setIsButtonVisible(true);
  }, []);

  const handleMouseLeave = useCallback(() => {
    if (!isVisible) {
      setIsButtonVisible(false);
    }
  }, [isVisible]);

  // Mouse zone detection for auto-reveal (right side)
  React.useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      const windowWidth = window.innerWidth;
      if (e.clientX >= windowWidth - 100) { // Within 100px of right edge
        handleMouseEnter();
      } else if (!isVisible) {
        handleMouseLeave();
      }
    };

    document.addEventListener('mousemove', handleMouseMove);
    return () => document.removeEventListener('mousemove', handleMouseMove);
  }, [isVisible, handleMouseEnter, handleMouseLeave]);

  // Filter flow nodes based on search
  const filteredNodes = flowNodeTypes.filter(node =>
    node.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    node.description.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="fixed top-0 right-0 h-full z-50 pointer-events-none">
      {/* Flow Node Library Button */}
      <AnimatePresence>
        {isButtonVisible && (
          <motion.div
            initial={{ x: 100, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            exit={{ x: 100, opacity: 0 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className="absolute top-1/2 -translate-y-1/2 right-4 pointer-events-auto"
          >
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={onToggleVisibility}
              className="w-12 h-12 bg-gray-800 hover:bg-gray-700 border border-gray-600 rounded-lg flex items-center justify-center text-white shadow-lg transition-all duration-200"
              title="Flow Node Library"
            >
              <Layers className="w-5 h-5" />
            </motion.button>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Flow Nodes Panel - Slides out from the right */}
      <AnimatePresence>
        {isVisible && (
          <motion.div
            initial={{ x: "100%" }}
            animate={{ x: 0 }}
            exit={{ x: "100%" }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="w-80 h-full bg-gray-900 border-l border-gray-700 shadow-2xl pointer-events-auto flex flex-col"
          >
            {/* Header */}
            <div className="p-4 border-b border-gray-700">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-lg font-semibold text-white">Flow Nodes</h3>
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={onToggleVisibility}
                  className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-md transition-all duration-200"
                >
                  <X className="w-4 h-4" />
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
                  className="w-full pl-10 pr-4 py-2 bg-gray-800 border border-gray-600 rounded-lg text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>

            {/* Flow Nodes List */}
            <div className="flex-1 overflow-y-auto p-4">
              <div className="space-y-2">
                {filteredNodes.map((node) => {
                  const Icon = node.icon;
                  return (
                    <motion.div
                      key={node.type}
                      draggable
                      onDragStart={(event) => {
                        event.stopPropagation();
                        onDragStart(event, node);
                      }}
                      whileHover={{ scale: 1.02 }}
                      whileTap={{ scale: 0.98 }}
                      className="flex items-center space-x-3 p-3 bg-gray-800 hover:bg-gray-700 rounded-lg cursor-grab active:cursor-grabbing transition-all duration-200 group"
                    >
                      <div className={`p-2 rounded-md bg-gray-700 group-hover:bg-gray-600 transition-colors`}>
                        <Icon className={`w-4 h-4 ${node.color} drop-shadow-sm`} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-white">
                          {node.name}
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
                <div className="text-center py-8">
                  <div className="text-gray-400 text-sm">
                    No flow nodes found
                  </div>
                  <div className="text-gray-500 text-xs mt-1">
                    Try adjusting your search terms
                  </div>
                </div>
              )}
            </div>

            {/* Footer */}
            <div className="p-4 border-t border-gray-700">
              <div className="text-xs text-gray-400 text-center">
                Drag nodes to the flow canvas to add them
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
