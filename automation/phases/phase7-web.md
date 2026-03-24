You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox.

Read CLAUDE.md, docs/development-plan.md. Read the API handlers in pkg/api/handlers.go to understand the backend endpoints.

## Your Task: Phase 7 - Web Dashboard

Build the React + TypeScript dashboard in the web/ directory.

### 1. Install additional dependencies
```bash
cd /c/Users/Administrator/ai-sandbox/web
npm install react-router-dom @tanstack/react-query lucide-react
npm install -D @types/react-router-dom
```

### 2. web/src/lib/api.ts
TypeScript API client:
- Base URL configurable (default http://localhost:8080/api/v1)
- Functions for all API endpoints:
  - `listSandboxes()`, `createSandbox(config)`, `getSandbox(id)`
  - `startSandbox(id)`, `stopSandbox(id)`, `destroySandbox(id)`
  - `executeAction(id, action)`, `getTraces(id)`, `startReplay(id)`
  - `connectWebSocket(id, onEvent)` - WebSocket client for real-time traces
- Proper TypeScript types matching the Go types
- Error handling with typed errors

### 3. web/src/App.tsx
- React Router setup with routes:
  - `/` - Dashboard
  - `/sandboxes` - Sandbox list
  - `/sandboxes/:id` - Sandbox detail with traces
  - `/sandboxes/:id/replay` - Replay view
  - `/policies` - Policy editor
- Dark mode by default using Tailwind
- Responsive sidebar navigation

### 4. web/src/components/Dashboard.tsx
Main dashboard:
- Stats cards: Active sandboxes, Total actions, Denied actions, Avg response time
- Recent activity feed (last 20 events across all sandboxes)
- Quick actions: Create sandbox, View all sandboxes

### 5. web/src/components/SandboxList.tsx
- Table of sandboxes with columns: Name, Status (color-coded badge), Actions count, Created, Controls
- Create new sandbox button with modal form
- Start/Stop/Delete actions per sandbox
- Auto-refresh every 5 seconds using react-query

### 6. web/src/components/TraceViewer.tsx
Visual trace timeline:
- Tree view showing parent-child spans
- Each node shows: event type (icon), action details, duration, result (success/fail)
- Color coding: green=allowed, red=denied, yellow=audit, gray=info
- Click to expand full event details (JSON view)
- Timeline bar visualization showing relative timing
- Filter by event type, action type

### 7. web/src/components/PolicyEditor.tsx
- Split view: visual rule builder (left) + YAML preview (right)
- Add/edit/delete rules with form fields
- Action type dropdown, resource pattern input, effect radio buttons
- Live YAML generation as you edit
- Validate button that calls the API
- Load from file / save to file

### 8. web/src/components/ReplayControls.tsx
Trace replay UI:
- Play / Pause / Step Forward / Step Back / Rewind buttons
- Speed control (0.5x, 1x, 2x, 5x)
- Progress bar showing position in trace
- Current event detail panel
- Decision tree visualization showing which policy rule matched

### 9. Styling
- Use Tailwind CSS throughout
- Dark mode (bg-gray-900, text-gray-100)
- Consistent spacing and typography
- Responsive design (works on mobile)
- Use lucide-react for icons

### Verification:
1. `cd web && npm run build` - builds without errors
2. `cd web && npm run dev` - starts dev server (just verify it starts, then kill it)
3. Git commit: "feat: implement React dashboard with trace viewer, policy editor, and replay controls"
