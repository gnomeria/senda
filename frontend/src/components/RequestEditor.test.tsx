import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@solidjs/testing-library";
import { BulkKVSection } from "./RequestEditor";
import type { KV } from "../lib/api";

// Regression guard for the frozen-handler bug: the bulk-edit toggle button used
// `onClick={bulk() ? exitBulk : enterBulk}`, which Solid binds once at render —
// so the handler stayed wired to enterBulk and "← Table" never committed edits.
// The fix wraps it in an arrow so the signal is read at click time. These tests
// exercise the toggle through a full enter → edit → exit cycle.
describe("BulkKVSection toggle", () => {
  const rows: KV[] = [{ key: "A", value: "1", enabled: true }];

  it("commits textarea edits back to rows when leaving bulk mode", () => {
    const onChange = vi.fn();
    const { container } = render(() => <BulkKVSection rows={rows} onChange={onChange} />);

    fireEvent.click(screen.getByRole("button", { name: "Bulk Edit" }));
    const area = container.querySelector("textarea") as HTMLTextAreaElement;
    expect(area.value).toBe("A=1");

    fireEvent.input(area, { target: { value: "A=1\nB=2" } });
    fireEvent.click(screen.getByRole("button", { name: "← Table" }));

    // The exit path (exitBulk) must run — not enterBulk again.
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith([
      { key: "A", value: "1", enabled: true },
      { key: "B", value: "2", enabled: true },
    ]);
    // And we are back in table mode (textarea gone, button toggled back).
    expect(container.querySelector("textarea")).toBeNull();
    expect(screen.getByRole("button", { name: "Bulk Edit" })).toBeInTheDocument();
  });

  it("toggles label each click instead of sticking on one handler", () => {
    render(() => <BulkKVSection rows={rows} onChange={() => {}} />);
    const btn = () => screen.getByRole("button", { name: /Bulk Edit|← Table/ });
    expect(btn()).toHaveTextContent("Bulk Edit");
    fireEvent.click(btn());
    expect(btn()).toHaveTextContent("← Table");
    fireEvent.click(btn());
    expect(btn()).toHaveTextContent("Bulk Edit");
  });
});
