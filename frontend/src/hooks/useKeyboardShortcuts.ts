import { useEffect, useCallback } from 'react';

interface KeyboardShortcut {
  key: string;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
  altKey?: boolean;
  action: () => void;
  description: string;
}

interface UseKeyboardShortcutsProps {
  shortcuts: KeyboardShortcut[];
  enabled?: boolean;
}

export const useKeyboardShortcuts = ({ shortcuts, enabled = true }: UseKeyboardShortcutsProps) => {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return;

      // Don't trigger shortcuts when typing in inputs
      const target = event.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.contentEditable === 'true'
      ) {
        return;
      }

      const matchingShortcut = shortcuts.find((shortcut) => {
        const keyMatch = shortcut.key.toLowerCase() === event.key.toLowerCase();
        const ctrlMatch = !!shortcut.ctrlKey === event.ctrlKey;
        const metaMatch = !!shortcut.metaKey === event.metaKey;
        const shiftMatch = !!shortcut.shiftKey === event.shiftKey;
        const altMatch = !!shortcut.altKey === event.altKey;

        return keyMatch && ctrlMatch && metaMatch && shiftMatch && altMatch;
      });

      if (matchingShortcut) {
        event.preventDefault();
        matchingShortcut.action();
      }
    },
    [shortcuts, enabled]
  );

  useEffect(() => {
    if (enabled) {
      document.addEventListener('keydown', handleKeyDown);
      return () => document.removeEventListener('keydown', handleKeyDown);
    }
  }, [handleKeyDown, enabled]);

  return shortcuts;
};

// Common shortcut configurations
export const getCommonShortcuts = (actions: {
  copy: () => void;
  paste: () => void;
  delete: () => void;
  group: () => void;
  undo: () => void;
  redo: () => void;
  save: () => void;
  selectAll: () => void;
  duplicate: () => void;
  focusMode: () => void;
}): KeyboardShortcut[] => [
  {
    key: 'c',
    ctrlKey: true,
    action: actions.copy,
    description: 'Copy selected items',
  },
  {
    key: 'c',
    metaKey: true,
    action: actions.copy,
    description: 'Copy selected items (Mac)',
  },
  {
    key: 'v',
    ctrlKey: true,
    action: actions.paste,
    description: 'Paste items',
  },
  {
    key: 'v',
    metaKey: true,
    action: actions.paste,
    description: 'Paste items (Mac)',
  },
  {
    key: 'Delete',
    action: actions.delete,
    description: 'Delete selected items',
  },
  {
    key: 'Backspace',
    action: actions.delete,
    description: 'Delete selected items',
  },
  {
    key: 'g',
    ctrlKey: true,
    action: actions.group,
    description: 'Group selected items',
  },
  {
    key: 'g',
    metaKey: true,
    action: actions.group,
    description: 'Group selected items (Mac)',
  },
  {
    key: 'z',
    ctrlKey: true,
    action: actions.undo,
    description: 'Undo',
  },
  {
    key: 'z',
    metaKey: true,
    action: actions.undo,
    description: 'Undo (Mac)',
  },
  {
    key: 'y',
    ctrlKey: true,
    action: actions.redo,
    description: 'Redo',
  },
  {
    key: 'z',
    ctrlKey: true,
    shiftKey: true,
    action: actions.redo,
    description: 'Redo (Shift+Ctrl+Z)',
  },
  {
    key: 'z',
    metaKey: true,
    shiftKey: true,
    action: actions.redo,
    description: 'Redo (Mac)',
  },
  {
    key: 's',
    ctrlKey: true,
    action: actions.save,
    description: 'Save project',
  },
  {
    key: 's',
    metaKey: true,
    action: actions.save,
    description: 'Save project (Mac)',
  },
  {
    key: 'a',
    ctrlKey: true,
    action: actions.selectAll,
    description: 'Select all',
  },
  {
    key: 'a',
    metaKey: true,
    action: actions.selectAll,
    description: 'Select all (Mac)',
  },
  {
    key: 'd',
    ctrlKey: true,
    action: actions.duplicate,
    description: 'Duplicate selected items',
  },
  {
    key: 'd',
    metaKey: true,
    action: actions.duplicate,
    description: 'Duplicate selected items (Mac)',
  },
  {
    key: 'f',
    ctrlKey: true,
    shiftKey: true,
    action: actions.focusMode,
    description: 'Toggle focus mode',
  },
  {
    key: 'f',
    metaKey: true,
    shiftKey: true,
    action: actions.focusMode,
    description: 'Toggle focus mode (Mac)',
  },
];