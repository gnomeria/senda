import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  DEFAULT_DARK,
  DEFAULT_LIGHT,
  applyTheme,
  darkThemes,
  initTheme,
  lightThemes,
  resolvedKind,
  setDarkTheme,
  setLightTheme,
  setThemeMode,
  themeById,
  themeMode,
  themes,
} from "./theme";

// Install a controllable matchMedia. jsdom has none, and "system" mode and
// initTheme both depend on it.
function mockMatchMedia(prefersDark: boolean) {
  const listeners: Array<() => void> = [];
  const mql = {
    matches: prefersDark,
    media: "(prefers-color-scheme: dark)",
    addEventListener: (_: string, fn: () => void) => listeners.push(fn),
    removeEventListener: (_: string, fn: () => void) => {
      const i = listeners.indexOf(fn);
      if (i >= 0) listeners.splice(i, 1);
    },
  };
  vi.stubGlobal("matchMedia", vi.fn().mockReturnValue(mql));
  return {
    mql,
    listeners,
    setPrefersDark(v: boolean) {
      mql.matches = v;
      for (const fn of [...listeners]) fn();
    },
  };
}

beforeEach(() => {
  localStorage.clear();
  vi.unstubAllGlobals();
  // Reset module-level signals back to defaults between tests.
  setThemeMode("system");
  setLightTheme(DEFAULT_LIGHT);
  setDarkTheme(DEFAULT_DARK);
  localStorage.clear();
});

describe("theme registry", () => {
  it("has unique ids and non-empty names", () => {
    const ids = themes.map((t) => t.id);
    expect(new Set(ids).size).toBe(ids.length);
    for (const t of themes) expect(t.name.length).toBeGreaterThan(0);
  });

  it("splits into light and dark groups that cover the registry", () => {
    expect(lightThemes.length).toBeGreaterThanOrEqual(2);
    expect(darkThemes.length).toBeGreaterThanOrEqual(2);
    expect(lightThemes.length + darkThemes.length).toBe(themes.length);
    for (const t of lightThemes) expect(t.kind).toBe("light");
    for (const t of darkThemes) expect(t.kind).toBe("dark");
  });

  it("includes the defaults", () => {
    expect(lightThemes.some((t) => t.id === DEFAULT_LIGHT)).toBe(true);
    expect(darkThemes.some((t) => t.id === DEFAULT_DARK)).toBe(true);
  });

  it("every theme defines the same token set as the default dark theme", () => {
    const want = Object.keys(themeById(DEFAULT_DARK, "dark").tokens).sort();
    for (const t of themes) {
      expect(Object.keys(t.tokens).sort(), `theme ${t.id}`).toEqual(want);
    }
  });

  it("every token is a non-empty CSS color value", () => {
    for (const t of themes) {
      for (const [k, v] of Object.entries(t.tokens)) {
        expect(v, `${t.id} ${k}`).toMatch(/^#[0-9a-f]{6}([0-9a-f]{2})?$/i);
      }
    }
  });

  it("default dark tokens stay in sync with the :root fallback in styles.css", () => {
    // Tripwire: if these change, update :root in styles.css too.
    const dark = themeById(DEFAULT_DARK, "dark").tokens;
    expect(dark["--bg"]).toBe("#111215");
    expect(dark["--accent"]).toBe("#6e8bff");
    expect(dark["--err"]).toBe("#f85149");
  });
});

describe("themeById", () => {
  it("resolves a known id", () => {
    expect(themeById("nord", "dark").name).toBe("Nord");
  });

  it("falls back to the kind default for an unknown id", () => {
    expect(themeById("does-not-exist", "dark").id).toBe(DEFAULT_DARK);
    expect(themeById("does-not-exist", "light").id).toBe(DEFAULT_LIGHT);
  });

  it("does not resolve an id across kinds", () => {
    // "nord" is dark; asking for it as a light theme yields the light default.
    expect(themeById("nord", "light").id).toBe(DEFAULT_LIGHT);
  });
});

describe("mode resolution", () => {
  it("explicit light/dark modes resolve to themselves", () => {
    expect(resolvedKind("light")).toBe("light");
    expect(resolvedKind("dark")).toBe("dark");
  });

  it("system mode follows the OS preference", () => {
    mockMatchMedia(true);
    expect(resolvedKind("system")).toBe("dark");
    mockMatchMedia(false);
    expect(resolvedKind("system")).toBe("light");
  });

  it("system mode defaults to dark when matchMedia is unavailable", () => {
    // jsdom: no matchMedia at all.
    expect(typeof window.matchMedia).not.toBe("function");
    expect(resolvedKind("system")).toBe("dark");
  });
});

describe("persistence", () => {
  it("setThemeMode persists and updates the signal", () => {
    setThemeMode("light");
    expect(themeMode()).toBe("light");
    expect(localStorage.getItem("senda.themeMode")).toBe("light");
  });

  it("setLightTheme / setDarkTheme persist their choices independently", () => {
    setLightTheme("catppuccin-latte");
    setDarkTheme("nord");
    expect(localStorage.getItem("senda.themeLight")).toBe("catppuccin-latte");
    expect(localStorage.getItem("senda.themeDark")).toBe("nord");
  });

  it("an unknown id is normalized to the default before persisting", () => {
    setDarkTheme("bogus");
    expect(localStorage.getItem("senda.themeDark")).toBe(DEFAULT_DARK);
  });

  it("a fresh module load restores persisted state and survives garbage", async () => {
    localStorage.setItem("senda.themeMode", "dark");
    localStorage.setItem("senda.themeDark", "catppuccin-mocha");
    vi.resetModules();
    let mod = await import("./theme");
    expect(mod.themeMode()).toBe("dark");
    expect(mod.activeTheme().id).toBe("catppuccin-mocha");

    localStorage.setItem("senda.themeMode", "purple"); // not a valid mode
    localStorage.setItem("senda.themeDark", "not-a-theme");
    vi.resetModules();
    mod = await import("./theme");
    expect(mod.themeMode()).toBe("system");
    expect(mod.activeTheme().id).toBe(mod.DEFAULT_DARK);
  });
});

describe("applyTheme", () => {
  it("writes every token, color-scheme and data-theme onto the root", () => {
    const el = document.createElement("div");
    const nord = themeById("nord", "dark");
    applyTheme(nord, el);
    for (const [k, v] of Object.entries(nord.tokens)) {
      expect(el.style.getPropertyValue(k)).toBe(v);
    }
    expect(el.style.colorScheme).toBe("dark");
    expect(el.dataset.theme).toBe("nord");
  });

  it("switching themes overwrites previous values", () => {
    const el = document.createElement("div");
    applyTheme(themeById("nord", "dark"), el);
    const latte = themeById("catppuccin-latte", "light");
    applyTheme(latte, el);
    expect(el.style.getPropertyValue("--bg")).toBe(latte.tokens["--bg"]);
    expect(el.style.colorScheme).toBe("light");
    expect(el.dataset.theme).toBe("catppuccin-latte");
  });

  it("setters re-apply to document.documentElement", () => {
    setThemeMode("dark");
    setDarkTheme("vscode-dark");
    const root = document.documentElement;
    expect(root.dataset.theme).toBe("vscode-dark");
    expect(root.style.getPropertyValue("--bg")).toBe("#1e1e1e");
  });
});

describe("initTheme", () => {
  it("applies the active theme immediately", () => {
    mockMatchMedia(true);
    setThemeMode("system");
    setDarkTheme("catppuccin-frappe");
    const cleanup = initTheme();
    expect(document.documentElement.dataset.theme).toBe("catppuccin-frappe");
    cleanup();
  });

  it("re-applies when the OS preference flips in system mode", () => {
    const media = mockMatchMedia(true);
    setThemeMode("system");
    setLightTheme("catppuccin-latte");
    setDarkTheme("nord");
    const cleanup = initTheme();
    expect(document.documentElement.dataset.theme).toBe("nord");
    media.setPrefersDark(false);
    expect(document.documentElement.dataset.theme).toBe("catppuccin-latte");
    cleanup();
  });

  it("ignores OS flips in an explicit mode and stops after cleanup", () => {
    const media = mockMatchMedia(true);
    setThemeMode("dark");
    setDarkTheme("nord");
    const cleanup = initTheme();
    media.setPrefersDark(false);
    expect(document.documentElement.dataset.theme).toBe("nord");
    cleanup();
    expect(media.listeners.length).toBe(0);
  });
});
