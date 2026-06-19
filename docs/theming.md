---
layout: page
title: Theming
permalink: /theming/
---

# Senda — Theming

> Status: Shipped · Last updated: 2026-06-13
>
> How the appearance system works, what guarantees it makes, and how to add a theme.

## User model

Appearance is controlled from the **palette icon in the titlebar** (or the
command palette → "Appearance: change theme…"). The dialog has two independent
axes, mirroring editors like VS Code and clients like Bruno:

1. **Mode** — `light`, `dark`, or `system`. System follows the OS
   `prefers-color-scheme` and updates live when the OS preference flips.
2. **Theme per side** — one chosen theme for light, one for dark. Both choices
   are always visible and editable; the column currently in effect shows an
   **Active** badge. Picking a dark theme while light mode is showing simply
   stores the choice for the next time dark is in effect.

Built-in themes:

| Light | Dark |
|-------|------|
| Light | Dark |
| Light Monochrome | OLED Dark |
| Light Pastel | Dark Monochrome |
| Catppuccin Latte | Dark Pastel |
| VS Code Light | Catppuccin Frappé |
| | Catppuccin Macchiato |
| | Catppuccin Mocha |
| | Nord |
| | VS Code Dark |

**OLED Dark** uses a pure-black `--bg` (`#000000`) with small elevation steps
(`#0a0a0c` / `#141519`) so surfaces stay visible while background pixels go
fully dark on OLED panels. It reuses the default Dark theme's accent and status
colors.

Choices persist across launches in `localStorage`:

| Key | Value |
|-----|-------|
| `senda.themeMode` | `light` \| `dark` \| `system` |
| `senda.themeLight` | id of the chosen light theme |
| `senda.themeDark` | id of the chosen dark theme |

Invalid/stale stored values (e.g. a theme id removed in a later version) fall
back to the defaults (`light` / `dark`, mode `system`) instead of breaking.

## Architecture

Everything lives in `frontend/src/lib/theme.ts` + `frontend/src/components/ThemeSettings.tsx`.

### Tokens, not stylesheets

A theme is a flat map of **CSS custom properties → hex values**
(`ThemeTokens`). The entire UI styles itself off these variables (declared
once in `styles.css`), so applying a theme is just writing ~18 inline
variables onto `<html>`:

```
applyTheme(theme) →
  for each token: documentElement.style.setProperty("--bg", …)
  documentElement.style.colorScheme = theme.kind   // native widgets follow
  documentElement.dataset.theme = theme.id          // debuggable / testable
```

`:root` in `styles.css` declares the same variables with the **dark theme's
values** as a pre-init fallback, so the first paint (before `initTheme()` runs
in `App`'s `onMount`) is never unstyled. `theme.test.ts` pins a few values to
keep the two in sync.

The token set (see `ThemeTokens` in `theme.ts`):

| Token | Role |
|-------|------|
| `--bg` / `--bg-elev` / `--bg-elev2` | base background and two elevation steps |
| `--border` / `--border-soft` | strong and hairline borders |
| `--text` / `--text-dim` / `--text-faint` | three-step text hierarchy |
| `--accent` / `--accent-dim` / `--accent-fg` | primary action color, focus/hover borders, text on accent fills |
| `--selection` / `--selection-fg` | selected row background + its text |
| `--hover` | subtle hover overlay (translucent) |
| `--ok` / `--warn` / `--err` / `--redirect` | status colors (2xx / pending / 4xx-5xx / 3xx-info) |

`--selection` exists separately from `--accent-dim` because dark themes can
reuse a dim accent as a selected-row background with white text, but light
themes need a pale tint with dark text — one variable can't serve both roles.

### Mode resolution

```
resolvedKind(): mode "light"/"dark" → itself
               mode "system"        → matchMedia("(prefers-color-scheme: dark)")
activeTheme():  resolvedKind() == "light" ? themeById(lightThemeId) : themeById(darkThemeId)
```

`initTheme()` applies the persisted theme once and registers a `matchMedia`
change listener so system mode tracks the OS live; it returns a cleanup
function which `App` wires into `onCleanup`. Where `matchMedia` doesn't exist
(jsdom), system mode resolves to dark — the app's native look.

### Why inline variables instead of `data-theme` CSS blocks?

13 themes × ~18 tokens as CSS rules would mean ~230 lines of generated CSS and
a build step, or hand-maintained duplication. A TS registry keeps themes
typed (a missing token is a compile error **and** a test failure), trivially
unit-testable, and means adding a theme touches exactly one file.

## Adding a theme

1. Append a `Theme` object to the `themes` array in `frontend/src/lib/theme.ts`
   with a unique `id`, display `name`, `kind`, and a complete `ThemeTokens` map.
2. That's it — the picker lists themes from the registry, and
   `theme.test.ts` will fail if the token set is incomplete or the id collides.

Guidelines for picking values:

- `--bg-elev`/`--bg-elev2` should step *away* from `--bg` (lighter in dark
  themes, usually darker/grayer in light themes).
- `--accent-fg` is the text drawn on solid `--accent` fills (Send button):
  white for saturated accents, near-black for pale ones (e.g. Dark Monochrome).
- Keep `--ok`/`--warn`/`--err` recognizably green/yellow/red even in
  monochrome themes — they encode HTTP semantics, not decoration.

## Testing

- `frontend/src/lib/theme.test.ts` — registry integrity (unique ids, complete
  token sets, valid hex), mode resolution incl. mocked `matchMedia`,
  persistence round-trips and garbage-tolerant loading, `applyTheme` /
  `initTheme` behavior incl. live OS-preference flips and listener cleanup.
- `frontend/src/components/ThemeSettings.test.tsx` — dialog rendering, Active
  badge placement, picking themes per side, mode buttons, backdrop close.
- Visual: theme switching is exercised in the browser (vite `--mode test` +
  Playwright) — see `docs/testing.md` §“Running the UI without Wails”.

## Icon sizing

Icons are [`lucide-solid`](https://lucide.dev) components. **Never pass a literal
`size`** — sizes are centralized in `frontend/src/lib/icons.ts` as the `ICON`
scale, so the whole app's icon sizing lives in one file. Import and pick the
token whose px fits the spot:

```tsx
import { ICON } from "../lib/icons";
<Play size={ICON.xxl} />
```

`ICON` keys are **size labels, not roles** — `md` means 16px, nothing more. The
same UI element may use different tokens in different contexts (a caret can be
`md` inline vs `lg` in the sidebar); that's expected, pick by eye. Examples
below are illustrative, not binding:

| token | px | Example uses |
|-------|----|--------------|
| `ICON.xxl` | 22 | Titlebar palette+gear, tree-row play/pencil/delete |
| `ICON.xl`  | 20 | Sidebar header action buttons (new/import/history/settings/open) |
| `ICON.lg`  | 18 | Sidebar folder caret |
| `ICON.md`  | 16 | Modal/panel close (X), JSON-tree caret, dropdown chevrons, tab close/new |
| `ICON.sm`  | 15 | Icon beside a text label (context menus, assert marks) |
| `ICON.xs`  | 14 | Small inline glyphs (search box, "Add row", external link) — floor |

Floor is 14px; smaller reads as broken on dark themes. Bumping a value in
`icons.ts` shifts every call site using that token.

`.icon-btn` boxes: 26px default, 30px in `.titlebar-actions` / `.sidebar-actions`
/ `.env-switcher`, 26px for `.icon-btn.tiny` (tree-row hover actions) — see
`styles.css`. `.icon-btn` icons draw in `--text-dim` and brighten to `--text`
on hover.

> **Editing note:** change icon sizes via the `ICON` map (or the Edit tool for
> component code), never `sed`/shell redirects — Bash file edits run sandboxed
> and silently revert.
