import type { TraceEvent } from "./api";

// Build a tree from flat events using parent_id
export function buildTree(events: TraceEvent[]): TraceEvent[] {
  const map = new Map<string, TraceEvent>();
  const roots: TraceEvent[] = [];

  for (const ev of events) {
    map.set(ev.id, { ...ev, children: [] });
  }

  for (const ev of events) {
    const node = map.get(ev.id)!;
    if (ev.parent_id && map.has(ev.parent_id)) {
      map.get(ev.parent_id)!.children!.push(node);
    } else {
      roots.push(node);
    }
  }

  return roots;
}
