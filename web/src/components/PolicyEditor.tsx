import { useState, useMemo } from "react";
import { Plus, Trash2, Check, Upload, Download } from "lucide-react";
import type { PolicyRule, Policy, ActionType, Effect } from "../lib/api";

// ─── YAML generation ─────────────────────────────────────────────────────────

function toYaml(policy: Policy): string {
  const lines: string[] = [];
  lines.push(`name: ${policy.name}`);
  if (policy.description) lines.push(`description: ${policy.description}`);
  lines.push(`version: "${policy.version}"`);
  lines.push("rules:");
  for (const rule of policy.rules) {
    lines.push(`  - name: ${rule.name}`);
    if (rule.description) lines.push(`    description: ${rule.description}`);
    lines.push(`    action_type: ${rule.action_type}`);
    lines.push(`    resource_pattern: "${rule.resource_pattern}"`);
    if (rule.operations && rule.operations.length > 0) {
      lines.push(`    operations: [${rule.operations.map((o) => `"${o}"`).join(", ")}]`);
    }
    lines.push(`    effect: ${rule.effect}`);
    if (rule.priority !== undefined) lines.push(`    priority: ${rule.priority}`);
  }
  return lines.join("\n") + "\n";
}

// ─── Defaults ────────────────────────────────────────────────────────────────

const defaultRule: PolicyRule = {
  name: "",
  action_type: "file",
  resource_pattern: "*",
  effect: "deny",
  operations: [],
  priority: 0,
};

const defaultPolicy: Policy = {
  name: "custom-policy",
  description: "Custom security policy",
  version: "1.0",
  rules: [],
};

const actionTypes: ActionType[] = ["file", "network", "process", "shell"];
const effects: Effect[] = ["allow", "deny", "audit"];

// ─── Rule form ───────────────────────────────────────────────────────────────

interface RuleFormProps {
  rule: PolicyRule;
  index: number;
  onChange: (index: number, rule: PolicyRule) => void;
  onDelete: (index: number) => void;
}

function RuleForm({ rule, index, onChange, onDelete }: RuleFormProps) {
  function update(patch: Partial<PolicyRule>) {
    onChange(index, { ...rule, ...patch });
  }

  return (
    <div className="p-4 bg-gray-900 rounded-lg border border-gray-700 space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-xs text-gray-500 uppercase">Rule {index + 1}</span>
        <button
          onClick={() => onDelete(index)}
          className="p-1 rounded hover:bg-gray-700 text-red-400"
          title="Delete rule"
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label className="block text-xs text-gray-400 mb-1">Name</label>
          <input
            type="text"
            value={rule.name}
            onChange={(e) => update({ name: e.target.value })}
            placeholder="rule-name"
            className="w-full px-2 py-1.5 bg-gray-800 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-400 mb-1">Action Type</label>
          <select
            value={rule.action_type}
            onChange={(e) => update({ action_type: e.target.value as ActionType })}
            className="w-full px-2 py-1.5 bg-gray-800 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500"
          >
            {actionTypes.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs text-gray-400 mb-1">Resource Pattern</label>
          <input
            type="text"
            value={rule.resource_pattern}
            onChange={(e) => update({ resource_pattern: e.target.value })}
            placeholder="/tmp/**"
            className="w-full px-2 py-1.5 bg-gray-800 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-400 mb-1">Operations</label>
          <input
            type="text"
            value={rule.operations?.join(", ") ?? ""}
            onChange={(e) =>
              update({
                operations: e.target.value
                  .split(",")
                  .map((s) => s.trim())
                  .filter(Boolean),
              })
            }
            placeholder="read, write"
            className="w-full px-2 py-1.5 bg-gray-800 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500"
          />
        </div>
      </div>

      {/* Effect radio buttons */}
      <div>
        <label className="block text-xs text-gray-400 mb-1">Effect</label>
        <div className="flex gap-4">
          {effects.map((eff) => (
            <label key={eff} className="flex items-center gap-1.5 text-sm cursor-pointer">
              <input
                type="radio"
                name={`effect-${index}`}
                checked={rule.effect === eff}
                onChange={() => update({ effect: eff })}
                className="accent-blue-500"
              />
              <span
                className={
                  eff === "allow"
                    ? "text-green-400"
                    : eff === "deny"
                      ? "text-red-400"
                      : "text-yellow-400"
                }
              >
                {eff}
              </span>
            </label>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export default function PolicyEditor() {
  const [policy, setPolicy] = useState<Policy>(defaultPolicy);
  const [validated, setValidated] = useState<boolean | null>(null);

  const yaml = useMemo(() => toYaml(policy), [policy]);

  function addRule() {
    setPolicy((p) => ({
      ...p,
      rules: [...p.rules, { ...defaultRule, name: `rule-${p.rules.length + 1}` }],
    }));
    setValidated(null);
  }

  function updateRule(index: number, rule: PolicyRule) {
    setPolicy((p) => ({
      ...p,
      rules: p.rules.map((r, i) => (i === index ? rule : r)),
    }));
    setValidated(null);
  }

  function deleteRule(index: number) {
    setPolicy((p) => ({
      ...p,
      rules: p.rules.filter((_, i) => i !== index),
    }));
    setValidated(null);
  }

  function validate() {
    const valid = policy.rules.every(
      (r) => r.name.length > 0 && r.resource_pattern.length > 0,
    );
    setValidated(valid);
  }

  function saveToFile() {
    const blob = new Blob([yaml], { type: "text/yaml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${policy.name}.yaml`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function loadFromFile() {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".yaml,.yml";
    input.onchange = async () => {
      const file = input.files?.[0];
      if (!file) return;
      const text = await file.text();
      // Simple YAML parse (name and rules extraction)
      try {
        const nameMatch = text.match(/^name:\s*(.+)$/m);
        const descMatch = text.match(/^description:\s*(.+)$/m);
        const versionMatch = text.match(/^version:\s*"?(.+?)"?\s*$/m);
        setPolicy({
          name: nameMatch?.[1] ?? "imported-policy",
          description: descMatch?.[1] ?? "",
          version: versionMatch?.[1] ?? "1.0",
          rules: policy.rules, // keep existing rules since basic YAML parsing is limited
        });
      } catch {
        /* ignore parse errors */
      }
    };
    input.click();
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Policy Editor</h1>
        <div className="flex gap-2">
          <button
            onClick={loadFromFile}
            className="flex items-center gap-1.5 px-3 py-2 text-sm bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
          >
            <Upload className="w-4 h-4" />
            Load
          </button>
          <button
            onClick={saveToFile}
            className="flex items-center gap-1.5 px-3 py-2 text-sm bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            Save
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Visual rule builder */}
        <div className="space-y-4">
          {/* Policy metadata */}
          <div className="bg-gray-800 rounded-xl border border-gray-700 p-4 space-y-3">
            <h2 className="text-sm font-semibold text-gray-300">Policy Metadata</h2>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-gray-400 mb-1">Name</label>
                <input
                  type="text"
                  value={policy.name}
                  onChange={(e) => setPolicy((p) => ({ ...p, name: e.target.value }))}
                  className="w-full px-2 py-1.5 bg-gray-900 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">Version</label>
                <input
                  type="text"
                  value={policy.version}
                  onChange={(e) => setPolicy((p) => ({ ...p, version: e.target.value }))}
                  className="w-full px-2 py-1.5 bg-gray-900 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>
            <div>
              <label className="block text-xs text-gray-400 mb-1">Description</label>
              <input
                type="text"
                value={policy.description ?? ""}
                onChange={(e) => setPolicy((p) => ({ ...p, description: e.target.value }))}
                className="w-full px-2 py-1.5 bg-gray-900 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500"
              />
            </div>
          </div>

          {/* Rules */}
          <div className="space-y-3">
            {policy.rules.map((rule, i) => (
              <RuleForm
                key={i}
                rule={rule}
                index={i}
                onChange={updateRule}
                onDelete={deleteRule}
              />
            ))}
          </div>

          <div className="flex gap-2">
            <button
              onClick={addRule}
              className="flex items-center gap-1.5 px-3 py-2 text-sm bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
            >
              <Plus className="w-4 h-4" />
              Add Rule
            </button>
            <button
              onClick={validate}
              className="flex items-center gap-1.5 px-3 py-2 text-sm bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
            >
              <Check className="w-4 h-4" />
              Validate
            </button>
            {validated === true && (
              <span className="flex items-center text-sm text-green-400">Valid</span>
            )}
            {validated === false && (
              <span className="flex items-center text-sm text-red-400">
                Invalid: all rules need a name and resource pattern
              </span>
            )}
          </div>
        </div>

        {/* Right: YAML preview */}
        <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
          <div className="px-4 py-3 border-b border-gray-700 text-sm font-semibold text-gray-300">
            YAML Preview
          </div>
          <pre className="p-4 text-sm font-mono text-gray-300 overflow-x-auto whitespace-pre leading-relaxed">
            {yaml}
          </pre>
        </div>
      </div>
    </div>
  );
}
