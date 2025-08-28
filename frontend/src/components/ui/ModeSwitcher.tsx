import React from 'react';
import { motion } from 'framer-motion';
import { Box, Layers } from 'lucide-react';
import { useUIStore, type DesignMode } from '../../store/uiStore';

export const ModeSwitcher: React.FC = () => {
  const { designMode, setDesignMode } = useUIStore();

  const modes = [
    {
      id: 'system-design' as DesignMode,
      label: 'System Design',
      icon: Layers,
      description: 'Design systems using components'
    },
    {
      id: 'component-design' as DesignMode,
      label: 'Component Design',
      icon: Box,
      description: 'Design custom components using engines'
    }
  ];

  return (
    <div className="flex items-center justify-center">
      <div className="flex items-center space-x-1 bg-gray-100 dark:bg-gray-800 rounded-lg p-1 border border-gray-200 dark:border-gray-700 shadow-sm">
        {modes.map((mode) => {
          const Icon = mode.icon;
          const isActive = designMode === mode.id;

          return (
            <motion.button
              key={mode.id}
              onClick={() => setDesignMode(mode.id)}
              className={`flex items-center space-x-1.5 px-2.5 py-1.5 rounded-md text-sm font-medium transition-all duration-200 ${
                isActive
                  ? 'bg-gray-600 text-white border border-gray-500 shadow-sm'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-200 dark:hover:bg-gray-700'
              }`}
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              title={mode.description}
            >
              <Icon className="w-4 h-4" />
              <span className="whitespace-nowrap">{mode.label}</span>
            </motion.button>
          );
        })}
      </div>
    </div>
  );
};
