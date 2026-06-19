// Live smoke-check for example-collection/public-apis.postman_collection.json:
// walks the collection and fires every request, printing per-request status.
import { readFileSync } from "node:fs";

const doc = JSON.parse(
  readFileSync(new URL("../example-collection/public-apis.postman_collection.json", import.meta.url), "utf8")
);

function* walk(items, dir = "") {
  for (const it of items) {
    if (it.item) yield* walk(it.item, dir + it.name + "/");
    else if (it.request) yield { name: dir + it.name, request: it.request };
  }
}

let pass = 0, fail = 0;
for (const { name, request } of walk(doc.item)) {
  const r = typeof request === "string" ? { method: "GET", url: request } : request;
  const url = typeof r.url === "string" ? r.url : r.url.raw;
  const headers = {};
  for (const h of r.header ?? []) headers[h.key] = h.value;
  let body;
  if (r.body?.mode === "raw") {
    body = r.body.raw;
  } else if (r.body?.mode === "urlencoded") {
    body = new URLSearchParams(r.body.urlencoded.map((kv) => [kv.key, kv.value]));
  } else if (r.body?.mode === "graphql") {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify({
      query: r.body.graphql.query,
      ...(r.body.graphql.variables ? { variables: JSON.parse(r.body.graphql.variables) } : {}),
    });
  }
  if (r.auth?.type === "basic") {
    const u = r.auth.basic.find((x) => x.key === "username").value;
    const p = r.auth.basic.find((x) => x.key === "password").value;
    headers["Authorization"] = "Basic " + btoa(`${u}:${p}`);
  }
  try {
    const res = await fetch(url, { method: r.method ?? "GET", headers, body });
    const ok = res.status >= 200 && res.status < 400;
    ok ? pass++ : fail++;
    console.log(`${ok ? "✓" : "✗"} ${res.status} ${name}`);
  } catch (e) {
    fail++;
    console.log(`✗ ERR ${name}: ${e}`);
  }
}
console.log(`\n${pass} ok, ${fail} failed`);
process.exit(fail ? 1 : 0);
