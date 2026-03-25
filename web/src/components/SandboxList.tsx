import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import {
  Plus,
  Play,
  Square,
  Trash2,
  X,
} from "lucide-react";
import {
  listSandboxes,
  createSandbox,
  startSandbox,
  stopSandbox,
  destroySandbox,
} from "../lib/api";
import type { Sandbox, SandboxConfig, SandboxStatus } from "../lib/api";

function statusBadge(status: SandboxStatus) {
  const styles: Record<SandboxStatus, string> = {
    created: "bg-gray-600 text-gray-200",
    running: "bg-green-600/20 text-green-400 border border-green-500/30",
    stopped: "bg-yellow-600/20 text-yellow-400 border border-yellow-500/30",
    destroyed: "bg-red-600/20 text-red-400 border border-red-500/30",
  };
  return (
    <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${styles[status]}`}>
      {status}
    </span>
  );
}

function CreateModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const [policy, setPolicy] = useState("default");

  const create = useMutation({
    mutationFn: (cfg: SandboxConfig) => createSandbox(cfg),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sandboxes"] });
      onClose();
    },
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="bg-gray-800 rounded-xl border border-gray-700 w-full max-w-md mx-4">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-700">
          <h2 className="text-lg font-semibold">Create Sandbox</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-100">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form
          className="p-5 space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            create.mutate({ name, policy });
          }}
        >
          <div>
            <label className="block text-sm text-gray-400 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              placeholder="my-sandbox"
              className="w-full px-3 py-2 bg-gray-900 border border-gray-600 rounded-lg text-sm focus:outline-none focus:border-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Policy</label>
            <select
              value={policy}
              onChange={(e) => setPolicy(e.target.value)}
              className="w-full px-3 py-2 bg-gray-900 border border-gray-600 rounded-lg text-sm focus:outline-none focus:border-blue-500"
            >
              <option value="default">Default (Restrictive)</option>
              <option value="permissive">Permissive</option>
              <option value="strict">Strict</option>
              <option value="coding-agent">Coding Agent</option>
            </select>
          </div>
          {create.isError && (
            <p className="text-sm text-red-400">
              Failed to create sandbox. Is the API server running?
            </p>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={create.isPending}
              className="px-4 py-2 text-sm bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50"
            >
              {create.isPending ? "Creating..." : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function SandboxList() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [showCreate, setShowCreate] = useState(searchParams.get("create") === "1");
  const qc = useQueryClient();

  const sandboxes = useQuery({
    queryKey: ["sandboxes"],
    queryFn: listSandboxes,
    refetchInterval: 5000,
  });

  const start = useMutation({
    mutationFn: startSandbox,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sandboxes"] }),
  });
  const stop = useMutation({
    mutationFn: stopSandbox,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sandboxes"] }),
  });
  const destroy = useMutation({
    mutationFn: destroySandbox,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sandboxes"] }),
  });

  function openCreate() {
    setShowCreate(true);
    setSearchParams({});
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Sandboxes</h1>
        <button
          onClick={openCreate}
          className="flex items-center gap-1.5 px-3 py-2 text-sm bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
        >
          <Plus className="w-4 h-4" />
          New Sandbox
        </button>
      </div>

      {showCreate && <CreateModal onClose={() => setShowCreate(false)} />}

      <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="text-xs text-gray-400 uppercase bg-gray-800/50 border-b border-gray-700">
              <tr>
                <th className="px-5 py-3">Name</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Actions</th>
                <th className="px-5 py-3">Created</th>
                <th className="px-5 py-3 text-right">Controls</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700">
              {sandboxes.isLoading && (
                <tr>
                  <td colSpan={5} className="px-5 py-8 text-center text-gray-500">
                    Loading...
                  </td>
                </tr>
              )}
              {sandboxes.isError && (
                <tr>
                  <td colSpan={5} className="px-5 py-8 text-center text-gray-500">
                    Unable to load sandboxes. Is the API server running?
                  </td>
                </tr>
              )}
              {sandboxes.data?.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-5 py-8 text-center text-gray-500">
                    No sandboxes yet. Create one to get started.
                  </td>
                </tr>
              )}
              {sandboxes.data?.map((sb: Sandbox) => (
                <tr key={sb.id} className="hover:bg-gray-700/50">
                  <td className="px-5 py-3">
                    <Link
                      to={`/sandboxes/${sb.id}`}
                      className="text-blue-400 hover:underline font-medium"
                    >
                      {sb.name}
                    </Link>
                  </td>
                  <td className="px-5 py-3">{statusBadge(sb.status)}</td>
                  <td className="px-5 py-3 text-gray-300">
                    {sb.action_count}
                    {sb.denied_count > 0 && (
                      <span className="ml-1 text-red-400">
                        ({sb.denied_count} denied)
                      </span>
                    )}
                  </td>
                  <td className="px-5 py-3 text-gray-400">
                    {new Date(sb.created_at).toLocaleString()}
                  </td>
                  <td className="px-5 py-3">
                    <div className="flex items-center justify-end gap-1">
                      {sb.status !== "running" && sb.status !== "destroyed" && (
                        <button
                          onClick={() => start.mutate(sb.id)}
                          title="Start"
                          className="p-1.5 rounded hover:bg-gray-600 text-green-400"
                        >
                          <Play className="w-4 h-4" />
                        </button>
                      )}
                      {sb.status === "running" && (
                        <button
                          onClick={() => stop.mutate(sb.id)}
                          title="Stop"
                          className="p-1.5 rounded hover:bg-gray-600 text-yellow-400"
                        >
                          <Square className="w-4 h-4" />
                        </button>
                      )}
                      {sb.status !== "destroyed" && (
                        <button
                          onClick={() => {
                            if (confirm(`Destroy sandbox "${sb.name}"?`)) {
                              destroy.mutate(sb.id);
                            }
                          }}
                          title="Destroy"
                          className="p-1.5 rounded hover:bg-gray-600 text-red-400"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
