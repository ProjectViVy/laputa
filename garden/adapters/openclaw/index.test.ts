import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createPromptHandler, type GardenPromptConfig, type NativeContext } from "./index.js";

function fixture() {
 const dir = mkdtempSync(join(tmpdir(), "garden-openclaw-"));
 const credential = join(dir, "credential"); writeFileSync(credential, "fixture-private-token", { mode: 0o600 });
 const soul = join(dir,"SOUL.md");writeFileSync(soul,"EXISTING_NATIVE_SOUL");
 const config: GardenPromptConfig = { endpoint: "http://127.0.0.1:4321", bindingID: "binding-A", credentialFile: credential, hostInstance: "existing-test-host", agentID: "main", workspaceDir: dir, hostVersion: "2026.10.2", isEmbeddedContext: () => true, diagnose: () => {} };
 const ctx: NativeContext = { agentId: "main", workspaceDir: dir, sessionKey: "agent:main:main", sessionId: "session-A", modelProviderId: "fixture", modelId: "fixture-model" };
 return {dir,soul,config,ctx};
}
test("exact additive contribution, pinned scope/session, no prompt/history or file writes", async()=>{
 const f=fixture();try{
 let body:any,headers:Headers,url="";
 const fetcher: typeof fetch = async(input,init)=>{url=String(input);body=JSON.parse(String(init?.body));headers=new Headers(init?.headers);return Response.json({binding_id:"binding-A",source_revisions:{identity:3},hash:"hash",markdown:"[Garden Persona]\n## IDENTITY.MD (revision 3)\nLATEST\n[/Garden Persona]"})};
 const result=await createPromptHandler(f.config,fetcher)({prompt:"PRIVATE_PROMPT",messages:["PRIVATE_HISTORY"]},f.ctx);
 assert.deepEqual(Object.keys(result??{}),["appendSystemContext"]);assert.match(result!.appendSystemContext,/revision 3/);assert.equal(url,f.config.endpoint+"/v2/host-bindings/binding-A/projection/prompt");assert.equal(body.session_id,"session-A");assert.equal(body.host_version,"2026.10.2");assert.match(body.scope_fingerprint,/^[a-f0-9]{64}$/);assert.equal(headers!.get("X-Garden-Session"),"session-A");assert.equal(headers!.get("Authorization"),"Bearer fixture-private-token");assert.equal(Object.keys(body).length,5);assert.doesNotMatch(JSON.stringify(body),/PRIVATE_PROMPT|PRIVATE_HISTORY/);assert.equal(readFileSync(f.soul,"utf8"),"EXISTING_NATIVE_SOUL");
 }finally{rmSync(f.dir,{recursive:true,force:true})}
});
test("wrong scope, worker, unsupported harness/version skip before transport",async()=>{
 const f=fixture();try{let calls=0;const fetcher:typeof fetch=async()=>{calls++;throw Error("must not call")};
 for(const ctx of [{...f.ctx,agentId:"other"},{...f.ctx,workspaceDir:tmpdir()},{...f.ctx,sessionKey:"agent:main:subagent:worker"},{...f.ctx,sessionId:undefined}])assert.equal(await createPromptHandler(f.config,fetcher)({},ctx),undefined);
 assert.equal(await createPromptHandler({...f.config,isEmbeddedContext:()=>false},fetcher)({},f.ctx),undefined);assert.equal(await createPromptHandler({...f.config,hostVersion:"unknown"},fetcher)({},f.ctx),undefined);assert.equal(calls,0);
 }finally{rmSync(f.dir,{recursive:true,force:true})}
});
test("disabled/locked/offline omit context without cached fallback or token diagnostics",async()=>{
 const f=fixture();try{const diagnostics:string[]=[];f.config.diagnose=(code)=>diagnostics.push(code);let status=200;const fetcher:typeof fetch=async()=>status===200?Response.json({binding_id:"binding-A",source_revisions:{identity:1},hash:"hash",markdown:"OLD_BODY"}):new Response("fixture-private-token PRIVATE_BODY",{status});const handler=createPromptHandler(f.config,fetcher);assert.equal((await handler({},f.ctx))?.appendSystemContext,"OLD_BODY");for(const denied of [403,409,503]){status=denied;assert.equal(await handler({},f.ctx),undefined)}assert.doesNotMatch(diagnostics.join(" "),/OLD_BODY|private-token|PRIVATE_BODY/);
 assert.equal(await createPromptHandler(f.config,async()=>{throw Error("fixture-private-token")})({},f.ctx),undefined);
 }finally{rmSync(f.dir,{recursive:true,force:true})}
});
test("HTTP timeout aborts at one second and emits no body",async()=>{
 const f=fixture();try{let signal:AbortSignal|undefined;const started=Date.now();const fetcher:typeof fetch=async(_url,init)=>new Promise((_resolve,reject)=>{signal=init?.signal??undefined;signal?.addEventListener("abort",()=>reject(Error("timeout")),{once:true})});const result=await createPromptHandler(f.config,fetcher)({},f.ctx);assert.equal(result,undefined);assert.equal(signal?.aborted,true);assert.ok(Date.now()-started<1500);
 }finally{rmSync(f.dir,{recursive:true,force:true})}
});
