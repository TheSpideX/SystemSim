import React, { useCallback, useState, useRef, useEffect, useMemo } from 'react';
import {
  ReactFlow,
  addEdge,
  useNodesState,
  useEdgesState,
  Background,
  BackgroundVariant,
  Panel,
  MiniMap,
  Controls,
  useReactFlow,
  ReactFlowProvider,
} from '@xyflow/react';
import type { Node, Edge, Connection } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Plus, Zap, Activity, MousePointer, Hand, Link,
  ZoomIn, ZoomOut, Maximize, RotateCcw, Save,
  Play, Pause, Square, Settings, Share2, Box,
  Copy, Trash2, Users, Eye, Grid3X3, Layers,
  Target, Move, Crosshair, Navigation, Compass,
  Magnet, Ruler, Palette, Lightbulb, TrendingUp
} from 'lucide-react';
import toast from 'react-hot-toast';

import { ImprovedSystemNode, type SystemNodeData } from './ImprovedSystemNode';
import { ProfessionalEdge } from './ProfessionalEdge';
import { NodeDetailsPanel } from './NodeDetailsPanel';
import { ContextMenu, getNodeContextMenu, getCanvasContextMenu } from '../ui/ContextMenu';
import { LassoSelection } from '../ui/LassoSelection';
import { useKeyboardShortcuts, getCommonShortcuts } from '../../hooks/useKeyboardShortcuts';
import { useUIStore } from '../../store/uiStore';
import { useThemeStore } from '../../store/themeStore';

// Custom node and edge types - we'll create this dynamically to pass the onDoubleClick handler

const edgeTypes = {
  professional: ProfessionalEdge,
};

// Clean initial setup
const initialNodes: Node[] = [
  {
    id: 'client',
    type: 'system',
    position: { x: 400, y: 50 },
    data: {
      label: 'Client',
      type: 'client',
      status: 'healthy',
      rps: 25000,
      latency: 2,
      cpu: 15,
      memory: 30,
    },
  },
  {
    id: 'load-balancer',
    type: 'system',
    position: { x: 400, y: 200 },
    data: {
      label: 'Load Balancer',
      type: 'loadbalancer',
      status: 'healthy',
      rps: 25000,
      latency: 5,
      cpu: 45,
      memory: 60,
    },
  },
  {
    id: 'web-server-1',
    type: 'system',
    position: { x: 250, y: 350 },
    data: {
      label: 'Web Server 1',
      type: 'webserver',
      status: 'healthy',
      rps: 12500,
      latency: 12,
      cpu: 65,
      memory: 70,
    },
  },
  {
    id: 'web-server-2',
    type: 'system',
    position: { x: 550, y: 350 },
    data: {
      label: 'Web Server 2',
      type: 'webserver',
      status: 'healthy',
      rps: 12500,
      latency: 15,
      cpu: 58,
      memory: 65,
    },
  },
  {
    id: 'database',
    type: 'system',
    position: { x: 400, y: 500 },
    data: {
      label: 'PostgreSQL',
      type: 'database',
      status: 'warning',
      rps: 8000,
      latency: 45,
      cpu: 85,
      memory: 90,
    },
  },
  {
    id: 'cache',
    type: 'system',
    position: { x: 150, y: 500 },
    data: {
      label: 'Redis Cache',
      type: 'cache',
      status: 'healthy',
      rps: 15000,
      latency: 2,
      cpu: 25,
      memory: 40,
    },
  },
];

// Clean professional edges
const initialEdges: Edge[] = [
  {
    id: 'client-lb',
    source: 'client',
    target: 'load-balancer',
    type: 'professional',
    animated: false,
    data: { throughput: 25000, label: 'HTTPS' },
  },
  {
    id: 'lb-ws1',
    source: 'load-balancer',
    target: 'web-server-1',
    type: 'professional',
    animated: false,
    data: { throughput: 12500, label: 'HTTP' },
  },
  {
    id: 'lb-ws2',
    source: 'load-balancer',
    target: 'web-server-2',
    type: 'professional',
    animated: false,
    data: { throughput: 12500, label: 'HTTP' },
  },
  {
    id: 'ws1-db',
    source: 'web-server-1',
    target: 'database',
    type: 'professional',
    animated: false,
    data: { throughput: 4000, label: 'SQL' },
  },
  {
    id: 'ws2-db',
    source: 'web-server-2',
    target: 'database',
    type: 'professional',
    animated: false,
    data: { throughput: 4000, label: 'SQL' },
  },
  {
    id: 'ws1-cache',
    source: 'web-server-1',
    target: 'cache',
    type: 'professional',
    animated: false,
    data: { throughput: 8000, label: 'Redis' },
  },
  {
    id: 'ws2-cache',
    source: 'web-server-2',
    target: 'cache',
    type: 'professional',
    animated: false,
    data: { throughput: 7000, label: 'Redis' },
  },
];

interface EnhancedSystemDesignCanvasProps {
  isSimulationRunning: boolean;
  selectedComponent: string | null;
  onSelectComponent: (id: string | null) => void;
  onToggleSimulation: () => void;
}

const EnhancedSystemDesignCanvasInner: React.FC<EnhancedSystemDesignCanvasProps> = ({
  isSimulationRunning,
  selectedComponent,
  onSelectComponent,
  onToggleSimulation,
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [currentTool, setCurrentTool] = useState<'select' | 'pan' | 'connect' | 'lasso'>('select');
  const [zoomLevel, setZoomLevel] = useState(100);
  const [contextMenu, setContextMenu] = useState<{
    isOpen: boolean;
    position: { x: number; y: number };
    type: 'node' | 'canvas';
    nodeId?: string;
  }>({ isOpen: false, position: { x: 0, y: 0 }, type: 'canvas' });

  const [clipboard, setClipboard] = useState<Node[]>([]);
  const [selectedNodes, setSelectedNodes] = useState<string[]>([]);
  const [selectedNodeData, setSelectedNodeData] = useState<SystemNodeData | null>(null);

  // Enhanced canvas features
  const [showGrid, setShowGrid] = useState(true);
  const [showMiniMap, setShowMiniMap] = useState(true);
  const [showMetricsOverlay, setShowMetricsOverlay] = useState(false);
  const [showConnectionLabels, setShowConnectionLabels] = useState(true);
  const [snapToGrid, setSnapToGrid] = useState(true);
  const [showPerformanceHeatmap, setShowPerformanceHeatmap] = useState(false);
  const [canvasMode, setCanvasMode] = useState<'design' | 'simulation' | 'analysis'>('design');

  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const { project, zoomIn, zoomOut, fitView, getZoom } = useReactFlow();
  const { theme } = useThemeStore();
  const { setSaveStatus } = useUIStore();

  // Handle connection creation
  const onConnect = useCallback(
    (params: Connection) => {
      const newEdge = {
        ...params,
        type: 'professional',
        animated: isSimulationRunning,
        data: { 
          throughput: Math.floor(Math.random() * 10000) + 1000,
          label: 'API'
        },
      };
      setEdges((eds) => addEdge(newEdge, eds));
      setSaveStatus('unsaved');
      toast.success('Connection created');
    },
    [setEdges, isSimulationRunning, setSaveStatus]
  );

  // Handle node selection
  const onNodeClick = useCallback(
    (event: React.MouseEvent, node: Node) => {
      event.stopPropagation();
      
      if (event.ctrlKey || event.metaKey) {
        // Multi-select
        setSelectedNodes(prev => 
          prev.includes(node.id) 
            ? prev.filter(id => id !== node.id)
            : [...prev, node.id]
        );
      } else {
        setSelectedNodes([node.id]);
        onSelectComponent(node.id);
      }
    },
    [onSelectComponent]
  );

  // Handle canvas click (deselect)
  const onPaneClick = useCallback(() => {
    setSelectedNodes([]);
    onSelectComponent(null);
    setContextMenu({ isOpen: false, position: { x: 0, y: 0 }, type: 'canvas' });
  }, [onSelectComponent]);

  // Handle node double-click for details panel
  const handleNodeDoubleClick = useCallback((event: React.MouseEvent, node: Node) => {
    event.stopPropagation();
    setSelectedNodeData(node.data as SystemNodeData);
  }, []);

  // Close details panel
  const closeDetailsPanel = useCallback(() => {
    setSelectedNodeData(null);
  }, []);

  // Handle node double-click from component
  const handleNodeComponentDoubleClick = useCallback((nodeData: SystemNodeData) => {
    setSelectedNodeData(nodeData);
  }, []);

  // Create node types with memoized double-click handler
  const nodeTypes = useMemo(() => ({
    system: (props: any) => (
      <ImprovedSystemNode
        {...props}
        onDoubleClick={handleNodeComponentDoubleClick}
      />
    ),
  }), [handleNodeComponentDoubleClick]);

  // Memoized styles to prevent re-renders
  const connectionLineStyle = useMemo(() => ({
    stroke: theme === 'dark' ? '#fff' : '#000',
    strokeWidth: 2,
  }), [theme]);

  const defaultEdgeOptions = useMemo(() => ({
    style: { stroke: theme === 'dark' ? '#fff' : '#000', strokeWidth: 2 },
    type: 'professional',
  }), [theme]);

  // Handle right-click context menu
  const onNodeContextMenu = useCallback((event: React.MouseEvent, node: Node) => {
    event.preventDefault();
    setContextMenu({
      isOpen: true,
      position: { x: event.clientX, y: event.clientY },
      type: 'node',
      nodeId: node.id,
    });
  }, []);

  const onPaneContextMenu = useCallback((event: React.MouseEvent) => {
    event.preventDefault();
    setContextMenu({
      isOpen: true,
      position: { x: event.clientX, y: event.clientY },
      type: 'canvas',
    });
  }, []);

  // Handle drag over for drop functionality
  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  // Handle drop to add new components
  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const reactFlowBounds = reactFlowWrapper.current?.getBoundingClientRect();
      if (!reactFlowBounds) return;

      const type = event.dataTransfer.getData('application/reactflow');
      if (!type) return;

      const componentData = JSON.parse(type);
      const position = project({
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      });

      const newNode = {
        id: `${componentData.data.type}-${Date.now()}`,
        type: componentData.type,
        position,
        data: componentData.data,
      };

      setNodes((nds) => nds.concat(newNode));
      setSaveStatus('unsaved');
      toast.success(`${componentData.data.label} added to canvas`);
    },
    [project, setNodes, setSaveStatus]
  );

  // Lasso selection handler
  const handleLassoSelection = useCallback((boundingBox: { x: number; y: number; width: number; height: number }) => {
    const selectedNodeIds = nodes
      .filter(node => {
        const nodeX = node.position.x;
        const nodeY = node.position.y;
        return (
          nodeX >= boundingBox.x &&
          nodeX <= boundingBox.x + boundingBox.width &&
          nodeY >= boundingBox.y &&
          nodeY <= boundingBox.y + boundingBox.height
        );
      })
      .map(node => node.id);
    
    setSelectedNodes(selectedNodeIds);
    if (selectedNodeIds.length > 0) {
      toast.success(`Selected ${selectedNodeIds.length} components`);
    }
  }, [nodes]);

  // Context menu actions
  const duplicateNode = useCallback((nodeId: string) => {
    const node = nodes.find(n => n.id === nodeId);
    if (!node) return;

    const newNode = {
      ...node,
      id: `${node.id}-copy-${Date.now()}`,
      position: { x: node.position.x + 50, y: node.position.y + 50 },
    };

    setNodes(nds => [...nds, newNode]);
    setSaveStatus('unsaved');
    toast.success('Component duplicated');
  }, [nodes, setNodes, setSaveStatus]);

  const deleteNode = useCallback((nodeId: string) => {
    setNodes(nds => nds.filter(n => n.id !== nodeId));
    setEdges(eds => eds.filter(e => e.source !== nodeId && e.target !== nodeId));
    setSaveStatus('unsaved');
    toast.success('Component deleted');
  }, [setNodes, setEdges, setSaveStatus]);

  const copyNode = useCallback((nodeId: string) => {
    const node = nodes.find(n => n.id === nodeId);
    if (node) {
      setClipboard([node]);
      toast.success('Component copied');
    }
  }, [nodes]);

  const pasteNodes = useCallback(() => {
    if (clipboard.length === 0) return;

    const newNodes = clipboard.map(node => ({
      ...node,
      id: `${node.id}-paste-${Date.now()}`,
      position: { x: node.position.x + 100, y: node.position.y + 100 },
    }));

    setNodes(nds => [...nds, ...newNodes]);
    setSaveStatus('unsaved');
    toast.success(`Pasted ${newNodes.length} component(s)`);
  }, [clipboard, setNodes, setSaveStatus]);

  const groupNodes = useCallback(() => {
    if (selectedNodes.length < 2) {
      toast.error('Select at least 2 components to group');
      return;
    }
    toast.success(`Grouped ${selectedNodes.length} components`);
  }, [selectedNodes]);

  const selectAllNodes = useCallback(() => {
    setSelectedNodes(nodes.map(n => n.id));
    toast.success(`Selected all ${nodes.length} components`);
  }, [nodes]);

  const duplicateSelected = useCallback(() => {
    selectedNodes.forEach(nodeId => duplicateNode(nodeId));
  }, [selectedNodes, duplicateNode]);

  const deleteSelected = useCallback(() => {
    if (selectedNodes.length === 0) return;
    
    setNodes(nds => nds.filter(n => !selectedNodes.includes(n.id)));
    setEdges(eds => eds.filter(e => !selectedNodes.includes(e.source) && !selectedNodes.includes(e.target)));
    setSelectedNodes([]);
    setSaveStatus('unsaved');
    toast.success(`Deleted ${selectedNodes.length} component(s)`);
  }, [selectedNodes, setNodes, setEdges, setSaveStatus]);

  const saveProject = useCallback(async () => {
    setSaveStatus('saving');
    // Simulate save
    await new Promise(resolve => setTimeout(resolve, 1000));
    setSaveStatus('saved');
    toast.success('Project saved successfully');
  }, [setSaveStatus]);

  // Keyboard shortcuts
  useKeyboardShortcuts({
    shortcuts: getCommonShortcuts({
      copy: () => selectedNodes.length > 0 && copyNode(selectedNodes[0]),
      paste: pasteNodes,
      delete: deleteSelected,
      group: groupNodes,
      undo: () => toast.info('Undo not implemented yet'),
      redo: () => toast.info('Redo not implemented yet'),
      save: saveProject,
      selectAll: selectAllNodes,
      duplicate: duplicateSelected,
      focusMode: () => toast.info('Focus mode toggled'),
    }),
    enabled: !contextMenu.isOpen,
  });

  // Update edge animations based on simulation state
  useEffect(() => {
    setEdges((eds) =>
      eds.map((edge) => ({
        ...edge,
        animated: isSimulationRunning,
      }))
    );
  }, [isSimulationRunning, setEdges]);

  // Update zoom level display
  useEffect(() => {
    const updateZoom = () => {
      setZoomLevel(Math.round(getZoom() * 100));
    };
    
    const interval = setInterval(updateZoom, 100);
    return () => clearInterval(interval);
  }, [getZoom]);

  // Simulate real-time metrics updates
  useEffect(() => {
    if (!isSimulationRunning) return;

    const interval = setInterval(() => {
      setNodes((nds) =>
        nds.map((node) => ({
          ...node,
          data: {
            ...node.data,
            rps: Math.max(0, node.data.rps + Math.floor(Math.random() * 400 - 200)),
            latency: Math.max(1, node.data.latency + Math.floor(Math.random() * 10 - 5)),
            cpu: Math.max(0, Math.min(100, node.data.cpu + Math.floor(Math.random() * 10 - 5))),
            memory: Math.max(0, Math.min(100, node.data.memory + Math.floor(Math.random() * 8 - 4))),
          },
        }))
      );
    }, 1000);

    return () => clearInterval(interval);
  }, [isSimulationRunning, setNodes]);

  return (
    <div className="w-full h-full relative bg-white dark:bg-black" ref={reactFlowWrapper}>
      <LassoSelection
        isActive={currentTool === 'lasso'}
        onSelection={handleLassoSelection}
      >
        <ReactFlow
          nodes={nodes.map(node => ({
            ...node,
            selected: selectedNodes.includes(node.id),
          }))}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          onNodeDoubleClick={handleNodeDoubleClick}
          onPaneClick={onPaneClick}
          onNodeContextMenu={onNodeContextMenu}
          onPaneContextMenu={onPaneContextMenu}
          onDrop={onDrop}
          onDragOver={onDragOver}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          fitView
          attributionPosition="bottom-left"
          className={`${theme === 'dark' ? 'bg-black' : 'bg-gray-50'}`}
          connectionLineStyle={connectionLineStyle}
          defaultEdgeOptions={defaultEdgeOptions}
          panOnScroll
          selectionOnDrag={currentTool === 'select'}
          panOnDrag={currentTool === 'pan'}
          nodesDraggable={currentTool === 'select'}
          nodesConnectable={currentTool === 'connect'}
          elementsSelectable={currentTool === 'select'}
          multiSelectionKeyCode="Control"
          onlyRenderVisibleElements={false}
        >
          {/* Enhanced Background */}
          {showGrid && (
            <Background
              variant={snapToGrid ? BackgroundVariant.Dots : BackgroundVariant.Lines}
              gap={snapToGrid ? 20 : 40}
              size={snapToGrid ? 1 : 0.5}
              color={theme === 'dark' ? '#444' : '#ddd'}
            />
          )}

          {/* Performance Heatmap Overlay */}
          {showPerformanceHeatmap && (
            <div className="absolute inset-0 pointer-events-none z-10">
              <div className="absolute top-4 right-4 bg-white dark:bg-gray-900 rounded-lg p-3 shadow-lg border border-gray-200 dark:border-gray-700">
                <div className="text-xs font-medium text-gray-700 dark:text-gray-300 mb-2">Performance Heat</div>
                <div className="flex items-center space-x-2 text-xs">
                  <div className="flex items-center space-x-1">
                    <div className="w-3 h-3 bg-green-500 rounded"></div>
                    <span className="text-gray-600 dark:text-gray-400">Low</span>
                  </div>
                  <div className="flex items-center space-x-1">
                    <div className="w-3 h-3 bg-yellow-500 rounded"></div>
                    <span className="text-gray-600 dark:text-gray-400">Med</span>
                  </div>
                  <div className="flex items-center space-x-1">
                    <div className="w-3 h-3 bg-red-500 rounded"></div>
                    <span className="text-gray-600 dark:text-gray-400">High</span>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Enhanced Toolbar */}
          <Panel position="top-center" className="m-4">
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              className="bg-gray-800 border border-gray-700 rounded-lg shadow-lg px-2 py-2 flex items-center space-x-1"
            >
              {/* Tool Selection */}
              <div className="flex items-center space-x-1 pr-2 border-r border-gray-600">
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setCurrentTool('select')}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    currentTool === 'select'
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Select Tool - Click and drag to select components (V)"
                >
                  <MousePointer className="w-4 h-4" />
                </motion.button>
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setCurrentTool('pan')}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    currentTool === 'pan'
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Pan Tool - Click and drag to move around the canvas (H)"
                >
                  <Hand className="w-4 h-4" />
                </motion.button>
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setCurrentTool('connect')}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    currentTool === 'connect'
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Connect Tool - Click between components to create connections (C)"
                >
                  <Link className="w-4 h-4" />
                </motion.button>
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setCurrentTool('lasso')}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    currentTool === 'lasso'
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Lasso Select - Draw to select multiple components (L)"
                >
                  <Box className="w-4 h-4" />
                </motion.button>
              </div>

              {/* Zoom Controls */}
              <div className="flex items-center space-x-1 px-2 border-r border-gray-600">
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => zoomOut()}
                  className="p-2 text-white hover:bg-gray-700 rounded transition-all duration-200"
                  title="Zoom Out - Decrease canvas zoom level (-)"
                >
                  <ZoomOut className="w-4 h-4" />
                </motion.button>
                <div className="px-2 py-1 text-sm font-medium text-white min-w-[50px] text-center bg-gray-700 rounded">
                  {zoomLevel}%
                </div>
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => zoomIn()}
                  className="p-2 text-white hover:bg-gray-700 rounded transition-all duration-200"
                  title="Zoom In - Increase canvas zoom level (+)"
                >
                  <ZoomIn className="w-4 h-4" />
                </motion.button>
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => fitView()}
                  className="p-2 text-white hover:bg-gray-700 rounded transition-all duration-200"
                  title="Fit to Screen - Show all components in view (F)"
                >
                  <Maximize className="w-4 h-4" />
                </motion.button>
              </div>

              {/* Simulation Controls */}
              <div className="flex items-center space-x-1 px-2 border-r border-gray-600">
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={onToggleSimulation}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    isSimulationRunning
                      ? 'bg-green-600 text-white border border-green-500 shadow-sm'
                      : 'bg-gray-600 text-white border border-gray-500 shadow-sm hover:bg-gray-700'
                  }`}
                  title={isSimulationRunning ? 'Pause Simulation - Stop the live simulation (Space)' : 'Start Simulation - Begin live system simulation (Space)'}
                >
                  {isSimulationRunning ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                </motion.button>
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  className="p-2 text-white hover:bg-gray-700 rounded-md transition-all duration-200"
                  title="Reset Simulation - Reset all metrics to initial state (R)"
                >
                  <RotateCcw className="w-4 h-4" />
                </motion.button>
              </div>

              {/* Canvas Mode Toggle */}
              <div className="flex items-center space-x-1 px-2 border-r border-gray-600">
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setCanvasMode('design')}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    canvasMode === 'design'
                      ? 'bg-blue-600 text-white border border-blue-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Design Mode - Create and arrange system components"
                >
                  <Compass className="w-4 h-4" />
                </motion.button>

                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setCanvasMode('simulation')}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    canvasMode === 'simulation'
                      ? 'bg-purple-600 text-white border border-purple-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Simulation Mode - Run live system simulation"
                >
                  <Activity className="w-4 h-4" />
                </motion.button>

                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setCanvasMode('analysis')}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    canvasMode === 'analysis'
                      ? 'bg-orange-600 text-white border border-orange-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Analysis Mode - View performance metrics and insights"
                >
                  <TrendingUp className="w-4 h-4" />
                </motion.button>
              </div>

              {/* View Options */}
              <div className="flex items-center space-x-1 px-2 border-r border-gray-600">
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setShowGrid(!showGrid)}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    showGrid
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Toggle Grid - Show/hide alignment grid on canvas"
                >
                  <Grid3X3 className="w-4 h-4" />
                </motion.button>

                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setSnapToGrid(!snapToGrid)}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    snapToGrid
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Snap to Grid - Components automatically align to grid"
                >
                  <Magnet className="w-4 h-4" />
                </motion.button>

                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setShowPerformanceHeatmap(!showPerformanceHeatmap)}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    showPerformanceHeatmap
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Performance Heatmap - Color-code components by performance"
                >
                  <Palette className="w-4 h-4" />
                </motion.button>

                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => setShowMiniMap(!showMiniMap)}
                  className={`p-2 rounded-md transition-all duration-200 ${
                    showMiniMap
                      ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                      : 'text-white hover:bg-gray-700'
                  }`}
                  title="Toggle Mini Map - Show/hide navigation overview"
                >
                  <Navigation className="w-4 h-4" />
                </motion.button>
              </div>

              {/* Actions */}
              <div className="flex items-center space-x-1 pl-2">
                <motion.button
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={saveProject}
                  className="p-2 text-white hover:bg-gray-700 rounded transition-all duration-200"
                  title="Save Project - Save current system design (Ctrl+S)"
                >
                  <Save className="w-4 h-4" />
                </motion.button>
              </div>
            </motion.div>
          </Panel>

          {/* Status Panel */}
          <Panel position="bottom-left" className="m-4">
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              className="bg-gray-800 border border-gray-700 rounded-lg shadow-lg px-4 py-3"
            >
              <div className="flex items-center space-x-6 text-sm">
                <div className="flex items-center space-x-2" title="Simulation status">
                  <div className={`w-2 h-2 rounded-full ${
                    isSimulationRunning ? 'bg-white animate-pulse' : 'bg-gray-500'
                  }`} />
                  <span className="text-white font-medium">
                    {isSimulationRunning ? 'Live' : 'Paused'}
                  </span>
                </div>
                <div className="text-gray-300" title="Total requests per second across all components">
                  <span className="font-medium text-white">
                    {nodes.reduce((sum, node) => sum + (node.data?.rps || 0), 0).toLocaleString()}
                  </span> RPS
                </div>
                <div className="text-gray-300" title="Average response latency across all components">
                  <span className="font-medium text-white">
                    {Math.round(nodes.reduce((sum, node) => sum + (node.data?.latency || 0), 0) / nodes.length)}ms
                  </span> Latency
                </div>
                <div className="text-gray-300" title="Total number of components in the system">
                  <span className="font-medium text-white">{nodes.length}</span> Components
                </div>
                {selectedNodes.length > 0 && (
                  <div className="text-gray-300" title="Number of currently selected components">
                    <span className="font-medium text-white">{selectedNodes.length}</span> Selected
                  </div>
                )}
              </div>
            </motion.div>
          </Panel>

          {/* Enhanced MiniMap */}
          {showMiniMap && (
            <MiniMap
              nodeColor={(node) => {
                if (selectedNodes.includes(node.id)) return '#3b82f6';
                if (node.selected) return '#3b82f6';
                if (showPerformanceHeatmap) {
                  // Color based on performance (mock data for now)
                  const performance = node.data?.performance || Math.random();
                  if (performance > 0.8) return '#ef4444'; // High load - red
                  if (performance > 0.5) return '#f59e0b'; // Medium load - yellow
                  return '#10b981'; // Low load - green
                }
                return theme === 'dark' ? '#555' : '#999';
              }}
              nodeStrokeColor={theme === 'dark' ? '#000' : '#fff'}
              nodeBorderRadius={4}
              className={`!border !rounded-lg !shadow-lg ${
                theme === 'dark'
                  ? '!bg-gray-900 !border-gray-700'
                  : '!bg-white !border-gray-200'
              }`}
              maskColor={theme === 'dark' ? 'rgba(0, 0, 0, 0.8)' : 'rgba(255, 255, 255, 0.8)'}
              pannable
              zoomable
              position="bottom-right"
            />
          )}

          {/* Enhanced Controls */}
          <Controls
            className={`!border !rounded-lg !shadow-lg ${
              theme === 'dark'
                ? '!bg-gray-900 !border-gray-700'
                : '!bg-white !border-gray-200'
            }`}
            showZoom={false}
            showFitView={false}
            showInteractive={false}
          />
        </ReactFlow>
      </LassoSelection>

      {/* Context Menu */}
      <ContextMenu
        isOpen={contextMenu.isOpen}
        position={contextMenu.position}
        items={
          contextMenu.type === 'node' && contextMenu.nodeId
            ? getNodeContextMenu(
                contextMenu.nodeId,
                duplicateNode,
                deleteNode,
                copyNode,
                (id) => {
                  onSelectComponent(id);
                  toast.info('Viewing metrics');
                },
                (id) => {
                  onSelectComponent(id);
                  toast.info('Configuring component');
                }
              )
            : getCanvasContextMenu(
                pasteNodes,
                () => toast.info('Add component from library'),
                () => fitView(),
                clipboard.length > 0
              )
        }
        onClose={() => setContextMenu({ isOpen: false, position: { x: 0, y: 0 }, type: 'canvas' })}
      />

      {/* Node Details Panel */}
      <NodeDetailsPanel
        nodeData={selectedNodeData}
        onClose={closeDetailsPanel}
      />
    </div>
  );
};

export const EnhancedSystemDesignCanvas: React.FC<EnhancedSystemDesignCanvasProps> = (props) => {
  return (
    <ReactFlowProvider>
      <EnhancedSystemDesignCanvasInner {...props} />
    </ReactFlowProvider>
  );
};