import React, { useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Copy, Trash2, Settings, BarChart, CopyPlus,
  Plus, Clipboard, Eye, Maximize
} from 'lucide-react';
import { useClickOutside } from '../../hooks/useGlobalMouse';

export interface ContextMenuItem {
  id: string;
  label: string;
  icon: React.ReactNode;
  action: () => void;
  disabled?: boolean;
  separator?: boolean;
}

interface ContextMenuProps {
  isOpen: boolean;
  position: { x: number; y: number };
  items: ContextMenuItem[];
  onClose: () => void;
}

export const ContextMenu: React.FC<ContextMenuProps> = ({
  isOpen,
  position,
  items,
  onClose,
}) => {
  const menuRef = useRef<HTMLDivElement>(null);

  // Use global click outside detection
  useClickOutside(menuRef, onClose, isOpen);

  // Handle escape key (keep this separate as it's not mouse-related)
  React.useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      return () => document.removeEventListener('keydown', handleEscape);
    }
  }, [isOpen, onClose]);

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          ref={menuRef}
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.95 }}
          transition={{ duration: 0.1 }}
          className="fixed z-50 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg py-1 min-w-[180px]"
          style={{
            left: position.x,
            top: position.y,
          }}
        >
          {items.map((item, index) => (
            <React.Fragment key={item.id}>
              {item.separator ? (
                <div className="h-px bg-gray-200 dark:bg-gray-700 my-1" />
              ) : (
                <button
                  onClick={() => {
                    if (!item.disabled) {
                      item.action();
                      onClose();
                    }
                  }}
                  disabled={item.disabled}
                  className={`w-full px-3 py-2 text-left flex items-center space-x-3 text-sm transition-colors ${
                    item.disabled
                      ? 'text-gray-400 dark:text-gray-600 cursor-not-allowed'
                      : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
                  }`}
                >
                  <span className="w-4 h-4 flex-shrink-0">{item.icon}</span>
                  <span>{item.label}</span>
                </button>
              )}
            </React.Fragment>
          ))}
        </motion.div>
      )}
    </AnimatePresence>
  );
};

// Predefined context menu configurations
export const getNodeContextMenu = (
  nodeId: string,
  onDuplicate: (id: string) => void,
  onDelete: (id: string) => void,
  onCopy: (id: string) => void,
  onViewMetrics: (id: string) => void,
  onConfigure: (id: string) => void
): ContextMenuItem[] => [
  {
    id: 'duplicate',
    label: 'Duplicate',
    icon: <CopyPlus className="w-4 h-4" />,
    action: () => onDuplicate(nodeId),
  },
  {
    id: 'copy',
    label: 'Copy',
    icon: <Copy className="w-4 h-4" />,
    action: () => onCopy(nodeId),
  },
  {
    id: 'separator1',
    label: '',
    icon: null,
    action: () => {},
    separator: true,
  },
  {
    id: 'metrics',
    label: 'View Metrics',
    icon: <BarChart className="w-4 h-4" />,
    action: () => onViewMetrics(nodeId),
  },
  {
    id: 'configure',
    label: 'Configure',
    icon: <Settings className="w-4 h-4" />,
    action: () => onConfigure(nodeId),
  },
  {
    id: 'separator2',
    label: '',
    icon: null,
    action: () => {},
    separator: true,
  },
  {
    id: 'delete',
    label: 'Delete',
    icon: <Trash2 className="w-4 h-4" />,
    action: () => onDelete(nodeId),
  },
];

export const getCanvasContextMenu = (
  onPaste: () => void,
  onAddComponent: () => void,
  onFitToView: () => void,
  canPaste: boolean = false
): ContextMenuItem[] => [
  {
    id: 'paste',
    label: 'Paste',
    icon: <Clipboard className="w-4 h-4" />,
    action: onPaste,
    disabled: !canPaste,
  },
  {
    id: 'separator1',
    label: '',
    icon: null,
    action: () => {},
    separator: true,
  },
  {
    id: 'add',
    label: 'Add Component',
    icon: <Plus className="w-4 h-4" />,
    action: onAddComponent,
  },
  {
    id: 'separator2',
    label: '',
    icon: null,
    action: () => {},
    separator: true,
  },
  {
    id: 'fit',
    label: 'Fit to View',
    icon: <Maximize className="w-4 h-4" />,
    action: onFitToView,
  },
];