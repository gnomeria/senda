// Appearance dialog: light/dark/system mode toggle plus a theme picker for
// each side. Both columns stay visible so switching mode never loses the
// other side's choice; the column currently in effect carries an "Active"
// badge.
import { For, Show } from "solid-js";
import { Check, Monitor, Moon, Sun, X } from "lucide-solid";
import { ICON } from "../lib/icons";
import {
  darkThemeId,
  darkThemes,
  lightThemeId,
  lightThemes,
  resolvedKind,
  setDarkTheme,
  setLightTheme,
  setThemeMode,
  themeMode,
  type Theme,
  type ThemeMode,
} from "../lib/theme";

const modes: { mode: ThemeMode; title: string; icon: typeof Sun }[] = [
  { mode: "light", title: "Light mode", icon: Sun },
  { mode: "dark", title: "Dark mode", icon: Moon },
  { mode: "system", title: "Follow system", icon: Monitor },
];

function ThemeList(props: {
  label: string;
  themes: Theme[];
  selected: string;
  active: boolean;
  onPick: (id: string) => void;
}) {
  return (
    <div class="appearance-col">
      <div class="appearance-col-head">
        <span class="appearance-col-label">{props.label}</span>
        <Show when={props.active}>
          <span class="appearance-active-badge">Active</span>
        </Show>
      </div>
      <div class="appearance-list" role="listbox" aria-label={props.label}>
        <For each={props.themes}>
          {(t) => (
            <button
              class="appearance-item"
              classList={{ selected: t.id === props.selected }}
              role="option"
              aria-selected={t.id === props.selected}
              onClick={() => props.onPick(t.id)}
            >
              <span>{t.name}</span>
              <Show when={t.id === props.selected}>
                <Check size={ICON.xs} class="appearance-check" />
              </Show>
            </button>
          )}
        </For>
      </div>
    </div>
  );
}

export default function ThemeSettings(props: { onClose: () => void }) {
  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal appearance-modal" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="modal-title">Appearance</span>
          <button class="icon-btn" title="Close" onClick={props.onClose}>
            <X size={ICON.md} />
          </button>
        </div>
        <div class="appearance-modes">
          <For each={modes}>
            {(m) => (
              <button
                class="appearance-mode-btn"
                classList={{ active: themeMode() === m.mode }}
                title={m.title}
                aria-label={m.title}
                aria-pressed={themeMode() === m.mode}
                onClick={() => setThemeMode(m.mode)}
              >
                <m.icon size={ICON.md} />
              </button>
            )}
          </For>
        </div>
        <div class="appearance-cols">
          <ThemeList
            label="Light theme"
            themes={lightThemes}
            selected={lightThemeId()}
            active={resolvedKind() === "light"}
            onPick={setLightTheme}
          />
          <ThemeList
            label="Dark theme"
            themes={darkThemes}
            selected={darkThemeId()}
            active={resolvedKind() === "dark"}
            onPick={setDarkTheme}
          />
        </div>
      </div>
    </div>
  );
}
