import { describe, expect, it } from "vitest";
import { blankRequest, formatBytes, statusClass } from "./factory";

describe("factory", () => {
  it("blankRequest has GET method and empty collections", () => {
    const r = blankRequest("x");
    expect(r.method).toBe("GET");
    expect(r.name).toBe("x");
    expect(r.params).toEqual([]);
    expect(r.body.type).toBe("none");
  });

  it("formatBytes scales units", () => {
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(2048)).toBe("2.0 KB");
    expect(formatBytes(5 * 1024 * 1024)).toBe("5.00 MB");
  });

  it("statusClass buckets status codes", () => {
    expect(statusClass(200)).toBe("ok");
    expect(statusClass(301)).toBe("redirect");
    expect(statusClass(404)).toBe("err");
    expect(statusClass(0)).toBe("");
  });
});
