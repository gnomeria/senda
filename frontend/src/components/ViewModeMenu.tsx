// Small labelled dropdown (Bruno-style "JSON ▾"). Click toggles a popover list;
// selecting an option or clicking outside closes it.
import { createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { Check, ChevronDown } from "lucide-solid";
import { ICON } from "../lib/icons";

export type Option<T extends string> = { value: T; label: string };

export default function ViewModeMenu<T extends string>(props: {
  value: T;
  options: Option<T>[];
  onChange: (v: T) => void;
}) {
  const [open, setOpen] = createSignal(false);
  let root!: HTMLDivElement;

  const onDocDown = (e: MouseEvent) => {
    if (open() && root && !root.contains(e.target as HTMLElement)) setOpen(false);
  };
  onMount(() => document.addEventListener("mousedown", onDocDown));
  onCleanup(() => document.removeEventListener("mousedown", onDocDown));

  const current = () => props.options.find((o) => o.value === props.value);

  return (
    <div class="viewmode" ref={root}>
      <button class="viewmode-btn" onClick={() => setOpen(!open())}>
        {current()?.label ?? props.value}
        <span class="viewmode-chev"><ChevronDown size={ICON.md} /></span>
      </button>
      <Show when={open()}>
        <div class="viewmode-menu">
          <For each={props.options}>
            {(o) => (
              <button
                class="viewmode-item"
                classList={{ active: o.value === props.value }}
                onClick={() => {
                  props.onChange(o.value);
                  setOpen(false);
                }}
              >
                {o.label}
                <Show when={o.value === props.value}>
                  <span class="viewmode-check"><Check size={ICON.xs} /></span>
                </Show>
              </button>
            )}
          </For>
        </div>
      </Show>
    </div>
  );
}
