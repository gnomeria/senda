# Senda — Testing & Visual-Feedback Strategy

> Status: Draft · Last updated: 2026-06-12
>
> How we continuously verify what's being built — logic correctness *and* whether the UI actually looks right.

## 0. The core problem

Senda is a **Wails desktop app** (Go + OS webview). Can't "open it in a browser" normally, and a headless agent can't look at a native window. Solution:

> **`wails dev` serves the frontend at `http://localhost:34115` with the Go bindings bridged in.**

So the running app — real Go backend included — is reachable over HTTP. A headless browser can load it, interact, and **screenshot**. Those screenshots are read back and visually inspected as features land. This is the whole feedback loop.

## 1. Test layers (the pyramid)

| Layer | Tool | Scope | Speed | Runs |
|-------|------|-------|-------|------|
| Go unit | `go test` | `httpclient`, `store`, `vars`, `model` | fast | every change + CI |
| Frontend unit | Vitest + `@solidjs/testing-library` + jsdom | components, signals, formatting | fast | every change + CI |
| Go↔JS contract | Vitest against generated bindings (mocked) | binding shapes match Go | fast | CI |
| E2E + visual | Playwright → `localhost:34115` | real app, real backend | slow | per feature + pre-release |
| Manual smoke | `wails dev` + screenshot read-back | "does it look good?" | manual | per feature |

Keep the base fast and broad; reserve slow E2E/visual for release gates.

## 2. Go unit tests

- Table-driven. Live beside source (`store/store_test.go` etc).
- **`store`**: golden-file round-trip — YAML → struct → YAML must be stable. Cover multiline bodies, unicode, disabled rows, empty sections.
- **`vars`**: precedence (request→env→collection), unresolved-var warnings, nested `{{a}}` containing resolved value.
- **`httpclient`**: spin up `httptest.Server`; assert method/headers/body built right, timing/size captured, large-body truncation path.
- Gate: `go test ./...` green before anything ships.

## 3. Frontend unit tests

- **Vitest** + **@solidjs/testing-library** + jsdom.
- Cover: KV-row editor (add/remove/toggle), body-type switch, response formatting helpers, var-substitution preview, tab state.
- Pure render logic only — no real HTTP (backend is mocked at the `api.ts` boundary).

**Current suites** (`bun run test` from `frontend/`):

| Suite | Covers |
|-------|--------|
| `lib/format.test.ts` | base64/hex formatting helpers |
| `lib/factory.test.ts` | blank-model factories, byte formatting, status bucketing |
| `lib/store.test.ts` | tab lifecycle: open/focus-if-open, scratch reuse, switch snapshots, close variants (single/others/left/right/saved), cycle wrap, clone, save/revert, localStorage persistence incl. corrupt data |
| `lib/theme.test.ts` | theme registry integrity, mode resolution (incl. mocked `matchMedia`), persistence + fallback on garbage, `applyTheme`/`initTheme` |
| `components/ThemeSettings.test.tsx` | appearance dialog: lists, selection marks, Active badge, mode buttons, backdrop close |

**Bindings stubs:** the generated Wails bindings (`frontend/bindings/`, gitignored)
don't exist on a fresh clone or in CI, so `vite.config.ts` aliases the bindings
imports to hand-written stubs in `src/test-stubs/` when mode is `test`. Unit
tests therefore run with nothing but `bun install` — no Go toolchain, no
`wails3 generate bindings`. Keep the stub enums in sync with
`internal/model/model.go`.

**Running the UI without Wails:** the same stubs make the whole frontend
servable in a plain browser — `bunx vite --mode test` + the dev mock
(`lib/devMock.ts`) gives a clickable app for Playwright screenshots when the
Wails toolchain or webkit2gtk isn't available (e.g. cloud sandboxes).

## 4. E2E + visual (the "see it" layer)

**Stack:** Playwright (Chromium), installed as a frontend devDependency. lightpanda is *not* usable here — it doesn't render graphically, so it can't screenshot. Playwright+Chromium required for visuals.

**Flow:**
1. Start `wails dev` (background) — frontend + Go backend live at `localhost:34115`.
2. Playwright loads that URL, runs interaction scripts.
3. Capture screenshots to `tests/visual/__screenshots__/`.
4. Screenshots are **read back and visually inspected** — this is how "does the UI look good yet?" gets answered as features land.

**Key visual checkpoints** (saved as named screenshots):
- Empty 3-pane shell.
- Request filled in + response rendered.
- Large-response truncation banner; big-body scroll.
- Populated collection tree; create/rename flows.
- Env switcher open; `{{var}}` resolved in URL bar.

**Visual regression (lightweight):** Playwright `toHaveScreenshot()` baselines committed under `tests/visual/`. Diffs flagged on change; baselines updated deliberately, never blindly.

**E2E happy paths** (assertions, not just pixels):
- Send a request → response status/body appear.
- Save a request → YAML file exists on disk with right content.
- Switch env → interpolated URL changes.

## 5. The inner loop (how I work each session)

```
1. wails dev   (background, serves :34115 + Go bindings)
2. make change (Go or Solid)        ── hot reload
3. Playwright screenshot relevant view
4. read screenshot → judge layout / spacing / states
5. add/extend unit test for the logic touched
6. go test ./... + vitest green
7. before a release → full Playwright E2E + visual pass
```

## 6. CI (GitHub Actions)

The `ci.yml` workflow runs on every push/PR:

- `go test ./...` + `go vet`.
- `vitest run` + `tsc --noEmit`.
- Build check: `wails3 build` (Linux) compiles.
- E2E/visual: still run manually (headless Chromium) to avoid flake; the
  Screenshots workflow regenerates the docs images.

## 7. Platform note

The default Linux build targets **GTK4 / webkitgtk-6.0** — no build tag needed.
The legacy **GTK3 / webkit2gtk-4.1** stack is an opt-out via the `gtk3` tag
(`wails3 task linux:build PROD_TAGS="production gtk3"`).

## 8. Out of scope (for now)

- Cross-OS visual matrix (mac WebKit / Windows WebView2) — revisit near release.
- Load/perf benchmarking beyond the 20 MB large-response freeze check.
- Fuzzing the YAML parser — golden tests suffice initially.
