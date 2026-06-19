import { chromium } from "playwright";
const mock = () => {
  const tree = { name:"demo",path:"/demo",isDir:true,children:[
    { name:"create-user",path:"/demo/create-user.yaml",isDir:false },
    { name:"list-users",path:"/demo/list-users.yaml",isDir:false },
  ]};
  const reqFor = (p) => ({ name:p.includes("create")?"Create user":"List users",
    method:p.includes("create")?"POST":"GET",
    url:p.includes("create")?"https://x/create":"https://x/list",
    params:[],headers:[],body:{type:"none"} });
  window.go={main:{App:{
    Ping:async()=>"ok",
    OpenCollection:async()=>({name:"demo",path:"/demo",vars:[],tree}),
    ListEnvironments:async()=>[],
    ReadRequest:async(p)=>reqFor(p),
    SaveRequest:async()=>{},DeleteRequest:async()=>{},CreateFolder:async()=>{},
    SaveEnvironment:async()=>{},
    SendRequest:async()=>({status:200,statusText:"OK",durationMs:1,sizeBytes:2,headers:{},body:"{}",truncated:false}),
  }}};
  window.runtime={};
};
const browser = await chromium.launch();
const page = await browser.newPage({ viewport:{width:1280,height:820} });
const errs=[]; page.on("console", m=>{ if(m.type()==="error") errs.push(m.text()); });
page.on("pageerror", e=>errs.push("PAGEERROR: "+e.message));
await page.addInitScript(mock);
await page.goto("http://localhost:5173/",{waitUntil:"networkidle"});
await page.evaluate(()=>{ localStorage.setItem("senda.lastCollection","/demo"); });
await page.reload({waitUntil:"networkidle"});
await page.waitForTimeout(300);
await page.getByText("create-user",{exact:true}).click();
await page.waitForTimeout(150);
await page.getByText("list-users",{exact:true}).click();
await page.waitForTimeout(150);
const urlAfterOpen = await page.inputValue(".url-input");
const tabCount = await page.locator(".tabbar .tab").count();
// click first tab
await page.locator(".tabbar .tab").first().click();
await page.waitForTimeout(200);
const urlAfterSwitch = await page.inputValue(".url-input");
const activeTitle = await page.locator(".tabbar .tab.active .tab-title").innerText().catch(()=>"<none>");
console.log(JSON.stringify({tabCount, urlAfterOpen, urlAfterSwitch, activeTitle, errs},null,2));
await browser.close();
