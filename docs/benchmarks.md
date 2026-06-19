---
layout: page
title: Benchmarks — Footprint vs Bruno & Insomnia
permalink: /benchmarks/
---

# Benchmarks: Senda Footprint vs Bruno & Insomnia

> Authored: 2026-06-16 · Owner: @gnomeria
>
> Goal: quantify Senda's resource footprint against Electron-based API clients
> (Bruno, Insomnia) to back the "no Chromium, fraction of the RAM" positioning.

---

## TL;DR

| Metric | Senda | Bruno | Insomnia | Senda edge vs Bruno |
|---|---|---|---|---|
| **Install size** | **31 MB** | 450 MB | 1.0 GB | **14.5× smaller** |
| **Main process RAM (idle)** | **114 MB** | 270 MB | — | **2.4× lighter** |
| **Full process tree RAM (idle)** | ~235–391 MB | 632 MB | — | **~1.6–2.7× lighter** |

Headline claims these support:

- **14× smaller download** than Bruno, **33× smaller** than Insomnia.
- **Less than half the idle RAM** of Bruno.
- **No bundled Chromium** — one shared system webview, not a 450 MB browser per app.

---

## Why Senda is lighter

Senda is built on **Wails v3**: a Go binary that renders its UI in the OS's
**system webview** (WebKit on macOS, WebView2 on Windows). There is no bundled
browser engine.

Electron apps (Bruno, Insomnia) ship a **full Chromium + Node runtime** inside
every app bundle, and spawn a multi-process Chromium tree (main + GPU + renderer
+ utility helpers) at runtime.

| | Senda (Wails) | Bruno / Insomnia (Electron) |
|---|---|---|
| Render engine | OS system webview (shared) | Bundled Chromium (per app) |
| Runtime procs (idle) | 1 main + 1 WebKit renderer | 4+ (main + GPU + renderers + utility) |
| Engine in bundle | none | ~full Chromium |

---

## Method

Measured on macOS (Darwin 25.5.0), 2026-06-16. Apps launched **idle** (no
collection loaded), allowed to settle (~8 s), then sampled with `ps`.

- **Install size** — `du -sh` on the `.app` bundle (Bruno, Insomnia) vs the
  local `senda-desktop` binary (31 MB; the packaged `.app` adds only a small
  Info.plist + icon).

  > **Build-variant note:** 31 MB is the locally-built (unstripped) binary
  > measured here. A clean stripped `wails3 build` is ~24 MB. **31 MB is the
  > conservative number** — the real shipped size is smaller, so the size
  > advantage below is a floor, not a ceiling. Re-measure a release artifact
  > before quoting. (An earlier ~5.7 MB figure floating around was a broken
  > build with the mandatory `production` tag missing — a stub that errors at
  > runtime, not a real binary.)
- **Main process RAM** — RSS of the single largest owning process per app.
- **Full tree RAM** — Bruno: sum RSS of all name-matched `Bruno*` processes
  (clean — Chromium helpers are named). Senda: main RSS + the marginal WebKit
  renderer RSS, derived by diffing system WebKit RSS before/after launch (Senda
  spawned **+1** `WebContent` process).

### Raw numbers

```
Install size
  senda-desktop (binary)   31 MB
  Bruno.app               450 MB
  Insomnia.app            1.0 GB

RAM (idle, RSS)
  Senda  — main 114 MB · full tree ~235–391 MB (1 main + 1 WebKit renderer)
  Bruno  — main 270 MB · full tree 632 MB (4 procs)
```

---

## Caveats

Be honest when quoting these:

- **Single run, not averaged.** One launch per app.
- **Idle / empty collection.** Loading a large collection grows both; not yet
  measured.
- **Full-tree RAM is a band, not a point.** The test machine had ~10 GB of other
  apps resident and macOS WebKit XPC processes are shared system-wide, so
  attributing Senda's exact renderer share is noisy. The **main-process number
  (114 MB) is the rock-solid figure**; the full tree is an estimate.
- **Startup time NOT measured.** GUI cold-start (time-to-window-painted) is the
  "feels snappy" metric and the hardest to automate reliably. Pending.

---

## Open follow-ups

1. **Cold-start time** — instrument `main.go` to log a window-ready timestamp,
   diff from launch. Strongest proof for the "Bruno feels clunky" claim.
2. **Averaged RAM** — 3+ launches on a quiet machine for clean point numbers.
3. **Loaded-collection RAM** — open an identical large collection in each.
4. **Linux / Windows footprint** — WebView2 (Windows) and WebKitGTK (Linux)
   numbers will differ from macOS WebKit.

## Reproduce

```sh
# install size
du -sh /Applications/Bruno.app /Applications/Insomnia.app
ls -lh bin/senda-desktop

# RAM (launch app, let settle, then)
ps -axo rss,command | grep -i 'Bruno'  | grep -v grep   # Bruno tree
ps -axo rss,comm    | grep senda-desktop | grep -v grep  # Senda main
```
