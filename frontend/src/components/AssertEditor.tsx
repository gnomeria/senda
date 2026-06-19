// Editable list of declarative response asserts (the request's Tests tab).
// Each row: target expression, operator, expected value.
import { Index, Show } from "solid-js";
import { Plus, X } from "lucide-solid";
import { ICON } from "../lib/icons";
import type { Assert } from "../lib/api";
import { blankAssert } from "../lib/factory";

const OPS = [
  "eq",
  "neq",
  "contains",
  "notcontains",
  "gt",
  "gte",
  "lt",
  "lte",
  "exists",
  "notexists",
  "matches",
];

// Operators that take no expected value.
const UNARY = new Set(["exists", "notexists"]);

type Props = {
  rows: Assert[];
  onChange: (rows: Assert[]) => void;
};

export default function AssertEditor(props: Props) {
  const update = (i: number, patch: Partial<Assert>) => {
    const next = props.rows.map((r, idx) => (idx === i ? { ...r, ...patch } : r));
    props.onChange(next as Assert[]);
  };
  const remove = (i: number) =>
    props.onChange(props.rows.filter((_, idx) => idx !== i));
  const add = () => props.onChange([...props.rows, blankAssert()]);

  return (
    <div class="kv-editor assert-editor">
      <Index each={props.rows}>
        {(row, i) => (
          <div class="kv-row" classList={{ disabled: !row().enabled }}>
            <input
              type="checkbox"
              checked={row().enabled}
              onChange={(e) => update(i, { enabled: e.currentTarget.checked })}
              title="Enable / disable"
            />
            <input
              class="kv-key"
              placeholder="status · json.user.id · header.Content-Type"
              value={row().target}
              onInput={(e) => update(i, { target: e.currentTarget.value })}
            />
            <select
              class="assert-op"
              value={row().op}
              onChange={(e) => update(i, { op: e.currentTarget.value })}
            >
              {OPS.map((op) => (
                <option value={op}>{op}</option>
              ))}
            </select>
            <Show
              when={!UNARY.has(row().op)}
              fallback={<span class="kv-val assert-noval" />}
            >
              <input
                class="kv-val"
                placeholder="expected value"
                value={row().value ?? ""}
                onInput={(e) => update(i, { value: e.currentTarget.value })}
              />
            </Show>
            <button class="icon-btn" onClick={() => remove(i)} title="Remove">
              <X size={ICON.sm} />
            </button>
          </div>
        )}
      </Index>
      <button class="add-row" onClick={add}>
        <Plus size={ICON.xs} /> Add assert
      </button>
      <div class="empty-hint assert-hint">
        Targets: <code>status</code>, <code>duration</code>, <code>size</code>,{" "}
        <code>body</code>, <code>header.&lt;Name&gt;</code>,{" "}
        <code>json.&lt;path&gt;</code> (e.g. <code>json.items[0].id</code>)
      </div>
    </div>
  );
}
