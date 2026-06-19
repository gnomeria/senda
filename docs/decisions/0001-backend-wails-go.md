# ADR-0001 — Backend runtime: Wails (Go)

> Status: Accepted · 2026-06-09

## Context

Need a desktop shell + backend. Candidates: Electron (Node), Tauri (Rust), Wails (Go).
Performance of Tauri vs Wails is ~equal — both use the OS webview, both crush Electron on binary size and RAM. So the deciding factor is **language productivity**, not runtime speed.

The backend is HTTP-, file-, and git-heavy. That work is trivial and idiomatic in Go's stdlib (`net/http`, `os`, `encoding`).

## Decision

Use **Wails v2 with a Go backend.**

## Consequences

- ✅ Fast to write the backend (HTTP/file/YAML) in Go.
- ✅ Small binary (~10–20MB), low RAM, native webview.
- ✅ Easy single-binary distribution.
- ⚠️ Future scripting needs `goja` (Go JS interpreter), slower than V8/QuickJS. Accepted — scripting is deferred (ADR-0004) and perf there is non-critical.
- ⚠️ Smaller plugin ecosystem than Tauri; some things (updater, secure store) are more DIY.
- ❌ No mobile target (Wails is desktop-only). Acceptable — mobile is a non-goal.

## Alternatives rejected

- **Electron** — large binary/RAM, the exact thing we're escaping.
- **Tauri (Rust)** — smaller binary + richer plugins + mobile path, but Rust slows feature velocity for a backend-logic-heavy app, and the perf delta over Wails is marginal for this use case.
