You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read existing code in pkg/api/handlers.go (WebSocket handler) and web/src/ components.

## Your Task: Connect WebSocket real-time trace streaming to React frontend

### 1. Review existing WebSocket backend
Read pkg/api/handlers.go to understand the existing WS endpoint at `/api/v1/sandboxes/:id/ws`.

### 2. web/src/hooks/useTraceStream.ts
Create a custom React hook for WebSocket trace streaming:
```typescript
export function useTraceStream(sandboxId: string | null) {
  // Connect to ws://localhost:8080/api/v1/sandboxes/{id}/ws
  // Return: { events: TraceEvent[], isConnected: boolean, error: string | null }
  // Auto-reconnect on disconnect
  // Clean up on unmount
}
```

### 3. Update web/src/components/TraceViewer.tsx
- Import and use `useTraceStream` hook
- Show real-time events as they arrive via WebSocket
- Show connection status indicator (green dot = connected, red = disconnected)
- New events should appear at the top of the list with a subtle animation/highlight

### 4. Update web/src/components/Dashboard.tsx
- Use WebSocket or polling to keep recent activity updated
- Show a "live" indicator when connected

### 5. web/src/components/ConnectionStatus.tsx
Create a small reusable component:
- Green dot + "Connected" when WS is open
- Red dot + "Disconnected" when WS is closed
- Yellow dot + "Connecting..." when connecting

### Verification:
1. `cd web && npm run build` (verify TypeScript compiles)
2. Commit: `feat(web): integrate WebSocket real-time trace streaming with React components`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
