// Collection-level settings modal. Currently edits the collection's default
// auth, which per-request "Inherit" auth falls back to. Saves to senda.meta.yaml via
// SaveCollection and updates the in-memory collection.
import { createSignal, Show } from "solid-js";
import { X } from "lucide-solid";
import { ICON } from "../lib/icons";
import { api } from "../lib/api";
import type { Auth, Collection } from "../lib/api";
import { collection, setCollection } from "../lib/store";
import { blankAuth } from "../lib/factory";
import AuthEditor from "./AuthEditor";

export default function CollectionSettings(props: { onClose: () => void }) {
  const coll = collection()!;
  const [auth, setAuth] = createSignal<Auth>(
    (coll.auth && coll.auth.type ? coll.auth : blankAuth()) as Auth
  );
  const [saving, setSaving] = createSignal(false);
  const [error, setError] = createSignal("");

  const save = async () => {
    setSaving(true);
    setError("");
    try {
      const next: Collection = { ...coll, auth: auth() };
      await api.saveCollection(next);
      setCollection(next);
      props.onClose();
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="modal-title">{coll.name} — Settings</span>
          <button class="icon-btn" title="Close" onClick={props.onClose}>
            <X size={ICON.md} />
          </button>
        </div>
        <div class="modal-body">
          <div class="modal-section-label">Default authentication</div>
          <AuthEditor
            auth={auth()}
            onChange={setAuth}
            allowInherit={false}
          />
        </div>
        <Show when={error()}>
          <div class="modal-error">{error()}</div>
        </Show>
        <div class="modal-foot">
          <button class="btn ghost" onClick={props.onClose}>
            Cancel
          </button>
          <button class="btn" onClick={save} disabled={saving()}>
            {saving() ? "Saving…" : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}
