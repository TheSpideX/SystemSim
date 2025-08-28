# Frontend Design Specification

## Overview

This document outlines the frontend architecture for the Simulation Engine platform, featuring a dual-panel design system with Google Docs-style real-time collaboration.

## Architecture Overview

### Two-Level Design System

The frontend operates on two distinct but mirrored design levels:

1. **Component Design Mode** - Design custom components using engines
2. **System Design Mode** - Design systems using components

Both modes share the same interface pattern: **Left Canvas + Right Graph Editor**

## UI Layout Architecture

### Top-Level Structure

```
┌─────────────────────────────────────────────────────────────────┐
│ Header: [Logo] [Project Name] [Collaborators] [Share] [Profile] │
├─────────────────────────────────────────────────────────────────┤
│ Mode Switcher: [System Design] [Component Design]              │
├─────────────────────────────────────────────────────────────────┤
│                    Main Design Area                             │
│  ┌─────────────────────┬─────────────────────────────────────┐  │
│  │ LEFT: Canvas        │ RIGHT: Graph Editor                 │  │
│  │                     │                                     │  │
│  │ (Content changes    │ (Content changes based on mode)    │  │
│  │  based on mode)     │                                     │  │
│  │                     │                                     │  │
│  └─────────────────────┴─────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│ Bottom Panel: [Simulation Controls] [Metrics] [Logs]           │
└─────────────────────────────────────────────────────────────────┘
```

## Component Design Mode

### Purpose
Design custom components by combining and configuring the 4 base engines (CPU, Memory, Storage, Network).

### Left Panel: Engine Canvas
```
┌─────────────────────┐
│ Engine Palette      │
│ ┌─────────────────┐ │
│ │ [CPU Engine]    │ │
│ │ [Memory Engine] │ │
│ │ [Storage Engine]│ │
│ │ [Network Engine]│ │
│ └─────────────────┘ │
│                     │
│ Canvas Area         │
│ ┌─────────────────┐ │
│ │                 │ │
│ │  Drag engines   │ │
│ │  here to build  │ │
│ │  component      │ │
│ │                 │ │
│ │  [CPU]  [MEM]   │ │
│ │    │      │     │ │
│ │  [NET]  [STO]   │ │
│ │                 │ │
│ └─────────────────┘ │
│                     │
│ Component Config    │
│ ┌─────────────────┐ │
│ │ Name: [DB Srv]  │ │
│ │ Type: [Custom]  │ │
│ │ Resources: [⚙️] │ │
│ │ Profiles: [📝]  │ │
│ └─────────────────┘ │
└─────────────────────┘
```

### Right Panel: Engine Flow Graph
```
┌─────────────────────────────────────┐
│ Engine Flow Designer                │
│                                     │
│ ┌─Network─┐    ┌─CPU─────┐          │
│ │ receive │───▶│ parse   │          │
│ │ request │    │ query   │          │
│ └─────────┘    └────┬────┘          │
│                     │               │
│                     ▼               │
│ ┌─Memory──┐    ┌─Storage─┐          │
│ │ cache   │◄──▶│ read    │          │
│ │ lookup  │    │ data    │          │
│ └────┬────┘    └────┬────┘          │
│      │              │               │
│      └──────┬───────┘               │
│             ▼                       │
│ ┌─Network─────────┐                 │
│ │ send response   │                 │
│ └─────────────────┘                 │
│                                     │
│ Flow Controls:                      │
│ [▶️ Test Flow] [🔄 Validate]        │
│                                     │
│ Conditions Editor:                  │
│ • cache_hit → send_response         │
│ • cache_miss → storage_read         │
│ • error → error_handler             │
└─────────────────────────────────────┘
```

## System Design Mode

### Purpose
Design complete systems by combining custom and pre-built components, with user flow definitions.

### Left Panel: Component Canvas
```
┌─────────────────────┐
│ Component Palette   │
│ ┌─────────────────┐ │
│ │ Pre-built:      │ │
│ │ [Database Srv]  │ │
│ │ [Web Server]    │ │
│ │ [Cache Server]  │ │
│ │ [Load Balancer] │ │
│ │                 │ │
│ │ Custom:         │ │
│ │ [My DB Server]  │ │
│ │ [API Gateway]   │ │
│ └─────────────────┘ │
│                     │
│ System Canvas       │
│ ┌─────────────────┐ │
│ │                 │ │
│ │ [Load Balancer] │ │
│ │        │        │ │
│ │        ▼        │ │
│ │   [Web Srv] ────┤ │
│ │        │        │ │
│ │        ▼        │ │
│ │   [Database]    │ │
│ │                 │ │
│ └─────────────────┘ │
│                     │
│ System Config       │
│ ┌─────────────────┐ │
│ │ Name: [E-comm]  │ │
│ │ Load: [1K RPS]  │ │
│ │ Users: [10K]    │ │
│ │ Region: [US-E]  │ │
│ └─────────────────┘ │
└─────────────────────┘
```

### Right Panel: User Flow Graph
```
┌─────────────────────────────────────┐
│ User Flow Designer                  │
│                                     │
│ Flow: "Purchase Product"            │
│                                     │
│ ┌─User────┐    ┌─Load Bal─┐         │
│ │ Browse  │───▶│ Route    │         │
│ │ Product │    │ Request  │         │
│ └─────────┘    └────┬─────┘         │
│                     │               │
│                     ▼               │
│ ┌─Web Server───────────────┐        │
│ │ • Render catalog         │        │
│ │ • Handle product clicks  │        │
│ │ • Process cart actions   │        │
│ └─────────┬─────────────────┘        │
│           │                         │
│           ▼                         │
│ ┌─Database─────────────────┐        │
│ │ • Product lookup         │        │
│ │ • Inventory check        │        │
│ │ • Order processing       │        │
│ └─────────┬─────────────────┘        │
│           │                         │
│           ▼                         │
│ ┌─User────────────────────┐         │
│ │ Receive confirmation    │         │
│ └─────────────────────────┘         │
│                                     │
│ Flow Metrics:                       │
│ • Avg Response: 250ms               │
│ • Success Rate: 99.2%               │
│ • Bottleneck: Database              │
└─────────────────────────────────────┘
```

## Real-Time Collaboration Features

### Google Docs-Style Collaboration

#### Live Presence Indicators
```
┌─────────────────────────────────────────────────────────────────┐
│ Header: [Logo] [Project Name]                                   │
│                                                                 │
│ Active Collaborators:                                           │
│ ┌──────┐ ┌──────┐ ┌──────┐                                     │
│ │ 👤 A │ │ 👤 B │ │ 👤 C │  [+ Invite]                        │
│ │Green │ │Blue  │ │Red   │                                     │
│ └──────┘ └──────┘ └──────┘                                     │
└─────────────────────────────────────────────────────────────────┘
```

#### Live Cursors & Selections
- **Colored cursors** showing where each user is working
- **Selection highlights** showing what each user has selected
- **User labels** showing who is editing what component/node

#### Real-Time Updates
- **Instant synchronization** of all design changes
- **Conflict resolution** using operational transforms
- **Change indicators** showing recent modifications

### Collaboration Controls

#### Sharing & Permissions
```
┌─────────────────────────────────────┐
│ Share Project                       │
│                                     │
│ 👤 alice@company.com    [Editor ▼] │
│ 👤 bob@company.com      [Viewer ▼] │
│ 👤 carol@company.com    [Admin  ▼] │
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ Add people: [email@domain.com]  │ │
│ │ Permission: [Editor ▼] [Add]    │ │
│ └─────────────────────────────────┘ │
│                                     │
│ Link Sharing:                       │
│ 🔗 Anyone with link can [View ▼]    │
│                                     │
│ [Copy Link] [Done]                  │
└─────────────────────────────────────┘
```

#### Comments & Suggestions
```
┌─────────────────────────────────────┐
│ 💬 Comments                         │
│                                     │
│ 👤 Alice: "Should we add caching    │
│           here for better perf?"    │
│    └─ 👤 Bob: "Good idea! I'll add  │
│              a Redis component"     │
│                                     │
│ 💡 Suggestions                      │
│                                     │
│ 👤 Carol suggested:                 │
│ "Change CPU complexity from 2 to 3" │
│ [Accept] [Reject] [Reply]           │
└─────────────────────────────────────┘
```

## State Management Architecture

### Global App State
```typescript
interface AppState {
  // Authentication & User
  user: User | null
  
  // Project Management
  currentProject: Project | null
  projects: Project[]
  
  // Collaboration
  collaborators: ActiveUser[]
  presence: PresenceState
  
  // Design Mode
  mode: 'component-design' | 'system-design'
  
  // Component Design State
  componentDesign: {
    selectedComponent: ComponentDesign | null
    enginePalette: EngineType[]
    canvas: EngineInstance[]
    flowGraph: EngineFlowGraph
    isEditing: boolean
  }
  
  // System Design State  
  systemDesign: {
    selectedSystem: SystemDesign | null
    componentPalette: ComponentDesign[]
    canvas: ComponentInstance[]
    userFlows: UserFlowGraph[]
    isEditing: boolean
  }
  
  // Simulation State
  runningSimulations: SimulationInstance[]
  simulationMetrics: MetricsData
  
  // UI State
  selectedTool: DesignTool
  zoomLevel: number
  panPosition: Point
  showGrid: boolean
  showMetrics: boolean
}
```

### Real-Time Collaboration State
```typescript
interface CollaborationState {
  // Active Users
  activeUsers: Map<string, ActiveUser>
  
  // Live Cursors
  cursors: Map<string, CursorPosition>
  
  // Selections
  selections: Map<string, Selection[]>
  
  // Resource Locks
  locks: Map<string, ResourceLock>
  
  // Change Stream
  operations: Operation[]
  pendingOperations: Operation[]
  
  // Conflict Resolution
  conflicts: Conflict[]
}
```

## Component Architecture

### React Component Hierarchy
```
App
├── AuthProvider
├── ProjectProvider
├── CollaborationProvider
└── Router
    ├── ProjectDashboard
    ├── ProjectEditor
    │   ├── Header
    │   │   ├── ProjectInfo
    │   │   ├── CollaboratorList
    │   │   └── ShareButton
    │   ├── ModeSwitch
    │   ├── DesignArea
    │   │   ├── LeftPanel
    │   │   │   ├── ComponentMode
    │   │   │   │   ├── EnginePalette
    │   │   │   │   ├── EngineCanvas
    │   │   │   │   └── ComponentConfig
    │   │   │   └── SystemMode
    │   │   │       ├── ComponentPalette
    │   │   │       ├── SystemCanvas
    │   │   │       └── SystemConfig
    │   │   └── RightPanel
    │   │       ├── ComponentMode
    │   │       │   └── EngineFlowEditor
    │   │       └── SystemMode
    │   │           └── UserFlowEditor
    │   └── BottomPanel
    │       ├── SimulationControls
    │       ├── MetricsPanel
    │       └── LogsPanel
    └── SimulationDashboard
```

## Technology Stack

### Frontend Framework
- **React 18** with TypeScript
- **Next.js 14** for SSR and routing
- **Tailwind CSS** for styling
- **Framer Motion** for animations

### State Management
- **Zustand** for global state
- **React Query** for server state
- **Immer** for immutable updates

### Real-Time Communication
- **Socket.io** for WebSocket connections
- **Y.js** for operational transforms
- **WebRTC** for peer-to-peer collaboration

### Canvas & Graphics
- **React Flow** for node-based editors
- **Konva.js** for canvas manipulation
- **D3.js** for data visualization

### UI Components
- **Radix UI** for accessible components
- **React Hook Form** for forms
- **React Hot Toast** for notifications

## Performance Considerations

### Optimization Strategies
- **Virtual scrolling** for large component lists
- **Canvas virtualization** for large system designs
- **Debounced updates** for real-time collaboration
- **Lazy loading** for component palettes
- **Memoization** for expensive calculations

### Collaboration Performance
- **Operational transform batching** to reduce network calls
- **Local-first architecture** for instant UI updates
- **Conflict-free replicated data types** for consistency
- **WebSocket connection pooling** for scalability

## Accessibility

### WCAG 2.1 AA Compliance
- **Keyboard navigation** for all design tools
- **Screen reader support** for canvas elements
- **High contrast mode** for visual elements
- **Focus management** for modal dialogs
- **ARIA labels** for complex interactions

## Next Steps

1. **Create design system** - Build reusable UI components
2. **Implement canvas framework** - Build drag & drop foundation
3. **Add real-time collaboration** - Implement operational transforms
4. **Build component/system editors** - Create the dual-panel interface
5. **Integrate with backend services** - Connect to Project & Simulation services
6. **Add simulation visualization** - Real-time metrics and monitoring
7. **Implement advanced features** - Comments, suggestions, version history
