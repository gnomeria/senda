import { beforeEach, describe, expect, it } from "vitest";
import { blankRequest } from "./factory";
import {
  activePath,
  activeTabId,
  closeAllTabs,
  closeOtherTabs,
  closeSavedTabs,
  closeTab,
  closeTabsToLeft,
  closeTabsToRight,
  cloneTab,
  cycleTab,
  dirty,
  markActiveSaved,
  newTab,
  openInTab,
  request,
  revertPath,
  revertTabTo,
  savedTabs,
  setDirty,
  setRequest,
  switchTab,
  tabs,
} from "./store";

// Tab state is module-level (one live editor per window); reset between tests.
beforeEach(() => {
  localStorage.clear();
  closeAllTabs();
  setDirty(false);
});

function open(name: string, path: string) {
  openInTab(blankRequest(name), path);
}

function activeTab() {
  return tabs.find((t) => t.id === activeTabId())!;
}

describe("openInTab", () => {
  it("reuses a clean scratch tab instead of leaving it behind", () => {
    expect(tabs.length).toBe(1);
    open("users", "api/users.yaml");
    expect(tabs.length).toBe(1);
    expect(activePath()).toBe("api/users.yaml");
    expect(activeTab().title).toBe("users");
  });

  it("keeps a dirty scratch tab and opens alongside it", () => {
    setRequest("url", "http://draft");
    setDirty(true);
    open("users", "api/users.yaml");
    expect(tabs.length).toBe(2);
    expect(activePath()).toBe("api/users.yaml");
  });

  it("focuses the existing tab when the same path is opened again", () => {
    open("users", "api/users.yaml");
    open("orders", "api/orders.yaml");
    expect(tabs.length).toBe(2);
    open("users again", "api/users.yaml");
    expect(tabs.length).toBe(2);
    expect(activePath()).toBe("api/users.yaml");
  });

  it("derives the tab title from the file basename, stripping .yaml/.yml", () => {
    open("ignored", "deep/nested/list-users.yaml");
    expect(activeTab().title).toBe("list-users");
    open("ignored", "other.yml");
    expect(tabs.find((t) => t.path === "other.yml")!.title).toBe("other");
  });
});

describe("switchTab", () => {
  it("snapshots live edits and restores them when switching back", () => {
    open("a", "a.yaml");
    const aId = activeTabId();
    setRequest("url", "http://edited-a");
    setDirty(true);

    open("b", "b.yaml");
    expect(request.url).toBe(""); // b's fresh model is live
    expect(dirty()).toBe(false);

    switchTab(aId);
    expect(request.url).toBe("http://edited-a");
    expect(dirty()).toBe(true);
  });
});

describe("closeTab", () => {
  it("activates the right-hand neighbour (or last tab) on close", () => {
    open("a", "a.yaml");
    open("b", "b.yaml");
    open("c", "c.yaml");
    switchTab(tabs[1].id);
    closeTab(tabs[1].id);
    expect(activePath()).toBe("c.yaml");
    closeTab(activeTabId());
    expect(activePath()).toBe("a.yaml");
  });

  it("closing the last tab falls back to a fresh scratch tab", () => {
    open("a", "a.yaml");
    closeTab(activeTabId());
    expect(tabs.length).toBe(1);
    expect(activePath()).toBe("");
    expect(dirty()).toBe(false);
  });

  it("closing an inactive tab keeps the active one live", () => {
    open("a", "a.yaml");
    open("b", "b.yaml");
    closeTab(tabs[0].id);
    expect(activePath()).toBe("b.yaml");
    expect(tabs.length).toBe(1);
  });
});

describe("bulk close operations", () => {
  beforeEach(() => {
    open("a", "a.yaml");
    open("b", "b.yaml");
    open("c", "c.yaml");
  });

  it("closeOtherTabs keeps only the given tab and activates it", () => {
    closeOtherTabs(tabs[0].id);
    expect(tabs.length).toBe(1);
    expect(activePath()).toBe("a.yaml");
  });

  it("closeTabsToRight trims and re-activates when the active tab was cut", () => {
    closeTabsToRight(tabs[0].id); // active tab (c) is to the right
    expect(tabs.map((t) => t.path)).toEqual(["a.yaml"]);
    expect(activePath()).toBe("a.yaml");
  });

  it("closeTabsToLeft trims and keeps the active tab when it survives", () => {
    closeTabsToLeft(tabs[2].id);
    expect(tabs.map((t) => t.path)).toEqual(["c.yaml"]);
    expect(activePath()).toBe("c.yaml");
  });

  it("closeSavedTabs keeps dirty and unsaved tabs only", () => {
    switchTab(tabs[1].id);
    setDirty(true); // b is dirty
    newTab(); // unsaved scratch
    closeSavedTabs();
    expect(tabs.map((t) => t.path)).toEqual(["b.yaml", ""]);
  });

  it("closeSavedTabs falls back to a scratch tab when everything was saved", () => {
    closeSavedTabs();
    expect(tabs.length).toBe(1);
    expect(activePath()).toBe("");
  });
});

describe("cycleTab", () => {
  it("wraps in both directions", () => {
    open("a", "a.yaml");
    open("b", "b.yaml");
    open("c", "c.yaml"); // active
    cycleTab(1);
    expect(activePath()).toBe("a.yaml"); // wrapped right
    cycleTab(-1);
    expect(activePath()).toBe("c.yaml"); // wrapped left
  });

  it("is a no-op with a single tab", () => {
    const id = activeTabId();
    cycleTab(1);
    expect(activeTabId()).toBe(id);
  });
});

describe("cloneTab", () => {
  it("duplicates into an unsaved scratch tab with a ' copy' name", () => {
    open("users", "api/users.yaml");
    setRequest("url", "http://live-edit");
    cloneTab(activeTabId());
    expect(tabs.length).toBe(2);
    expect(activePath()).toBe(""); // clone is unsaved
    expect(request.name).toBe("users copy");
    expect(request.url).toBe("http://live-edit"); // clones the LIVE model
  });
});

describe("save / revert", () => {
  it("markActiveSaved clears dirty and records the new path + title", () => {
    setRequest("name", "draft");
    setDirty(true);
    markActiveSaved("api/draft.yaml");
    expect(dirty()).toBe(false);
    expect(activePath()).toBe("api/draft.yaml");
    expect(activeTab().title).toBe("draft");
  });

  it("revertPath returns the disk path for saved tabs and '' for scratch", () => {
    open("a", "a.yaml");
    expect(revertPath(activeTabId())).toBe("a.yaml");
    newTab();
    expect(revertPath(activeTabId())).toBe("");
  });

  it("revertTabTo restores the model and clears dirty on the active tab", () => {
    open("a", "a.yaml");
    setRequest("url", "http://edited");
    setDirty(true);
    const fresh = blankRequest("a");
    fresh.url = "http://from-disk";
    revertTabTo(activeTabId(), fresh);
    expect(request.url).toBe("http://from-disk");
    expect(dirty()).toBe(false);
  });
});

describe("tab persistence", () => {
  it("savedTabs round-trips open saved tabs and the active path", () => {
    open("a", "a.yaml");
    open("b", "b.yaml");
    newTab(); // scratch tabs are not persisted
    switchTab(tabs[0].id);
    expect(savedTabs()).toEqual({ paths: ["a.yaml", "b.yaml"], active: "a.yaml" });
  });

  it("savedTabs tolerates corrupt or missing localStorage", () => {
    localStorage.setItem("senda.openTabs", "{not json");
    expect(savedTabs()).toEqual({ paths: [], active: "" });
    localStorage.removeItem("senda.openTabs");
    expect(savedTabs()).toEqual({ paths: [], active: "" });
    localStorage.setItem("senda.openTabs", JSON.stringify({ paths: "nope", active: 7 }));
    expect(savedTabs()).toEqual({ paths: [], active: "" });
  });
});
