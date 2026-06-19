// Verify session fixes: body pane follows radio; auth select readable.
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";

const OUT = new URL("./__screenshots__/", import.meta.url).pathname;
mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 820 } });
await page.goto("http://localhost:5173/", { waitUntil: "networkidle" });
await page.waitForTimeout(400);

// Body pane sync
await page.click('.tabs button:has-text("Body")');
await page.click('.body-types label:has-text("json") input');
await page.waitForTimeout(200);
const jsonPane = await page
  .locator(".body-editor")
  .evaluate((el) => (el.querySelector(".cm-editor") ? "code-editor" : "WRONG"));
await page.screenshot({ path: OUT + "fix-body-json.png" });

// Auth select contrast
await page.click('.tabs button:has-text("Auth")');
await page.waitForTimeout(200);
const sel = page.locator(".auth-field select").first();
const colors = await sel.evaluate((el) => {
  const cs = getComputedStyle(el);
  return { color: cs.color, background: cs.backgroundColor };
});
await page.screenshot({ path: OUT + "fix-auth-select.png" });

console.log(JSON.stringify({ jsonPane, authSelect: colors }, null, 2));
await browser.close();
