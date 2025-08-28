import React, { useCallback, useState, useRef, useEffect } from 'react';
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
  Handle,
  Position,
} from '@xyflow/react';
import type { Node, Edge, Connection } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { motion } from 'framer-motion';
import {
  Play, Pause, RotateCcw, Save, MousePointer, Hand, Link,
  ZoomIn, ZoomOut, Maximize, Grid3X3, Eye, EyeOff, Cpu, MemoryStick, HardDrive, Network,
  Compass, Activity, TrendingUp, Layers, Copy, Trash2, CopyPlus, Lasso,
  BarChart, Settings, Plus, Clipboard
} from 'lucide-react';
import toast from 'react-hot-toast';
import { useThemeStore } from '../../store/themeStore';
import { useUIStore } from '../../store/uiStore';
import { ContextMenu, getNodeContextMenu, getCanvasContextMenu } from '../ui/ContextMenu';
import { LassoSelection } from '../ui/LassoSelection';
import { useKeyboardShortcuts, getCommonShortcuts } from '../../hooks/useKeyboardShortcuts';
import { ProfessionalEdge } from './ProfessionalEdge';

// Engine node component - styled exactly like simulation nodes
const EngineNode = ({ data, selected }: { data: any; selected: boolean }) => {
  // Map engine type to icon component
  const getEngineIcon = (type: string) => {
    switch (type) {
      case 'cpu': return Cpu;
      case 'memory': return MemoryStick;
      case 'storage': return HardDrive;
      case 'network': return Network;
      default: return Cpu;
    }
  };

  const Icon = getEngineIcon(data.type);

  return (
    <motion.div
      initial={{ scale: 0.9, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      whileHover={{ scale: 1.05 }}
      className={`relative group cursor-pointer transition-all duration-200 ${
        selected ? 'z-10' : 'z-0'
      }`}
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

        {/* Main Icon - Black and White like simulation nodes */}
        <div className="w-full h-full flex items-center justify-center">
          <Icon className="text-white drop-shadow-sm" size={28} />
        </div>
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
            {data.type} Engine
          </div>
          <div className="text-xs text-gray-500 mt-1">
            {data.description}
          </div>
        </div>
        {/* Arrow pointing up to the node */}
        <div className="absolute -top-1 left-1/2 transform -translate-x-1/2 w-2 h-2 bg-gray-900 border-l border-t border-gray-600 rotate-45" />
      </motion.div>
    </motion.div>
  );
};

// Edge types
const edgeTypes = {
  professional: ProfessionalEdge,
};

const nodeTypes = {
  engine: EngineNode,
};

interface ComponentDesignCanvasProps {
  isSimulationRunning: boolean;
  selectedComponent: string | null;
  onSelectComponent: (id: string | null) => void;
  onToggleSimulation: () => void;
}

const ComponentDesignCanvasInner: React.FC<ComponentDesignCanvasProps> = ({
  isSimulationRunning,
  selectedComponent,
  onSelectComponent,
  onToggleSimulation,
}) => {
  const { theme } = useThemeStore();
  const { setSaveStatus } = useUIStore();
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const { screenToFlowPosition, fitView } = useReactFlow();

  // Canvas state
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [currentTool, setCurrentTool] = useState<'select' | 'pan' | 'connect' | 'lasso'>('select');
  const [zoomLevel, setZoomLevel] = useState(100);

  // Context menu state
  const [contextMenu, setContextMenu] = useState<{
    isOpen: boolean;
    position: { x: number; y: number };
    type: 'node' | 'canvas';
    nodeId?: string;
  }>({ isOpen: false, position: { x: 0, y: 0 }, type: 'canvas' });

  // Selection and clipboard
  const [clipboard, setClipboard] = useState<Node[]>([]);
  const [selectedNodes, setSelectedNodes] = useState<string[]>([]);

  // Enhanced canvas features
  const [showGrid, setShowGrid] = useState(true);
  const [showMiniMap, setShowMiniMap] = useState(true);
  const [showMetricsOverlay, setShowMetricsOverlay] = useState(false);
  const [showConnectionLabels, setShowConnectionLabels] = useState(true);
  const [snapToGrid, setSnapToGrid] = useState(true);
  const [showPerformanceHeatmap, setShowPerformanceHeatmap] = useState(false);
  const [canvasMode, setCanvasMode] = useState<'design' | 'simulation' | 'analysis'>('design');

  // Connection line style
  const connectionLineStyle = {
    strokeWidth: 2,
    stroke: '#6b7280',
  };

  const defaultEdgeOptions = {
    type: 'professional',
    style: { strokeWidth: 2, stroke: '#6b7280' },
    animated: false,
  };

  // Handle connections between engines
  const onConnect = useCallback(
    (params: Connection) => {
      const edge = {
        ...params,
        id: `${params.source}-${params.target}`,
        type: 'professional',
        animated: isSimulationRunning,
        data: {
          throughput: Math.floor(Math.random() * 1000) + 100,
          label: 'Data Flow',
          status: 'healthy' as const
        },
      };
      setEdges((eds) => addEdge(edge, eds));
      setSaveStatus('unsaved');
      toast.success('Engines connected');
    },
    [setEdges, setSaveStatus, isSimulationRunning]
  );

  // Handle node selection
  const onNodeClick = useCallback(
    (event: React.MouseEvent, node: Node) => {
      onSelectComponent(node.id);
    },
    [onSelectComponent]
  );

  // Handle canvas click (deselect)
  const onPaneClick = useCallback(() => {
    onSelectComponent(null);
    setSelectedNodes([]);
  }, [onSelectComponent]);

  // Context menu handlers
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

  // Node operations
  const duplicateNode = useCallback((nodeId: string) => {
    const nodeToDuplicate = nodes.find(n => n.id === nodeId);
    if (!nodeToDuplicate) return;

    const newNode = {
      ...nodeToDuplicate,
      id: `${nodeToDuplicate.data.type}-${Date.now()}`,
      position: {
        x: nodeToDuplicate.position.x + 50,
        y: nodeToDuplicate.position.y + 50,
      },
    };

    setNodes(nds => [...nds, newNode]);
    setSaveStatus('unsaved');
    toast.success(`${nodeToDuplicate.data.label} duplicated`);
  }, [nodes, setNodes, setSaveStatus]);

  const deleteNode = useCallback((nodeId: string) => {
    setNodes(nds => nds.filter(n => n.id !== nodeId));
    setEdges(eds => eds.filter(e => e.source !== nodeId && e.target !== nodeId));
    setSaveStatus('unsaved');
    toast.success('Engine deleted');
  }, [setNodes, setEdges, setSaveStatus]);

  const copyNode = useCallback((nodeId: string) => {
    const nodeToCopy = nodes.find(n => n.id === nodeId);
    if (nodeToCopy) {
      setClipboard([nodeToCopy]);
      toast.success('Engine copied to clipboard');
    }
  }, [nodes]);

  // Handle drop to add new engines
  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const reactFlowBounds = reactFlowWrapper.current?.getBoundingClientRect();
      if (!reactFlowBounds) return;

      const type = event.dataTransfer.getData('application/reactflow');
      if (!type) return;

      const engineData = JSON.parse(type);

      const position = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      const newNode = {
        id: `${engineData.data.type}-${Date.now()}`,
        type: 'engine',
        position,
        data: engineData.data,
      };

      setNodes((nds) => nds.concat(newNode));
      setSaveStatus('unsaved');
      toast.success(`${engineData.data.label} added to component`);
    },
    [screenToFlowPosition, setNodes, setSaveStatus]
  );

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  // Additional node operations
  const pasteNodes = useCallback(() => {
    if (clipboard.length === 0) return;

    const newNodes = clipboard.map(node => ({
      ...node,
      id: `${node.data.type}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      position: {
        x: node.position.x + 100,
        y: node.position.y + 100,
      },
    }));

    setNodes(nds => [...nds, ...newNodes]);
    setSaveStatus('unsaved');
    toast.success(`Pasted ${newNodes.length} engine(s)`);
  }, [clipboard, setNodes, setSaveStatus]);

  const selectAllNodes = useCallback(() => {
    const allNodeIds = nodes.map(n => n.id);
    setSelectedNodes(allNodeIds);
    toast.info(`Selected ${allNodeIds.length} engines`);
  }, [nodes]);

  const duplicateSelected = useCallback(() => {
    if (selectedNodes.length === 0) return;

    const nodesToDuplicate = nodes.filter(n => selectedNodes.includes(n.id));
    const newNodes = nodesToDuplicate.map(node => ({
      ...node,
      id: `${node.data.type}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      position: {
        x: node.position.x + 50,
        y: node.position.y + 50,
      },
    }));

    setNodes(nds => [...nds, ...newNodes]);
    setSaveStatus('unsaved');
    toast.success(`Duplicated ${newNodes.length} engine(s)`);
  }, [selectedNodes, nodes, setNodes, setSaveStatus]);

  const deleteSelected = useCallback(() => {
    if (selectedNodes.length === 0) return;

    setNodes(nds => nds.filter(n => !selectedNodes.includes(n.id)));
    setEdges(eds => eds.filter(e => !selectedNodes.includes(e.source) && !selectedNodes.includes(e.target)));
    setSelectedNodes([]);
    setSaveStatus('unsaved');
    toast.success(`Deleted ${selectedNodes.length} engine(s)`);
  }, [selectedNodes, setNodes, setEdges, setSaveStatus]);

  const saveProject = useCallback(async () => {
    setSaveStatus('saving');
    // Simulate save
    await new Promise(resolve => setTimeout(resolve, 1000));
    setSaveStatus('saved');
    toast.success('Component saved successfully');
  }, [setSaveStatus]);

  // Keyboard shortcuts
  useKeyboardShortcuts({
    shortcuts: getCommonShortcuts({
      copy: () => selectedNodes.length > 0 && copyNode(selectedNodes[0]),
      paste: pasteNodes,
      delete: deleteSelected,
      group: () => toast.info('Grouping not implemented yet'),
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

  // Handle node selection changes
  useEffect(() => {
    const selectedNodeIds = nodes.filter(node => node.selected).map(node => node.id);
    setSelectedNodes(selectedNodeIds);
  }, [nodes]);

  return (
    <div className="h-full w-full relative" ref={reactFlowWrapper}>
      <LassoSelection
        isActive={currentTool === 'lasso'}
        onSelection={(boundingBox) => {
          // Handle lasso selection
          console.log('Lasso selection:', boundingBox);
          toast.info('Lasso selection completed');
        }}
      >
        <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
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
        snapToGrid={snapToGrid}
        snapGrid={[20, 20]}
        multiSelectionKeyCode="Shift"
        deleteKeyCode="Delete"
      >
        {/* Background */}
        {showGrid && (
          <Background
            variant={BackgroundVariant.Dots}
            gap={20}
            size={1}
            color={theme === 'dark' ? '#444' : '#ddd'}
          />
        )}

        {/* Toolbar */}
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
                title="Select Tool (V)"
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
                title="Pan Tool (H)"
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
                title="Connect Tool (C)"
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
                title="Lasso Selection Tool (L)"
              >
                <Lasso className="w-4 h-4" />
              </motion.button>
            </div>

            {/* View Controls */}
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
                title="Toggle Grid (G)"
              >
                <Grid3X3 className="w-4 h-4" />
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
                title="Toggle MiniMap (M)"
              >
                {showMiniMap ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
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
                title="Toggle Snap to Grid"
              >
                <Maximize className="w-4 h-4" />
              </motion.button>

              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                onClick={() => setShowConnectionLabels(!showConnectionLabels)}
                className={`p-2 rounded-md transition-all duration-200 ${
                  showConnectionLabels
                    ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                    : 'text-white hover:bg-gray-700'
                }`}
                title="Toggle Connection Labels"
              >
                <Layers className="w-4 h-4" />
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
                title="Design Mode - Create and arrange engine components"
              >
                <Compass className="w-4 h-4" />
              </motion.button>

              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                onClick={() => setCanvasMode('simulation')}
                className={`p-2 rounded-md transition-all duration-200 ${
                  canvasMode === 'simulation'
                    ? 'bg-green-600 text-white border border-green-500 shadow-sm'
                    : 'text-white hover:bg-gray-700'
                }`}
                title="Simulation Mode - Test component behavior"
              >
                <Activity className="w-4 h-4" />
              </motion.button>

              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                onClick={() => setCanvasMode('analysis')}
                className={`p-2 rounded-md transition-all duration-200 ${
                  canvasMode === 'analysis'
                    ? 'bg-purple-600 text-white border border-purple-500 shadow-sm'
                    : 'text-white hover:bg-gray-700'
                }`}
                title="Analysis Mode - View performance metrics"
              >
                <TrendingUp className="w-4 h-4" />
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
                title={isSimulationRunning ? 'Pause Component Test (Space)' : 'Start Component Test (Space)'}
              >
                {isSimulationRunning ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
              </motion.button>

              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                onClick={() => {
                  // Reset simulation
                  setEdges(eds => eds.map(edge => ({ ...edge, animated: false })));
                  toast.success('Component simulation reset');
                }}
                className="p-2 text-white hover:bg-gray-700 rounded-md transition-all duration-200"
                title="Reset Simulation - Reset all metrics to initial state (R)"
              >
                <RotateCcw className="w-4 h-4" />
              </motion.button>
            </div>

            {/* Save Controls */}
            <div className="flex items-center space-x-1 px-2">
              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                onClick={saveProject}
                className="p-2 text-white hover:bg-gray-700 rounded-md transition-all duration-200"
                title="Save Component (Ctrl+S)"
              >
                <Save className="w-4 h-4" />
              </motion.button>
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
                  toast.info('Viewing engine metrics');
                },
                (id) => {
                  onSelectComponent(id);
                  toast.info('Configuring engine');
                }
              )
            : getCanvasContextMenu(
                pasteNodes,
                () => toast.info('Add engine from library'),
                () => fitView(),
                clipboard.length > 0
              )
        }
        onClose={() => setContextMenu({ isOpen: false, position: { x: 0, y: 0 }, type: 'canvas' })}
      />
    </div>
  );
};

export const ComponentDesignCanvas: React.FC<ComponentDesignCanvasProps> = (props) => {
  return (
    <ReactFlowProvider>
      <ComponentDesignCanvasInner {...props} />
    </ReactFlowProvider>
  );
};
