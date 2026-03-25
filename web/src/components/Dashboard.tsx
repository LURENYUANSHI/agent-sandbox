import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  Box,
  Activity,
  ShieldOff,
  Clock,
  Plus,
  List,
  Radio,
} from "lucide-react";
import { getDashboardStats, getRecentActivity } from "../lib/api";
import type { DashboardStats, ActivityEvent } from "../lib/api";

function StatCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: typeof Box;
  label: string;
  value: string | number;
  color: string;
}) {
  return (
    <div className="bg-gray-800 rounded-xl p-5 border border-gray-700">
      <div className="flex items-center gap-3 mb-3">
        <div className={`p-2 rounded-lg ${color}`}>
          <Icon className="w-5 h-5" />
        </div>
        <span className="text-sm text-gray-400">{label}</span>
      </div>
      <p className="text-2xl font-bold">{value}</p>
    </div>
  );
}

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

function formatTime(ts: string): string {
  return new Date(ts).toLocaleTimeString();
}

const defaultStats: DashboardStats = {
  active_sandboxes: 0,
  total_actions: 0,
  denied_actions: 0,
  avg_response_ms: 0,
};

export default function Dashboard() {
  const stats = useQuery({
    queryKey: ["dashboard-stats"],
    queryFn: getDashboardStats,
    refetchInterval: 5000,
  });

  const activity = useQuery({
    queryKey: ["dashboard-activity"],
    queryFn: getRecentActivity,
    refetchInterval: 5000,
  });

  const s = stats.data ?? defaultStats;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <div className="flex gap-2">
          <Link
            to="/sandboxes"
            className="flex items-center gap-1.5 px-3 py-2 text-sm bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          >
            <List className="w-4 h-4" />
            View All
          </Link>
          <Link
            to="/sandboxes?create=1"
            className="flex items-center gap-1.5 px-3 py-2 text-sm bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            New Sandbox
          </Link>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          icon={Box}
          label="Active Sandboxes"
          value={s.active_sandboxes}
          color="bg-blue-500/20 text-blue-400"
        />
        <StatCard
          icon={Activity}
          label="Total Actions"
          value={s.total_actions}
          color="bg-green-500/20 text-green-400"
        />
        <StatCard
          icon={ShieldOff}
          label="Denied Actions"
          value={s.denied_actions}
          color="bg-red-500/20 text-red-400"
        />
        <StatCard
          icon={Clock}
          label="Avg Response"
          value={`${s.avg_response_ms.toFixed(1)} ms`}
          color="bg-yellow-500/20 text-yellow-400"
        />
      </div>

      {/* Recent Activity */}
      <div className="bg-gray-800 rounded-xl border border-gray-700">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-700">
          <h2 className="text-lg font-semibold">Recent Activity</h2>
          {!activity.isError && (
            <span className="inline-flex items-center gap-1.5 text-xs text-gray-400">
              <Radio className="w-3 h-3 text-green-400 animate-pulse" />
              Live
            </span>
          )}
        </div>
        <div className="divide-y divide-gray-700">
          {activity.isLoading && (
            <p className="px-5 py-8 text-center text-gray-500">Loading...</p>
          )}
          {activity.isError && (
            <p className="px-5 py-8 text-center text-gray-500">
              Unable to load activity. Is the API server running?
            </p>
          )}
          {activity.data?.length === 0 && (
            <p className="px-5 py-8 text-center text-gray-500">
              No recent activity
            </p>
          )}
          {activity.data?.slice(0, 20).map((ev: ActivityEvent) => (
            <div
              key={ev.id}
              className="flex items-center gap-3 px-5 py-3 text-sm"
            >
              <span className={`font-mono ${effectColor(ev.effect)}`}>
                {ev.effect.toUpperCase()}
              </span>
              <span className="flex-1 truncate text-gray-300">
                {ev.action_detail}
              </span>
              <span className="text-gray-500 text-xs whitespace-nowrap">
                {ev.duration_ms}ms
              </span>
              <span className="text-gray-500 text-xs whitespace-nowrap">
                {formatTime(ev.timestamp)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
