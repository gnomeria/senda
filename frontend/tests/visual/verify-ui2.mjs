// Verify: pane splitters drag + persist; JSON body Format button works.
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";

const OUT = new URL("./__screenshots__/", import.meta.url).pathname;
mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 820 } });
await page.goto("http://localhost:5173/", { waitUntil: "networkidle" });
await page.waitForTimeout(400);

// --- splitter drag: sidebar 250 -> ~380
const before = await page.locator(".sidebar").evaluate((el) => el.getBoundingClientRect().width);
const side = page.locator(".splitter").first();
const box = await side.boundingBox();
await page.mouse.move(box.x + 2, box.y + 300);
await page.mouse.down();
await page.mouse.move(380, box.y + 300, { steps: 5 });
await page.mouse.up();
const after = await page.locator(".sidebar").evaluate((el) => el.getBoundingClientRect().width);
const persisted = await page.evaluate(() => localStorage.getItem("senda.sideW"));

// --- format button
await page.click('.tabs button:has-text("Body")');
await page.click('.body-types label:has-text("json") input');
await page.waitForTimeout(150);
await page.locator(".body-editor .cm-content").click();
await page.keyboard.type('{"a":1,"b":{"c":[1,2]}}');
await page.click('.body-toolbar button:has-text("Format")');
await page.waitForTimeout(150);
const formatted = await page
  .locator(".body-editor .cm-content")
  .evaluate((el) => el.textContent);

// invalid JSON path
await page.keyboard.type("garbage");
await page.click('.body-toolbar button:has-text("Format")');
await page.waitForTimeout(100);
const fmtError = await page.locator(".fmt-error").count();

await page.screenshot({ path: OUT + "fix-splitter-format.png" });
console.log(
  JSON.stringify(
    {
      sidebarBefore: before,
      sidebarAfter: after,
      persisted,
      formattedHasNewlines: formatted.includes('"b"') && formatted.includes("  "),
      invalidJsonErrorShown: fmtError === 1,
    },
    null,
    2
  )
);
await browser.close();
