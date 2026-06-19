# CLI screenshots

The still and walkthrough GIF of `senda-cli` referenced in the root README live
here. They are generated automatically — don't edit them by hand.

## Regenerating

Like the TUI images, these are rendered **headlessly in pure Go** — but the
content is a *real run*, not a storyboard. The generator stands up a local
in-process HTTP server, writes a small temporary collection, and sends every
request through the actual pipeline (the same path `senda-cli` uses). It then
paints `senda-cli`'s real stdout to PNG via
[`internal/termimg`](../../../internal/termimg), and builds the GIF by revealing
the run frame by frame as each result streams in. No network, no PTY, no
`ffmpeg`.

```bash
task shots:cli          # from the repo root → writes into docs/screenshots/cli/
```

or directly:

```bash
SENDA_CLI_SHOTS=1 SENDA_CLI_SHOT_DIR="$PWD/docs/screenshots/cli" \
  go test ./cmd/senda-cli -run TestCLIShot -count=1
```

### Fonts

Same requirement as the TUI shots: **DejaVu Sans Mono** + **FreeMono** on the
system font path (`apt-get install -y fonts-dejavu-core fonts-freefont-ttf`, or
point `SENDA_TUI_FONT*` at your own). `SENDA_CLI_GIF=0` skips the GIF.

## Files

| File | Shown |
|------|-------|
| `01-run.png` | A folder run against an environment — per-request status, timing, assertion tally, summary |
| `walkthrough.gif` | The same run, animated: type the command, watch each result stream in, then the summary |
