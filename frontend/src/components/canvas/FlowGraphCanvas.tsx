import React, { useCallback, useState, useRef } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  BackgroundVariant,
  useReactFlow,
  ReactFlowProvider,
} from '@xyflow/react';
import type { Connection, Node, Edge } from '@xyflow/react';
import { motion } from 'framer-motion';
import {
  Play, Pause, RotateCcw, Save, MousePointer, Hand, Link,
  ZoomIn, ZoomOut, Grid3X3, Eye, EyeOff, ArrowRight, Database,
  Filter, Shuffle, GitBranch, Zap, Clock, CheckCircle
} from 'lucide-react';
import toast from 'react-hot-toast';
import { useThemeStore } from '../../store/themeStore';

// Flow node types for internal component logic
const flowNodeTypes = [
  { type: 'input', label: 'Input', icon: ArrowRight, color: 'text-green-400' },
  { type: 'process', label: 'Process', icon: Zap, color: 'text-blue-400' },
  { type: 'decision', label: 'Decision', icon: GitBranch, color: 'text-yellow-400' },
  { type: 'data', label: 'Data Store', icon: Database, color: 'text-purple-400' },
  { type: 'filter', label: 'Filter', icon: Filter, color: 'text-orange-400' },
  { type: 'transform', label: 'Transform', icon: Shuffle, color: 'text-pink-400' },
  { type: 'timer', label: 'Timer', icon: Clock, color: 'text-indigo-400' },
  { type: 'output', label: 'Output', icon: CheckCircle, color: 'text-red-400' },
];

// Flow node component
const FlowNode = ({ data, selected }: { data: any; selected: boolean }) => {
  const Icon = data.icon;
  
  return (
    <div
      className={`relative bg-gray-800 border-2 rounded-lg p-3 min-w-[120px] transition-all duration-200 ${
        selected
          ? 'border-blue-500 shadow-lg shadow-blue-500/20'
          : 'border-gray-600 hover:border-gray-500'
      }`}
    >
      {/* Node Header */}
      <div className="flex items-center space-x-2 mb-2">
        <Icon className={`w-4 h-4 ${data.color}`} />
        <span className="text-white text-sm font-medium">{data.label}</span>
      </div>
      
      {/* Node Description */}
      {data.description && (
        <p className="text-gray-400 text-xs">{data.description}</p>
      )}

      {/* Connection Handles */}
      <div className="absolute -left-2 top-1/2 transform -translate-y-1/2 w-4 h-4 bg-blue-500 rounded-full border-2 border-gray-800 opacity-0 group-hover:opacity-100 transition-opacity" />
      <div className="absolute -right-2 top-1/2 transform -translate-y-1/2 w-4 h-4 bg-blue-500 rounded-full border-2 border-gray-800 opacity-0 group-hover:opacity-100 transition-opacity" />
    </div>
  );
};

const nodeTypes = {
  flow: FlowNode,
};

interface FlowGraphCanvasProps {
  selectedComponent: string | null;
  isSimulationRunning: boolean;
  onToggleSimulation: () => void;
}

const FlowGraphCanvasInner: React.FC<FlowGraphCanvasProps> = ({
  selectedComponent,
  isSimulationRunning,
  onToggleSimulation,
}) => {
  const { theme } = useThemeStore();
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const { screenToFlowPosition } = useReactFlow();

  // Canvas state
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [currentTool, setCurrentTool] = useState<'select' | 'pan' | 'connect'>('select');
  const [showGrid, setShowGrid] = useState(true);
  const [showMiniMap, setShowMiniMap] = useState(false);

  // Handle connections
  const onConnect = useCallback(
    (params: Connection) => {
      const edge = {
        ...params,
        type: 'smoothstep',
        style: { stroke: '#6b7280', strokeWidth: 2 },
        animated: isSimulationRunning,
      };
      setEdges((eds) => addEdge(edge, eds));
      toast.success('Flow nodes connected');
    },
    [setEdges, isSimulationRunning]
  );

  // Handle node click
  const onNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
    // Handle node selection
  }, []);

  // Handle canvas click
  const onPaneClick = useCallback(() => {
    // Deselect nodes
  }, []);

  // Handle drop to add new flow nodes
  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const reactFlowBounds = reactFlowWrapper.current?.getBoundingClientRect();
      if (!reactFlowBounds) return;

      const type = event.dataTransfer.getData('application/reactflow');
      if (!type) return;

      const flowData = JSON.parse(type);
      const position = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      const newNode = {
        id: `${flowData.type}-${Date.now()}`,
        type: 'flow',
        position,
        data: flowData,
      };

      setNodes((nds) => nds.concat(newNode));
      toast.success(`${flowData.label} added to flow`);
    },
    [screenToFlowPosition, setNodes]
  );

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  return (
    <div className="h-full w-full flex flex-col bg-gray-900">
      {/* Flow Toolbar */}
      <div className="flex items-center justify-between p-2 bg-gray-800 border-b border-gray-700">
        <div className="flex items-center space-x-2">
          {/* Tool Selection */}
          <div className="flex items-center space-x-1">
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => setCurrentTool('select')}
              className={`p-1.5 rounded-md transition-all duration-200 ${
                currentTool === 'select'
                  ? 'bg-blue-600 text-white'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700'
              }`}
              title="Select Tool"
            >
              <MousePointer className="w-4 h-4" />
            </motion.button>

            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => setCurrentTool('pan')}
              className={`p-1.5 rounded-md transition-all duration-200 ${
                currentTool === 'pan'
                  ? 'bg-blue-600 text-white'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700'
              }`}
              title="Pan Tool"
            >
              <Hand className="w-4 h-4" />
            </motion.button>

            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => setCurrentTool('connect')}
              className={`p-1.5 rounded-md transition-all duration-200 ${
                currentTool === 'connect'
                  ? 'bg-blue-600 text-white'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700'
              }`}
              title="Connect Tool"
            >
              <Link className="w-4 h-4" />
            </motion.button>
          </div>

          {/* View Controls */}
          <div className="flex items-center space-x-1 ml-4">
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => setShowGrid(!showGrid)}
              className={`p-1.5 rounded-md transition-all duration-200 ${
                showGrid
                  ? 'bg-gray-600 text-white'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700'
              }`}
              title="Toggle Grid"
            >
              <Grid3X3 className="w-4 h-4" />
            </motion.button>

            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => setShowMiniMap(!showMiniMap)}
              className={`p-1.5 rounded-md transition-all duration-200 ${
                showMiniMap
                  ? 'bg-gray-600 text-white'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700'
              }`}
              title="Toggle MiniMap"
            >
              {showMiniMap ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
            </motion.button>
          </div>
        </div>

        {/* Simulation Controls */}
        <div className="flex items-center space-x-2">
          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={onToggleSimulation}
            className={`p-1.5 rounded-md transition-all duration-200 ${
              isSimulationRunning
                ? 'bg-green-600 text-white'
                : 'bg-gray-600 text-white hover:bg-gray-700'
            }`}
            title={isSimulationRunning ? 'Pause Flow' : 'Start Flow'}
          >
            {isSimulationRunning ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          </motion.button>

          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={() => toast.success('Flow reset')}
            className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded-md transition-all duration-200"
            title="Reset Flow"
          >
            <RotateCcw className="w-4 h-4" />
          </motion.button>
        </div>
      </div>

      {/* Flow Canvas */}
      <div className="flex-1" ref={reactFlowWrapper}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          onPaneClick={onPaneClick}
          onDrop={onDrop}
          onDragOver={onDragOver}
          nodeTypes={nodeTypes}
          fitView
          className="bg-gray-900"
          panOnScroll
          selectionOnDrag={currentTool === 'select'}
          panOnDrag={currentTool === 'pan'}
          nodesDraggable={currentTool === 'select'}
          nodesConnectable={currentTool === 'connect'}
          elementsSelectable={currentTool === 'select'}
        >
          {/* Background */}
          {showGrid && (
            <Background
              variant={BackgroundVariant.Dots}
              gap={20}
              size={1}
              color={theme === 'dark' ? '#374151' : '#d1d5db'}
            />
          )}

          {/* MiniMap */}
          {showMiniMap && (
            <MiniMap
              nodeColor="#374151"
              nodeStrokeColor="#000"
              nodeBorderRadius={4}
              className="!bg-gray-800 !border-gray-700"
              position="bottom-right"
            />
          )}

          {/* Controls */}
          <Controls
            className="!bg-gray-800 !border-gray-700"
            showZoom={false}
            showFitView={false}
            showInteractive={false}
          />
        </ReactFlow>
      </div>
    </div>
  );
};

export const FlowGraphCanvas: React.FC<FlowGraphCanvasProps> = (props) => {
  return (
    <ReactFlowProvider>
      <FlowGraphCanvasInner {...props} />
    </ReactFlowProvider>
  );
};
