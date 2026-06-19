// Per-request detail pane for the folder runner: status line plus
// Response / Headers / Tests tabs for one RunResult, Bruno-style.
import { createMemo, createSignal, For, Show, Switch, Match } from "solid-js";
import { Check, X } from "lucide-solid";
import { ICON } from "../lib/icons";
import type { RunResult } from "../lib/api";
import { formatBytes, statusClass } from "../lib/factory";
import CodeEditor from "./CodeEditor";
import JsonTree from "./JsonTree";
import Timeline from "./Timeline";

type Tab = "body" | "headers" | "timeline" | "tests";

export default function RunDetail(props: { result: RunResult; onClose: () => void }) {
  const [tab, setTab] = createSignal<Tab>("body");
  const resp = () => props.result.response;
  const asserts = () => resp()?.asserts ?? [];
  const failed = () => asserts().filter((a) => !a.pass).length;

  const looksJSON = createMemo(() => {
    const b = resp()?.body?.trimStart() ?? "";
    return b.startsWith("{") || b.startsWith("[");
  });

  return (
    <div class="run-detail">
      <div class="run-detail-head">
        <span class="run-dot" classList={{ ok: props.result.ok, fail: !props.result.ok }} />
        <span class="run-detail-name">{props.result.name || props.result.path}</span>
        <button class="icon-btn" title="Close detail" onClick={props.onClose}>
          <X size={ICON.md} />
        </button>
      </div>

      <Show
        when={resp()}
        fallback={<div class="resp-error">{props.result.error || "No response captured."}</div>}
      >
        {(r) => (
          <>
            <div class="status-line">
              <span class={`status-badge ${statusClass(r().status)}`}>
                {r().status} {r().statusText}
              </span>
              <span class="meta">{r().durationMs} ms</span>
              <span class="meta">{formatBytes(r().sizeBytes)}</span>
              <Show when={r().truncated}>
                <span class="meta warn">truncated</span>
              </Show>
            </div>

            <div class="tabs">
              <button classList={{ active: tab() === "body" }} onClick={() => setTab("body")}>
                Response
              </button>
              <button classList={{ active: tab() === "headers" }} onClick={() => setTab("headers")}>
                Headers
              </button>
              <Show when={r().timing}>
                <button classList={{ active: tab() === "timeline" }} onClick={() => setTab("timeline")}>
                  Timeline
                </button>
              </Show>
              <Show when={asserts().length}>
                <button classList={{ active: tab() === "tests" }} onClick={() => setTab("tests")}>
                  Tests{" "}
                  <span class="assert-badge" classList={{ ok: failed() === 0, err: failed() > 0 }}>
                    {asserts().length - failed()}/{asserts().length}
                  </span>
                </button>
              </Show>
            </div>

            <div class="run-detail-body">
              <Switch>
                <Match when={tab() === "body"}>
                  <Show
                    when={looksJSON()}
                    fallback={<CodeEditor value={r().body} language="text" readOnly />}
                  >
                    <JsonTree text={r().body} />
                  </Show>
                </Match>
                <Match when={tab() === "headers"}>
                  <div class="resp-headers">
                    <For each={Object.entries(r().headers ?? {})}>
                      {([k, vals]) => (
                        <div class="hdr-row">
                          <span class="hdr-key">{k}</span>
                          <span class="hdr-val">{(vals ?? []).join(", ")}</span>
                        </div>
                      )}
                    </For>
                  </div>
                </Match>
                <Match when={tab() === "timeline"}>
                  <Timeline response={r()} />
                </Match>
                <Match when={tab() === "tests"}>
                  <div class="assert-results">
                    <For each={asserts()}>
                      {(a) => (
                        <div class="assert-row" classList={{ pass: a.pass, fail: !a.pass }}>
                          <span class="assert-mark">{a.pass ? <Check size={ICON.sm} /> : <X size={ICON.sm} />}</span>
                          <span class="assert-expr">
                            {a.target} {a.op}
                            {a.value ? ` ${a.value}` : ""}
                          </span>
                          <Show when={!a.pass}>
                            <span class="assert-detail">
                              {a.error ? a.error : `actual: ${a.actual ?? ""}`}
                            </span>
                          </Show>
                        </div>
                      )}
                    </For>
                  </div>
                </Match>
              </Switch>
            </div>
          </>
        )}
      </Show>
    </div>
  );
}
