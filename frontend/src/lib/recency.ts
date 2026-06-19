// Helpers for the sidebar recency pills: turn a request's last-run history
// into a compact "time ago" label and a colour bucket (fresh/recent/stale/
// error). Folders roll their descendants up: the pill shows the newest run,
// and flags an error whenever any request beneath it last returned non-2xx.
import type { Activity } from "./api";
import type { TreeNode } from "./api";

export type RecencyClass = "err" | "fresh" | "recent" | "stale";

export type NodeRecency = {
  at: string; // RFC3339 of the newest run under this node
  cls: RecencyClass; // colour bucket; "err" if any descendant failed
  failing: number; // descendants whose last run was non-2xx
  ran: number; // descendants that have run at least once
};

const MIN = 60_000;
const HOUR = 60 * MIN;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;

// fmtAgo renders the elapsed time since `at` (RFC3339) as a short label like
// "now", "5m", "2h", "3d", "2w", "4mo". Empty for an unparseable timestamp.
export function fmtAgo(at: string, now: number = Date.now()): string {
  const t = Date.parse(at);
  if (Number.isNaN(t)) return "";
  const d = Math.max(0, now - t);
  if (d < MIN) return "now";
  if (d < HOUR) return `${Math.floor(d / MIN)}m`;
  if (d < DAY) return `${Math.floor(d / HOUR)}h`;
  if (d < WEEK) return `${Math.floor(d / DAY)}d`;
  if (d < 4 * WEEK) return `${Math.floor(d / WEEK)}w`;
  return `${Math.floor(d / (4 * WEEK))}mo`;
}

// ageClass buckets a timestamp by recency: green under an hour, amber under a
// day, grey beyond.
function ageClass(at: string, now: number): RecencyClass {
  const t = Date.parse(at);
  if (Number.isNaN(t)) return "stale";
  const d = now - t;
  if (d < HOUR) return "fresh";
  if (d < DAY) return "recent";
  return "stale";
}

// isFail reports whether a run did not complete with a 2xx/3xx status.
function isFail(a: Activity): boolean {
  return a.error || a.status === 0 || a.status >= 400;
}

// nodeRecency resolves the pill to show on a tree node. Leaves reflect their
// own last run; folders aggregate all descendants — newest run for the time/
// colour, plus a failure count so a broken request anywhere inside turns the
// folder pill red. Returns null when nothing under the node has ever run.
export function nodeRecency(
  node: TreeNode,
  map: Record<string, Activity>,
  now: number = Date.now(),
): NodeRecency | null {
  if (!node.isDir) {
    const a = map[node.path];
    if (!a) return null;
    const fail = isFail(a);
    return { at: a.at, cls: fail ? "err" : ageClass(a.at, now), failing: fail ? 1 : 0, ran: 1 };
  }

  let newest = "";
  let newestT = -1;
  let failing = 0;
  let ran = 0;
  const visit = (n: TreeNode) => {
    if (!n.isDir) {
      const a = map[n.path];
      if (a) {
        ran++;
        if (isFail(a)) failing++;
        const t = Date.parse(a.at);
        if (!Number.isNaN(t) && t > newestT) {
          newestT = t;
          newest = a.at;
        }
      }
      return;
    }
    for (const c of n.children ?? []) if (c) visit(c);
  };
  visit(node);

  if (ran === 0) return null;
  return { at: newest, cls: failing > 0 ? "err" : ageClass(newest, now), failing, ran };
}
