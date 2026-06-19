// Verify keyboard shortcuts + command palette (defensive: counts, no waits).
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";

const OUT = new URL("./__screenshots__/", import.meta.url).pathname;
mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 820 } });
page.on("console", (m) => m.type() === "error" && console.log("PAGE-ERR:", m.text()));
page.on("pageerror", (e) => console.log("PAGE-EXC:", String(e)));
await page.goto("http://localhost:5173/", { waitUntil: "networkidle" });
await page.waitForTimeout(500);

const tabCount = () => page.locator(".tabbar .tab").count();
const r = {};

r.tabsInitial = await tabCount();
await page.keyboard.press("Control+t");
await page.waitForTimeout(120);
r.tabsAfterCtrlT = await tabCount();
await page.keyboard.press("Control+w");
await page.waitForTimeout(120);
r.tabsAfterCtrlW = await tabCount();

await page.keyboard.press("Control+k");
await page.waitForTimeout(200);
r.paletteOpen = await page.locator(".palette").count();
r.itemsNoQuery = await page.locator(".palette-item").count();
await page.keyboard.type("new req");
await page.waitForTimeout(150);
r.itemsFiltered = await page.locator(".palette-item").count();
if (r.itemsFiltered > 0) {
  r.firstItem = await page.locator(".palette-item").first().textContent();
  await page.keyboard.press("Enter");
  await page.waitForTimeout(150);
  r.paletteClosedAfterEnter = (await page.locator(".palette").count()) === 0;
  r.tabsAfterPaletteNew = await tabCount();
}
await page.screenshot({ path: OUT + "palette.png" });
console.log(JSON.stringify(r, null, 2));
await browser.close();
