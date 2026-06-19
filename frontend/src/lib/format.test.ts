import { describe, expect, it } from "vitest";
import { toBase64, toHex } from "./format";

describe("format", () => {
  it("toBase64 matches btoa for ascii", () => {
    expect(toBase64("hello")).toBe("aGVsbG8=");
  });

  it("toBase64 is unicode-safe", () => {
    // round-trips through UTF-8, unlike a bare btoa(str)
    expect(toBase64("héllo→")).toBe(
      btoa(String.fromCharCode(...new TextEncoder().encode("héllo→")))
    );
  });

  it("toHex formats offset + bytes + ascii", () => {
    const out = toHex("ABC");
    // single row: offset, hex 41 42 43, ascii ABC
    expect(out.startsWith("00000000  41 42 43")).toBe(true);
    expect(out.endsWith("  ABC")).toBe(true);
  });

  it("toHex wraps at 16 bytes per row", () => {
    const out = toHex("0123456789abcdefXY"); // 18 bytes -> 2 rows
    const rows = out.split("\n");
    expect(rows.length).toBe(2);
    expect(rows[1].startsWith("00000010")).toBe(true);
  });
});
