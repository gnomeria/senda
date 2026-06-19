import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@solidjs/testing-library";
import ThemeSettings from "./ThemeSettings";
import {
  DEFAULT_DARK,
  DEFAULT_LIGHT,
  darkThemes,
  lightThemes,
  setDarkTheme,
  setLightTheme,
  setThemeMode,
} from "../lib/theme";

beforeEach(() => {
  localStorage.clear();
  setThemeMode("dark");
  setLightTheme(DEFAULT_LIGHT);
  setDarkTheme(DEFAULT_DARK);
  localStorage.clear();
});

describe("ThemeSettings", () => {
  it("lists every light and dark theme", () => {
    render(() => <ThemeSettings onClose={() => {}} />);
    for (const t of [...lightThemes, ...darkThemes]) {
      expect(screen.getByRole("option", { name: t.name })).toBeInTheDocument();
    }
  });

  it("marks the selected theme in each column", () => {
    render(() => <ThemeSettings onClose={() => {}} />);
    expect(screen.getByRole("option", { name: "Dark" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
    expect(screen.getByRole("option", { name: "Light" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
    expect(screen.getByRole("option", { name: "Nord" })).toHaveAttribute(
      "aria-selected",
      "false"
    );
  });

  it("shows the Active badge on the column matching the resolved mode", () => {
    render(() => <ThemeSettings onClose={() => {}} />);
    const badge = screen.getByText("Active");
    expect(badge.closest(".appearance-col")).toContainElement(
      screen.getByText("Dark theme")
    );
    // Flip to light mode: the badge moves to the light column.
    fireEvent.click(screen.getByRole("button", { name: "Light mode" }));
    expect(screen.getByText("Active").closest(".appearance-col")).toContainElement(
      screen.getByText("Light theme")
    );
  });

  it("picking a dark theme applies and persists it without touching the light pick", () => {
    render(() => <ThemeSettings onClose={() => {}} />);
    fireEvent.click(screen.getByRole("option", { name: "Nord" }));
    expect(document.documentElement.dataset.theme).toBe("nord");
    expect(localStorage.getItem("senda.themeDark")).toBe("nord");
    expect(localStorage.getItem("senda.themeLight")).toBeNull();
    expect(screen.getByRole("option", { name: "Nord" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  it("picking a light theme while dark mode is active stores the choice for later", () => {
    render(() => <ThemeSettings onClose={() => {}} />);
    fireEvent.click(screen.getByRole("option", { name: "Catppuccin Latte" }));
    // Still dark on screen…
    expect(document.documentElement.style.colorScheme).toBe("dark");
    expect(localStorage.getItem("senda.themeLight")).toBe("catppuccin-latte");
    // …until the mode flips.
    fireEvent.click(screen.getByRole("button", { name: "Light mode" }));
    expect(document.documentElement.dataset.theme).toBe("catppuccin-latte");
  });

  it("mode buttons reflect and change the mode", () => {
    render(() => <ThemeSettings onClose={() => {}} />);
    const dark = screen.getByRole("button", { name: "Dark mode" });
    const light = screen.getByRole("button", { name: "Light mode" });
    expect(dark).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(light);
    expect(light).toHaveAttribute("aria-pressed", "true");
    expect(dark).toHaveAttribute("aria-pressed", "false");
    expect(localStorage.getItem("senda.themeMode")).toBe("light");
  });

  it("closes on backdrop click but not on clicks inside the dialog", () => {
    const onClose = vi.fn();
    const { container } = render(() => <ThemeSettings onClose={onClose} />);
    fireEvent.click(container.querySelector(".appearance-modal")!);
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(container.querySelector(".modal-backdrop")!);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
