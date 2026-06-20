---
layout: default
title: Mock server
---

# Mock server

Senda ships a built-in local HTTP mock server. It serves canned or generated
responses from plain YAML files so you can develop a frontend against an API that
doesn't exist yet, reproduce edge cases (slow responses, error codes), or run a
stateful fake backend in tests — without touching the real service.

Like everything else in Senda, mocks are **just files on disk** — one YAML file
per route in a `mocks/` directory inside your collection. They are git-trackable
and diff cleanly.

There are two kinds of mock files:

- **Rule routes** — match a request, return a response. Responses can be selected
  by request conditions or by the active *scenario*, and their bodies pass through
  a `{{...}}` template engine.
- **Resource routes** — a `resource:` declaration auto-wires full REST CRUD over
  an in-memory store (json-server style).

An optional `mocks/_config.yaml` adds a proxy passthrough, CORS, and a default
scenario. The server **hot-reloads** when you edit files.

---

## Quick start

1. In an open collection, create a `mocks/` directory.
2. Add `*.yaml` files (rule and/or resource routes).
3. Start it: in the desktop app open the **Mock Server** panel and click **Start**,
   or from the terminal run `senda mock -collection . -addr :8787`.
4. Send requests to the bound address — e.g. `curl http://localhost:8787/users/42`.

```
my-collection/
├── requests/
│   └── create-user.yaml
└── mocks/
    ├── _config.yaml      # optional server config
    ├── get-user.yaml     # rule route
    └── users.yaml        # resource route
```

---

## Rule routes

```yaml
name: Get user
method: GET                 # empty = any method; case-insensitive
path: /users/:id            # :id is a path parameter
enabled: true               # default true; set false to disable the file
priority: 0                 # higher wins when several routes match
match:                      # optional extra conditions (all must hold)
  query:   { expand: "true" }
  headers: { x-api-key: secret }
  body:    { role: admin }  # deep-contains against the JSON request body
responses:                  # one or more; first matching one wins
  - status: 200
    headers: { Content-Type: application/json }
    body:                   # NATIVE object (also: string, or omit + schema)
      id: "{{params.id}}"
      name: "{{faker.name}}"
      expand: "{{query.expand}}"
    delay: 100              # optional latency in ms
    when:    { query: { v: "2" } }   # optional per-response condition
    scenario: success                 # optional named scenario
```

**Route selection.** Among all routes whose `method`, `path` and `match` fit the
request, Senda picks the highest `priority`, then the most specific path (more
literal segments — `/users/me` beats `/users/:id`), then file order.

**Response selection** within a route:
1. a response tagged with the active **scenario** whose `when` matches, else
2. an untagged (default) response whose `when` matches, else
3. any response whose `when` matches.

### Backward compatibility (v1)

Old single-response files still work. When `responses` is absent, the top-level
`status` / `headers` / `body` / `delay` / `schema` fields are treated as one
response:

```yaml
method: GET
path: /ping
status: 200
body: '{"ok": true}'
```

---

## Templating

Response bodies and header values pass through a small template engine. Native
object bodies are walked recursively, so every string leaf is templated.

| Token | Resolves to |
|---|---|
| `{{params.id}}` | path parameter `:id` |
| `{{query.x}}` | query-string value `?x=` |
| `{{headers.x-api-key}}` | request header (case-insensitive) |
| `{{body.user.name}}` | dotted path into the JSON request body |
| `{{faker.name}}` | fake full name (also `firstname`, `lastname`, `email`, `username`, `uuid`, `int`, `float`, `bool`, `word`, `sentence`, `city`, `country`, `company`, `phone`, `date`, `datetime`) |
| `{{uuid}}` | a random UUID v4 |
| `{{randomInt 1 100}}` | random integer in `[1,100]` |
| `{{now}}` / `{{now "2006-01-02"}}` | current UTC time (RFC 3339, or a Go layout) |

Unknown tokens are left untouched.

---

## Resource routes (stateful CRUD)

A file with a `resource:` declaration auto-generates a REST collection backed by
an **in-memory** store:

```yaml
resource: users
path: /users
key: id                     # id field, default "id"
seed:
  - { id: 1, name: Alice }
  - { id: 2, name: Bob }
```

| Method & path | Action |
|---|---|
| `GET /users` | list all records |
| `GET /users/:id` | fetch one (404 if missing) |
| `POST /users` | create; assigns `key` if omitted (auto-increment) |
| `PUT /users/:id` | replace (preserves `key`) |
| `PATCH /users/:id` | merge fields |
| `DELETE /users/:id` | remove (204) |

State lives in RAM: it seeds on start and **resets on restart** or via the panel's
**Reset state** button. Nothing is written to disk.

---

## Scenarios

Tag responses with `scenario:` to model alternate worlds (happy path, server
error, rate-limited…). Switch the active scenario from the panel's **Scenario**
dropdown, the `_config.yaml` `scenario:` field, or `senda mock -scenario error`.

```yaml
method: GET
path: /checkout
responses:
  - status: 200
    body: { ok: true }
  - scenario: error
    status: 500
    body: { error: "boom" }
```

---

## Server config — `mocks/_config.yaml`

```yaml
proxy: https://real.api     # forward unmatched requests to a real backend
cors: true                  # default true: CORS headers + OPTIONS preflight
scenario: success           # default active scenario
```

- **proxy** — on no route match, the request is forwarded (method, path, query,
  headers, body) to this base URL and the upstream response is returned. Lets you
  mock a few endpoints and pass the rest through.
- **cors** — on by default (dev convenience): adds `Access-Control-Allow-*`
  headers and answers `OPTIONS` preflight with `204`.

---

## CLI

```bash
senda mock -collection ./my-api -addr :8787 -scenario error
```

Pure Go, no webview. Loads `./my-api/mocks/`, serves, logs each request to stdout,
and runs until `Ctrl-C`. Use `-addr :0` to pick a free port.

---

## Save a response as a mock

After sending a request, click **Save as mock** in the response toolbar. Senda
writes a `mocks/<name>.yaml` capturing the request method + path and the response
status, headers and body — turning a real round-trip into a fixture you can edit.

---

## Request log

While running, every handled request is recorded (`method`, `path`, `status`,
source) and streamed live to the Mock Server panel (last 200 entries). The
**source** tells you how it was served: `mock` (rule route), `state` (resource
CRUD), or `proxy` (passthrough).

---

## Hot reload

Editing any file under `mocks/` reloads routes and config automatically (200 ms
debounce). Resource records you've already mutated are preserved across reloads;
new resources are seeded and removed ones dropped.

---

## Under the hood

| Concern | Location |
|---|---|
| Schema / types | `internal/mockserver/model.go` |
| Loading & path compile | `internal/mockserver/load.go` |
| Matching & ordering | `internal/mockserver/match.go` |
| Templating | `internal/mockserver/template.go` |
| Faker & schema fake | `internal/mockserver/fake.go` |
| Resource store (CRUD) | `internal/mockserver/state.go` |
| Proxy passthrough | `internal/mockserver/proxy.go` |
| Server / handler / hot-reload | `internal/mockserver/mockserver.go` |
| App bindings | `app_features.go` |
| UI panel | `frontend/src/components/MockServerPanel.tsx` |
| CLI | `cmd/senda/mock.go` |

Everything — matching, templating, fake data, the resource store — runs in Go.
The faker is hand-rolled (no dependency) to keep the binary small.
