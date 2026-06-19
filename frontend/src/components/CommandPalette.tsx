// Command palette (Ctrl+K / Ctrl+P): fuzzy-filtered actions, environment
// switching, and every request in the open collection.
import { createMemo, createSignal, For, onMount, Show } from "solid-js";
import { api } from "../lib/api";
import type { TreeNode } from "../lib/api";
import {
  activeEnv,
  activeTabId,
  closeTab,
  collection,
  environments,
  newTab,
  openInTab,
  rememberEnv,
  setActiveEnv,
} from "../lib/store";
import { openCollectionDialog, openZipCollectionDialog, saveActive, sendActive } from "../lib/actions";

type Item = {
  label: string;
  hint: string;
  run: () => void | Promise<void>;
};

// Flatten the collection tree into "folder/sub/name" request entries.
function flattenRequests(node: TreeNode | null | undefined, prefix: string, out: { label: string; path: string }[]) {
  if (!node) return out;
  for (const child of node.children ?? []) {
    if (!child) continue;
    if (child.isDir) {
      flattenRequests(child, prefix + child.name + "/", out);
    } else {
      out.push({ label: prefix + child.name, path: child.path });
    }
  }
  return out;
}

// Subsequence fuzzy match: every query char appears in order.
function fuzzy(query: string, target: string): boolean {
  let qi = 0;
  const q = query.toLowerCase();
  const t = target.toLowerCase();
  for (let ti = 0; qi < q.length && ti < t.length; ti++) {
    if (t[ti] === q[qi]) qi++;
  }
  return qi === q.length;
}

export default function CommandPalette(props: {
  onClose: () => void;
  onOpenTheme?: () => void;
}) {
  const [query, setQuery] = createSignal("");
  const [selected, setSelected] = createSignal(0);
  let input!: HTMLInputElement;

  onMount(() => input.focus());

  const items = createMemo<Item[]>(() => {
    const out: Item[] = [
      { label: "New request tab", hint: "Ctrl+T", run: () => newTab() },
      { label: "Send request", hint: "Ctrl+Enter", run: () => void sendActive() },
      { label: "Save request", hint: "Ctrl+S", run: () => void saveActive() },
      { label: "Close tab", hint: "Ctrl+W", run: () => closeTab(activeTabId()) },
      { label: "Open collection…", hint: "folder", run: () => void openCollectionDialog() },
      { label: "Open .zip collection…", hint: "archive", run: () => void openZipCollectionDialog() },
      { label: "Appearance: change theme…", hint: "", run: () => props.onOpenTheme?.() },
      { label: "Clear cookies", hint: "session", run: () => void api.clearCookies() },
      { label: "Clear runtime vars", hint: "session", run: () => void api.clearRuntimeVars() },
    ];
    for (const env of environments()) {
      if (env.name === activeEnv()) continue;
      out.push({
        label: `Switch environment: ${env.name}`,
        hint: "env",
        run: () => {
          setActiveEnv(env.name);
          rememberEnv(env.name);
        },
      });
    }
    for (const r of flattenRequests(collection()?.tree, "", [])) {
      out.push({
        label: r.label,
        hint: "open",
        run: async () => openInTab(await api.readRequest(r.path), r.path),
      });
    }
    return out;
  });

  const filtered = createMemo(() => {
    const q = query().trim();
    const all = items();
    return q ? all.filter((it) => fuzzy(q, it.label)) : all;
  });

  const pick = async (it: Item | undefined) => {
    if (!it) return;
    props.onClose();
    await it.run();
  };

  const onKey = (e: KeyboardEvent) => {
    const n = filtered().length;
    if (e.key === "Escape") {
      e.preventDefault();
      props.onClose();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelected((s) => (n ? (s + 1) % n : 0));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelected((s) => (n ? (s - 1 + n) % n : 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      void pick(filtered()[selected()]);
    }
  };

  return (
    <div class="palette-backdrop" onClick={props.onClose}>
      <div class="palette" onClick={(e) => e.stopPropagation()}>
        <input
          ref={input}
          class="palette-input"
          placeholder="Type a command or request name…"
          value={query()}
          onInput={(e) => {
            setQuery(e.currentTarget.value);
            setSelected(0);
          }}
          onKeyDown={onKey}
        />
        <div class="palette-list">
          <For each={filtered().slice(0, 50)}>
            {(it, i) => (
              <div
                class="palette-item"
                classList={{ selected: i() === selected() }}
                onMouseEnter={() => setSelected(i())}
                onClick={() => void pick(it)}
              >
                <span class="palette-label">{it.label}</span>
                <Show when={it.hint}>
                  <span class="palette-hint">{it.hint}</span>
                </Show>
              </div>
            )}
          </For>
          <Show when={filtered().length === 0}>
            <div class="palette-empty">No matches.</div>
          </Show>
        </div>
      </div>
    </div>
  );
}
