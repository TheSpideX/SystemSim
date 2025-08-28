import React, { useState, useCallback, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Search, Package, Cpu, HardDrive, Database, Wifi,
  Zap, Activity, MemoryStick, Network
} from 'lucide-react';

interface EngineItem {
  id: string;
  name: string;
  type: string;
  description: string;
  category: string;
  icon: React.ComponentType<any>;
  color: string;
}

const engineLibrary: EngineItem[] = [
  {
    id: 'cpu',
    name: 'CPU Engine',
    type: 'cpu',
    description: 'Handles computational work and processing',
    category: 'engines',
    icon: Cpu,
    color: 'text-white'
  },
  {
    id: 'memory',
    name: 'Memory Engine',
    type: 'memory',
    description: 'Manages RAM and cache operations',
    category: 'engines',
    icon: MemoryStick,
    color: 'text-white'
  },
  {
    id: 'storage',
    name: 'Storage Engine',
    type: 'storage',
    description: 'Handles disk I/O and persistent storage',
    category: 'engines',
    icon: HardDrive,
    color: 'text-white'
  },
  {
    id: 'network',
    name: 'Network Engine',
    type: 'network',
    description: 'Manages network communication and protocols',
    category: 'engines',
    icon: Network,
    color: 'text-white'
  }
];

const EngineItem = React.memo<{
  engine: EngineItem;
  onDragStart: (event: React.DragEvent, engine: EngineItem) => void;
}>(({ engine, onDragStart }) => {
  const Icon = engine.icon;

  return (
    <motion.div
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: -10 }}
      whileHover={{ x: 4 }}
      className="group"
    >
      <div
        draggable={true}
        onDragStart={(event) => {
          event.stopPropagation();
          onDragStart(event, engine);
        }}
        className="flex items-center space-x-3 p-3 hover:bg-gray-800 rounded-lg cursor-grab active:cursor-grabbing transition-all duration-200"
      >
        {/* Icon - Black and White like simulation components */}
        <div className="w-10 h-10 bg-gray-900 rounded-lg flex items-center justify-center border border-gray-600 group-hover:border-gray-500 transition-colors flex-shrink-0">
          <Icon className="text-white" size={20} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center space-x-2">
            <h3 className="text-white font-medium text-sm truncate">{engine.name}</h3>
          </div>
          <p className="text-gray-400 text-xs truncate">{engine.description}</p>
        </div>
      </div>
    </motion.div>
  );
});

interface EngineLibraryProps {
  onDragStart: (event: React.DragEvent, engine: EngineItem) => void;
  isVisible?: boolean;
  onToggleVisibility?: () => void;
}

export const EngineLibrary: React.FC<EngineLibraryProps> = ({
  onDragStart,
  isVisible = false,
  onToggleVisibility
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [isButtonVisible, setIsButtonVisible] = useState(false);

  const panelRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  // Auto-reveal functionality - show button when mouse is near left edge
  const handleMouseEnter = useCallback(() => {
    setIsButtonVisible(true);
  }, []);

  const handleMouseLeave = useCallback(() => {
    if (!isVisible) {
      setIsButtonVisible(false);
    }
  }, [isVisible]);

  // Mouse zone detection for auto-reveal
  React.useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (e.clientX <= 100) { // Within 100px of left edge
        handleMouseEnter();
      } else if (!isVisible) {
        handleMouseLeave();
      }
    };

    document.addEventListener('mousemove', handleMouseMove);
    return () => document.removeEventListener('mousemove', handleMouseMove);
  }, [isVisible, handleMouseEnter, handleMouseLeave]);

  // Filter engines based on search
  const filteredEngines = engineLibrary.filter(engine =>
    engine.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    engine.description.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <>
      {/* Auto-reveal Expandable Button Container */}
      <AnimatePresence>
        {(isButtonVisible || isVisible) && (
          <motion.div
            ref={panelRef}
            initial={{ opacity: 0 }}
            animate={{
              opacity: 1,
              height: isVisible ? 'auto' : '56px',
              width: isVisible ? 'auto' : '56px'
            }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3, ease: "easeOut" }}
            className="fixed top-1/2 left-4 transform -translate-y-1/2 z-50 bg-gray-800 border border-gray-700 rounded-lg shadow-lg overflow-hidden"
          >
            {!isVisible ? (
              /* Collapsed State - Just the Engine Button */
              <motion.button
                ref={triggerRef}
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                onClick={onToggleVisibility}
                className="w-14 h-14 flex items-center justify-center text-white hover:bg-gray-700 transition-all duration-200"
                title="Show engine library"
              >
                <Zap className="w-5 h-5" />
              </motion.button>
            ) : (
              /* Expanded State - Vertical Stack */
              <div className="p-2 flex flex-col items-center space-y-2">
                {/* Engine Icon Button */}
                <motion.button
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.8 }}
                  transition={{ delay: 0.1 }}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  className="w-10 h-10 flex items-center justify-center bg-gray-600 text-white border border-gray-500 shadow-sm rounded-md transition-all duration-200"
                  title="Engines - CPU, Memory, Storage, Network"
                >
                  <Zap className="w-4 h-4" />
                </motion.button>

                {/* Separator */}
                <div className="w-8 h-px bg-gray-600 my-1"></div>

                {/* Search Button */}
                <motion.button
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ delay: 0.2 }}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  className="w-10 h-10 flex items-center justify-center text-white hover:bg-gray-700 rounded-md transition-all duration-200"
                  title="Search engines"
                >
                  <Search className="w-4 h-4" />
                </motion.button>

                {/* Close Button */}
                <motion.button
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ delay: 0.3 }}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={onToggleVisibility}
                  className="w-10 h-10 flex items-center justify-center text-white hover:bg-gray-700 rounded-md transition-all duration-200"
                  title="Close engine library"
                >
                  ×
                </motion.button>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>

      {/* Engines Panel - Slides out from the toolbar */}
      <AnimatePresence>
        {isVisible && (
          <motion.div
            initial={{ opacity: 0, x: -20, scaleX: 0.8, y: '-50%' }}
            animate={{ opacity: 1, x: 0, scaleX: 1, y: '-50%' }}
            exit={{ opacity: 0, x: -20, scaleX: 0.8, y: '-50%' }}
            transition={{ duration: 0.2, delay: 0.1 }}
            className="fixed top-1/2 left-24 z-30 bg-gray-800 border border-gray-700 rounded-lg shadow-lg overflow-hidden"
          >
            <div className="w-80 max-h-96 flex flex-col">
              {/* Panel Header */}
              <div className="p-3 border-b border-gray-700">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    <Zap size={16} className="text-white" />
                    <h3 className="text-white font-medium text-sm">Engines</h3>
                  </div>
                </div>

                {/* Search within panel */}
                <div className="relative">
                  <Search className="absolute left-2 top-1/2 transform -translate-y-1/2 text-gray-400 w-3 h-3" />
                  <input
                    type="text"
                    placeholder="Search engines..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="w-full pl-7 pr-2 py-1 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400 text-xs focus:outline-none focus:border-gray-500"
                  />
                </div>
              </div>

              {/* Engines List */}
              <div className="flex-1 overflow-y-auto p-2">
                <AnimatePresence mode="wait">
                  {filteredEngines.length > 0 ? (
                    <motion.div
                      key="engines"
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      className="space-y-1"
                    >
                      {filteredEngines.map((engine) => (
                        <EngineItem
                          key={engine.id}
                          engine={engine}
                          onDragStart={onDragStart}
                        />
                      ))}
                    </motion.div>
                  ) : (
                    <motion.div
                      key="empty"
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      className="flex flex-col items-center justify-center h-32 text-center p-4"
                    >
                      <Zap className="w-8 h-8 text-gray-600 mb-2" />
                      <p className="text-gray-400 text-sm">
                        {searchTerm ? 'No matches found' : 'No engines'}
                      </p>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
};

export type { EngineItem };
