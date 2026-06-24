import { beforeEach, describe, expect, it, vi } from "vitest";

// Mock the Go binding layer. saveActive's save-as path calls saveRequest then
// refreshCollection (openCollection + listEnvironments + collectionActivity).
vi.mock("./api", () => ({
  api: {
    saveRequest: vi.fn(() => Promise.resolve()),
    openCollection: vi.fn(() => Promise.resolve({ name: "c", path: "/c" })),
    listEnvironments: vi.fn(() => Promise.resolve([])),
    collectionActivity: vi.fn(() => Promise.resolve({})),
  },
}));

import { api } from "./api";
import { saveActive } from "./actions";
import { activePath, dirty, newTab, setCollection, setDirty } from "./store";

beforeEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
});

describe("saveActive", () => {
  it("save-as: a scratch tab with no path prompts for a name and writes into the collection root", async () => {
    setCollection({ name: "c", path: "/c" } as any);
    newTab(); // fresh scratch tab, path === ""
    setDirty(true);
    vi.stubGlobal("prompt", () => "my-req");

    await saveActive();

    expect(api.saveRequest).toHaveBeenCalledTimes(1);
    expect(vi.mocked(api.saveRequest).mock.calls[0][0]).toBe("/c/my-req.yaml");
    // tab is now backed by the new file and clean
    expect(activePath()).toBe("/c/my-req.yaml");
    expect(dirty()).toBe(false);

    vi.unstubAllGlobals();
  });

  it("save-as: cancelling the prompt writes nothing", async () => {
    setCollection({ name: "c", path: "/c" } as any);
    newTab();
    setDirty(true);
    vi.stubGlobal("prompt", () => null);

    await saveActive();

    expect(api.saveRequest).not.toHaveBeenCalled();
    vi.unstubAllGlobals();
  });

  it("no collection open: save-as does nothing", async () => {
    setCollection(null);
    newTab();
    setDirty(true);
    vi.stubGlobal("prompt", () => "x");

    await saveActive();

    expect(api.saveRequest).not.toHaveBeenCalled();
    vi.unstubAllGlobals();
  });
});
