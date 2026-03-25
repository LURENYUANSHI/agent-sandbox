import { useState, useEffect, useRef, useCallback } from "react";
import {
  Play,
  Pause,
  SkipForward,
  SkipBack,
  Rewind,
  GitBranch,
} from "lucide-react";
import type { TraceEvent } from "../lib/api";

// ─── Types ───────────────────────────────────────────────────────────────────

interface ReplayControlsProps {
  events: TraceEvent[];
}

const speeds = [0.5, 1, 2, 5] as const;

function effectColor(effect: string): string {
  switch (effect) {
    case "allow":
      return "text-green-400";
    case "deny":
      return "text-red-400";
    case "audit":
      return "text-yellow-400";
    default:
      return "text-gray-400";
  }
}

function effectBg(effect: string): string {
  switch (effect) {
    case "allow":
      return "bg-green-500";
    case "deny":
      return "bg-red-500";
    case "audit":
      return "bg-yellow-500";
    default:
      return "bg-gray-500";
  }
}

// ─── Component ───────────────────────────────────────────────────────────────

export default function ReplayControls({ events }: ReplayControlsProps) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [speed, setSpeed] = useState<number>(1);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const total = events.length;
  const current = total > 0 ? events[currentIndex] : null;

  const stop = useCallback(() => {
    setPlaying(false);
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const tick = useCallback(() => {
    setCurrentIndex((prev) => {
      if (prev >= total - 1) {
        stop();
        return prev;
      }
      return prev + 1;
    });
  }, [total, stop]);

  useEffect(() => {
    if (playing && total > 0) {
      const ms = 1000 / speed;
      timerRef.current = setTimeout(tick, ms);
    }
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [playing, currentIndex, speed, tick, total]);

  function togglePlay() {
    if (playing) {
      stop();
    } else if (currentIndex >= total - 1) {
      setCurrentIndex(0);
      setPlaying(true);
    } else {
      setPlaying(true);
    }
  }

  function stepForward() {
    stop();
    setCurrentIndex((i) => Math.min(i + 1, total - 1));
  }

  function stepBack() {
    stop();
    setCurrentIndex((i) => Math.max(i - 1, 0));
  }

  function rewind() {
    stop();
    setCurrentIndex(0);
  }

  const progress = total > 0 ? ((currentIndex + 1) / total) * 100 : 0;

  if (total === 0) {
    return (
      <div className="bg-gray-800 rounded-xl border border-gray-700 p-8 text-center text-gray-500">
        No trace events to replay
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Transport controls */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 p-4">
        <div className="flex items-center justify-center gap-3 mb-4">
          <button
            onClick={rewind}
            className="p-2 rounded-lg hover:bg-gray-700 text-gray-300 transition-colors"
            title="Rewind"
          >
            <Rewind className="w-5 h-5" />
          </button>
          <button
            onClick={stepBack}
            className="p-2 rounded-lg hover:bg-gray-700 text-gray-300 transition-colors"
            title="Step back"
          >
            <SkipBack className="w-5 h-5" />
          </button>
          <button
            onClick={togglePlay}
            className="p-3 rounded-full bg-blue-600 hover:bg-blue-500 text-white transition-colors"
            title={playing ? "Pause" : "Play"}
          >
            {playing ? (
              <Pause className="w-6 h-6" />
            ) : (
              <Play className="w-6 h-6" />
            )}
          </button>
          <button
            onClick={stepForward}
            className="p-2 rounded-lg hover:bg-gray-700 text-gray-300 transition-colors"
            title="Step forward"
          >
            <SkipForward className="w-5 h-5" />
          </button>
        </div>

        {/* Progress bar */}
        <div className="mb-3">
          <div
            className="w-full h-2 bg-gray-700 rounded-full overflow-hidden cursor-pointer"
            onClick={(e) => {
              const rect = e.currentTarget.getBoundingClientRect();
              const pct = (e.clientX - rect.left) / rect.width;
              const idx = Math.round(pct * (total - 1));
              stop();
              setCurrentIndex(Math.max(0, Math.min(idx, total - 1)));
            }}
          >
            <div
              className="h-full bg-blue-500 rounded-full transition-all duration-150"
              style={{ width: `${progress}%` }}
            />
          </div>
          <div className="flex justify-between mt-1 text-xs text-gray-500">
            <span>
              {currentIndex + 1} / {total}
            </span>
            <span>{current?.timestamp ? new Date(current.timestamp).toLocaleTimeString() : ""}</span>
          </div>
        </div>

        {/* Speed control */}
        <div className="flex items-center justify-center gap-2">
          <span className="text-xs text-gray-400">Speed:</span>
          {speeds.map((s) => (
            <button
              key={s}
              onClick={() => setSpeed(s)}
              className={`px-2 py-1 text-xs rounded ${
                speed === s
                  ? "bg-blue-600 text-white"
                  : "bg-gray-700 text-gray-300 hover:bg-gray-600"
              }`}
            >
              {s}x
            </button>
          ))}
        </div>
      </div>

      {/* Current event detail */}
      {current && (
        <div className="bg-gray-800 rounded-xl border border-gray-700 p-5">
          <div className="flex items-center gap-3 mb-3">
            <span className={`text-lg font-bold ${effectColor(current.effect)}`}>
              {current.effect.toUpperCase()}
            </span>
            <span className="text-sm text-gray-400">
              {current.event_type} / {current.action_type ?? "info"}
            </span>
            <span className="text-xs text-gray-500 ml-auto">
              {current.duration_ms}ms
            </span>
          </div>
          <p className="text-sm text-gray-200 mb-4">{current.action_detail}</p>

          {/* Decision tree visualization */}
          <div className="bg-gray-900 rounded-lg p-4 border border-gray-700">
            <div className="flex items-center gap-2 mb-2 text-xs text-gray-400">
              <GitBranch className="w-4 h-4" />
              <span>Policy Decision</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-blue-500" />
              <span className="text-xs text-gray-300">Action: {current.action_type ?? "unknown"}</span>
            </div>
            <div className="ml-1.5 border-l border-gray-600 pl-4 my-1">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-gray-500" />
                <span className="text-xs text-gray-300">
                  Resource: {current.action_detail}
                </span>
              </div>
            </div>
            <div className="ml-1.5 border-l border-gray-600 pl-4 my-1">
              <div className="flex items-center gap-2">
                <div className={`w-3 h-3 rounded-full ${effectBg(current.effect)}`} />
                <span className={`text-xs font-medium ${effectColor(current.effect)}`}>
                  Result: {current.effect}
                </span>
              </div>
            </div>
          </div>

          {/* Raw JSON */}
          {current.attributes && Object.keys(current.attributes).length > 0 && (
            <details className="mt-3">
              <summary className="text-xs text-gray-400 cursor-pointer hover:text-gray-300">
                Attributes
              </summary>
              <pre className="mt-2 p-3 bg-gray-900 rounded-lg text-xs font-mono text-gray-300 overflow-x-auto">
                {JSON.stringify(current.attributes, null, 2)}
              </pre>
            </details>
          )}
        </div>
      )}
    </div>
  );
}
