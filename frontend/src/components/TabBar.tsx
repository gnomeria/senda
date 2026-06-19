// Strip of open request tabs above the editor. Each tab carries its own
// request model, response, and dirty state (see store.ts tab layer).
import { createSignal, For, onCleanup, Show } from "solid-js";
import { Plus, X } from "lucide-solid";
import { ICON } from "../lib/icons";
import { api } from "../lib/api";
import { attachCtxDismiss } from "../lib/ctxMenu";
import {
  activeTabId,
  closeAllTabs,
  closeOtherTabs,
  closeSavedTabs,
  closeTab,
  closeTabsToLeft,
  closeTabsToRight,
  cloneTab,
  dirty,
  newTab,
  revertPath,
  revertTabTo,
  switchTab,
  tabs,
} from "../lib/store";

type CtxMenu = { x: number; y: number; tabId: string };

export default function TabBar() {
  const [ctxMenu, setCtxMenu] = createSignal<CtxMenu | null>(null);
  let closeCtx: (() => void) | null = null;

  const openCtxMenu = (e: MouseEvent, tabId: string) => {
    e.preventDefault();
    e.stopPropagation();
    setCtxMenu({ x: e.clientX, y: e.clientY, tabId });
    closeCtx = attachCtxDismiss(() => setCtxMenu(null));
  };
  onCleanup(() => closeCtx?.());

  const act = (fn: () => void) => { closeCtx?.(); fn(); };

  const revert = async (id: string) => {
    closeCtx?.();
    const path = revertPath(id);
    if (!path) return;
    try {
      revertTabTo(id, await api.readRequest(path));
    } catch {
      /* file gone; leave the tab as-is */
    }
  };

  const onWheel = (e: WheelEvent) => {
    const el = e.currentTarget as HTMLElement;
    if (el.scrollWidth > el.clientWidth && e.deltaY !== 0) {
      e.preventDefault();
      el.scrollLeft += e.deltaY;
    }
  };

  return (
    <div class="tabbar" onWheel={onWheel}>
      <For each={tabs}>
        {(t) => {
          const isActive = () => t.id === activeTabId();
          const isDirty = () => (isActive() ? dirty() : t.dirty);
          return (
            <div
              class="tab"
              classList={{ active: isActive() }}
              onClick={() => switchTab(t.id)}
              onMouseDown={(e) => {
                if (e.button === 1) { e.preventDefault(); closeTab(t.id); }
              }}
              onContextMenu={(e) => openCtxMenu(e, t.id)}
              title={t.path || t.title}
            >
              <span class="tab-dot" classList={{ on: isDirty() }} />
              <span class="tab-title">{t.title}</span>
              <button
                class="tab-close"
                title="Close tab"
                onClick={(e) => {
                  e.stopPropagation();
                  closeTab(t.id);
                }}
              >
                <X size={ICON.md} />
              </button>
            </div>
          );
        }}
      </For>
      <button class="tab-new" title="New tab" onClick={() => newTab()}>
        <Plus size={ICON.md} />
      </button>

      <Show when={ctxMenu()}>
        {(() => {
          const m = ctxMenu()!;
          const tab = () => tabs.find((t) => t.id === m.tabId);
          const idx = () => tabs.findIndex((t) => t.id === m.tabId);
          const isLast = () => tabs.length <= 1;
          const tabDirty = () => {
            const t = tab();
            if (!t) return false;
            return t.id === activeTabId() ? dirty() : t.dirty;
          };
          const noSaved = () => !tabs.some((t) => t.path && !(t.id === activeTabId() ? dirty() : t.dirty));
          return (
            <div
              class="ctx-menu"
              style={{ left: `${m.x}px`, top: `${m.y}px` }}
              onClick={(e) => e.stopPropagation()}
            >
              <button class="ctx-item" onClick={() => act(() => newTab())}>
                New request
              </button>
              <button class="ctx-item" onClick={() => act(() => cloneTab(m.tabId))}>
                Clone request
              </button>
              <button
                class="ctx-item"
                disabled={!tab()?.path || !tabDirty()}
                onClick={() => void revert(m.tabId)}
              >
                Revert changes
              </button>
              <div class="ctx-sep" />
              <button class="ctx-item ctx-item-danger" onClick={() => act(() => closeTab(m.tabId))}>
                <X size={ICON.sm} /> Close
              </button>
              <button class="ctx-item" disabled={isLast()} onClick={() => act(() => closeOtherTabs(m.tabId))}>
                Close others
              </button>
              <button
                class="ctx-item"
                disabled={idx() <= 0}
                onClick={() => act(() => closeTabsToLeft(m.tabId))}
              >
                Close to the left
              </button>
              <button
                class="ctx-item"
                disabled={tabs[tabs.length - 1]?.id === m.tabId}
                onClick={() => act(() => closeTabsToRight(m.tabId))}
              >
                Close to the right
              </button>
              <button class="ctx-item" disabled={noSaved()} onClick={() => act(() => closeSavedTabs())}>
                Close saved
              </button>
              <button class="ctx-item ctx-item-danger" onClick={() => act(() => closeAllTabs())}>
                Close all
              </button>
            </div>
          );
        })()}
      </Show>
    </div>
  );
}
