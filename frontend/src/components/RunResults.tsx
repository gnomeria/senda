// Folder-run modal: two tabs — sequential Run and concurrent Load Test.
// Run tab streams results live via "run:start"/"run:result" Wails events,
// filterable by outcome (all/passed/failed), re-runnable, and exportable as a
// JSON report; clicking a result opens its full response in a side pane.
// Load Test tab lets the user configure VUs/duration and start on demand.
import { createMemo, createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { Download, X } from "lucide-solid";
import { ICON } from "../lib/icons";
import { Events } from "@wailsio/runtime";
import { api } from "../lib/api";
import type { RunResult } from "../lib/api";
import { activeEnv, collection, refreshActivity } from "../lib/store";
import { statusClass } from "../lib/factory";
import LoadTest from "./LoadTest";
import RunDetail from "./RunDetail";

type Outcome = "all" | "passed" | "failed";

export default function RunResults(props: {
  folderPath: string;
  folderName: string;
  initialTab?: "run" | "load";
  onClose: () => void;
}) {
  const [tab, setTab] = createSignal<"run" | "load">(props.initialTab ?? "run");

  // ---- sequential run state ----
  const [results, setResults] = createSignal<RunResult[]>([]);
  const [total, setTotal] = createSignal(0);
  const [running, setRunning] = createSignal(false);
  const [error, setError] = createSignal("");
  const [filter, setFilter] = createSignal<Outcome>("all");
  const [startedAt, setStartedAt] = createSignal("");
  const [selected, setSelected] = createSignal<RunResult | null>(null);

  let call: ReturnType<typeof api.runFolder> | undefined;
  let offEvents: (() => void)[] = [];

  const passed = createMemo(() => results().filter((r) => r.ok).length);
  const failed = createMemo(() => results().length - passed());
  const visible = createMemo(() => {
    switch (filter()) {
      case "passed":
        return results().filter((r) => r.ok);
      case "failed":
        return results().filter((r) => !r.ok);
      default:
        return results();
    }
  });

  const startRun = async () => {
    if (running()) return;
    offEvents.forEach((f) => f());
    offEvents = [];
    setResults([]);
    setTotal(0);
    setError("");
    setFilter("all");
    setSelected(null);
    setStartedAt(new Date().toISOString());
    setRunning(true);
    offEvents.push(
      Events.On("run:start", (e: any) => setTotal(e.data?.total ?? 0)),
      Events.On("run:result", (e: any) =>
        setResults((prev) => [...prev, e.data as RunResult])
      ),
    );
    try {
      const coll = collection();
      call = api.runFolder(props.folderPath, coll?.path ?? "", activeEnv());
      const out = await call;
      // Use return value only as a fallback when no events arrived
      // (events are the primary stream; overwriting would cause duplicates
      // if late-queued Wails event callbacks fire after this line).
      if (out?.length && results().length === 0) setResults(out);
    } catch (e) {
      setError(String(e));
    } finally {
      offEvents.forEach((f) => f());
      offEvents = [];
      setRunning(false);
      void refreshActivity(collection()?.path ?? "");
    }
  };

  const reset = () => {
    setResults([]);
    setTotal(0);
    setError("");
    setFilter("all");
    setSelected(null);
  };

  const downloadReport = async () => {
    // Strip the heavy response bodies from the report; keep the summary rows.
    const rows = results().map(({ response, ...rest }) => rest);
    const report = {
      collection: collection()?.name ?? "",
      folder: props.folderName,
      environment: activeEnv(),
      startedAt: startedAt(),
      total: results().length,
      passed: passed(),
      failed: failed(),
      results: rows,
    };
    const stamp = startedAt().slice(0, 19).replaceAll(":", "-");
    try {
      await api.exportFile(
        `senda-run-${props.folderName}-${stamp}.json`,
        JSON.stringify(report, null, 2)
      );
    } catch (e) {
      setError(String(e));
    }
  };

  onMount(() => {
    if (tab() === "run") void startRun();
  });

  onCleanup(() => {
    call?.cancel();
    offEvents.forEach((f) => f());
  });

  const close = () => {
    if (running()) call?.cancel();
    props.onClose();
  };

  return (
    <div class="modal-backdrop" onClick={close}>
      <div class="modal modal-wide" classList={{ "modal-xwide": !!selected() }} onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="modal-title">
            {props.folderName}
            <Show when={tab() === "run"}>
              <Show when={running() && total() > 0}>
                <span class="run-summary"> — {results().length}/{total()}</span>
              </Show>
              <Show when={!running() && results().length > 0}>
                <span class="run-summary"> — {passed()}/{results().length} passed</span>
              </Show>
            </Show>
          </span>
          <button class="icon-btn" title="Close" onClick={close}>
            <X size={ICON.md} />
          </button>
        </div>

        <div class="run-tabs">
          <button
            class="run-tab"
            classList={{ active: tab() === "run" }}
            onClick={() => setTab("run")}
          >
            Run
          </button>
          <button
            class="run-tab"
            classList={{ active: tab() === "load" }}
            onClick={() => setTab("load")}
          >
            Load Test
          </button>
        </div>

        <div class="modal-body" classList={{ "run-body": tab() === "run" }}>
          <Show when={tab() === "run"}>
            <div class="run-toolbar">
              <div class="run-filters">
                <span class="run-filter-label">Filter:</span>
                <button
                  class="run-chip"
                  classList={{ active: filter() === "all" }}
                  onClick={() => setFilter("all")}
                >
                  All <span class="run-chip-count">{results().length}</span>
                </button>
                <button
                  class="run-chip"
                  classList={{ active: filter() === "passed" }}
                  onClick={() => setFilter("passed")}
                >
                  Passed <span class="run-chip-count ok">{passed()}</span>
                </button>
                <button
                  class="run-chip"
                  classList={{ active: filter() === "failed" }}
                  onClick={() => setFilter("failed")}
                >
                  Failed <span class="run-chip-count" classList={{ err: failed() > 0 }}>{failed()}</span>
                </button>
              </div>
              <div class="run-toolbar-actions">
                <button class="btn" disabled={running()} onClick={() => void startRun()}>
                  Run Again
                </button>
                <button class="btn" disabled={running() || results().length === 0} onClick={reset}>
                  Reset
                </button>
                <button
                  class="btn run-report-btn"
                  disabled={running() || results().length === 0}
                  onClick={() => void downloadReport()}
                >
                  <Download size={ICON.xs} /> Download Report
                </button>
              </div>
            </div>

            <Show when={error()}>
              <div class="modal-error">{error()}</div>
            </Show>
            <Show when={running() && results().length === 0}>
              <div class="empty-hint">
                Running{total() > 0 ? ` ${total()} requests` : ""}…
              </div>
            </Show>
            <div class="run-main" classList={{ split: !!selected() }}>
              <div class="run-list">
                <For each={visible()}>
                  {(r) => (
                    <div
                      class="run-row run-row-clickable"
                      classList={{ selected: selected() === r }}
                      onClick={() => setSelected(selected() === r ? null : r)}
                    >
                      <span class="run-dot" classList={{ ok: r.ok, fail: !r.ok }} />
                      <span class="run-name">{r.name || r.path}</span>
                      <span class="run-method">{r.method}</span>
                      <Show
                        when={!r.error}
                        fallback={<span class="run-err" title={r.error}>{r.error}</span>}
                      >
                        <span class={`status-badge ${statusClass(r.status)}`}>{r.status}</span>
                        <Show when={r.assertPass + r.assertFail > 0}>
                          <span
                            class="assert-badge"
                            classList={{ ok: r.assertFail === 0, err: r.assertFail > 0 }}
                            title="Asserts passed / total"
                          >
                            {r.assertPass}/{r.assertPass + r.assertFail}
                          </span>
                        </Show>
                        <span class="run-time">{r.durationMs}ms</span>
                      </Show>
                    </div>
                  )}
                </For>
                <Show when={running() && results().length > 0}>
                  <div class="run-row pending">
                    <span class="run-dot" />
                    <span class="run-name dim">sending…</span>
                  </div>
                </Show>
                <Show when={!running() && results().length > 0 && visible().length === 0}>
                  <div class="empty-hint">No {filter()} results.</div>
                </Show>
              </div>
              <Show when={selected()}>
                <RunDetail result={selected()!} onClose={() => setSelected(null)} />
              </Show>
            </div>
          </Show>

          <Show when={tab() === "load"}>
            <LoadTest folderPath={props.folderPath} />
          </Show>
        </div>

        <div class="modal-foot">
          <button class="btn" onClick={close}>
            {tab() === "run" && running() ? "Cancel" : "Close"}
          </button>
        </div>
      </div>
    </div>
  );
}
