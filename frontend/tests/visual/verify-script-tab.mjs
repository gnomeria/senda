// Verify Script tab: pre/post editors, typing marks dirty, badge dot.
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";

const OUT = new URL("./__screenshots__/", import.meta.url).pathname;
mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 820 } });
await page.goto("http://localhost:5173/", { waitUntil: "networkidle" });
await page.waitForTimeout(400);

const r = {};
await page.click('.tabs button:has-text("Script")');
await page.waitForTimeout(150);
r.preActive = await page.locator(".script-switch button.active").textContent();
await page.locator(".script-editor .cm-content").click();
await page.keyboard.type('senda.setVar("x", "1");');
await page.waitForTimeout(150);
r.scriptBadge = await page.locator('.tabs button:has-text("Script")').textContent();

await page.click('.script-switch button:has-text("Post-response")');
await page.waitForTimeout(150);
r.postEmpty = await page.locator(".script-editor .cm-content").textContent();
await page.locator(".script-editor .cm-content").click();
await page.keyboard.type('senda.setVar("token", res.json.token);');
await page.click('.script-switch button:has-text("Pre-request")');
await page.waitForTimeout(150);
r.preKept = await page.locator(".script-editor .cm-content").textContent();
await page.screenshot({ path: OUT + "script-tab.png" });
console.log(JSON.stringify(r, null, 2));
await browser.close();
