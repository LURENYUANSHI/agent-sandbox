import { useParams, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Play, Eye } from "lucide-react";
import { getSandbox, getTraces } from "../lib/api";
import TraceViewer from "./TraceViewer";

export default function SandboxDetail() {
  const { id } = useParams<{ id: string }>();

  const sandbox = useQuery({
    queryKey: ["sandbox", id],
    queryFn: () => getSandbox(id!),
    enabled: !!id,
    refetchInterval: 5000,
  });

  const traces = useQuery({
    queryKey: ["traces", id],
    queryFn: () => getTraces(id!),
    enabled: !!id,
    refetchInterval: 5000,
  });

  const sb = sandbox.data;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to="/sandboxes"
          className="p-2 rounded-lg hover:bg-gray-800 text-gray-400 transition-colors"
        >
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold">{sb?.name ?? "Sandbox"}</h1>
          {sb && (
            <p className="text-sm text-gray-400">
              {sb.status} &middot; {sb.action_count} actions &middot; ID: {sb.id}
            </p>
          )}
        </div>
        {id && (
          <Link
            to={`/sandboxes/${id}/replay`}
            className="flex items-center gap-1.5 px-3 py-2 text-sm bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
          >
            <Play className="w-4 h-4" />
            Replay
          </Link>
        )}
      </div>

      {sandbox.isLoading && (
        <p className="text-gray-500">Loading sandbox...</p>
      )}
      {sandbox.isError && (
        <p className="text-gray-500">
          Unable to load sandbox. Is the API server running?
        </p>
      )}

      {/* Sandbox info */}
      {sb && (
        <div className="bg-gray-800 rounded-xl border border-gray-700 p-5">
          <h2 className="text-sm font-semibold text-gray-300 mb-3 flex items-center gap-2">
            <Eye className="w-4 h-4" />
            Details
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-gray-500">Status</span>
              <p className="font-medium">{sb.status}</p>
            </div>
            <div>
              <span className="text-gray-500">Actions</span>
              <p className="font-medium">{sb.action_count}</p>
            </div>
            <div>
              <span className="text-gray-500">Denied</span>
              <p className="font-medium text-red-400">{sb.denied_count}</p>
            </div>
            <div>
              <span className="text-gray-500">Created</span>
              <p className="font-medium">{new Date(sb.created_at).toLocaleString()}</p>
            </div>
          </div>
        </div>
      )}

      {/* Trace viewer */}
      <div>
        <h2 className="text-lg font-semibold mb-3">Trace Events</h2>
        {traces.isLoading && <p className="text-gray-500">Loading traces...</p>}
        {traces.isError && (
          <p className="text-gray-500">Unable to load traces.</p>
        )}
        {traces.data && <TraceViewer events={traces.data} />}
      </div>
    </div>
  );
}
