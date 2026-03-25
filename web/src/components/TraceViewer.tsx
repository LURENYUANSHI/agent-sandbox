import { useState, useMemo } from "react";
import {
  ChevronRight,
  ChevronDown,
  File,
  Globe,
  Terminal,
  Cpu,
  Info,
  type LucideProps,
} from "lucide-react";
import type { TraceEvent, ActionType, Effect } from "../lib/api";
import { buildTree } from "../lib/trace-utils";
import { useTraceStream } from "../hooks/useTraceStream";
import ConnectionStatus from "./ConnectionStatus";

// ─── Helpers ─────────────────────────────────────────────────────────────────

function ActionIcon({ type, ...props }: { type?: ActionType } & LucideProps) {
  switch (type) {
    case "file": return <File {...props} />;
    case "network": return <Globe {...props} />;
    case "shell": return <Terminal {...props} />;
    case "process": return <Cpu {...props} />;
    default: return <Info {...props} />;
  }
}

function effectStyles(effect: Effect): string {
  switch (effect) {
    case "allow":
      return "bg-green-500/20 text-green-400 border-green-500/30";
    case "deny":
      return "bg-red-500/20 text-red-400 border-red-500/30";
    case "audit":
      return "bg-yellow-500/20 text-yellow-400 border-yellow-500/30";
  }
}

function effectBarColor(effect: Effect): string {
  switch (effect) {
    case "allow":
      return "bg-green-500";
    case "deny":
      return "bg-red-500";
    case "audit":
      return "bg-yellow-500";
  }
}

// ─── Filter bar ──────────────────────────────────────────────────────────────

const eventTypes = ["all", "file", "network", "process", "shell"] as const;
const effectTypes = ["all", "allow", "deny", "audit"] as const;

interface FilterBarProps {
  eventType: string;
  effectType: string;
  onEventType: (v: string) => void;
  onEffectType: (v: string) => void;
}

function FilterBar({ eventType, effectType, onEventType, onEffectType }: FilterBarProps) {
  return (
    <div className="flex flex-wrap gap-4 mb-4">
      <div className="flex items-center gap-2">
        <span className="text-xs text-gray-400 uppercase">Action:</span>
        {eventTypes.map((t) => (
          <button
            key={t}
            onClick={() => onEventType(t)}
            className={`px-2 py-1 text-xs rounded ${
              eventType === t
                ? "bg-blue-600 text-white"
                : "bg-gray-700 text-gray-300 hover:bg-gray-600"
            }`}
          >
            {t}
          </button>
        ))}
      </div>
      <div className="flex items-center gap-2">
        <span className="text-xs text-gray-400 uppercase">Effect:</span>
        {effectTypes.map((t) => (
          <button
            key={t}
            onClick={() => onEffectType(t)}
            className={`px-2 py-1 text-xs rounded ${
              effectType === t
                ? "bg-blue-600 text-white"
                : "bg-gray-700 text-gray-300 hover:bg-gray-600"
            }`}
          >
            {t}
          </button>
        ))}
      </div>
    </div>
  );
}

// ─── Tree node ───────────────────────────────────────────────────────────────

interface TreeNodeProps {
  event: TraceEvent;
  maxDuration: number;
  depth: number;
}

function TreeNode({ event, maxDuration, depth }: TreeNodeProps) {
  const [expanded, setExpanded] = useState(depth < 2);
  const [showDetail, setShowDetail] = useState(false);
  const hasChildren = event.children && event.children.length > 0;
  const barWidth = maxDuration > 0 ? (event.duration_ms / maxDuration) * 100 : 0;

  return (
    <div>
      <div
        className="flex items-center gap-2 py-1.5 px-2 hover:bg-gray-700/50 rounded cursor-pointer group"
        style={{ paddingLeft: `${depth * 20 + 8}px` }}
      >
        {/* Expand toggle */}
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-4 h-4 flex items-center justify-center text-gray-500"
        >
          {hasChildren ? (
            expanded ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronRight className="w-4 h-4" />
            )
          ) : (
            <span className="w-4" />
          )}
        </button>

        {/* Icon + Effect badge */}
        <ActionIcon type={event.action_type} className="w-4 h-4 text-gray-400 shrink-0" />
        <span
          className={`px-1.5 py-0.5 text-[10px] font-semibold uppercase rounded border ${effectStyles(event.effect)}`}
        >
          {event.effect}
        </span>

        {/* Detail */}
        <button
          onClick={() => setShowDetail(!showDetail)}
          className="flex-1 text-left text-sm truncate text-gray-200"
        >
          {event.action_detail}
        </button>

        {/* Duration bar */}
        <div className="w-24 h-2 bg-gray-700 rounded-full overflow-hidden shrink-0">
          <div
            className={`h-full rounded-full ${effectBarColor(event.effect)}`}
            style={{ width: `${Math.max(barWidth, 2)}%` }}
          />
        </div>
        <span className="text-xs text-gray-500 w-14 text-right shrink-0">
          {event.duration_ms}ms
        </span>
      </div>

      {/* JSON detail */}
      {showDetail && (
        <div
          className="mx-2 mb-2 p-3 bg-gray-900 rounded-lg border border-gray-700 text-xs font-mono overflow-x-auto"
          style={{ marginLeft: `${depth * 20 + 28}px` }}
        >
          <pre className="text-gray-300 whitespace-pre-wrap">
            {JSON.stringify(event, null, 2)}
          </pre>
        </div>
      )}

      {/* Children */}
      {expanded &&
        hasChildren &&
        event.children!.map((child) => (
          <TreeNode
            key={child.id}
            event={child}
            maxDuration={maxDuration}
            depth={depth + 1}
          />
        ))}
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

interface TraceViewerProps {
  events: TraceEvent[];
  sandboxId?: string | null;
}

export default function TraceViewer({ events, sandboxId }: TraceViewerProps) {
  const stream = useTraceStream(sandboxId ?? null);
  const [eventType, setEventType] = useState("all");
  const [effectType, setEffectType] = useState("all");

  // Merge: live events first (newest on top), then historical
  const merged = useMemo(() => {
    const liveIds = new Set(stream.events.map((e) => e.id));
    const deduped = events.filter((e) => !liveIds.has(e.id));
    return [...stream.events, ...deduped];
  }, [stream.events, events]);

  const filtered = merged.filter((ev) => {
    if (eventType !== "all" && ev.action_type !== eventType) return false;
    if (effectType !== "all" && ev.effect !== effectType) return false;
    return true;
  });

  const tree = buildTree(filtered);
  const maxDuration = Math.max(...filtered.map((e) => e.duration_ms), 1);

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <FilterBar
          eventType={eventType}
          effectType={effectType}
          onEventType={setEventType}
          onEffectType={setEffectType}
        />
        {sandboxId && <ConnectionStatus state={stream.connectionState} />}
      </div>
      {stream.error && (
        <p className="mb-2 text-xs text-red-400">{stream.error}</p>
      )}
      <div className="bg-gray-800 rounded-xl border border-gray-700 py-2">
        {tree.length === 0 && (
          <p className="px-5 py-8 text-center text-gray-500">No trace events</p>
        )}
        {tree.map((ev, i) => (
          <div
            key={ev.id}
            className={
              i < stream.events.length
                ? "animate-[fadeIn_0.3s_ease-in-out]"
                : ""
            }
          >
            <TreeNode event={ev} maxDuration={maxDuration} depth={0} />
          </div>
        ))}
      </div>
    </div>
  );
}
