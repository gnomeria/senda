// Shared dismiss logic for right-click context menus (TabBar, Sidebar).
//
// Previous bug: each opener registered two independent {once:true} document
// listeners (click + contextmenu). The .ctx-menu element stops click
// propagation, so picking a menu item never reached document — the once-click
// listener fired only on an *outside* click, and the once-contextmenu listener
// often never fired at all. A leftover contextmenu listener then swallowed the
// *next* right-click (it closed the menu the new right-click had just opened),
// so the context menu appeared dead after using any menu action — notably one
// that opens a modal. This centralizes teardown so listeners never accumulate.

let activeTeardown: (() => void) | null = null;

// attachCtxDismiss wires document/window listeners that close an open context
// menu on any outside click, another right-click, Escape, or window blur.
// `close` should clear the menu's own signal. Returns a teardown to call when
// the menu is closed via a menu action. Opening a new menu tears down the
// previous one first, so exactly one set of listeners is ever live.
export function attachCtxDismiss(close: () => void): () => void {
  activeTeardown?.();

  const onKey = (e: KeyboardEvent) => {
    if (e.key === "Escape") teardown();
  };
  const onDismiss = () => teardown();

  const teardown = () => {
    document.removeEventListener("click", onDismiss);
    document.removeEventListener("contextmenu", onDismiss);
    document.removeEventListener("keydown", onKey);
    window.removeEventListener("blur", onDismiss);
    if (activeTeardown === teardown) activeTeardown = null;
    close();
  };

  // Defer attaching so the click/contextmenu event that opened this menu does
  // not immediately dismiss it. A fresh right-click on another target tears
  // this menu down (via activeTeardown) before its own listeners are added,
  // so the new menu survives.
  setTimeout(() => {
    document.addEventListener("click", onDismiss);
    document.addEventListener("contextmenu", onDismiss);
    document.addEventListener("keydown", onKey);
    window.addEventListener("blur", onDismiss);
  }, 0);

  activeTeardown = teardown;
  return teardown;
}
