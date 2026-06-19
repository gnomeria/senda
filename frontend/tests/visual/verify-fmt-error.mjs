// Verify Format button surfaces invalid-JSON error.
import { chromium } from "playwright";

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 820 } });
await page.goto("http://localhost:5173/", { waitUntil: "networkidle" });
await page.waitForTimeout(400);

await page.click('.tabs button:has-text("Body")');
await page.click('.body-types label:has-text("json") input');
await page.waitForTimeout(150);
await page.locator(".body-editor .cm-content").click();
await page.keyboard.type("{not valid json");
await page.click('.body-toolbar button:has-text("Format")');
await page.waitForTimeout(150);
const shown = await page.locator(".fmt-error").count();
// typing again clears nothing automatically; clicking radio resets
console.log(JSON.stringify({ invalidJsonErrorShown: shown === 1 }));
await browser.close();
