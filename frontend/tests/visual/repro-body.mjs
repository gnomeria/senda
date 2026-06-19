// Repro: fresh scratch tab -> Body tab -> click "json" radio.
// Expect CodeMirror editor; bug shows stale none/form pane instead.
import { chromium } from "playwright";

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 820 } });
await page.goto("http://localhost:5173/", { waitUntil: "networkidle" });
await page.waitForTimeout(400);

await page.click('.tabs button:has-text("Body")');
const states = {};
const dump = async (label) => {
  states[label] = {
    checked: await page
      .locator(".body-types input:checked")
      .evaluate((el) => el.parentElement.textContent.trim())
      .catch(() => "none-checked"),
    pane: await page.locator(".body-editor").evaluate((el) => {
      if (el.querySelector(".cm-editor")) return "code-editor";
      if (el.querySelector(".add-row")) return "kv-editor";
      if (el.querySelector(".empty-hint")) return "empty-hint";
      return "nothing";
    }),
  };
};

await dump("initial");
await page.click('.body-types label:has-text("json") input');
await page.waitForTimeout(200);
await dump("after-json-click");
await page.click('.body-types label:has-text("form") input');
await page.waitForTimeout(200);
await dump("after-form-click");
await page.click('.body-types label:has-text("json") input');
await page.waitForTimeout(200);
await dump("after-json-again");

console.log(JSON.stringify(states, null, 2));
await page.screenshot({ path: new URL("./__screenshots__/repro-body.png", import.meta.url).pathname });
await browser.close();
