import type { ConnectionState } from "../hooks/useTraceStream";

interface ConnectionStatusProps {
  state: ConnectionState;
}

const config: Record<ConnectionState, { color: string; label: string }> = {
  connected: { color: "bg-green-500", label: "Connected" },
  disconnected: { color: "bg-red-500", label: "Disconnected" },
  connecting: { color: "bg-yellow-500", label: "Connecting..." },
};

export default function ConnectionStatus({ state }: ConnectionStatusProps) {
  const { color, label } = config[state];
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-gray-400">
      <span
        className={`w-2 h-2 rounded-full ${color} ${state === "connecting" ? "animate-pulse" : ""}`}
      />
      {label}
    </span>
  );
}
