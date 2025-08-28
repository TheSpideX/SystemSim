import { create } from 'zustand';

export type DesignMode = 'system-design' | 'component-design';

interface UIState {
  // Panel states
  isLeftPanelCollapsed: boolean;
  isRightPanelCollapsed: boolean;
  leftPanelWidth: number;
  rightPanelWidth: number;

  // Focus mode
  isFocusMode: boolean;

  // Design mode
  designMode: DesignMode;

  // Panel content
  showProperties: boolean;
  showMetrics: boolean;

  // Project state
  projectTitle: string;
  saveStatus: 'saved' | 'saving' | 'unsaved';

  // Selection and grouping
  selectedNodes: string[];
  selectedGroups: string[];

  // Actions
  toggleLeftPanel: () => void;
  toggleRightPanel: () => void;
  setLeftPanelWidth: (width: number) => void;
  setRightPanelWidth: (width: number) => void;
  toggleFocusMode: () => void;
  setDesignMode: (mode: DesignMode) => void;
  setShowProperties: (show: boolean) => void;
  setShowMetrics: (show: boolean) => void;
  setProjectTitle: (title: string) => void;
  setSaveStatus: (status: 'saved' | 'saving' | 'unsaved') => void;
  setSelectedNodes: (nodes: string[]) => void;
  addSelectedNode: (nodeId: string) => void;
  removeSelectedNode: (nodeId: string) => void;
  clearSelection: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  // Initial state
  isLeftPanelCollapsed: false,
  isRightPanelCollapsed: false,
  leftPanelWidth: 320,
  rightPanelWidth: 320,
  isFocusMode: false,
  designMode: 'system-design',
  showProperties: true,
  showMetrics: true,
  projectTitle: 'Untitled Project',
  saveStatus: 'saved',
  selectedNodes: [],
  selectedGroups: [],

  // Actions
  toggleLeftPanel: () =>
    set((state) => ({ isLeftPanelCollapsed: !state.isLeftPanelCollapsed })),

  toggleRightPanel: () =>
    set((state) => ({ isRightPanelCollapsed: !state.isRightPanelCollapsed })),

  setLeftPanelWidth: (width) =>
    set({ leftPanelWidth: Math.max(200, Math.min(600, width)) }),

  setRightPanelWidth: (width) =>
    set({ rightPanelWidth: Math.max(200, Math.min(600, width)) }),

  toggleFocusMode: () =>
    set((state) => ({
      isFocusMode: !state.isFocusMode,
      isLeftPanelCollapsed: !state.isFocusMode ? true : state.isLeftPanelCollapsed,
      isRightPanelCollapsed: !state.isFocusMode ? true : state.isRightPanelCollapsed,
    })),

  setDesignMode: (mode) => set({ designMode: mode }),
  setShowProperties: (show) => set({ showProperties: show }),
  setShowMetrics: (show) => set({ showMetrics: show }),
  setProjectTitle: (title) => set({ projectTitle: title, saveStatus: 'unsaved' }),
  setSaveStatus: (status) => set({ saveStatus: status }),

  setSelectedNodes: (nodes) => set({ selectedNodes: nodes }),
  addSelectedNode: (nodeId) =>
    set((state) => ({
      selectedNodes: state.selectedNodes.includes(nodeId)
        ? state.selectedNodes
        : [...state.selectedNodes, nodeId],
    })),
  removeSelectedNode: (nodeId) =>
    set((state) => ({
      selectedNodes: state.selectedNodes.filter((id) => id !== nodeId),
    })),
  clearSelection: () => set({ selectedNodes: [], selectedGroups: [] }),
}));