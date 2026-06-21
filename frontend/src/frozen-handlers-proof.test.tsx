import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@solidjs/testing-library";
import { createSignal } from "solid-js";

// Empirical proof that the "frozen handler" bug is real in this exact Solid
// version — not a recollection. Two identical toggles, one buggy, one fixed.
// If Solid actually rebinds handlers reactively, the "buggy" case would pass
// too and this test would fail, telling us the lint rule is bogus.

function Toggle(props: { wrap: boolean; log: string[] }) {
  const [on, setOn] = createSignal(false);
  const a = () => { props.log.push("a"); setOn(true); };
  const b = () => { props.log.push("b"); setOn(false); };
  return (
    <button onClick={props.wrap ? () => (on() ? b() : a()) : (on() ? b : a)}>
      {on() ? "ON" : "OFF"}
    </button>
  );
}

describe("Solid event-handler binding semantics", () => {
  it("BARE ternary handler is frozen at render (proves the bug)", () => {
    const log: string[] = [];
    render(() => <Toggle wrap={false} log={log} />);
    const btn = screen.getByRole("button");
    fireEvent.click(btn); // a → on=true, label ON
    fireEvent.click(btn); // frozen: a AGAIN (not b)
    expect(log).toEqual(["a", "a"]);
    expect(btn).toHaveTextContent("ON"); // stuck — never reached b
  });

  it("ARROW-WRAPPED handler reads the signal per click (proves the fix)", () => {
    const log: string[] = [];
    render(() => <Toggle wrap={true} log={log} />);
    const btn = screen.getByRole("button");
    fireEvent.click(btn); // a → on=true
    fireEvent.click(btn); // b → on=false
    expect(log).toEqual(["a", "b"]);
    expect(btn).toHaveTextContent("OFF");
  });
});
