// Code-generation modal: renders the active request as a snippet for a chosen
// target (curl, fetch, httpie, python, go) and copies it to the clipboard.
import { createResource, createSignal, For, Show } from "solid-js";
import { X } from "lucide-solid";
import { ICON } from "../lib/icons";
import { api } from "../lib/api";
import { request } from "../lib/store";

const TARGETS = ["curl", "fetch", "httpie", "python", "go"];
const LABELS: Record<string, string> = {
  curl: "cURL",
  fetch: "JS fetch",
  httpie: "HTTPie",
  python: "Python",
  go: "Go",
};

export default function CodeGenDialog(props: { onClose: () => void }) {
  const [target, setTarget] = createSignal("curl");
  const [copied, setCopied] = createSignal(false);

  // Snapshot the request once so the snippet is stable while the dialog is open.
  const snapshot = JSON.parse(JSON.stringify(request));
  const [code] = createResource(target, (t) => api.generateCode(snapshot, t));

  const copy = async () => {
    const c = code();
    if (!c) return;
    await navigator.clipboard.writeText(c);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal modal-wide" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="modal-title">Generate code</span>
          <button class="icon-btn" title="Close" onClick={props.onClose}>
            <X size={ICON.md} />
          </button>
        </div>
        <div class="modal-body">
          <div class="seg">
            <For each={TARGETS}>
              {(t) => (
                <button classList={{ active: target() === t }} onClick={() => setTarget(t)}>
                  {LABELS[t]}
                </button>
              )}
            </For>
          </div>
          <pre class="code-out">{code.loading ? "…" : (code() ?? "")}</pre>
        </div>
        <div class="modal-foot">
          <button class="btn ghost" onClick={props.onClose}>
            Close
          </button>
          <button class="btn" onClick={copy}>
            <Show when={copied()} fallback="Copy">
              Copied ✓
            </Show>
          </button>
        </div>
      </div>
    </div>
  );
}
