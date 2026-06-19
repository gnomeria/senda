// Theme system: a registry of named color themes plus the appearance state
// (light / dark / system mode and the chosen theme for each side).
//
// Themes are plain maps of CSS custom-property name → value, applied as
// inline styles on <html>. styles.css declares the same variables on :root
// (matching the "dark" theme) so the app renders correctly before initTheme
// runs and in contexts where it never runs (tests, storybook-style harnesses).
import { createSignal } from "solid-js";

export type ThemeKind = "light" | "dark";
export type ThemeMode = ThemeKind | "system";

// Every theme must define exactly this set of variables — themes.test.ts
// enforces it so a new theme can't silently miss a token.
export type ThemeTokens = {
  "--bg": string; // base background
  "--bg-elev": string; // raised surfaces (sidebar, titlebar, modals)
  "--bg-elev2": string; // higher surfaces (inputs, menus, badges)
  "--border": string;
  "--border-soft": string;
  "--text": string;
  "--text-dim": string;
  "--text-faint": string;
  "--accent": string; // primary action color
  "--accent-dim": string; // focus borders, hover borders
  "--accent-fg": string; // text on solid accent fills
  "--selection": string; // selected row / active item background
  "--selection-fg": string; // text on --selection
  "--hover": string; // subtle hover overlay
  "--ok": string;
  "--warn": string;
  "--err": string;
  "--redirect": string; // 3xx + informational blue
};

export type Theme = {
  id: string;
  name: string;
  kind: ThemeKind;
  tokens: ThemeTokens;
};

export const DEFAULT_LIGHT = "light";
export const DEFAULT_DARK = "dark";

export const themes: Theme[] = [
  // ---- light ----
  {
    id: "light",
    name: "Light",
    kind: "light",
    tokens: {
      "--bg": "#ffffff",
      "--bg-elev": "#f6f8fa",
      "--bg-elev2": "#eaeef2",
      "--border": "#d0d7de",
      "--border-soft": "#e2e8f0",
      "--text": "#1f2328",
      "--text-dim": "#57606a",
      "--text-faint": "#818b98",
      "--accent": "#4263eb",
      "--accent-dim": "#a5b4fc",
      "--accent-fg": "#ffffff",
      "--selection": "#dbe3ff",
      "--selection-fg": "#1f2c66",
      "--hover": "#00000008",
      "--ok": "#1a7f37",
      "--warn": "#9a6700",
      "--err": "#cf222e",
      "--redirect": "#0969da",
    },
  },
  {
    id: "light-monochrome",
    name: "Light Monochrome",
    kind: "light",
    tokens: {
      "--bg": "#ffffff",
      "--bg-elev": "#f7f7f7",
      "--bg-elev2": "#ededed",
      "--border": "#d4d4d4",
      "--border-soft": "#e4e4e4",
      "--text": "#1a1a1a",
      "--text-dim": "#565656",
      "--text-faint": "#8c8c8c",
      "--accent": "#333333",
      "--accent-dim": "#9a9a9a",
      "--accent-fg": "#ffffff",
      "--selection": "#e2e2e2",
      "--selection-fg": "#111111",
      "--hover": "#00000008",
      "--ok": "#1a7f37",
      "--warn": "#9a6700",
      "--err": "#cf222e",
      "--redirect": "#57606a",
    },
  },
  {
    id: "light-pastel",
    name: "Light Pastel",
    kind: "light",
    tokens: {
      "--bg": "#faf9fc",
      "--bg-elev": "#f3f1f8",
      "--bg-elev2": "#eae7f3",
      "--border": "#d8d3e8",
      "--border-soft": "#e5e1f0",
      "--text": "#2a2440",
      "--text-dim": "#5f5878",
      "--text-faint": "#948daa",
      "--accent": "#8b6fc7",
      "--accent-dim": "#c3b2e3",
      "--accent-fg": "#ffffff",
      "--selection": "#e6dcf7",
      "--selection-fg": "#43306b",
      "--hover": "#00000008",
      "--ok": "#4c9a6a",
      "--warn": "#c08a3e",
      "--err": "#d26a7c",
      "--redirect": "#6a8fd8",
    },
  },
  {
    id: "catppuccin-latte",
    name: "Catppuccin Latte",
    kind: "light",
    tokens: {
      "--bg": "#eff1f5",
      "--bg-elev": "#e6e9ef",
      "--bg-elev2": "#dce0e8",
      "--border": "#bcc0cc",
      "--border-soft": "#ccd0da",
      "--text": "#4c4f69",
      "--text-dim": "#6c6f85",
      "--text-faint": "#9ca0b0",
      "--accent": "#1e66f5",
      "--accent-dim": "#7287fd",
      "--accent-fg": "#ffffff",
      "--selection": "#d2dcf8",
      "--selection-fg": "#2a3d8f",
      "--hover": "#00000008",
      "--ok": "#40a02b",
      "--warn": "#df8e1d",
      "--err": "#d20f39",
      "--redirect": "#209fb5",
    },
  },
  {
    id: "vscode-light",
    name: "VS Code Light",
    kind: "light",
    tokens: {
      "--bg": "#ffffff",
      "--bg-elev": "#f3f3f3",
      "--bg-elev2": "#ececec",
      "--border": "#d4d4d4",
      "--border-soft": "#e5e5e5",
      "--text": "#1f1f1f",
      "--text-dim": "#616161",
      "--text-faint": "#8e8e8e",
      "--accent": "#007acc",
      "--accent-dim": "#98c8ec",
      "--accent-fg": "#ffffff",
      "--selection": "#add6ff",
      "--selection-fg": "#1f1f1f",
      "--hover": "#00000008",
      "--ok": "#008000",
      "--warn": "#bf8803",
      "--err": "#e51400",
      "--redirect": "#0070c1",
    },
  },
  // ---- dark ----
  {
    id: "dark",
    name: "Dark",
    kind: "dark",
    // Must stay in sync with the :root block in styles.css (the pre-init
    // fallback). themes.test.ts pins a few values as a tripwire.
    tokens: {
      "--bg": "#111215",
      "--bg-elev": "#16181c",
      "--bg-elev2": "#1c1f25",
      "--border": "#262a31",
      "--border-soft": "#20242a",
      "--text": "#e6e8ec",
      "--text-dim": "#9aa1ac",
      "--text-faint": "#5a616d",
      "--accent": "#6e8bff",
      "--accent-dim": "#3d4f8a",
      "--accent-fg": "#ffffff",
      "--selection": "#3d4f8a",
      "--selection-fg": "#ffffff",
      "--hover": "#ffffff08",
      "--ok": "#3fb950",
      "--warn": "#d29922",
      "--err": "#f85149",
      "--redirect": "#58a6ff",
    },
  },
  {
    id: "oled",
    name: "OLED Dark",
    kind: "dark",
    // Pure-black base for OLED panels (zero-light pixels). Surfaces step up
    // from #000 in small increments so elevation stays visible without leaking
    // gray onto the background.
    tokens: {
      "--bg": "#000000",
      "--bg-elev": "#0a0a0c",
      "--bg-elev2": "#141519",
      "--border": "#23262e",
      "--border-soft": "#181a20",
      "--text": "#e9ebf0",
      "--text-dim": "#9097a3",
      "--text-faint": "#565d68",
      "--accent": "#6e8bff",
      "--accent-dim": "#34406e",
      "--accent-fg": "#ffffff",
      "--selection": "#34406e",
      "--selection-fg": "#ffffff",
      "--hover": "#ffffff0d",
      "--ok": "#3fb950",
      "--warn": "#d29922",
      "--err": "#f85149",
      "--redirect": "#58a6ff",
    },
  },
  {
    id: "dark-monochrome",
    name: "Dark Monochrome",
    kind: "dark",
    tokens: {
      "--bg": "#0e0e0e",
      "--bg-elev": "#151515",
      "--bg-elev2": "#1d1d1d",
      "--border": "#2a2a2a",
      "--border-soft": "#232323",
      "--text": "#e8e8e8",
      "--text-dim": "#a0a0a0",
      "--text-faint": "#5e5e5e",
      "--accent": "#d4d4d4",
      "--accent-dim": "#4a4a4a",
      "--accent-fg": "#111111",
      "--selection": "#333333",
      "--selection-fg": "#ffffff",
      "--hover": "#ffffff08",
      "--ok": "#3fb950",
      "--warn": "#d29922",
      "--err": "#f85149",
      "--redirect": "#9aa1ac",
    },
  },
  {
    id: "dark-pastel",
    name: "Dark Pastel",
    kind: "dark",
    tokens: {
      "--bg": "#16161e",
      "--bg-elev": "#1b1b26",
      "--bg-elev2": "#22222f",
      "--border": "#2e2e3f",
      "--border-soft": "#282836",
      "--text": "#e9e6f2",
      "--text-dim": "#a8a3bd",
      "--text-faint": "#645f7a",
      "--accent": "#c4a7e7",
      "--accent-dim": "#4f3d6b",
      "--accent-fg": "#1d1430",
      "--selection": "#4f3d6b",
      "--selection-fg": "#f2ecff",
      "--hover": "#ffffff0a",
      "--ok": "#8fd9a8",
      "--warn": "#f0c987",
      "--err": "#ee8499",
      "--redirect": "#8ec5fc",
    },
  },
  {
    id: "catppuccin-frappe",
    name: "Catppuccin Frappé",
    kind: "dark",
    tokens: {
      "--bg": "#303446",
      "--bg-elev": "#292c3c",
      "--bg-elev2": "#414559",
      "--border": "#51576d",
      "--border-soft": "#414559",
      "--text": "#c6d0f5",
      "--text-dim": "#a5adce",
      "--text-faint": "#737994",
      "--accent": "#8caaee",
      "--accent-dim": "#45527a",
      "--accent-fg": "#232634",
      "--selection": "#45527a",
      "--selection-fg": "#e8eeff",
      "--hover": "#ffffff08",
      "--ok": "#a6d189",
      "--warn": "#e5c890",
      "--err": "#e78284",
      "--redirect": "#85c1dc",
    },
  },
  {
    id: "catppuccin-macchiato",
    name: "Catppuccin Macchiato",
    kind: "dark",
    tokens: {
      "--bg": "#24273a",
      "--bg-elev": "#1e2030",
      "--bg-elev2": "#363a4f",
      "--border": "#494d64",
      "--border-soft": "#363a4f",
      "--text": "#cad3f5",
      "--text-dim": "#a5adcb",
      "--text-faint": "#6e738d",
      "--accent": "#8aadf4",
      "--accent-dim": "#44507a",
      "--accent-fg": "#181926",
      "--selection": "#44507a",
      "--selection-fg": "#e8eeff",
      "--hover": "#ffffff08",
      "--ok": "#a6da95",
      "--warn": "#eed49f",
      "--err": "#ed8796",
      "--redirect": "#7dc4e4",
    },
  },
  {
    id: "catppuccin-mocha",
    name: "Catppuccin Mocha",
    kind: "dark",
    tokens: {
      "--bg": "#1e1e2e",
      "--bg-elev": "#181825",
      "--bg-elev2": "#313244",
      "--border": "#45475a",
      "--border-soft": "#313244",
      "--text": "#cdd6f4",
      "--text-dim": "#a6adc8",
      "--text-faint": "#6c7086",
      "--accent": "#89b4fa",
      "--accent-dim": "#41527e",
      "--accent-fg": "#11111b",
      "--selection": "#41527e",
      "--selection-fg": "#e8eeff",
      "--hover": "#ffffff08",
      "--ok": "#a6e3a1",
      "--warn": "#f9e2af",
      "--err": "#f38ba8",
      "--redirect": "#74c7ec",
    },
  },
  {
    id: "nord",
    name: "Nord",
    kind: "dark",
    tokens: {
      "--bg": "#2e3440",
      "--bg-elev": "#3b4252",
      "--bg-elev2": "#434c5e",
      "--border": "#4c566a",
      "--border-soft": "#434c5e",
      "--text": "#eceff4",
      "--text-dim": "#d8dee9",
      "--text-faint": "#7b88a1",
      "--accent": "#88c0d0",
      "--accent-dim": "#5e81ac",
      "--accent-fg": "#2e3440",
      "--selection": "#5e81ac",
      "--selection-fg": "#eceff4",
      "--hover": "#ffffff08",
      "--ok": "#a3be8c",
      "--warn": "#ebcb8b",
      "--err": "#bf616a",
      "--redirect": "#81a1c1",
    },
  },
  {
    id: "vscode-dark",
    name: "VS Code Dark",
    kind: "dark",
    tokens: {
      "--bg": "#1e1e1e",
      "--bg-elev": "#252526",
      "--bg-elev2": "#2d2d30",
      "--border": "#3e3e42",
      "--border-soft": "#333337",
      "--text": "#d4d4d4",
      "--text-dim": "#a6a6a6",
      "--text-faint": "#6e6e6e",
      "--accent": "#0e639c",
      "--accent-dim": "#094771",
      "--accent-fg": "#ffffff",
      "--selection": "#094771",
      "--selection-fg": "#e8e8e8",
      "--hover": "#ffffff08",
      "--ok": "#89d185",
      "--warn": "#cca700",
      "--err": "#f48771",
      "--redirect": "#569cd6",
    },
  },
];

export const lightThemes = themes.filter((t) => t.kind === "light");
export const darkThemes = themes.filter((t) => t.kind === "dark");

// themeById resolves an id to a theme of the given kind, falling back to the
// kind's default when the id is unknown (e.g. a theme was removed between
// versions but its id is still in localStorage).
export function themeById(id: string, kind: ThemeKind): Theme {
  const fallback = kind === "light" ? DEFAULT_LIGHT : DEFAULT_DARK;
  return (
    themes.find((t) => t.id === id && t.kind === kind) ??
    themes.find((t) => t.id === fallback)!
  );
}

// ---- persisted appearance state ------------------------------------------

const MODE_KEY = "senda.themeMode";
const LIGHT_KEY = "senda.themeLight";
const DARK_KEY = "senda.themeDark";

function storedMode(): ThemeMode {
  const v = localStorage.getItem(MODE_KEY);
  return v === "light" || v === "dark" || v === "system" ? v : "system";
}

export const [themeMode, setThemeModeSignal] = createSignal<ThemeMode>(storedMode());
export const [lightThemeId, setLightThemeIdSignal] = createSignal<string>(
  localStorage.getItem(LIGHT_KEY) ?? DEFAULT_LIGHT
);
export const [darkThemeId, setDarkThemeIdSignal] = createSignal<string>(
  localStorage.getItem(DARK_KEY) ?? DEFAULT_DARK
);

// systemPrefersDark reads the OS preference; defaults to dark where
// matchMedia is unavailable (jsdom) since dark is the app's native look.
export function systemPrefersDark(): boolean {
  if (typeof window.matchMedia !== "function") return true;
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

// resolvedKind maps the 3-state mode to the 2-state kind actually rendered.
export function resolvedKind(mode: ThemeMode = themeMode()): ThemeKind {
  if (mode === "system") return systemPrefersDark() ? "dark" : "light";
  return mode;
}

// activeTheme is the theme currently in effect given mode + per-kind choice.
export function activeTheme(): Theme {
  const kind = resolvedKind();
  return themeById(kind === "light" ? lightThemeId() : darkThemeId(), kind);
}

// applyTheme writes a theme's variables onto an element (the <html> root in
// production; injectable for tests).
export function applyTheme(
  theme: Theme = activeTheme(),
  root: HTMLElement = document.documentElement
) {
  for (const [k, v] of Object.entries(theme.tokens)) {
    root.style.setProperty(k, v);
  }
  root.style.colorScheme = theme.kind; // native widgets follow the theme
  root.dataset.theme = theme.id;
}

export function setThemeMode(mode: ThemeMode) {
  setThemeModeSignal(mode);
  localStorage.setItem(MODE_KEY, mode);
  applyTheme();
}

export function setLightTheme(id: string) {
  setLightThemeIdSignal(themeById(id, "light").id);
  localStorage.setItem(LIGHT_KEY, lightThemeId());
  applyTheme();
}

export function setDarkTheme(id: string) {
  setDarkThemeIdSignal(themeById(id, "dark").id);
  localStorage.setItem(DARK_KEY, darkThemeId());
  applyTheme();
}

// initTheme applies the persisted theme and tracks OS light/dark changes so
// "system" mode follows live. Returns a cleanup for the media listener.
export function initTheme(): () => void {
  applyTheme();
  if (typeof window.matchMedia !== "function") return () => {};
  const mq = window.matchMedia("(prefers-color-scheme: dark)");
  const onChange = () => {
    if (themeMode() === "system") applyTheme();
  };
  mq.addEventListener("change", onChange);
  return () => mq.removeEventListener("change", onChange);
}
