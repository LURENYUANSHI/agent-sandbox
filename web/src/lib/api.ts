// API client for AgentSandbox backend

const BASE_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1";

// ─── Types ───────────────────────────────────────────────────────────────────

export type SandboxStatus = "created" | "running" | "stopped" | "destroyed";

export type ActionType = "file" | "network" | "process" | "shell";

export type Effect = "allow" | "deny" | "audit";

export interface SandboxConfig {
  name: string;
  policy?: string;
  working_dir?: string;
  resource_limits?: ResourceLimits;
  network_rules?: NetworkRule[];
  allowed_paths?: string[];
}

export interface ResourceLimits {
  max_memory_mb: number;
  max_cpu_percent: number;
  max_processes: number;
  max_open_files: number;
  timeout_seconds: number;
}

export interface NetworkRule {
  host: string;
  port: number;
  protocol: string;
  effect: Effect;
}

export interface Sandbox {
  id: string;
  name: string;
  status: SandboxStatus;
  config: SandboxConfig;
  created_at: string;
  started_at?: string;
  stopped_at?: string;
  action_count: number;
  denied_count: number;
}

export interface Action {
  type: ActionType;
  resource: string;
  operation: string;
  params?: Record<string, unknown>;
}

export interface ActionResult {
  id: string;
  sandbox_id: string;
  action: Action;
  allowed: boolean;
  effect: Effect;
  matched_rule?: string;
  output?: string;
  error?: string;
  duration_ms: number;
  timestamp: string;
}

export interface TraceEvent {
  id: string;
  sandbox_id: string;
  parent_id?: string;
  span_id: string;
  trace_id: string;
  event_type: string;
  action_type?: ActionType;
  action_detail: string;
  effect: Effect;
  duration_ms: number;
  timestamp: string;
  children?: TraceEvent[];
  attributes?: Record<string, unknown>;
}

export interface PolicyRule {
  name: string;
  description?: string;
  action_type: ActionType;
  resource_pattern: string;
  operations?: string[];
  effect: Effect;
  priority?: number;
}

export interface Policy {
  name: string;
  description?: string;
  version: string;
  rules: PolicyRule[];
}

export interface ReplaySession {
  id: string;
  sandbox_id: string;
  trace_id: string;
  total_events: number;
  current_index: number;
  status: "playing" | "paused" | "stopped";
  speed: number;
}

export interface DashboardStats {
  active_sandboxes: number;
  total_actions: number;
  denied_actions: number;
  avg_response_ms: number;
}

export interface WebSocketEvent {
  type: string;
  sandbox_id: string;
  event: TraceEvent;
}

// ─── Error handling ──────────────────────────────────────────────────────────

export class ApiError extends Error {
  status: number;
  statusText: string;
  body?: unknown;

  constructor(status: number, statusText: string, body?: unknown) {
    super(`API Error ${status}: ${statusText}`);
    this.name = "ApiError";
    this.status = status;
    this.statusText = statusText;
    this.body = body;
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (!res.ok) {
    let body: unknown;
    try {
      body = await res.json();
    } catch {
      /* empty */
    }
    throw new ApiError(res.status, res.statusText, body);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// ─── Sandbox endpoints ──────────────────────────────────────────────────────

export function listSandboxes(): Promise<Sandbox[]> {
  return request<Sandbox[]>("/sandboxes");
}

export function createSandbox(config: SandboxConfig): Promise<Sandbox> {
  return request<Sandbox>("/sandboxes", {
    method: "POST",
    body: JSON.stringify(config),
  });
}

export function getSandbox(id: string): Promise<Sandbox> {
  return request<Sandbox>(`/sandboxes/${encodeURIComponent(id)}`);
}

export function startSandbox(id: string): Promise<Sandbox> {
  return request<Sandbox>(`/sandboxes/${encodeURIComponent(id)}/start`, {
    method: "POST",
  });
}

export function stopSandbox(id: string): Promise<Sandbox> {
  return request<Sandbox>(`/sandboxes/${encodeURIComponent(id)}/stop`, {
    method: "POST",
  });
}

export function destroySandbox(id: string): Promise<void> {
  return request<void>(`/sandboxes/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

// ─── Action / Trace endpoints ───────────────────────────────────────────────

export function executeAction(
  id: string,
  action: Action,
): Promise<ActionResult> {
  return request<ActionResult>(
    `/sandboxes/${encodeURIComponent(id)}/exec`,
    { method: "POST", body: JSON.stringify(action) },
  );
}

export function getTraces(id: string): Promise<TraceEvent[]> {
  return request<TraceEvent[]>(
    `/sandboxes/${encodeURIComponent(id)}/traces`,
  );
}

export function startReplay(id: string): Promise<ReplaySession> {
  return request<ReplaySession>(
    `/sandboxes/${encodeURIComponent(id)}/replay`,
    { method: "POST" },
  );
}

// ─── Dashboard ──────────────────────────────────────────────────────────────

export function getDashboardStats(): Promise<DashboardStats> {
  return request<DashboardStats>("/dashboard/stats");
}

export function getRecentActivity(): Promise<TraceEvent[]> {
  return request<TraceEvent[]>("/dashboard/activity");
}

// ─── WebSocket ──────────────────────────────────────────────────────────────

export function connectWebSocket(
  sandboxId: string,
  onEvent: (event: WebSocketEvent) => void,
): () => void {
  const wsBase = BASE_URL.replace(/^http/, "ws").replace(/\/api\/v1$/, "");
  const ws = new WebSocket(
    `${wsBase}/ws/sandboxes/${encodeURIComponent(sandboxId)}/traces`,
  );

  ws.onmessage = (msg) => {
    try {
      const data = JSON.parse(msg.data as string) as WebSocketEvent;
      onEvent(data);
    } catch {
      /* ignore malformed messages */
    }
  };

  // Return cleanup function
  return () => {
    ws.close();
  };
}
