import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import {
  Activity, TrendingUp, TrendingDown, Clock, Cpu, MemoryStick,
  Network, HardDrive, AlertTriangle, CheckCircle, Zap, Database,
  BarChart, TrendingUp as LineChartIcon, PieChart
} from 'lucide-react';

interface EnhancedMetricsPanelProps {
  isSimulationRunning: boolean;
  selectedComponent: string | null;
}

interface MetricData {
  timestamp: number;
  value: number;
  label?: string;
}

interface ComponentMetrics {
  rps: MetricData[];
  latency: MetricData[];
  cpu: MetricData[];
  memory: MetricData[];
  errorRate: MetricData[];
  connections: MetricData[];
}

// Generate mock real-time metrics
const generateMetricData = (baseValue: number, variance: number = 0.1): number => {
  const variation = (Math.random() - 0.5) * 2 * variance;
  return Math.max(0, baseValue * (1 + variation));
};

export const EnhancedMetricsPanel: React.FC<EnhancedMetricsPanelProps> = ({
  isSimulationRunning,
  selectedComponent,
}) => {
  const [metrics, setMetrics] = useState<ComponentMetrics>({
    rps: [],
    latency: [],
    cpu: [],
    memory: [],
    errorRate: [],
    connections: [],
  });

  const [currentMetrics, setCurrentMetrics] = useState({
    rps: 7500,
    latency: 15,
    cpu: 65,
    memory: 70,
    errorRate: 0.1,
    connections: 245,
  });

  const [selectedMetric, setSelectedMetric] = useState<keyof ComponentMetrics>('rps');

  // Update metrics in real-time when simulation is running
  useEffect(() => {
    if (!isSimulationRunning) return;

    const interval = setInterval(() => {
      const timestamp = Date.now();
      
      const newMetrics = {
        rps: generateMetricData(7500, 0.2),
        latency: generateMetricData(15, 0.3),
        cpu: generateMetricData(65, 0.15),
        memory: generateMetricData(70, 0.1),
        errorRate: generateMetricData(0.1, 0.5),
        connections: generateMetricData(245, 0.25),
      };

      setCurrentMetrics(newMetrics);

      setMetrics(prev => ({
        rps: [...prev.rps.slice(-59), { timestamp, value: newMetrics.rps }],
        latency: [...prev.latency.slice(-59), { timestamp, value: newMetrics.latency }],
        cpu: [...prev.cpu.slice(-59), { timestamp, value: newMetrics.cpu }],
        memory: [...prev.memory.slice(-59), { timestamp, value: newMetrics.memory }],
        errorRate: [...prev.errorRate.slice(-59), { timestamp, value: newMetrics.errorRate }],
        connections: [...prev.connections.slice(-59), { timestamp, value: newMetrics.connections }],
      }));
    }, 1000);

    return () => clearInterval(interval);
  }, [isSimulationRunning]);

  const MetricCard: React.FC<{
    title: string;
    value: number;
    unit: string;
    icon: React.ComponentType<any>;
    color: string;
    trend?: 'up' | 'down' | 'stable';
    data: MetricData[];
    format?: (value: number) => string;
    isSelected?: boolean;
    onClick?: () => void;
  }> = ({ title, value, unit, icon: Icon, color, trend, data, format, isSelected, onClick }) => {
    const formatValue = format || ((v: number) => v.toFixed(1));
    const trendIcon = trend === 'up' ? TrendingUp : trend === 'down' ? TrendingDown : null;
    
    return (
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.98 }}
        onClick={onClick}
        className={`border rounded-lg p-3 cursor-pointer transition-all duration-200 ${
          isSelected 
            ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-700' 
            : 'bg-gray-50 dark:bg-gray-700/30 border-gray-200 dark:border-gray-600/30 hover:bg-gray-100 dark:hover:bg-gray-700/50'
        }`}
      >
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center space-x-2">
            <Icon className={`w-4 h-4 ${color}`} />
            <span className="text-gray-700 dark:text-gray-300 text-sm font-medium">{title}</span>
          </div>
          {trendIcon && (
            <trendIcon className={`w-3 h-3 ${
              trend === 'up' ? 'text-red-400' : 'text-green-400'
            }`} />
          )}
        </div>
        
        <div className="flex items-baseline space-x-1 mb-3">
          <span className="text-gray-900 dark:text-white font-mono text-xl font-bold">{formatValue(value)}</span>
          <span className="text-gray-500 dark:text-gray-400 text-sm">{unit}</span>
        </div>

        {/* Mini Sparkline Chart */}
        <div className="h-8 flex items-end space-x-0.5">
          {data.slice(-20).map((point, index) => {
            const maxValue = Math.max(...data.map(d => d.value));
            const minValue = Math.min(...data.map(d => d.value));
            const range = maxValue - minValue || 1;
            const height = ((point.value - minValue) / range) * 100;
            
            return (
              <div
                key={index}
                className={`flex-1 bg-gradient-to-t ${color.replace('text-', 'from-').replace('-400', '-500')} to-transparent rounded-sm opacity-70`}
                style={{ height: `${Math.max(height, 5)}%` }}
              />
            );
          })}
        </div>
      </motion.div>
    );
  };

  const LineChart: React.FC<{
    data: MetricData[];
    title: string;
    color: string;
    unit: string;
  }> = ({ data, title, color, unit }) => {
    if (data.length === 0) return null;

    const maxValue = Math.max(...data.map(d => d.value));
    const minValue = Math.min(...data.map(d => d.value));
    const range = maxValue - minValue || 1;

    const points = data.map((point, index) => {
      const x = (index / (data.length - 1)) * 100;
      const y = 100 - ((point.value - minValue) / range) * 100;
      return `${x},${y}`;
    }).join(' ');

    return (
      <div className="bg-white dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h4 className="text-gray-900 dark:text-white font-medium">{title}</h4>
          <div className="text-sm text-gray-500 dark:text-gray-400">Last 60 seconds</div>
        </div>
        
        <div className="relative h-32 bg-gray-50 dark:bg-gray-900/50 rounded-lg overflow-hidden">
          <svg className="w-full h-full" viewBox="0 0 100 100" preserveAspectRatio="none">
            {/* Grid lines */}
            <defs>
              <pattern id="grid" width="10" height="10" patternUnits="userSpaceOnUse">
                <path d="M 10 0 L 0 0 0 10" fill="none" stroke="currentColor" strokeWidth="0.5" className="text-gray-300 dark:text-gray-600" opacity="0.3"/>
              </pattern>
            </defs>
            <rect width="100" height="100" fill="url(#grid)" />
            
            {/* Area under curve */}
            <path
              d={`M 0,100 L ${points} L 100,100 Z`}
              fill={`url(#gradient-${color})`}
              opacity="0.2"
            />
            
            {/* Line */}
            <polyline
              points={points}
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              className={color}
            />
            
            {/* Gradient definition */}
            <defs>
              <linearGradient id={`gradient-${color}`} x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" stopColor="currentColor" className={color} />
                <stop offset="100%" stopColor="currentColor" className={color} stopOpacity="0" />
              </linearGradient>
            </defs>
          </svg>
          
          {/* Value labels */}
          <div className="absolute top-2 left-2 text-xs text-gray-500 dark:text-gray-400">
            {maxValue.toFixed(1)} {unit}
          </div>
          <div className="absolute bottom-2 left-2 text-xs text-gray-500 dark:text-gray-400">
            {minValue.toFixed(1)} {unit}
          </div>
        </div>
      </div>
    );
  };

  const AlertCard: React.FC<{
    type: 'warning' | 'critical' | 'info';
    title: string;
    message: string;
  }> = ({ type, title, message }) => {
    const colors = {
      warning: 'border-yellow-200 dark:border-yellow-500/50 bg-yellow-50 dark:bg-yellow-500/10 text-yellow-700 dark:text-yellow-400',
      critical: 'border-red-200 dark:border-red-500/50 bg-red-50 dark:bg-red-500/10 text-red-700 dark:text-red-400',
      info: 'border-blue-200 dark:border-blue-500/50 bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-400',
    };

    const icons = {
      warning: AlertTriangle,
      critical: AlertTriangle,
      info: CheckCircle,
    };

    const Icon = icons[type];

    return (
      <motion.div
        initial={{ opacity: 0, x: 10 }}
        animate={{ opacity: 1, x: 0 }}
        className={`border rounded-lg p-3 ${colors[type]}`}
      >
        <div className="flex items-start space-x-2">
          <Icon className="w-4 h-4 mt-0.5 flex-shrink-0" />
          <div>
            <h4 className="font-medium text-sm">{title}</h4>
            <p className="text-xs opacity-80 mt-1">{message}</p>
          </div>
        </div>
      </motion.div>
    );
  };

  if (!selectedComponent) {
    return (
      <div className="h-full flex items-center justify-center p-8">
        <div className="text-center">
          <div className="w-16 h-16 bg-gray-100 dark:bg-gray-900 rounded-full flex items-center justify-center mx-auto mb-4">
            <Activity className="w-6 h-6 text-gray-400" />
          </div>
          <h3 className="text-gray-900 dark:text-white font-medium mb-2">No Component Selected</h3>
          <p className="text-gray-600 dark:text-gray-500 text-sm">
            Select a component to view real-time metrics and performance data
          </p>
        </div>
      </div>
    );
  }

  const metricConfigs = {
    rps: { title: 'Requests/sec', unit: 'RPS', icon: TrendingUp, color: 'text-blue-500', format: (v: number) => Math.round(v).toLocaleString() },
    latency: { title: 'Latency', unit: 'ms', icon: Clock, color: 'text-green-500', format: (v: number) => v.toFixed(1) },
    cpu: { title: 'CPU Usage', unit: '%', icon: Cpu, color: 'text-purple-500', format: (v: number) => v.toFixed(1) },
    memory: { title: 'Memory Usage', unit: '%', icon: MemoryStick, color: 'text-yellow-500', format: (v: number) => v.toFixed(1) },
    errorRate: { title: 'Error Rate', unit: '%', icon: AlertTriangle, color: 'text-red-500', format: (v: number) => v.toFixed(2) },
    connections: { title: 'Connections', unit: 'active', icon: Network, color: 'text-cyan-500', format: (v: number) => Math.round(v).toString() },
  };

  return (
    <div className="h-full overflow-y-auto bg-white dark:bg-black">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700/50">
        <div className="flex items-center space-x-2 mb-2">
          <Activity className="w-5 h-5 text-green-500" />
          <h3 className="text-gray-900 dark:text-white font-semibold">Real-time Metrics</h3>
        </div>
        <div className="flex items-center space-x-2">
          <div className={`w-2 h-2 rounded-full ${
            isSimulationRunning ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
          }`} />
          <span className="text-gray-600 dark:text-gray-400 text-sm">
            {isSimulationRunning ? 'Live Data' : 'Simulation Paused'}
          </span>
        </div>
      </div>

      <div className="p-4 space-y-6">
        {/* Metrics Overview Grid */}
        <div className="grid grid-cols-2 gap-3">
          {Object.entries(metricConfigs).map(([key, config]) => (
            <MetricCard
              key={key}
              title={config.title}
              value={currentMetrics[key as keyof typeof currentMetrics]}
              unit={config.unit}
              icon={config.icon}
              color={config.color}
              data={metrics[key as keyof ComponentMetrics]}
              format={config.format}
              isSelected={selectedMetric === key}
              onClick={() => setSelectedMetric(key as keyof ComponentMetrics)}
            />
          ))}
        </div>

        {/* Detailed Chart */}
        <LineChart
          data={metrics[selectedMetric]}
          title={metricConfigs[selectedMetric].title}
          color={metricConfigs[selectedMetric].color}
          unit={metricConfigs[selectedMetric].unit}
        />

        {/* Alerts & Recommendations */}
        <div className="space-y-3">
          <h4 className="text-gray-900 dark:text-gray-300 font-medium text-sm">Alerts & Recommendations</h4>
          <div className="space-y-2">
            {currentMetrics.cpu > 80 && (
              <AlertCard
                type="warning"
                title="High CPU Usage"
                message="CPU usage is above 80%. Consider scaling up or optimizing workload."
              />
            )}
            {currentMetrics.memory > 85 && (
              <AlertCard
                type="critical"
                title="Memory Pressure"
                message="Memory usage is critically high. Immediate action required."
              />
            )}
            {currentMetrics.errorRate > 1 && (
              <AlertCard
                type="critical"
                title="High Error Rate"
                message="Error rate is above acceptable threshold. Check logs for issues."
              />
            )}
            {currentMetrics.latency > 100 && (
              <AlertCard
                type="warning"
                title="High Latency"
                message="Response times are elevated. Consider optimizing or scaling."
              />
            )}
            {currentMetrics.cpu < 50 && currentMetrics.memory < 50 && currentMetrics.errorRate < 0.5 && (
              <AlertCard
                type="info"
                title="System Healthy"
                message="All metrics are within normal ranges. System performing optimally."
              />
            )}
          </div>
        </div>

        {/* Quick Actions */}
        <div className="space-y-3">
          <h4 className="text-gray-900 dark:text-gray-300 font-medium text-sm">Quick Actions</h4>
          <div className="grid grid-cols-2 gap-2">
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              className="p-3 bg-blue-50 dark:bg-blue-500/20 text-blue-600 dark:text-blue-400 rounded-lg hover:bg-blue-100 dark:hover:bg-blue-500/30 transition-colors text-sm font-medium"
            >
              Scale Up
            </motion.button>
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              className="p-3 bg-red-50 dark:bg-red-500/20 text-red-600 dark:text-red-400 rounded-lg hover:bg-red-100 dark:hover:bg-red-500/30 transition-colors text-sm font-medium"
            >
              Restart
            </motion.button>
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              className="p-3 bg-yellow-50 dark:bg-yellow-500/20 text-yellow-600 dark:text-yellow-400 rounded-lg hover:bg-yellow-100 dark:hover:bg-yellow-500/30 transition-colors text-sm font-medium"
            >
              Debug
            </motion.button>
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              className="p-3 bg-green-50 dark:bg-green-500/20 text-green-600 dark:text-green-400 rounded-lg hover:bg-green-100 dark:hover:bg-green-500/30 transition-colors text-sm font-medium"
            >
              Optimize
            </motion.button>
          </div>
        </div>
      </div>
    </div>
  );
};