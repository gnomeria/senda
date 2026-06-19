import { chromium } from "playwright";
const big = JSON.stringify(
  Array.from({length:500},(_,i)=>({postId:Math.floor(i/5)+1,id:i+1,name:"n"+i,email:"e"+i+"@x.io",body:"text "+i})),
  null, 2
);
const mock = () => {
  const tree={name:"d",path:"/d",isDir:true,children:[{name:"comments",path:"/d/comments.yaml",isDir:false}]};
  const req={name:"comments",method:"GET",url:"https://x/comments",params:[],headers:[],body:{type:"none"}};
  window.__BIG=window.__BIG;
  window.go={main:{App:{
    Ping:async()=>"ok",
    OpenCollection:async()=>({name:"d",path:"/d",vars:[],tree}),
    ListEnvironments:async()=>[],
    ReadRequest:async()=>req, SaveRequest:async()=>{}, DeleteRequest:async()=>{}, CreateFolder:async()=>{}, SaveEnvironment:async()=>{},
    SendRequest:async()=>({status:200,statusText:"OK",durationMs:29,sizeBytes:154000,headers:{"Content-Type":["application/json"]},body:window.__BIGBODY,truncated:false}),
  }}};
  window.runtime={};
};
const browser=await chromium.launch();
const page=await browser.newPage({viewport:{width:1280,height:820}});
const errs=[]; page.on("console",m=>{if(m.type()==="error")errs.push(m.text());});
page.on("pageerror",e=>errs.push("PAGEERROR: "+e.message));
await page.addInitScript(mock);
await page.addInitScript((b)=>{window.__BIGBODY=b;}, big);
await page.goto("http://localhost:5173/",{waitUntil:"networkidle"});
await page.evaluate(()=>localStorage.setItem("senda.lastCollection","/d"));
await page.reload({waitUntil:"networkidle"});
await page.waitForTimeout(200);
await page.getByText("comments",{exact:true}).click();
await page.waitForTimeout(150);
await page.getByRole("button",{name:"Send"}).click();
await page.waitForTimeout(300);
const rowsCollapsed = await page.locator(".json-tree .jt-row").count();
const rootSummary = await page.locator(".json-tree .jt-count").first().innerText().catch(()=>"<none>");
const modeLabel = await page.locator(".viewmode-btn").innerText().catch(()=>"<none>");
// expand first bucket
await page.locator(".json-tree .jt-clickable").nth(1).click();
await page.waitForTimeout(150);
const rowsAfterExpand = await page.locator(".json-tree .jt-row").count();
// switch to Raw
await page.locator(".viewmode-btn").click();
await page.waitForTimeout(80);
await page.getByText("Raw",{exact:true}).click();
await page.waitForTimeout(150);
const hasCM = await page.locator(".cm-editor").count();
// switch to Hex
await page.locator(".viewmode-btn").click(); await page.waitForTimeout(60);
await page.getByText("Hex",{exact:true}).click(); await page.waitForTimeout(150);
const hexFirst = await page.locator(".cm-line").first().innerText().catch(()=>"<none>");
console.log(JSON.stringify({rowsCollapsed,rootSummary,modeLabel,rowsAfterExpand,hasCM,hexFirst:hexFirst.slice(0,40),errs},null,2));
await browser.close();
