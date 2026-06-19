# ADR-0002 — Frontend: SolidJS

> Status: Accepted · 2026-06-09

## Context

The frontend is a webview-rendered UI. For an API client (forms, tree, tabs, code editor), framework choice has a **small raw-perf impact** but a meaningful effect on bundle size and how easy it is to *accidentally* make the UI slow.

Candidates: React, Svelte, SolidJS.

## Decision

Use **SolidJS + TypeScript.**

## Consequences

- ✅ Fine-grained reactivity, no virtual-DOM diff → top-tier runtime perf.
- ✅ Tiny bundle.
- ✅ Few re-render footguns vs React (no memo discipline needed).
- ⚠️ Smaller ecosystem than React; fewer ready-made CodeMirror/Solid wrappers → we write a thin custom integration (isolated to one component).
- ⚠️ Less AI/community example coverage than React. Acceptable.

## Note on perf reality

Framework is **not** the main perf lever for this app. The real levers (see architecture §5, §8):
1. Virtualize long lists + large responses.
2. Push heavy work to Go, off the UI thread.
3. CodeMirror 6 (virtualized) over Monaco.

Solid was chosen for clean code + low footgun risk, not because it alone makes the app fast.

## Alternatives rejected

- **React** — biggest ecosystem, but easiest to footgun into re-render storms; larger bundle.
- **Svelte** — close second; comparable safety, slightly behind Solid on raw reactivity. Viable fallback if Solid+CodeMirror integration proves painful.
