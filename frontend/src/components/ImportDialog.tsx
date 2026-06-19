// Import modal: paste a curl command (opens as a new tab) or paste a Postman
// v2.1 / OpenAPI 3 document (writes request files into the open collection).
import { createSignal, Match, Show, Switch } from "solid-js";
import { X } from "lucide-solid";
import { ICON } from "../lib/icons";
import { api } from "../lib/api";
import { collection, openInTab, setCollection, setEnvironments } from "../lib/store";

type Mode = "curl" | "postman" | "openapi";

export default function ImportDialog(props: { onClose: () => void }) {
  const [mode, setMode] = createSignal<Mode>("curl");
  const [text, setText] = createSignal("");
  const [subdir, setSubdir] = createSignal("");
  const [genMocks, setGenMocks] = createSignal(true);
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal("");
  const [done, setDone] = createSignal("");

  const run = async () => {
    setBusy(true);
    setError("");
    setDone("");
    try {
      if (mode() === "curl") {
        const req = await api.importCurl(text());
        openInTab(req, "");
        props.onClose();
        return;
      }
      const coll = collection();
      if (!coll) {
        setError("Open a collection first.");
        return;
      }
      const n = await api.importCollection(coll.path, mode(), text(), subdir());
      let msg = `Imported ${n} request${n === 1 ? "" : "s"}.`;
      // For OpenAPI specs, optionally also generate a runnable mock server from
      // the documented responses/examples (written into the collection's mocks/).
      if (mode() === "openapi" && genMocks()) {
        const m = await api.generateMocksFromOpenAPI(coll.path, text());
        msg += ` Generated ${m} mock${m === 1 ? "" : "s"} — start the mock server to serve them.`;
      }
      // Refresh tree so the imported requests appear.
      setCollection(await api.openCollection(coll.path));
      setEnvironments((await api.listEnvironments(coll.path)) ?? []);
      setDone(msg);
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const placeholder = () =>
    mode() === "curl"
      ? "curl -X POST https://api.example.com/login -H 'Content-Type: application/json' -d '{...}'"
      : mode() === "postman"
        ? "Paste a Postman Collection v2.1 JSON export…"
        : "Paste an OpenAPI 3 spec (JSON or YAML)…";

  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal modal-wide" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="modal-title">Import</span>
          <button class="icon-btn" title="Close" onClick={props.onClose}>
            <X size={ICON.md} />
          </button>
        </div>
        <div class="modal-body">
          <div class="seg">
            <button classList={{ active: mode() === "curl" }} onClick={() => setMode("curl")}>
              curl
            </button>
            <button classList={{ active: mode() === "postman" }} onClick={() => setMode("postman")}>
              Postman
            </button>
            <button classList={{ active: mode() === "openapi" }} onClick={() => setMode("openapi")}>
              OpenAPI
            </button>
          </div>

          <Show when={mode() !== "curl"}>
            <label class="field-row">
              <span class="field-label">Into subfolder (optional)</span>
              <input
                class="text-input"
                placeholder="imported"
                value={subdir()}
                onInput={(e) => setSubdir(e.currentTarget.value)}
              />
            </label>
          </Show>

          <Show when={mode() === "openapi"}>
            <label class="check-row">
              <input
                type="checkbox"
                checked={genMocks()}
                onChange={(e) => setGenMocks(e.currentTarget.checked)}
              />
              <span>Generate mock server from response examples</span>
            </label>
          </Show>

          <textarea
            class="import-text"
            spellcheck={false}
            placeholder={placeholder()}
            value={text()}
            onInput={(e) => setText(e.currentTarget.value)}
          />
        </div>
        <Show when={error()}>
          <div class="modal-error">{error()}</div>
        </Show>
        <Show when={done()}>
          <div class="modal-ok">{done()}</div>
        </Show>
        <div class="modal-foot">
          <button class="btn ghost" onClick={props.onClose}>
            Close
          </button>
          <button class="btn" onClick={run} disabled={busy() || !text().trim()}>
            <Switch fallback="Import">
              <Match when={busy()}>Importing…</Match>
              <Match when={mode() === "curl"}>Open as tab</Match>
            </Switch>
          </button>
        </div>
      </div>
    </div>
  );
}
