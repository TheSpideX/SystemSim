import React, { useState, useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Search, Star, Clock, ChevronDown, ChevronRight, Heart, Grid3X3,
  Filter, SortAsc, Zap, TrendingUp, Users, Layers, Box,
  Database, Server, Globe, Shield, MessageSquare, BarChart3,
  Sparkles, Target, BookOpen, Lightbulb
} from 'lucide-react';
import { getComponentIcon } from '../ui/CustomIcons';

interface ComponentItem {
  id: string;
  name: string;
  type: string;
  description: string;
  category: string;
  tags: string[];
  isFavorite?: boolean;
  lastUsed?: Date;
  // Enhanced metadata
  difficulty?: 'beginner' | 'intermediate' | 'advanced';
  popularity?: number; // 1-5 stars
  performance?: {
    cpu: 'low' | 'medium' | 'high';
    memory: 'low' | 'medium' | 'high';
    latency: 'low' | 'medium' | 'high';
  };
  useCases?: string[];
  connections?: string[]; // Common components this connects to
  pricing?: 'free' | 'low' | 'medium' | 'high';
  isNew?: boolean;
  isTrending?: boolean;
}

const componentLibrary: ComponentItem[] = [
  // Client Components
  {
    id: 'client',
    name: 'Client',
    type: 'client',
    description: 'Web browser or mobile app client',
    category: 'client',
    tags: ['browser', 'mobile', 'frontend', 'user'],
    isFavorite: true,
    lastUsed: new Date(Date.now() - 1000 * 60 * 30),
    difficulty: 'beginner',
    popularity: 5,
    performance: { cpu: 'low', memory: 'low', latency: 'low' },
    useCases: ['User Interface', 'Frontend', 'User Experience'],
    connections: ['web-server', 'load-balancer', 'cdn'],
    pricing: 'free',
  },
  {
    id: 'mobile',
    name: 'Mobile App',
    type: 'mobile',
    description: 'Native mobile application',
    category: 'client',
    tags: ['mobile', 'ios', 'android', 'app'],
    difficulty: 'intermediate',
    popularity: 4,
    performance: { cpu: 'medium', memory: 'medium', latency: 'low' },
    useCases: ['Mobile Interface', 'Native Apps', 'Push Notifications'],
    connections: ['api-gateway', 'microservice', 'push-service'],
    pricing: 'free',
    isTrending: true,
  },

  // Compute Components
  {
    id: 'web-server',
    name: 'Web Server',
    type: 'webserver',
    description: 'HTTP/HTTPS web server (NGINX, Apache)',
    category: 'compute',
    tags: ['http', 'nginx', 'apache', 'frontend'],
    isFavorite: true,
    lastUsed: new Date(Date.now() - 1000 * 60 * 15),
    difficulty: 'beginner',
    popularity: 5,
    performance: { cpu: 'medium', memory: 'low', latency: 'low' },
    useCases: ['Static Content', 'Reverse Proxy', 'SSL Termination'],
    connections: ['client', 'microservice', 'load-balancer'],
    pricing: 'free',
  },
  {
    id: 'microservice',
    name: 'Microservice',
    type: 'microservice',
    description: 'Business logic processing service',
    category: 'compute',
    tags: ['api', 'business-logic', 'microservice'],
    lastUsed: new Date(Date.now() - 1000 * 60 * 60),
    difficulty: 'intermediate',
    popularity: 5,
    performance: { cpu: 'medium', memory: 'medium', latency: 'medium' },
    useCases: ['Business Logic', 'API Endpoints', 'Data Processing'],
    connections: ['database', 'cache', 'message-queue'],
    pricing: 'medium',
    isTrending: true,
  },
  {
    id: 'load-balancer',
    name: 'Load Balancer',
    type: 'loadbalancer',
    description: 'Distribute traffic across multiple servers',
    category: 'compute',
    tags: ['traffic', 'distribution', 'scaling'],
    isFavorite: true,
    lastUsed: new Date(Date.now() - 1000 * 60 * 45),
    difficulty: 'intermediate',
    popularity: 4,
    performance: { cpu: 'low', memory: 'low', latency: 'low' },
    useCases: ['Traffic Distribution', 'High Availability', 'Auto Scaling'],
    connections: ['web-server', 'microservice', 'health-check'],
    pricing: 'low',
  },

  // Storage Components
  {
    id: 'database',
    name: 'Database',
    type: 'database',
    description: 'Relational database (PostgreSQL, MySQL)',
    category: 'storage',
    tags: ['sql', 'postgresql', 'mysql', 'relational'],
    lastUsed: new Date(Date.now() - 1000 * 60 * 20),
  },
  {
    id: 'cache',
    name: 'Cache',
    type: 'cache',
    description: 'In-memory cache (Redis, Memcached)',
    category: 'storage',
    tags: ['redis', 'memcached', 'memory', 'fast'],
    isFavorite: true,
  },
  {
    id: 'object-storage',
    name: 'Object Storage',
    type: 'storage',
    description: 'Scalable object storage (S3, GCS)',
    category: 'storage',
    tags: ['s3', 'blob', 'files', 'scalable'],
  },

  // Network Components
  {
    id: 'cdn',
    name: 'CDN',
    type: 'cdn',
    description: 'Content Delivery Network',
    category: 'network',
    tags: ['cdn', 'global', 'caching', 'performance'],
  },
  {
    id: 'api-gateway',
    name: 'API Gateway',
    type: 'gateway',
    description: 'API management and routing',
    category: 'network',
    tags: ['api', 'routing', 'auth', 'rate-limiting'],
  },

  // Processing Components
  {
    id: 'message-queue',
    name: 'Message Queue',
    type: 'queue',
    description: 'Async message processing (Kafka, RabbitMQ)',
    category: 'processing',
    tags: ['kafka', 'rabbitmq', 'async', 'messaging'],
  },

  // Security Components
  {
    id: 'firewall',
    name: 'Firewall',
    type: 'security',
    description: 'Network security and filtering',
    category: 'security',
    tags: ['security', 'filtering', 'protection'],
  },
];

const categories = [
  { id: 'client', name: 'Client', color: 'text-blue-500', bgColor: 'bg-blue-50 dark:bg-blue-900/20' },
  { id: 'compute', name: 'Compute', color: 'text-green-500', bgColor: 'bg-green-50 dark:bg-green-900/20' },
  { id: 'storage', name: 'Storage', color: 'text-purple-500', bgColor: 'bg-purple-50 dark:bg-purple-900/20' },
  { id: 'network', name: 'Network', color: 'text-orange-500', bgColor: 'bg-orange-50 dark:bg-orange-900/20' },
  { id: 'processing', name: 'Processing', color: 'text-yellow-500', bgColor: 'bg-yellow-50 dark:bg-yellow-900/20' },
  { id: 'security', name: 'Security', color: 'text-red-500', bgColor: 'bg-red-50 dark:bg-red-900/20' },
];

export const ImprovedComponentLibrary: React.FC = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
    new Set(['favorites', 'recent'])
  );
  const [favorites, setFavorites] = useState<Set<string>>(
    new Set(componentLibrary.filter(c => c.isFavorite).map(c => c.id))
  );
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');

  const toggleCategory = (categoryId: string) => {
    const newExpanded = new Set(expandedCategories);
    if (newExpanded.has(categoryId)) {
      newExpanded.delete(categoryId);
    } else {
      newExpanded.add(categoryId);
    }
    setExpandedCategories(newExpanded);
  };

  const toggleFavorite = (componentId: string) => {
    const newFavorites = new Set(favorites);
    if (newFavorites.has(componentId)) {
      newFavorites.delete(componentId);
    } else {
      newFavorites.add(componentId);
    }
    setFavorites(newFavorites);
  };

  const onDragStart = (event: React.DragEvent, component: ComponentItem) => {
    event.dataTransfer.setData('application/reactflow', JSON.stringify({
      type: 'system',
      data: {
        label: component.name,
        type: component.type,
        status: 'healthy',
        rps: Math.floor(Math.random() * 5000) + 1000,
        latency: Math.floor(Math.random() * 50) + 10,
        cpu: Math.floor(Math.random() * 60) + 20,
        memory: Math.floor(Math.random() * 60) + 20,
      }
    }));
    event.dataTransfer.effectAllowed = 'move';
  };

  const filteredComponents = componentLibrary.filter(component => {
    if (searchTerm === '') return true;
    return (
      component.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      component.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      component.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()))
    );
  });

  const favoriteComponents = filteredComponents.filter(c => favorites.has(c.id));
  const recentComponents = filteredComponents
    .filter(c => c.lastUsed)
    .sort((a, b) => (b.lastUsed?.getTime() || 0) - (a.lastUsed?.getTime() || 0))
    .slice(0, 5);

  const getComponentsByCategory = (categoryId: string) => {
    return filteredComponents.filter(c => c.category === categoryId);
  };

  const ComponentCard: React.FC<{ 
    component: ComponentItem; 
    showFavorite?: boolean;
    compact?: boolean;
  }> = ({ component, showFavorite = true, compact = false }) => {
    const category = categories.find(c => c.id === component.category);
    
    return (
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.98 }}
        draggable
        onDragStart={(e) => onDragStart(e, component)}
        className={`cursor-grab active:cursor-grabbing transition-all duration-200 bg-gray-800 hover:bg-gray-700 border border-gray-700 hover:border-gray-600 rounded shadow-sm hover:shadow-md ${
          compact ? 'p-2' : 'p-3'
        }`}
        title={`${component.name} - ${component.description}`}
      >
        <div className={`flex items-center ${compact ? 'space-x-2' : 'space-x-3'}`}>
          {/* Drawing Board Style Icon */}
          <div
            className={`${compact ? 'w-8 h-8' : 'w-10 h-10'} bg-gray-900 rounded flex items-center justify-center flex-shrink-0 border border-gray-600`}
            title={`${component.type} component icon`}
          >
            {getComponentIcon(component.type, {
              className: "text-white",
              size: compact ? 16 : 20
            })}
          </div>
          
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <h3 className={`text-white font-medium ${compact ? 'text-xs' : 'text-sm'}`}>
                  {component.name}
                </h3>
                {/* Status Badges */}
                <div className="flex items-center space-x-1">
                  {component.isNew && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                      <Sparkles className="w-3 h-3 mr-1" />
                      New
                    </span>
                  )}
                  {component.isTrending && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200">
                      <TrendingUp className="w-3 h-3 mr-1" />
                      Trending
                    </span>
                  )}
                  {component.difficulty && (
                    <span className={`inline-flex items-center px-1.5 py-0.5 rounded-full text-xs font-medium ${
                      component.difficulty === 'beginner'
                        ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                        : component.difficulty === 'intermediate'
                        ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                        : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                    }`}>
                      {component.difficulty === 'beginner' && <BookOpen className="w-3 h-3 mr-1" />}
                      {component.difficulty === 'intermediate' && <Target className="w-3 h-3 mr-1" />}
                      {component.difficulty === 'advanced' && <Zap className="w-3 h-3 mr-1" />}
                      {component.difficulty}
                    </span>
                  )}
                </div>
              </div>
              {showFavorite && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    toggleFavorite(component.id);
                  }}
                  className="opacity-0 group-hover:opacity-100 transition-opacity duration-200"
                >
                  <Heart 
                    className={`w-3 h-3 ${
                      favorites.has(component.id) 
                        ? 'text-red-500 fill-current' 
                        : 'text-gray-400 hover:text-red-500'
                    }`} 
                  />
                </button>
              )}
            </div>
            {!compact && (
              <>
                <p className="text-gray-600 dark:text-gray-400 text-xs leading-relaxed mb-2">
                  {component.description}
                </p>

                {/* Performance & Popularity Indicators */}
                <div className="flex items-center justify-between mb-2">
                  {/* Popularity Stars */}
                  {component.popularity && (
                    <div className="flex items-center space-x-1">
                      {[...Array(5)].map((_, i) => (
                        <Star
                          key={i}
                          className={`w-3 h-3 ${
                            i < component.popularity!
                              ? 'text-yellow-400 fill-current'
                              : 'text-gray-300 dark:text-gray-600'
                          }`}
                        />
                      ))}
                      <span className="text-xs text-gray-500 dark:text-gray-400 ml-1">
                        ({component.popularity}/5)
                      </span>
                    </div>
                  )}

                  {/* Performance Indicators */}
                  {component.performance && (
                    <div className="flex items-center space-x-2">
                      <div className="flex items-center space-x-1">
                        <div className={`w-2 h-2 rounded-full ${
                          component.performance.cpu === 'low' ? 'bg-green-400' :
                          component.performance.cpu === 'medium' ? 'bg-yellow-400' : 'bg-red-400'
                        }`} />
                        <span className="text-xs text-gray-500 dark:text-gray-400">CPU</span>
                      </div>
                      <div className="flex items-center space-x-1">
                        <div className={`w-2 h-2 rounded-full ${
                          component.performance.memory === 'low' ? 'bg-green-400' :
                          component.performance.memory === 'medium' ? 'bg-yellow-400' : 'bg-red-400'
                        }`} />
                        <span className="text-xs text-gray-500 dark:text-gray-400">MEM</span>
                      </div>
                      <div className="flex items-center space-x-1">
                        <div className={`w-2 h-2 rounded-full ${
                          component.performance.latency === 'low' ? 'bg-green-400' :
                          component.performance.latency === 'medium' ? 'bg-yellow-400' : 'bg-red-400'
                        }`} />
                        <span className="text-xs text-gray-500 dark:text-gray-400">LAT</span>
                      </div>
                    </div>
                  )}
                </div>
              </>
            )}

            {/* Tags */}
            <div className="flex flex-wrap gap-1">
              {component.tags.slice(0, compact ? 1 : 2).map((tag) => (
                <span
                  key={tag}
                  className={`px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 ${compact ? 'text-xs' : 'text-xs'} rounded`}
                >
                  {tag}
                </span>
              ))}
              {component.tags.length > (compact ? 1 : 2) && (
                <span className={`px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 ${compact ? 'text-xs' : 'text-xs'} rounded`}>
                  +{component.tags.length - (compact ? 1 : 2)}
                </span>
              )}
            </div>
          </div>
        </div>
      </motion.div>
    );
  };

  const CategorySection: React.FC<{ 
    id: string; 
    title: string; 
    icon: React.ComponentType<any>; 
    components: ComponentItem[];
    color?: string;
  }> = ({ id, title, icon: Icon, components, color = 'text-gray-400' }) => (
    <div className="mb-4">
      <button
        onClick={() => toggleCategory(id)}
        className="w-full flex items-center justify-between p-2 hover:bg-gray-50 dark:hover:bg-gray-800 rounded-lg transition-colors duration-200"
      >
        <div className="flex items-center space-x-2">
          <Icon className={`w-4 h-4 ${color}`} />
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{title}</span>
          <span className="text-xs text-gray-500 dark:text-gray-500">({components.length})</span>
        </div>
        {expandedCategories.has(id) ? (
          <ChevronDown className="w-4 h-4 text-gray-400" />
        ) : (
          <ChevronRight className="w-4 h-4 text-gray-400" />
        )}
      </button>
      
      <AnimatePresence>
        {expandedCategories.has(id) && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2 }}
            className="mt-2 pl-6"
          >
            {viewMode === 'grid' ? (
              <div className="grid grid-cols-1 gap-2">
                {components.map((component) => (
                  <ComponentCard 
                    key={component.id} 
                    component={component} 
                    compact={true}
                  />
                ))}
              </div>
            ) : (
              <div className="space-y-2">
                {components.map((component) => (
                  <ComponentCard 
                    key={component.id} 
                    component={component}
                  />
                ))}
              </div>
            )}
            {components.length === 0 && (
              <p className="text-xs text-gray-500 dark:text-gray-500 py-2">No components found</p>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );

  return (
    <div className="h-full flex flex-col bg-gray-800 border-r border-gray-700">
      {/* Enhanced Header */}
      <div className="p-4 border-b border-gray-700 bg-gray-700">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-white font-semibold flex items-center">
            <Box className="w-5 h-5 mr-2 text-white" />
            Components
          </h2>
          <div className="flex items-center space-x-2">
            <button
              onClick={() => setViewMode(viewMode === 'list' ? 'grid' : 'list')}
              className="p-2 text-white hover:bg-gray-700 transition-colors rounded"
              title={`Switch to ${viewMode === 'list' ? 'grid' : 'list'} view - Change how components are displayed`}
            >
              <Grid3X3 className="w-4 h-4" />
            </button>
            <button
              className="p-2 text-white hover:bg-gray-700 transition-colors rounded"
              title="Filter components - Show only specific types of components"
            >
              <Filter className="w-4 h-4" />
            </button>
            <button
              className="p-2 text-white hover:bg-gray-700 transition-colors rounded"
              title="Sort components - Change the order of components"
            >
              <SortAsc className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Enhanced Search */}
        <div className="relative mb-3">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search components, tags, or use cases..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400 focus:outline-none focus:border-gray-500 focus:bg-gray-600 transition-all duration-200"
            title="Search for components by name, description, tags, or use cases"
          />
        </div>

        {/* Quick Filters */}
        <div className="flex flex-wrap gap-2">
          <button
            className="inline-flex items-center px-2.5 py-1 rounded text-xs font-medium bg-gray-700 text-white border border-gray-600 hover:bg-gray-600 transition-colors"
            title="Show only new components"
          >
            <Sparkles className="w-3 h-3 mr-1" />
            New
          </button>
          <button
            className="inline-flex items-center px-2.5 py-1 rounded text-xs font-medium bg-gray-700 text-white border border-gray-600 hover:bg-gray-600 transition-colors"
            title="Show trending components"
          >
            <TrendingUp className="w-3 h-3 mr-1" />
            Trending
          </button>
          <button
            className="inline-flex items-center px-2.5 py-1 rounded text-xs font-medium bg-gray-700 text-white border border-gray-600 hover:bg-gray-600 transition-colors"
            title="Show beginner-friendly components"
          >
            <BookOpen className="w-3 h-3 mr-1" />
            Beginner
          </button>
          <button
            className="inline-flex items-center px-2.5 py-1 rounded text-xs font-medium bg-gray-700 text-white border border-gray-600 hover:bg-gray-600 transition-colors"
            title="Show most popular components"
          >
            <Star className="w-3 h-3 mr-1" />
            Popular
          </button>
        </div>
      </div>

      {/* Component List */}
      <div className="flex-1 overflow-y-auto p-4 space-y-1">
        {/* Favorites Section */}
        {favoriteComponents.length > 0 && (
          <CategorySection
            id="favorites"
            title="Favorites"
            icon={Star}
            components={favoriteComponents}
            color="text-yellow-500"
          />
        )}

        {/* Recently Used Section */}
        {recentComponents.length > 0 && (
          <CategorySection
            id="recent"
            title="Recently Used"
            icon={Clock}
            components={recentComponents}
            color="text-blue-500"
          />
        )}

        {/* Category Sections */}
        {categories.map((category) => (
          <CategorySection
            key={category.id}
            id={category.id}
            title={category.name}
            icon={() => getComponentIcon(getComponentsByCategory(category.id)[0]?.type || 'webserver', { size: 16 })}
            components={getComponentsByCategory(category.id)}
            color={category.color}
          />
        ))}

        {filteredComponents.length === 0 && (
          <div className="text-center py-8">
            <div className="w-16 h-16 bg-gray-100 dark:bg-gray-900 rounded-full flex items-center justify-center mx-auto mb-3">
              <Search className="w-6 h-6 text-gray-400" />
            </div>
            <p className="text-gray-500 text-sm">No components found</p>
            <p className="text-gray-400 text-xs mt-1">Try adjusting your search</p>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-gray-200 dark:border-gray-800">
        <div className="text-xs text-gray-500 text-center">
          Drag components to canvas
        </div>
      </div>
    </div>
  );
};