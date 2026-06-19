// Environment editor dialog: pick/create an environment, edit its variables.
// Writes through api.saveEnvironment (.senda/environments/<name>.yaml).
import { createMemo, createSignal, For, Show } from "solid-js";
import { X } from "lucide-solid";
import { ICON } from "../lib/icons";
import { api } from "../lib/api";
import type { KV } from "../lib/api";
import { collection, environments, setEnvironments } from "../lib/store";
import KVEditor from "./KVEditor";

export default function EnvEditor(props: { onClose: () => void }) {
  const [selected, setSelected] = createSignal(environments()[0]?.name ?? "");
  const [draftVars, setDraftVars] = createSignal<KV[] | null>(null);
  const [saving, setSaving] = createSignal(false);
  const [error, setError] = createSignal("");

  const current = createMemo(() =>
    environments().find((e) => e.name === selected())
  );
  const vars = () => draftVars() ?? current()?.vars ?? [];

  const pick = (name: string) => {
    setSelected(name);
    setDraftVars(null);
    setError("");
  };

  const addEnv = async () => {
    const name = prompt("New environment name:", "dev");
    if (!name) return;
    if (environments().some((e) => e.name === name)) {
      pick(name);
      return;
    }
    await save(name, []);
    pick(name);
  };

  const save = async (name: string, rows: KV[]) => {
    const coll = collection();
    if (!coll) return;
    setSaving(true);
    setError("");
    try {
      await api.saveEnvironment(coll.path, { name, vars: rows } as any);
      setEnvironments((await api.listEnvironments(coll.path)) ?? []);
      setDraftVars(null);
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal modal-wide" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="modal-title">Environments</span>
          <button class="icon-btn" title="Close" onClick={props.onClose}>
            <X size={ICON.md} />
          </button>
        </div>
        <div class="modal-body env-editor-body">
          <div class="env-list">
            <For each={environments()}>
              {(env) => (
                <button
                  class="env-item"
                  classList={{ active: env.name === selected() }}
                  onClick={() => pick(env.name)}
                >
                  {env.name}
                </button>
              )}
            </For>
            <button class="add-row" onClick={addEnv}>
              + New environment
            </button>
          </div>
          <div class="env-vars">
            <Show
              when={selected()}
              fallback={<div class="empty-hint">Create an environment to begin.</div>}
            >
              <KVEditor
                rows={vars()}
                keyPlaceholder="variable"
                onChange={(r) => setDraftVars(r)}
              />
            </Show>
          </div>
        </div>
        <Show when={error()}>
          <div class="modal-error">{error()}</div>
        </Show>
        <div class="modal-foot">
          <button class="btn" onClick={props.onClose}>
            Close
          </button>
          <button
            class="btn primary"
            disabled={!selected() || draftVars() === null || saving()}
            onClick={() => save(selected(), vars())}
          >
            {saving() ? "Saving…" : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}
