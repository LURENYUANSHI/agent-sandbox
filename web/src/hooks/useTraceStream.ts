import { useEffect, useRef, useState, useCallback } from "react";
import type { TraceEvent } from "../lib/api";

const BASE_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1";

function buildWsUrl(sandboxId: string): string {
  const wsBase = BASE_URL.replace(/^http/, "ws");
  return `${wsBase}/sandboxes/${encodeURIComponent(sandboxId)}/ws`;
}

const RECONNECT_DELAY_MS = 3000;
const MAX_EVENTS = 500;

export type ConnectionState = "connecting" | "connected" | "disconnected";

export interface UseTraceStreamReturn {
  events: TraceEvent[];
  isConnected: boolean;
  connectionState: ConnectionState;
  error: string | null;
}

export function useTraceStream(
  sandboxId: string | null,
): UseTraceStreamReturn {
  const [events, setEvents] = useState<TraceEvent[]>([]);
  const [connectionState, setConnectionState] =
    useState<ConnectionState>("disconnected");
  const [error, setError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mountedRef = useRef(true);

  const cleanup = useCallback(() => {
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current);
      reconnectTimer.current = null;
    }
    if (wsRef.current) {
      wsRef.current.onopen = null;
      wsRef.current.onclose = null;
      wsRef.current.onerror = null;
      wsRef.current.onmessage = null;
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  const connect = useCallback(
    (id: string) => {
      cleanup();
      if (!mountedRef.current) return;

      setConnectionState("connecting");
      setError(null);

      const ws = new WebSocket(buildWsUrl(id));
      wsRef.current = ws;

      ws.onopen = () => {
        if (!mountedRef.current) return;
        setConnectionState("connected");
        setError(null);
      };

      ws.onmessage = (msg) => {
        if (!mountedRef.current) return;
        try {
          const event = JSON.parse(msg.data as string) as TraceEvent;
          setEvents((prev) => [event, ...prev].slice(0, MAX_EVENTS));
        } catch {
          /* ignore malformed messages */
        }
      };

      ws.onclose = () => {
        if (!mountedRef.current) return;
        setConnectionState("disconnected");
        // Auto-reconnect
        reconnectTimer.current = setTimeout(() => {
          if (mountedRef.current) connect(id);
        }, RECONNECT_DELAY_MS);
      };

      ws.onerror = () => {
        if (!mountedRef.current) return;
        setError("WebSocket connection error");
      };
    },
    [cleanup],
  );

  useEffect(() => {
    mountedRef.current = true;

    if (sandboxId) {
      setEvents([]);
      connect(sandboxId);
    } else {
      cleanup();
      setConnectionState("disconnected");
    }

    return () => {
      mountedRef.current = false;
      cleanup();
    };
  }, [sandboxId, connect, cleanup]);

  return {
    events,
    isConnected: connectionState === "connected",
    connectionState,
    error,
  };
}
