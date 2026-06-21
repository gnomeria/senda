import { describe, expect, it } from "vitest";

// Guards against the Solid "frozen handler" bug class.
//
// Solid binds DOM event handlers ONCE at render and does not re-evaluate the
// JSX expression per event. So a handler written as a bare ternary —
//   onClick={running() ? stop : start}
// reads running() a single time (its initial value) and stays wired to that
// branch forever. The toggle button then runs the wrong handler on every later
// click. The fix is always to wrap in an arrow so the signal is read at click
// time:
//   onClick={() => (running() ? stop() : start())}
//
// This test scans every .tsx source file and fails if an on<Event>={...}
// handler value contains a ternary (`?` … `:`) without an arrow (`=>`).

// Vite reads the raw source of every component at test time — no node:fs needed.
const sources = import.meta.glob("./**/*.tsx", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

// Extract the balanced {...} value of every on<Event>= attribute in a file.
function handlerValues(src: string): string[] {
  const out: string[] = [];
  const re = /\son[A-Z][A-Za-z]*=\{/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(src))) {
    let depth = 1;
    let i = m.index + m[0].length;
    const start = i;
    for (; i < src.length && depth > 0; i++) {
      if (src[i] === "{") depth++;
      else if (src[i] === "}") depth--;
    }
    out.push(src.slice(start, i - 1));
  }
  return out;
}

// A ternary selecting a handler, with no arrow wrapping it, is frozen at render.
function isFrozenTernary(value: string): boolean {
  if (value.includes("=>")) return false; // wrapped in a function — re-read per event
  // Ignore optional chaining (?.) and nullish coalescing (??), which also use "?".
  const stripped = value.replace(/\?\./g, "").replace(/\?\?/g, "");
  return stripped.includes("?") && stripped.includes(":");
}

describe("no frozen event handlers", () => {
  const files = Object.entries(sources).filter(([p]) => !p.endsWith(".test.tsx"));

  it("finds .tsx sources to scan", () => {
    expect(files.length).toBeGreaterThan(0);
  });

  it("has no on<Event>={cond ? a : b} handler without an arrow wrapper", () => {
    const offenders: string[] = [];
    for (const [path, src] of files) {
      for (const value of handlerValues(src)) {
        if (isFrozenTernary(value)) offenders.push(`${path}: on…={${value.trim()}}`);
      }
    }
    expect(
      offenders,
      `Wrap these in an arrow: () => (cond() ? a() : b())\n${offenders.join("\n")}`
    ).toEqual([]);
  });
});
