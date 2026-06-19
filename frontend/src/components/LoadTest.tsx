// Load Test view. Pure UI over the load-test runtime in store.ts — the test
// itself runs in the background there, so this panel can be unmounted (run
// modal closed) and remounted without interrupting an in-flight test.
import { createMemo, createSignal, For, Show } from "solid-js";
import type { LoadTick } from "../lib/api";
import {
  loadConfig,
  setLoadConfig,
  loadHistory,
  loadSummary,
  loadError,
  loadTestRunning,
  startLoadTest,
  stopLoadTest,
} from "../lib/store";
import { statusClass } from "../lib/factory";

export default function LoadTest(props: { folderPath: string }) {
  const [hoverTick, setHoverTick] = createSignal<LoadTick | null>(null);
  const [hoverX, setHoverX] = createSignal(0);

  const running = loadTestRunning;
  const history = loadHistory;
  const error = loadError;

  const vus = () => loadConfig().vus;
  const mode = () => loadConfig().mode;
  const duration = () => loadConfig().duration;
  const iterations = () => loadConfig().iterations;
  const rampUp = () => loadConfig().rampUp;
  const patch = (p: Partial<ReturnType<typeof loadConfig>>) =>
    setLoadConfig((c) => ({ ...c, ...p }));

  const live = () => {
    const h = history();
    return h.length > 0 ? h[h.length - 1] : null;
  };
  const done = () => loadSummary();

  const fmt1 = (n: number) => n.toFixed(1);
  const fmtMs = (n: number) => `${n.toFixed(0)}ms`;


  // SVG chart from tick history
  const chart = createMemo(() => {
    const h = history();
    if (h.length < 2) return null;
    const W = 600;
    const H = 80;
    const maxRPS = Math.max(...h.map((t) => t.rps)) * 1.1 || 1;
    const maxP95 = Math.max(...h.map((t) => t.p95)) * 1.1 || 1;
    const maxElapsed = h[h.length - 1].elapsed || 1;

    const px = (t: LoadTick) => (t.elapsed / maxElapsed) * W;
    const pyRPS = (t: LoadTick) => H - (t.rps / maxRPS) * H;
    const pyP95 = (t: LoadTick) => H - (t.p95 / maxP95) * H;

    const rpsPoints = h.map((t) => `${px(t).toFixed(1)},${pyRPS(t).toFixed(1)}`).join(" ");
    const p95Points = h.map((t) => `${px(t).toFixed(1)},${pyP95(t).toFixed(1)}`).join(" ");

    return { rpsPoints, p95Points, W, H, maxRPS, maxP95 };
  });

  return (
    <div class="load-panel">
      <div class="load-config">
        <label class="load-field">
          <span class="load-label">VUs</span>
          <input
            class="load-input"
            type="number"
            min="1"
            max="500"
            value={vus()}
            onInput={(e) => patch({ vus: Math.max(1, +e.currentTarget.value) })}
            disabled={running()}
          />
        </label>
        <label class="load-field">
          <span class="load-label">Mode</span>
          <select
            class="load-select"
            value={mode()}
            onChange={(e) => patch({ mode: e.currentTarget.value as "duration" | "iterations" })}
            disabled={running()}
          >
            <option value="duration">Duration (s)</option>
            <option value="iterations">Iterations</option>
          </select>
        </label>
        <Show when={mode() === "duration"}>
          <label class="load-field">
            <span class="load-label">Seconds</span>
            <input
              class="load-input"
              type="number"
              min="1"
              value={duration()}
              onInput={(e) => patch({ duration: Math.max(1, +e.currentTarget.value) })}
              disabled={running()}
            />
          </label>
        </Show>
        <Show when={mode() === "iterations"}>
          <label class="load-field">
            <span class="load-label">Iterations</span>
            <input
              class="load-input"
              type="number"
              min="1"
              value={iterations()}
              onInput={(e) => patch({ iterations: Math.max(1, +e.currentTarget.value) })}
              disabled={running()}
            />
          </label>
        </Show>
        <label class="load-field">
          <span class="load-label" title="Seconds to ramp VUs from 0 to full count. 0 = instant.">Ramp-up (s)</span>
          <input
            class="load-input"
            type="number"
            min="0"
            value={rampUp()}
            onInput={(e) => patch({ rampUp: Math.max(0, +e.currentTarget.value) })}
            disabled={running()}
          />
        </label>
        <button class="btn load-start-btn" onClick={() => running() ? stopLoadTest() : startLoadTest(props.folderPath)}>
          {running() ? "Stop" : "Start"}
        </button>
      </div>

      <Show when={error()}>
        <div class="modal-error">{error()}</div>
      </Show>

      <Show when={live() || done()}>
        <div class="load-stats">
          <div class="load-stat-head">
            <span>Elapsed</span>
            <span>Requests</span>
            <span>RPS</span>
            <span>P50</span>
            <span>P95</span>
            <span>P99</span>
            <span>Errors</span>
            <span>Status</span>
          </div>
          <Show when={running() && live()}>
            <div class="load-stat-row">
              <span>{fmt1(live()!.elapsed)}s</span>
              <span>{live()!.total}</span>
              <span>{fmt1(live()!.rps)}</span>
              <span>{fmtMs(live()!.p50)}</span>
              <span>{fmtMs(live()!.p95)}</span>
              <span>{fmtMs(live()!.p99)}</span>
              <span class={live()!.errors > 0 ? "load-err" : ""}>{live()!.errors}</span>
              <div class="load-stat-dist">
                <For each={Object.entries(live()!.statusDist).sort(([a], [b]) => +a - +b)}>
                  {([status, count]) => (
                    <span class={`load-dist-badge status-badge ${statusClass(+status)}`}>
                      {status}×{count}
                    </span>
                  )}
                </For>
              </div>
            </div>
          </Show>
          <Show when={!running() && (done() || live())}>
            {(() => {
              const d = done();
              const t = live()!;
              const elapsed = d ? d.duration : t.elapsed;
              const total   = d ? d.total    : t.total;
              const rps     = d ? d.rps      : t.rps;
              const p50     = d ? d.p50      : t.p50;
              const p95     = d ? d.p95      : t.p95;
              const p99     = d ? d.p99      : t.p99;
              const errors  = d ? d.errors   : t.errors;
              const dist    = d ? d.statusDist : t.statusDist;
              return (
                <div class="load-stat-row load-stat-final">
                  <span>{fmt1(elapsed)}s</span>
                  <span>{total}</span>
                  <span>{fmt1(rps)}</span>
                  <span>{fmtMs(p50)}</span>
                  <span>{fmtMs(p95)}</span>
                  <span>{fmtMs(p99)}</span>
                  <span class={errors > 0 ? "load-err" : ""}>{errors}</span>
                  <div class="load-stat-dist">
                    <For each={Object.entries(dist).sort(([a], [b]) => +a - +b)}>
                      {([status, count]) => (
                        <span class={`load-dist-badge status-badge ${statusClass(+status)}`}>
                          {status}×{count}
                        </span>
                      )}
                    </For>
                  </div>
                </div>
              );
            })()}
          </Show>
        </div>

        {/* Over-time chart with hover tooltips */}
        <Show when={chart()}>
          <div class="load-chart-wrap">
            <div style="position:relative">
              <svg
                class="load-chart"
                viewBox={`0 0 ${chart()!.W} ${chart()!.H}`}
                preserveAspectRatio="none"
                onMouseMove={(e) => {
                  const rect = e.currentTarget.getBoundingClientRect();
                  const xFrac = (e.clientX - rect.left) / rect.width;
                  const h = history();
                  if (!h.length) return;
                  const idx = Math.min(h.length - 1, Math.round(xFrac * (h.length - 1)));
                  setHoverTick(h[idx]);
                  setHoverX(e.clientX - rect.left);
                }}
                onMouseLeave={() => setHoverTick(null)}
              >
                <polyline
                  class="load-chart-rps"
                  points={chart()!.rpsPoints}
                  fill="none"
                  stroke="var(--ok)"
                  stroke-width="1.5"
                />
                <polyline
                  class="load-chart-p95"
                  points={chart()!.p95Points}
                  fill="none"
                  stroke="var(--redirect)"
                  stroke-width="1.5"
                />
              </svg>
              <Show when={hoverTick()}>
                <div
                  class="load-chart-tooltip"
                  style={`left:${hoverX()}px`}
                >
                  <div>{fmt1(hoverTick()!.elapsed)}s</div>
                  <div>RPS: {fmt1(hoverTick()!.rps)}</div>
                  <div>P50: {fmtMs(hoverTick()!.p50)}</div>
                  <div>P95: {fmtMs(hoverTick()!.p95)}</div>
                  <div>Err: {hoverTick()!.errors}</div>
                </div>
              </Show>
            </div>
            <div class="load-chart-legend">
              <span class="load-legend-rps">● RPS (max {fmt1(chart()!.maxRPS)})</span>
              <span class="load-legend-p95">● P95 (max {fmtMs(chart()!.maxP95)})</span>
            </div>
          </div>
        </Show>
      </Show>

      <Show when={!running() && history().length === 0 && !done()}>
        <div class="empty-hint">Configure and press Start to begin the load test.</div>
      </Show>
    </div>
  );
}
