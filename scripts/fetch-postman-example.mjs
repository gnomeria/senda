// One-off: vendor postman-collection's example collection as importer testdata.
import { mkdirSync } from "node:fs";

const url =
  "https://raw.githubusercontent.com/postmanlabs/postman-collection/develop/examples/collection-v2.json";
const dest = new URL("../internal/importer/testdata/collection-v2.json", import.meta.url).pathname;
mkdirSync(new URL("../internal/importer/testdata/", import.meta.url).pathname, { recursive: true });
const res = await fetch(url);
if (!res.ok) throw new Error(`fetch failed: ${res.status}`);
await Bun.write(dest, await res.text());
console.log("wrote", dest);
