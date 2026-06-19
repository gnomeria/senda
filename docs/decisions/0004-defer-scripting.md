# ADR-0004 — Defer scripting to post-MVP

> Status: Accepted · 2026-06-09

## Context

Bruno/Postman support pre/post-request **JS scripting** (`bru.setVar`, `req.setHeader`, chai-style assertions). This is high-value but also the **single largest engineering cost** in a Bruno-like client:

- Go has no native JS engine. Requires embedding `goja` (pure-Go interpreter).
- The hard part isn't running JS — it's reimplementing the entire `bru`/`req`/`res` API surface and sandbox the scripts expect.
- This alone is multiple weeks and delays a usable build.

## Decision

**Exclude scripting from MVP (v0.1).** Park it for v0.2+.

MVP delivers: send/save/collections/environments/variable-interpolation — a fully usable client without scripting.

## Consequences

- ✅ MVP ships dramatically sooner.
- ✅ Core architecture (Go backend, YAML store, var resolver) stays simple.
- ✅ Variable interpolation (`{{var}}`) covers the most common dynamic need without scripting.
- ⚠️ No dynamic request chaining / computed values / assertions until v0.2.
- ⚠️ When added: choose `goja`, design the script API surface deliberately, give it its own ADR.

## Alternatives rejected

- **Include scripting in MVP** — multiplies scope and time-to-usable for a feature many simple flows don't need.
- **Embed Node for V8-grade scripting** — kills the single-binary advantage that motivated Wails.
