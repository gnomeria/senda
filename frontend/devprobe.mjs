import { chromium } from "playwright";
const browser=await chromium.launch();
const page=await browser.newPage({viewport:{width:1280,height:820}});
const errs=[],info=[];
page.on("console",m=>{ if(m.type()==="error")errs.push(m.text()); if(m.type()==="info")info.push(m.text()); });
page.on("pageerror",e=>errs.push("PAGEERROR: "+e.message));
// NO addInitScript mock — rely on devMock auto-install
await page.goto("http://localhost:5173/",{waitUntil:"networkidle"});
await page.waitForTimeout(500);
const treeVisible = await page.getByText("comments",{exact:true}).count();
await page.getByText("comments",{exact:true}).click();
await page.waitForTimeout(150);
await page.getByRole("button",{name:"Send"}).click();
await page.waitForTimeout(400);
const rootSummary = await page.locator(".json-tree .jt-count").first().innerText().catch(()=>"<none>");
console.log(JSON.stringify({treeVisible, rootSummary, info, errs},null,2));
await browser.close();
