import { Show } from "solid-js";
import { Server, Zap } from "lucide-solid";
import { ICON } from "../lib/icons";
import { mockServerAddr, setShowMockPanel, loadTestRunning, runPanelTarget, setRunPanelTarget, setShowRunPanel } from "../lib/store";

export default function StatusBar() {
  const hasAnything = () => mockServerAddr() || loadTestRunning();

  return (
    <Show when={hasAnything()}>
      <div class="status-bar">
        <Show when={loadTestRunning()}>
          <button
            class="status-bar-chip status-bar-load"
            onClick={() => {
              setRunPanelTarget((t) => (t ? { ...t, initialTab: "load" } : t));
              setShowRunPanel(true);
            }}
            title="Load test running — click to open"
          >
            <Zap size={ICON.xs} />
            <span>Load test{runPanelTarget() ? `: ${runPanelTarget()!.folderName}` : ""} running…</span>
          </button>
        </Show>
        <Show when={mockServerAddr()}>
          <button
            class="status-bar-chip status-bar-mock"
            onClick={() => setShowMockPanel(true)}
            title="Mock server running — click to open"
          >
            <Server size={ICON.xs} />
            <span>Mock {mockServerAddr()}</span>
          </button>
        </Show>
      </div>
    </Show>
  );
}
