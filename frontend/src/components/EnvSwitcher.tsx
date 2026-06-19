// Active-environment selector in the top bar. Persists the choice; the gear
// opens the environment editor.
import { createEffect, createSignal, For, Show } from "solid-js";
import { Settings } from "lucide-solid";
import { ICON } from "../lib/icons";
import { activeEnv, collection, environments, rememberEnv, setActiveEnv } from "../lib/store";
import EnvEditor from "./EnvEditor";

export default function EnvSwitcher() {
  let el!: HTMLSelectElement;
  const [showEditor, setShowEditor] = createSignal(false);

  // Re-apply the selected value whenever the option list or active env
  // changes — a <select>'s value won't stick if set before its <option>s
  // exist, which happens because environments load asynchronously.
  createEffect(() => {
    environments();
    if (el) el.value = activeEnv();
  });

  return (
    <div class="env-switcher">
      <select
        ref={el}
        onChange={(e) => {
          setActiveEnv(e.currentTarget.value);
          rememberEnv(e.currentTarget.value);
        }}
      >
        <option value="">No environment</option>
        <For each={environments()}>
          {(env) => <option value={env.name}>{env.name}</option>}
        </For>
      </select>
      <button
        class="icon-btn"
        title="Edit environments"
        disabled={!collection()}
        onClick={() => setShowEditor(true)}
      >
        <Settings size={ICON.xxl} />
      </button>
      <Show when={showEditor()}>
        <EnvEditor onClose={() => setShowEditor(false)} />
      </Show>
    </div>
  );
}
