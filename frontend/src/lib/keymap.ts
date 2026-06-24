// Pure keyboard-shortcut decisions, split out of App.tsx so they're unit
// testable. The tab-cycle logic in particular had a webview-only bug: on
// Linux/GTK, Shift+Tab emits the ISO_Left_Tab keysym, so WebKitGTK reports
// e.key as something other than "Tab" and Ctrl+Shift+Tab silently did
// nothing. Matching e.code (the physical key, modifier-independent) fixes it.
// Keep this keyed off e.code so a jsdom test can reproduce the GTK case by
// dispatching {code:"Tab"} with a non-"Tab" key.

// tabCycleDir returns the tab-switch direction for a keydown: +1 next, -1
// previous, 0 when the event isn't a tab-cycle shortcut.
export function tabCycleDir(e: KeyboardEvent): number {
  const mod = e.ctrlKey || e.metaKey;
  if (!mod || e.code !== "Tab") return 0;
  return e.shiftKey ? -1 : 1;
}
