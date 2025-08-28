import React, { useState, useMemo, useRef, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Search, Database, Server, Box, X, Heart, Monitor, Package
} from 'lucide-react';
import { getComponentIcon } from '../ui/CustomIcons';
import { useMouseZone, useClickOutside } from '../../hooks/useGlobalMouse';

interface ComponentItem {
  id: string;
  name: string;
  type: string;
  description: string;
  category: string;
  tags: string[];
  isFavorite?: boolean;
  isNew?: boolean;
  isTrending?: boolean;
  difficulty?: 'beginner' | 'intermediate' | 'advanced';
  popularity?: number;
}

const componentLibrary: ComponentItem[] = [
  // Frontend Components
  {
    id: 'client',
    name: 'Web Client',
    type: 'client',
    description: 'User interface for web applications',
    category: 'frontend',
    tags: ['browser', 'ui', 'frontend'],
    difficulty: 'beginner',
    popularity: 5,
    isFavorite: true
  },
  {
    id: 'mobile',
    name: 'Mobile App',
    type: 'mobile',
    description: 'Native or hybrid mobile application',
    category: 'frontend',
    tags: ['mobile', 'app', 'ios', 'android'],
    difficulty: 'intermediate',
    popularity: 4,
    isNew: true
  },
  
  // Backend Components
  {
    id: 'webserver',
    name: 'Web Server',
    type: 'webserver',
    description: 'HTTP server handling web requests',
    category: 'backend',
    tags: ['server', 'http', 'api'],
    difficulty: 'intermediate',
    popularity: 5,
    isTrending: true
  },
  {
    id: 'microservice',
    name: 'Microservice',
    type: 'microservice',
    description: 'Independent deployable service',
    category: 'backend',
    tags: ['microservice', 'api', 'service'],
    difficulty: 'advanced',
    popularity: 4
  },
  
  // Data Components
  {
    id: 'database',
    name: 'Database',
    type: 'database',
    description: 'Persistent data storage system',
    category: 'data',
    tags: ['database', 'storage', 'sql'],
    difficulty: 'intermediate',
    popularity: 5
  },
  {
    id: 'cache',
    name: 'Cache',
    type: 'cache',
    description: 'High-speed data caching layer',
    category: 'data',
    tags: ['cache', 'redis', 'memory'],
    difficulty: 'intermediate',
    popularity: 4
  },
  
  // Infrastructure Components
  {
    id: 'loadbalancer',
    name: 'Load Balancer',
    type: 'loadbalancer',
    description: 'Distributes traffic across servers',
    category: 'infrastructure',
    tags: ['loadbalancer', 'traffic', 'scaling'],
    difficulty: 'advanced',
    popularity: 4
  },
  {
    id: 'cdn',
    name: 'CDN',
    type: 'cdn',
    description: 'Content delivery network',
    category: 'infrastructure',
    tags: ['cdn', 'content', 'delivery'],
    difficulty: 'intermediate',
    popularity: 3
  }
];

const categories = [
  { id: 'frontend', name: 'Frontend', icon: Monitor, color: 'text-blue-400', description: 'User interfaces & client apps' },
  { id: 'backend', name: 'Backend', icon: Server, color: 'text-green-400', description: 'APIs & server components' },
  { id: 'data', name: 'Data', icon: Database, color: 'text-purple-400', description: 'Storage & data processing' },
  { id: 'infrastructure', name: 'Infrastructure', icon: Box, color: 'text-orange-400', description: 'Networking & deployment' },
  { id: 'favorites', name: 'Favorites', icon: Heart, color: 'text-red-400', description: 'Your favorite components' },
];

// Memoized ComponentItem to prevent unnecessary re-renders - restored original UI
const ComponentItem: React.FC<{
  component: ComponentItem;
  isFavorited: boolean;
  onToggleFavorite: (id: string) => void;
  onDragStart: (event: React.DragEvent, component: ComponentItem) => void;
}> = React.memo(({ component, isFavorited, onToggleFavorite, onDragStart }) => {
  return (
    <motion.div
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: -10 }}
      whileHover={{ x: 4 }}
      className="group"
    >
      <div
        draggable
        onDragStart={(e) => onDragStart(e, component)}
        className="flex items-center space-x-3 p-3 hover:bg-gray-800 rounded-lg cursor-grab active:cursor-grabbing transition-all duration-200"
      >
        {/* Icon */}
        <div className="w-10 h-10 bg-gray-900 rounded-lg flex items-center justify-center border border-gray-600 group-hover:border-gray-500 transition-colors flex-shrink-0">
          {getComponentIcon(component.type, {
            className: "text-white",
            size: 20
          })}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center space-x-2">
            <h3 className="text-white font-medium text-sm truncate">{component.name}</h3>
            {component.isNew && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-green-500/20 text-green-400">
                New
              </span>
            )}
            {component.isTrending && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-orange-500/20 text-orange-400">
                Hot
              </span>
            )}
          </div>
          <p className="text-gray-400 text-xs truncate">{component.description}</p>
        </div>

        {/* Actions */}
        <div className="flex items-center space-x-2 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggleFavorite(component.id);
            }}
            className="p-1 hover:bg-gray-700 rounded"
          >
            <Heart
              className={`w-4 h-4 ${
                isFavorited
                  ? 'text-red-500 fill-current'
                  : 'text-gray-400 hover:text-red-400'
              }`}
            />
          </button>
        </div>
      </div>
    </motion.div>
  );
});

interface ModernComponentLibraryProps {
  onDragStart: (event: React.DragEvent, component: ComponentItem) => void;
  isVisible?: boolean;
  onToggleVisibility?: () => void;
}

export const ModernComponentLibrary: React.FC<ModernComponentLibraryProps> = ({
  onDragStart,
  isVisible = false,
  onToggleVisibility
}) => {
  const [selectedCategory, setSelectedCategory] = useState<string | null>('frontend');
  const [searchTerm, setSearchTerm] = useState('');
  const [favorites, setFavorites] = useState(new Set(['client', 'webserver', 'database']));
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

  useMouseZone(
    { left: 0, right: 100 }, // Show when mouse is within 100px of left edge
    handleMouseEnter,
    handleMouseLeave
  );

  // Memoized function to get components for selected category
  const getComponentsForCategory = useCallback((categoryId: string) => {
    if (categoryId === 'favorites') {
      return componentLibrary.filter(component => favorites.has(component.id));
    }
    return componentLibrary.filter(component => component.category === categoryId);
  }, [favorites]);

  const filteredComponents = useMemo(() => {
    if (!selectedCategory) return [];

    const categoryComponents = getComponentsForCategory(selectedCategory);

    if (!searchTerm) return categoryComponents;

    return categoryComponents.filter(component =>
      component.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      component.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      component.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()))
    );
  }, [selectedCategory, searchTerm, getComponentsForCategory]);

  // Close panel when clicking outside using global click tracking
  useClickOutside(
    panelRef,
    () => {
      if (isVisible) {
        onToggleVisibility?.();
      }
    },
    isVisible // Only listen when panel is visible
  );

  const toggleFavorite = useCallback((componentId: string) => {
    setFavorites(prev => {
      const newFavorites = new Set(prev);
      if (newFavorites.has(componentId)) {
        newFavorites.delete(componentId);
      } else {
        newFavorites.add(componentId);
      }
      return newFavorites;
    });
  }, []);



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
          /* Collapsed State - Just the Package Button */
          <motion.button
            ref={triggerRef}
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={onToggleVisibility}
            className="w-14 h-14 flex items-center justify-center text-white hover:bg-gray-700 transition-all duration-200"
            title="Show component library"
          >
            <Package className="w-5 h-5" />
          </motion.button>
        ) : (
          /* Expanded State - Vertical Stack of Categories */
          <div className="p-2 flex flex-col items-center space-y-2">
            {/* Category Icons - Vertical stack */}
            {categories.map((category) => {
              const Icon = category.icon;
              const isSelected = selectedCategory === category.id;

              return (
                <motion.button
                  key={category.id}
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.8 }}
                  transition={{ delay: 0.1 }}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setSelectedCategory(category.id)}
                  className={`w-10 h-10 flex items-center justify-center rounded-md transition-all duration-200 ${
                    isSelected
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title={category.description}
                >
                  <Icon className="w-4 h-4" />
                </motion.button>
              );
            })}

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
              title="Search components"
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
              title="Close component library"
            >
              <X className="w-4 h-4" />
            </motion.button>
          </div>
        )}
      </motion.div>
        )}
      </AnimatePresence>

      {/* Components Panel - Slides out from the toolbar */}
      <AnimatePresence>
        {isVisible && selectedCategory && (
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
                    {(() => {
                      const category = categories.find(c => c.id === selectedCategory);
                      const Icon = category?.icon || Box;
                      return <Icon size={16} className={category?.color || 'text-gray-400'} />;
                    })()}
                    <h3 className="text-white font-medium text-sm">
                      {categories.find(c => c.id === selectedCategory)?.name}
                    </h3>
                  </div>
                </div>

                {/* Search within panel */}
                <div className="relative">
                  <Search className="absolute left-2 top-1/2 transform -translate-y-1/2 text-gray-400 w-3 h-3" />
                  <input
                    type="text"
                    placeholder="Search components..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="w-full pl-7 pr-2 py-1 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400 text-xs focus:outline-none focus:border-gray-500"
                  />
                </div>
              </div>

              {/* Components List */}
              <div className="flex-1 overflow-y-auto p-2">
                <AnimatePresence mode="wait">
                  {filteredComponents.length > 0 ? (
                    <motion.div
                      key="components"
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      className="space-y-1"
                    >
                      {filteredComponents.map((component) => (
                        <ComponentItem
                          key={component.id}
                          component={component}
                          isFavorited={favorites.has(component.id)}
                          onToggleFavorite={toggleFavorite}
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
                      <Box className="w-8 h-8 text-gray-600 mb-2" />
                      <p className="text-gray-400 text-sm">
                        {searchTerm ? 'No matches found' : 'No components'}
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
