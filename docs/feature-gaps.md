---
layout: page
title: Feature Gaps
permalink: /feature-gaps/
---

# Senda — Competitive Feature Gaps

> Snapshot analysis of features Senda is missing versus yaak.app, Bruno,
> Hoppscotch, Postman, and Insomnia. Intent is to guide roadmap priority, not a
> commitment. See [`roadmap.md`](roadmap.md) for what's actually queued.

_Last reviewed: 2026-06-27._

## Gaps by category

### Protocols (largest hole)

| Feature | Who has it | Senda |
|---------|-----------|-------|
| **gRPC** | Postman, Insomnia, yaak, Bruno | Roadmap only |
| **Socket.IO** | Hoppscotch, Postman | No |
| **MQTT** | Hoppscotch, Postman | No |
| **SOAP / XML-RPC** | Postman | No |
| **GraphQL subscriptions** (over WS) | Hoppscotch, others | No (has GQL + WS separately, not subscriptions) |

### Auth (concrete, low-effort win)

Senda ships 4: Bearer, Basic, API key, OAuth2.

Missing vs Postman/Insomnia: **Digest, AWS SigV4, NTLM, OAuth1, Hawk, Akamai
EdgeGrid, JWT bearer**. SigV4 + Digest most requested. Small work in
`internal/auth`.

### Request chaining

Insomnia/Postman expose **template-tag response refs** — pull a value from
another request's response via the UI, no script. Senda only supports
post-script `senda.setVar()`. Higher friction for the common chaining case.

### Extensibility

**Plugin system** — Insomnia, yaak, and Hoppscotch all support JS plugins.
Senda has none (closed feature set). Long game, but a real differentiator for
yaak/Insomnia.

### Traffic capture

**Interceptor / proxy debugger** — Postman and Hoppscotch capture browser or
system traffic into a collection. Senda: none.

### Response tooling

- **Visualizer** — custom HTML/chart rendering of a response body (Postman). Senda: no.
- **Diff / compare two responses** — Senda: roadmap "split view".

### Design / governance

- **OpenAPI design editor** — Insomnia, Postman. Senda: roadmap.
- **Spec linting / contract test** (Spectral) — Insomnia, Postman. Senda: no.

### Scheduling

**Monitors** (scheduled local/cloud runs) — Postman. Senda has the CLI for CI
but no built-in scheduler.

### Smaller UX gaps

- **Bulk edit** headers/params (paste raw block) — Postman. Senda: row-by-row only.
- **Collection/folder-level scripts** — Postman runs pre/post at folder scope. Senda: scripts per-request only (folders carry vars/auth, not scripts).
- **Cookie editor UI** — Senda has a jar; editable UI unclear.
- **Per-request settings** (SSL verify toggle, follow-redirect, timeout) — Postman/Insomnia expose per request. Senda: collection-level proxy/mTLS only.

## Where Senda is already ahead

- Git-native plain YAML (vs Postman cloud, Insomnia bru/db).
- Mock server, security scanning (nuclei), load testing, docgen — Postman gates these behind paid/cloud; Bruno/yaak lack them.
- TUI + headless pure-Go binary — nobody else ships this.
- ~24 MB binary / ~100 MB RAM vs Electron rivals.

## Suggested priority

1. **gRPC** — already on roadmap; table-stakes, every rival has it.
2. **More auth types** (SigV4, Digest, OAuth1) — small Go work, high value.
3. **Template-tag response refs** — removes scripting friction for chaining.
4. **Socket.IO / MQTT** — completes the realtime story vs Hoppscotch.
5. **Plugin system** — long game; yaak/Insomnia differentiator.

## Out of scope (Senda non-goals)

Cloud accounts, hosted sync, team workspaces, hosted doc publishing, telemetry.
See [`roadmap.md`](roadmap.md#non-goals).
