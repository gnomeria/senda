# Senda brand assets

The Senda mark is a single shape that carries the product's three ideas at once:

- **The path** — *senda* means "path / trail". The S-curve is the trail.
- **Request → response** — the two endpoint nodes are the client and the API,
  joined by the path the request travels.
- **The letter S** — the curve doubles as the Senda monogram.

Colors are from the **Catppuccin Mocha** palette the app ships with: a
`base → crust` background, with the path flowing blue `#89b4fa` (request) →
teal `#94e2d5` → green `#a6e3a1` (response).

## Files

| File | Use |
| --- | --- |
| `senda-mark.svg` | App icon / square mark (source of truth) |
| `senda-mark-512.png`, `-256.png`, `-128.png` | Rasterized mark |
| `senda-wordmark.svg` | Horizontal lockup (mark + "senda") |
| `senda-wordmark.png` | Rasterized lockup |
| `senda.ico` | Multi-size Windows icon (16/32/48/256) |

The favicon (`frontend/public/favicon.svg`) and in-app icon
(`frontend/public/appicon.png`) are copies of the mark.

## Regenerating the rasters

The SVGs are the source. PNG/ICO outputs are produced with
[`@resvg/resvg-js`](https://github.com/yisibl/resvg-js) +
[`png-to-ico`](https://github.com/steambap/png-to-ico):

```sh
bun add @resvg/resvg-js png-to-ico
# render each PNG at the desired width, then pack the .ico — see git history
# of this file for the exact one-off script.
```
