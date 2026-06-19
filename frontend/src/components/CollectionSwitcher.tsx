// Workspace switcher in the top bar: shows the active collection and drops a
// menu of recently-opened collections to jump between, plus an entry to open a
// new one. Recents persist in localStorage (see store.ts); opening a dead path
// prunes it from the list.
import { createSignal, For, Show } from "solid-js";
import { ChevronDown, FileArchive, FolderOpen, FolderPlus } from "lucide-solid";
import { ICON } from "../lib/icons";
import { collection, recents } from "../lib/store";
import { openCollectionDialog, openZipCollectionDialog, switchCollection } from "../lib/actions";

export default function CollectionSwitcher() {
  const [open, setOpen] = createSignal(false);

  const pick = async (path: string) => {
    setOpen(false);
    try {
      await switchCollection(path);
    } catch {
      /* moved/deleted — switchCollection already pruned it */
    }
  };

  const openNew = async () => {
    setOpen(false);
    await openCollectionDialog();
  };

  const openZip = async () => {
    setOpen(false);
    await openZipCollectionDialog();
  };

  return (
    <div class="coll-switcher">
      <button class="coll-switcher-btn" onClick={() => setOpen(!open())} title="Switch collection">
        <FolderOpen size={ICON.sm} />
        <span class="coll-switcher-name">{collection()?.name ?? "Open collection"}</span>
        <ChevronDown size={ICON.xs} />
      </button>

      <Show when={open()}>
        <div class="menu-backdrop" onClick={() => setOpen(false)} />
        <div class="coll-menu">
          <Show
            when={recents().length > 0}
            fallback={<div class="coll-menu-empty">No recent collections</div>}
          >
            <For each={recents()}>
              {(r) => (
                <button
                  class="coll-menu-item"
                  classList={{ "coll-menu-active": r.path === collection()?.path }}
                  onClick={() => pick(r.path)}
                  title={r.path}
                >
                  <span class="coll-menu-item-name">{r.name}</span>
                  <span class="coll-menu-item-path">{r.path}</span>
                </button>
              )}
            </For>
          </Show>
          <div class="coll-menu-sep" />
          <button class="coll-menu-item coll-menu-open" onClick={openNew}>
            <FolderPlus size={ICON.xs} />
            <span class="coll-menu-item-name">Open collection…</span>
          </button>
          <button class="coll-menu-item coll-menu-open" onClick={openZip}>
            <FileArchive size={ICON.xs} />
            <span class="coll-menu-item-name">Open .zip collection…</span>
          </button>
        </div>
      </Show>
    </div>
  );
}
