import React, { useState } from 'react';
import { motion } from 'framer-motion';
import {
  Settings, Cpu, MemoryStick, HardDrive, Network, Clock,
  Zap, Activity, Database, Server, Globe, Shield, Layers,
  ChevronDown, ChevronRight, Info, AlertTriangle, CheckCircle,
  Plus, Minus
} from 'lucide-react';

interface EnhancedPropertiesPanelProps {
  selectedComponent: string | null;
  onUpdateComponent: (id: string, updates: any) => void;
}

interface ComponentConfig {
  id: string;
  name: string;
  type: string;
  status: string;
  specs: {
    cpu: number;
    memory: number;
    storage: number;
    network: number;
  };
  performance: {
    maxRps: number;
    avgLatency: number;
    errorRate: number;
  };
  scaling: {
    minInstances: number;
    maxInstances: number;
    autoScaling: boolean;
  };
  monitoring: {
    healthCheck: boolean;
    metrics: boolean;
    alerts: boolean;
  };
}

// Mock component data
const mockComponentData: ComponentConfig = {
  id: 'web-server-1',
  name: 'Web Server 1',
  type: 'webserver',
  status: 'healthy',
  specs: {
    cpu: 4,
    memory: 8,
    storage: 100,
    network: 1000,
  },
  performance: {
    maxRps: 10000,
    avgLatency: 15,
    errorRate: 0.1,
  },
  scaling: {
    minInstances: 1,
    maxInstances: 10,
    autoScaling: true,
  },
  monitoring: {
    healthCheck: true,
    metrics: true,
    alerts: true,
  },
};

export const EnhancedPropertiesPanel: React.FC<EnhancedPropertiesPanelProps> = ({
  selectedComponent,
  onUpdateComponent,
}) => {
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(['specs', 'performance'])
  );
  const [componentData, setComponentData] = useState<ComponentConfig>(mockComponentData);

  const toggleSection = (section: string) => {
    const newExpanded = new Set(expandedSections);
    if (newExpanded.has(section)) {
      newExpanded.delete(section);
    } else {
      newExpanded.add(section);
    }
    setExpandedSections(newExpanded);
  };

  const updateComponentData = (path: string, value: any) => {
    const keys = path.split('.');
    const newData = { ...componentData };
    let current: any = newData;
    
    for (let i = 0; i < keys.length - 1; i++) {
      current = current[keys[i]];
    }
    current[keys[keys.length - 1]] = value;
    
    setComponentData(newData);
    onUpdateComponent(componentData.id, newData);
  };

  if (!selectedComponent) {
    return (
      <div className="h-full flex items-center justify-center p-8">
        <div className="text-center">
          <div className="w-16 h-16 bg-gray-100 dark:bg-gray-900 rounded-full flex items-center justify-center mx-auto mb-4">
            <Settings className="w-6 h-6 text-gray-400" />
          </div>
          <h3 className="text-gray-900 dark:text-white font-medium mb-2">No Component Selected</h3>
          <p className="text-gray-600 dark:text-gray-500 text-sm">
            Select a component on the canvas to view and edit its properties
          </p>
        </div>
      </div>
    );
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'warning': return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
      case 'critical': return <AlertTriangle className="w-4 h-4 text-red-500" />;
      default: return <Info className="w-4 h-4 text-gray-500" />;
    }
  };

  const Section: React.FC<{
    id: string;
    title: string;
    icon: React.ComponentType<any>;
    children: React.ReactNode;
  }> = ({ id, title, icon: Icon, children }) => {
    const isExpanded = expandedSections.has(id);
    
    return (
      <div className="border border-gray-200 dark:border-gray-700/50 rounded-lg overflow-hidden">
        <motion.button
          whileHover={{ backgroundColor: 'rgba(55, 65, 81, 0.1)' }}
          onClick={() => toggleSection(id)}
          className="w-full p-3 flex items-center justify-between bg-gray-50 dark:bg-gray-700/30 hover:bg-gray-100 dark:hover:bg-gray-700/50 transition-colors duration-200"
        >
          <div className="flex items-center space-x-2">
            <Icon className="w-4 h-4 text-blue-500" />
            <span className="text-gray-900 dark:text-white font-medium text-sm">{title}</span>
          </div>
          {isExpanded ? (
            <ChevronDown className="w-4 h-4 text-gray-500" />
          ) : (
            <ChevronRight className="w-4 h-4 text-gray-500" />
          )}
        </motion.button>
        
        {isExpanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="p-3 bg-white dark:bg-gray-800/30"
          >
            {children}
          </motion.div>
        )}
      </div>
    );
  };

  const SliderInput: React.FC<{
    label: string;
    value: number;
    onChange: (value: number) => void;
    min: number;
    max: number;
    unit?: string;
    step?: number;
  }> = ({ label, value, onChange, min, max, unit, step = 1 }) => (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="text-xs text-gray-600 dark:text-gray-400">{label}</label>
        <div className="flex items-center space-x-1">
          <span className="text-sm font-medium text-gray-900 dark:text-white">{value}</span>
          {unit && <span className="text-xs text-gray-500">{unit}</span>}
        </div>
      </div>
      <div className="relative">
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
          className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer slider"
        />
        <style jsx>{`
          .slider::-webkit-slider-thumb {
            appearance: none;
            height: 16px;
            width: 16px;
            border-radius: 50%;
            background: #3b82f6;
            cursor: pointer;
            border: 2px solid #ffffff;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
          }
          .slider::-moz-range-thumb {
            height: 16px;
            width: 16px;
            border-radius: 50%;
            background: #3b82f6;
            cursor: pointer;
            border: 2px solid #ffffff;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
          }
        `}</style>
      </div>
    </div>
  );

  const StepperInput: React.FC<{
    label: string;
    value: number;
    onChange: (value: number) => void;
    min: number;
    max: number;
    unit?: string;
    step?: number;
  }> = ({ label, value, onChange, min, max, unit, step = 1 }) => (
    <div className="space-y-1">
      <label className="text-xs text-gray-600 dark:text-gray-400">{label}</label>
      <div className="flex items-center space-x-2">
        <motion.button
          whileHover={{ scale: 1.1 }}
          whileTap={{ scale: 0.9 }}
          onClick={() => onChange(Math.max(min, value - step))}
          disabled={value <= min}
          className="w-8 h-8 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg flex items-center justify-center transition-colors duration-200"
        >
          <Minus className="w-4 h-4 text-gray-600 dark:text-gray-400" />
        </motion.button>
        
        <div className="flex-1 text-center">
          <input
            type="number"
            value={value}
            onChange={(e) => onChange(Number(e.target.value))}
            min={min}
            max={max}
            step={step}
            className="w-full px-2 py-1 bg-gray-50 dark:bg-gray-700/50 border border-gray-200 dark:border-gray-600/50 rounded text-gray-900 dark:text-white text-sm text-center focus:outline-none focus:border-blue-500"
          />
          {unit && <span className="text-xs text-gray-500 mt-1 block">{unit}</span>}
        </div>
        
        <motion.button
          whileHover={{ scale: 1.1 }}
          whileTap={{ scale: 0.9 }}
          onClick={() => onChange(Math.min(max, value + step))}
          disabled={value >= max}
          className="w-8 h-8 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg flex items-center justify-center transition-colors duration-200"
        >
          <Plus className="w-4 h-4 text-gray-600 dark:text-gray-400" />
        </motion.button>
      </div>
    </div>
  );

  const ToggleSwitch: React.FC<{
    label: string;
    value: boolean;
    onChange: (value: boolean) => void;
    description?: string;
  }> = ({ label, value, onChange, description }) => (
    <div className="flex items-center justify-between">
      <div>
        <span className="text-sm text-gray-900 dark:text-gray-300">{label}</span>
        {description && (
          <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">{description}</p>
        )}
      </div>
      <motion.button
        whileTap={{ scale: 0.95 }}
        onClick={() => onChange(!value)}
        className={`relative w-12 h-6 rounded-full transition-colors duration-200 ${
          value ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
        }`}
      >
        <motion.div
          animate={{ x: value ? 24 : 2 }}
          transition={{ type: 'spring', stiffness: 500, damping: 30 }}
          className="w-5 h-5 bg-white rounded-full mt-0.5 shadow-sm"
        />
      </motion.button>
    </div>
  );

  return (
    <div className="h-full overflow-y-auto bg-white dark:bg-black">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700/50">
        <div className="flex items-center space-x-3 mb-3">
          <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
            <Server className="w-4 h-4 text-white" />
          </div>
          <div>
            <h3 className="text-gray-900 dark:text-white font-semibold">{componentData.name}</h3>
            <p className="text-gray-600 dark:text-gray-400 text-xs capitalize">{componentData.type}</p>
          </div>
        </div>
        
        <div className="flex items-center space-x-2">
          {getStatusIcon(componentData.status)}
          <span className="text-sm text-gray-700 dark:text-gray-300 capitalize">{componentData.status}</span>
        </div>
      </div>

      {/* Properties Sections */}
      <div className="p-4 space-y-4">
        {/* Hardware Specifications */}
        <Section id="specs" title="Hardware Specs" icon={Cpu}>
          <div className="space-y-4">
            <SliderInput
              label="CPU Cores"
              value={componentData.specs.cpu}
              onChange={(value) => updateComponentData('specs.cpu', value)}
              min={1}
              max={64}
            />
            <SliderInput
              label="Memory"
              value={componentData.specs.memory}
              onChange={(value) => updateComponentData('specs.memory', value)}
              min={1}
              max={512}
              unit="GB"
            />
            <SliderInput
              label="Storage"
              value={componentData.specs.storage}
              onChange={(value) => updateComponentData('specs.storage', value)}
              min={10}
              max={10000}
              unit="GB"
              step={10}
            />
            <SliderInput
              label="Network Bandwidth"
              value={componentData.specs.network}
              onChange={(value) => updateComponentData('specs.network', value)}
              min={100}
              max={100000}
              unit="Mbps"
              step={100}
            />
          </div>
        </Section>

        {/* Performance Configuration */}
        <Section id="performance" title="Performance" icon={Activity}>
          <div className="space-y-4">
            <StepperInput
              label="Max RPS"
              value={componentData.performance.maxRps}
              onChange={(value) => updateComponentData('performance.maxRps', value)}
              min={100}
              max={1000000}
              step={1000}
            />
            <StepperInput
              label="Target Latency"
              value={componentData.performance.avgLatency}
              onChange={(value) => updateComponentData('performance.avgLatency', value)}
              min={1}
              max={10000}
              unit="ms"
            />
            <SliderInput
              label="Error Rate"
              value={componentData.performance.errorRate}
              onChange={(value) => updateComponentData('performance.errorRate', value)}
              min={0}
              max={10}
              unit="%"
              step={0.1}
            />
          </div>
        </Section>

        {/* Auto Scaling */}
        <Section id="scaling" title="Auto Scaling" icon={Zap}>
          <div className="space-y-4">
            <ToggleSwitch
              label="Auto Scaling"
              value={componentData.scaling.autoScaling}
              onChange={(value) => updateComponentData('scaling.autoScaling', value)}
              description="Automatically scale instances based on load"
            />
            
            {componentData.scaling.autoScaling && (
              <div className="space-y-4 pt-2">
                <StepperInput
                  label="Min Instances"
                  value={componentData.scaling.minInstances}
                  onChange={(value) => updateComponentData('scaling.minInstances', value)}
                  min={1}
                  max={100}
                />
                <StepperInput
                  label="Max Instances"
                  value={componentData.scaling.maxInstances}
                  onChange={(value) => updateComponentData('scaling.maxInstances', value)}
                  min={1}
                  max={1000}
                />
              </div>
            )}
          </div>
        </Section>

        {/* Monitoring */}
        <Section id="monitoring" title="Monitoring" icon={Activity}>
          <div className="space-y-4">
            <ToggleSwitch
              label="Health Checks"
              value={componentData.monitoring.healthCheck}
              onChange={(value) => updateComponentData('monitoring.healthCheck', value)}
              description="Enable automated health monitoring"
            />
            <ToggleSwitch
              label="Metrics Collection"
              value={componentData.monitoring.metrics}
              onChange={(value) => updateComponentData('monitoring.metrics', value)}
              description="Collect performance metrics"
            />
            <ToggleSwitch
              label="Alerts"
              value={componentData.monitoring.alerts}
              onChange={(value) => updateComponentData('monitoring.alerts', value)}
              description="Send alerts on threshold breaches"
            />
          </div>
        </Section>
      </div>
    </div>
  );
};