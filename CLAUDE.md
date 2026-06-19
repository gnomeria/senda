# CLAUDE.md

Senda — fast, git-native API client. Collections = plain folders of YAML. Built Wails v3 (Go shell) + SolidJS (UI). Also ships TUI + headless CLI.

## Architecture

- **Frontend** (`frontend/`, SolidJS + TS + CodeMirror 6): pure view + local UI state. No HTTP, no disk. Call Go via Wails bindings.
- **Backend** (Go): all network/disk/CPU work. Rule: **touch network, disk, or CPU-heavy → Go.** Frontend only render.
- **Disk = source of truth.** App stateless editor over `*.yaml` collection files.
- Three entrypoints: desktop (`main.go` + `app*.go`), TUI (`cmd/senda-tui`), CLI (`cmd/senda-cli`). TUI/CLI pure Go — no frontend/webview.

### Go packages (`internal/`)

`httpclient` build/send requests, timing/size. `store` read/write/walk collection dirs, YAML, file watch. `model` core structs (Collection, Request, Environment, Response). `vars` resolve `{{var}}`. `auth`, `assert`, `script` (Goja JS sandbox), `runner`, `pipeline`, `history`, `importer`, `codegen`, `docgen`, `schemaval`, `security`, `mockserver`, `load`, `sseclient`, `wsclient`, `aigen`, `buildinfo`. App-bound API surface in root `app*.go`.

## Build & run

Wails v3 via Taskfile (`wails3 build` / `wails3 dev` dispatch to OS-namespaced tasks).

```
wails3 dev              # desktop, live reload
wails3 build            # prod desktop binary (stripped, ~24 MB) into bin/
task build:tui          # bin/senda-tui (pure Go)
task build:cli          # bin/senda-cli (pure Go)
task tui -- <path>      # build + run TUI against a collection
task generate:bindings  # regen TS bindings from Go services
```

`DEBUG=1` keep symbols (unstripped). `VERSION=x.y.z` inject version via ldflags.

## Test & check

```
go test ./...                          # Go tests
cd frontend && bun run test            # vitest
cd frontend && bun run typecheck       # tsc --noEmit
```

## Conventions

- Use `bunx`, not `npx`. Frontend deps via `bun install`.
- Frontend never do I/O — add backend work as bound Go method, regen bindings.
- Go `go 1.25.7`. Module name `senda`.
- Commit messages: no AI attribution trailers — never add `Co-Authored-By: Claude` or `Claude-Session:` lines.