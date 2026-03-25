import { useParams, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { getTraces } from "../lib/api";
import ReplayControls from "./ReplayControls";

export default function ReplayView() {
  const { id } = useParams<{ id: string }>();

  const traces = useQuery({
    queryKey: ["traces", id],
    queryFn: () => getTraces(id!),
    enabled: !!id,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to={`/sandboxes/${id}`}
          className="p-2 rounded-lg hover:bg-gray-800 text-gray-400 transition-colors"
        >
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <h1 className="text-2xl font-bold">Trace Replay</h1>
      </div>

      {traces.isLoading && <p className="text-gray-500">Loading traces...</p>}
      {traces.isError && (
        <p className="text-gray-500">
          Unable to load traces. Is the API server running?
        </p>
      )}
      {traces.data && <ReplayControls events={traces.data} />}
    </div>
  );
}
