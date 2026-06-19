// Request timing breakdown (DNS / connect / TLS / server wait / download)
// with proportional bars, shown in the response viewer's Timeline tab.
import { createMemo, For, Show } from "solid-js";
import type { Response } from "../lib/api";

type Phase = { label: string; ms: number; cls: string };

export default function Timeline(props: { response: Response }) {
  const t = () => props.response.timing!;

  const phases = createMemo<Phase[]>(() => {
    const tm = t();
    // Server wait = first byte minus the connection setup phases.
    const setup = tm.dnsMs + tm.connectMs + tm.tlsMs;
    const wait = Math.max(0, tm.firstByteMs - setup);
    return [
      { label: "DNS", ms: tm.dnsMs, cls: "tl-dns" },
      { label: "Connect", ms: tm.connectMs, cls: "tl-connect" },
      { label: "TLS", ms: tm.tlsMs, cls: "tl-tls" },
      { label: "Server wait", ms: wait, cls: "tl-wait" },
      { label: "Download", ms: tm.downloadMs, cls: "tl-download" },
    ];
  });

  const max = () => Math.max(1, ...phases().map((p) => p.ms));

  return (
    <div class="timeline">
      <Show when={t().reused}>
        <div class="timeline-note">Connection reused — no DNS / connect / TLS cost.</div>
      </Show>
      <For each={phases()}>
        {(p) => (
          <div class="timeline-row">
            <span class="timeline-label">{p.label}</span>
            <span class="timeline-bar-track">
              <span
                class={`timeline-bar ${p.cls}`}
                style={{ width: `${Math.max(p.ms > 0 ? 2 : 0, (p.ms / max()) * 100)}%` }}
              />
            </span>
            <span class="timeline-ms">{p.ms} ms</span>
          </div>
        )}
      </For>
      <div class="timeline-row timeline-total">
        <span class="timeline-label">Total</span>
        <span class="timeline-bar-track" />
        <span class="timeline-ms">{props.response.durationMs} ms</span>
      </div>
    </div>
  );
}
