// Factory helpers for fresh model values used across the UI.
//
// These return PLAIN object literals, never `model.X.createFrom(...)`: the
// generated Wails classes have non-Object prototypes, which Solid's store
// refuses to proxy — nested fields (body.type, auth.type, …) silently stop
// being reactive and the editor UI desyncs from the model.
import { AuthType, BodyType } from "../../bindings/senda/internal/model/models";
import type * as model from "../../bindings/senda/internal/model/models";

export function blankKV(): model.KV {
  return { key: "", value: "", enabled: true };
}

export function blankAuth(): model.Auth {
  return { type: AuthType.AuthInherit };
}

export function blankAssert(): model.Assert {
  return { target: "status", op: "eq", value: "", enabled: true };
}

export function blankRequest(name = "New request"): model.Request {
  return {
    name,
    method: "GET",
    url: "",
    params: [],
    headers: [],
    body: { type: BodyType.BodyNone },
    auth: blankAuth(),
    asserts: [],
    preScript: "",
    postScript: "",
    docs: "",
  };
}

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(2)} MB`;
}

export function statusClass(status: number): string {
  if (status >= 200 && status < 300) return "ok";
  if (status >= 300 && status < 400) return "redirect";
  if (status >= 400) return "err";
  return "";
}
