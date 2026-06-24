import { describe, expect, it } from "vitest";
import { tabCycleDir } from "./keymap";

const key = (init: KeyboardEventInit) => new KeyboardEvent("keydown", init);

describe("tabCycleDir", () => {
  it("Ctrl+Tab cycles forward", () => {
    expect(tabCycleDir(key({ code: "Tab", ctrlKey: true }))).toBe(1);
  });

  it("Ctrl+Shift+Tab cycles backward", () => {
    expect(tabCycleDir(key({ code: "Tab", ctrlKey: true, shiftKey: true }))).toBe(-1);
  });

  it("Cmd+Tab cycles forward (mac)", () => {
    expect(tabCycleDir(key({ code: "Tab", metaKey: true }))).toBe(1);
  });

  // The regression this guards: on Linux/GTK, Shift+Tab emits ISO_Left_Tab so
  // e.key is NOT "Tab" while e.code stays "Tab". A handler keyed off e.key
  // would return 0 here and Ctrl+Shift+Tab would do nothing.
  it("works when e.key is not 'Tab' (GTK ISO_Left_Tab) as long as e.code is", () => {
    expect(
      tabCycleDir(key({ key: "ISO_Left_Tab", code: "Tab", ctrlKey: true, shiftKey: true })),
    ).toBe(-1);
  });

  it("ignores Tab without a modifier", () => {
    expect(tabCycleDir(key({ code: "Tab" }))).toBe(0);
  });

  it("ignores other keys with a modifier", () => {
    expect(tabCycleDir(key({ code: "KeyT", ctrlKey: true }))).toBe(0);
  });
});
