/** @type {import('tailwindcss').Config} */
// Tailwind is wired to the app's existing design tokens rather than its own
// palette: colors resolve to the CSS custom properties that theme.ts swaps per
// theme (so utilities follow the active theme automatically), and font sizes
// resolve to the --fs-* scale declared in styles.css (one place to tune text
// size instead of px littered across the stylesheet).
//
// Preflight is OFF so Tailwind utilities coexist with the hand-written
// styles.css; we adopt utilities incrementally without a global CSS reset.
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  corePlugins: { preflight: false },
  theme: {
    colors: {
      transparent: "transparent",
      current: "currentColor",
      bg: "var(--bg)",
      "bg-elev": "var(--bg-elev)",
      "bg-elev2": "var(--bg-elev2)",
      border: "var(--border)",
      "border-soft": "var(--border-soft)",
      text: {
        DEFAULT: "var(--text)",
        dim: "var(--text-dim)",
        faint: "var(--text-faint)",
      },
      accent: {
        DEFAULT: "var(--accent)",
        dim: "var(--accent-dim)",
        fg: "var(--accent-fg)",
      },
      selection: {
        DEFAULT: "var(--selection)",
        fg: "var(--selection-fg)",
      },
      ok: "var(--ok)",
      warn: "var(--warn)",
      err: "var(--err)",
      redirect: "var(--redirect)",
    },
    fontFamily: {
      sans: "var(--sans)",
      mono: "var(--mono)",
    },
    fontSize: {
      "3xs": "var(--fs-3xs)",
      "2xs": "var(--fs-2xs)",
      xs: "var(--fs-xs)",
      sm: "var(--fs-sm)",
      base: "var(--fs-base)",
      lg: "var(--fs-lg)",
      xl: "var(--fs-xl)",
    },
    extend: {},
  },
  plugins: [],
};
