import React, { useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Eye, EyeOff, Plus, Settings, Layers, X } from 'lucide-react';

import { Header } from './ui/Header';
import { EnhancedSystemDesignCanvas } from './canvas/EnhancedSystemDesignCanvas';
import { ComponentDesignCanvas } from './canvas/ComponentDesignCanvas';
import { ModernComponentLibrary } from './canvas/ModernComponentLibrary';
import { EngineLibrary } from './canvas/EngineLibrary';
import { FloatingFlowGraphWindow } from './canvas/FloatingFlowGraphWindow';
import { FlowNodeLibrary } from './canvas/FlowNodeLibrary';
import { EnhancedPropertiesPanel } from './canvas/EnhancedPropertiesPanel';
import { EnhancedMetricsPanel } from './canvas/EnhancedMetricsPanel';
import { ResizablePanel } from './ui/ResizablePanel';
import { useUIStore } from '../store/uiStore';
import { useKeyboardShortcuts } from '../hooks/useKeyboardShortcuts';
import toast from 'react-hot-toast';

interface EnhancedSimulationPageProps {
  onBack: () => void;
}

export const EnhancedSimulationPage: React.FC<EnhancedSimulationPageProps> = ({ onBack }) => {
  const [isSimulationRunning, setIsSimulationRunning] = useState(false);
  const [selectedComponent, setSelectedComponent] = useState<string | null>(null);
  const [isComponentLibraryVisible, setIsComponentLibraryVisible] = useState(false);
  const [isEngineLibraryVisible, setIsEngineLibraryVisible] = useState(false);
  const [isFlowNodeLibraryVisible, setIsFlowNodeLibraryVisible] = useState(false);
  const [isFlowGraphVisible, setIsFlowGraphVisible] = useState(false);
  
  const {
    isLeftPanelCollapsed,
    isRightPanelCollapsed,
    leftPanelWidth,
    rightPanelWidth,
    isFocusMode,
    designMode,
    showProperties,
    showMetrics,
    toggleLeftPanel,
    toggleRightPanel,
    setLeftPanelWidth,
    setRightPanelWidth,
    setShowProperties,
    setShowMetrics,
  } = useUIStore();

  const toggleSimulation = useCallback(() => {
    setIsSimulationRunning(!isSimulationRunning);
    toast.success(isSimulationRunning ? 'Simulation paused' : 'Simulation started');
  }, [isSimulationRunning]);

  // Keyboard shortcuts for panel toggles
  useKeyboardShortcuts({
    shortcuts: [
      {
        key: '1',
        ctrlKey: true,
        action: toggleLeftPanel,
        description: 'Toggle left panel',
      },
      {
        key: '2',
        ctrlKey: true,
        action: toggleRightPanel,
        description: 'Toggle right panel',
      },
      {
        key: ' ',
        action: (e) => {
          e?.preventDefault();
          toggleSimulation();
        },
        description: 'Toggle simulation',
      },
    ],
  });

  return (
    <div className="h-screen bg-white dark:bg-black flex flex-col overflow-hidden">
      {/* Enhanced Header */}
      <Header onBack={onBack} />

      {/* Dual Canvas Layout - Split Screen */}
      <div className="flex-1 flex overflow-hidden">
        {designMode === 'system-design' ? (
          /* System Design Mode - Single Canvas */
          <div className="flex-1 relative">
            <motion.div
              key={designMode}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3, ease: "easeInOut" }}
              className="h-full"
            >
              <EnhancedSystemDesignCanvas
                isSimulationRunning={isSimulationRunning}
                selectedComponent={selectedComponent}
                onSelectComponent={setSelectedComponent}
                onToggleSimulation={toggleSimulation}
              />
            </motion.div>
          </div>
        ) : (
          /* Component Design Mode - Clean Canvas with Right Overlay */
          <div className="flex-1 relative">
            {/* Main Component Design Canvas - Full Height */}
            <ComponentDesignCanvas
              isSimulationRunning={isSimulationRunning}
              selectedComponent={selectedComponent}
              onSelectComponent={setSelectedComponent}
              onToggleSimulation={toggleSimulation}
            />

            {/* Decision Graph Toggle Button - Floating */}
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => setIsFlowGraphVisible(!isFlowGraphVisible)}
              className={`absolute top-4 right-4 p-3 rounded-lg shadow-lg transition-all duration-200 z-40 ${
                isFlowGraphVisible
                  ? 'bg-green-600 text-white border border-green-500'
                  : 'bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white border border-gray-200 dark:border-gray-700 hover:shadow-xl'
              }`}
              title="Toggle Decision Graph"
            >
              <Layers className="w-5 h-5" />
            </motion.button>
          </div>
        )}
      </div>


      {/* Floating Component Library - Only in System Design Mode */}
      {designMode === 'system-design' && (
        <ModernComponentLibrary
          isVisible={isComponentLibraryVisible}
          onToggleVisibility={() => setIsComponentLibraryVisible(!isComponentLibraryVisible)}
          onDragStart={(event, component) => {
            const nodeData = {
              type: 'system',
              data: {
                label: component.name,
                type: component.type,
                status: 'healthy',
                rps: Math.floor(Math.random() * 5000) + 1000,
                latency: Math.floor(Math.random() * 100) + 10,
                cpu: Math.floor(Math.random() * 80) + 10,
                memory: Math.floor(Math.random() * 70) + 15,
                connections: Math.floor(Math.random() * 50) + 5,
              }
            };
            event.dataTransfer.setData('application/reactflow', JSON.stringify(nodeData));
            event.dataTransfer.effectAllowed = 'move';
          }}
        />
      )}

      {/* Floating Engine Library - Only in Component Design Mode */}
      {designMode === 'component-design' && (
        <EngineLibrary
          isVisible={isEngineLibraryVisible}
          onToggleVisibility={() => setIsEngineLibraryVisible(!isEngineLibraryVisible)}
          onDragStart={(event, engine) => {
            const nodeData = {
              type: 'engine',
              data: {
                label: engine.name,
                type: engine.type,
                iconType: engine.type, // Pass icon type instead of component
                color: engine.color,
                description: engine.description,
              }
            };
            event.dataTransfer.setData('application/reactflow', JSON.stringify(nodeData));
            event.dataTransfer.effectAllowed = 'move';
          }}
        />
      )}

      {/* Floating Flow Node Library - Only in Component Design Mode */}
      {designMode === 'component-design' && (
        <FlowNodeLibrary
          isVisible={isFlowNodeLibraryVisible}
          onToggleVisibility={() => setIsFlowNodeLibraryVisible(!isFlowNodeLibraryVisible)}
          onDragStart={(event, flowNode) => {
            const nodeData = {
              type: flowNode.type,
              label: flowNode.name,
              icon: flowNode.icon,
              color: flowNode.color,
              description: flowNode.description,
            };
            event.dataTransfer.setData('application/reactflow', JSON.stringify(nodeData));
            event.dataTransfer.effectAllowed = 'move';
          }}
        />
      )}

      {/* Floating Flow Graph Window - Only in Component Design Mode */}
      {designMode === 'component-design' && (
        <FloatingFlowGraphWindow
          isVisible={isFlowGraphVisible}
          onToggleVisibility={() => setIsFlowGraphVisible(!isFlowGraphVisible)}
          selectedComponent={selectedComponent}
          isSimulationRunning={isSimulationRunning}
          onToggleSimulation={toggleSimulation}
        />
      )}
    </div>
  );
};