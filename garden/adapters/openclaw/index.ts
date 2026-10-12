import { createHash } from "node:crypto";
import { closeSync, constants, fstatSync, openSync, readFileSync, realpathSync, statSync } from "node:fs";
import { dirname, isAbsolute } from "node:path";
import { definePluginEntry, buildJsonPluginConfigSchema, type OpenClawPluginApi } from "openclaw/plugin-sdk/plugin-entry";
import manifest from "./openclaw.plugin.json" with { type: "json" };
import { isSubagentSessionKey, parseAgentSessionKey } from "openclaw/plugin-sdk/routing";

export interface NativeContext {
 agentId?: string; workspaceDir?: string; sessionKey?: string; sessionId?: string;
 modelProviderId?: string; modelId?: string; hookInvocation?: { assertActive(): void };
}
export interface GardenPromptConfig {
 endpoint: string; bindingID: string; credentialFile: string; hostInstance: string;
 agentID: string; workspaceDir: string; hostVersion: string;
 // Resolved from the actual host SDK at registration, never from plugin JSON.
 isEmbeddedContext(context: NativeContext): boolean;
 diagnose(code: string): void;
}
function privateCredential(path: string): string {
 if (!isAbsolute(path)) throw Error("credential path");
 const directory=statSync(realpathSync(dirname(path)));
 if ((directory.mode & 0o077)!==0 || directory.uid!==process.getuid!()) throw Error("credential directory");
 const fd=openSync(path,constants.O_RDONLY|constants.O_NOFOLLOW);
 try { const info=fstatSync(fd);if(!info.isFile()||(info.mode&0o077)!==0||info.uid!==process.getuid!()||info.size>4096)throw Error("credential file");const token=readFileSync(fd,"utf8").trim();if(!token||token.length>512||/[\r\n\0]/.test(token))throw Error("credential format");return token; }
 finally {closeSync(fd)}
}
export function createPromptHandler(config: GardenPromptConfig,fetchImpl:typeof fetch){
 // Resolve the selected workspace once. Each invocation verifies its actual native scope.
 let workspace="",endpoint="";
 try { const url=new URL(config.endpoint);if(url.protocol!=="http:"||!["localhost","127.0.0.1","[::1]"].includes(url.hostname)||url.username||url.password||url.search||url.hash||url.pathname!=="/")throw Error("endpoint");endpoint=url.origin;workspace=realpathSync(config.workspaceDir); }
 catch { /* Invalid setup remains inert and is diagnosed only when invoked. */ }
 return async(_event:unknown,ctx:NativeContext):Promise<{appendSystemContext:string}|undefined>=>{
  try {
   if(process.platform!=="linux"||process.arch!=="x64"||config.hostVersion!=="2026.10.2"||!endpoint||!workspace){config.diagnose("unsupported_host");return}
   if(!ctx.sessionId||!ctx.sessionKey||ctx.agentId!==config.agentID||parseAgentSessionKey(ctx.sessionKey)?.agentId!==config.agentID||isSubagentSessionKey(ctx.sessionKey)||!ctx.workspaceDir||realpathSync(ctx.workspaceDir)!==workspace||!config.isEmbeddedContext(ctx)){config.diagnose("unsupported_scope");return}
   ctx.hookInvocation?.assertActive();
   const credential=privateCredential(config.credentialFile);
   const scopeFingerprint=createHash("sha256").update(config.agentID+"\n"+workspace).digest("hex");
   const response=await fetchImpl(endpoint+`/v2/host-bindings/${encodeURIComponent(config.bindingID)}/projection/prompt`,{method:"POST",redirect:"error",headers:{"Content-Type":"application/json",Authorization:"Bearer "+credential,"X-Garden-Session":ctx.sessionId},body:JSON.stringify({adapter_version:"garden-native-prompt/v1",host_version:config.hostVersion,host_instance:config.hostInstance,session_id:ctx.sessionId,scope_fingerprint:scopeFingerprint}),signal:AbortSignal.timeout(1000)});
   if(!response.ok){config.diagnose(`http_${response.status}`);return}
   const result=await response.json() as {binding_id?:unknown;source_revisions?:unknown;hash?:unknown;markdown?:unknown};
   if(result.binding_id!==config.bindingID||typeof result.markdown!=="string"||typeof result.hash!=="string"||!result.source_revisions||Array.from(result.markdown).length>6000){config.diagnose("invalid_projection");return}
   ctx.hookInvocation?.assertActive();return {appendSystemContext:result.markdown};
  }catch{config.diagnose("unavailable");return}
 };
}

function embeddedContext(api:OpenClawPluginApi,ctx:NativeContext):boolean{
 if(!ctx.sessionKey||!ctx.modelProviderId||!ctx.modelId)return false;
 const entry=api.runtime.agent.session.getSessionEntry({agentId:ctx.agentId,sessionKey:ctx.sessionKey,readConsistency:"latest"});
 if(entry?.spawnedBy||entry?.parentSessionId||(entry?.agentHarnessId&&entry.agentHarnessId!=="openclaw")||(entry?.agentRuntimeOverride&&entry.agentRuntimeOverride!=="openclaw"))return false;
 // Public SDK uses the same read-only model/runtime policy as native turns.
 return !!api.runtime.agent.resolveSessionCatalogCreateTarget({config:api.runtime.config.current() as OpenClawPluginApi["config"],requestedAgentId:ctx.agentId,provider:ctx.modelProviderId,modelIds:[ctx.modelId],agentRuntime:"openclaw"});
}
export default definePluginEntry({id:"garden",name:"Garden",description:"Opt-in native Garden Persona",configSchema:buildJsonPluginConfigSchema(manifest.configSchema),register(api){
 if(api.registrationMode!=="full")return;
 const cfg=api.pluginConfig??{};const names=["endpoint","bindingID","credentialFile","hostInstance","agentID","workspaceDir"] as const;
 if(names.some(name=>typeof cfg[name]!=="string"||!String(cfg[name]).trim())){api.logger.warn("Garden Persona inactive: incomplete configuration");return}
 const config={...Object.fromEntries(names.map(name=>[name,cfg[name]])),hostVersion:api.runtime.version,isEmbeddedContext:(ctx:NativeContext)=>embeddedContext(api,ctx),diagnose:(code:string)=>api.logger.warn(`Garden Persona omitted: ${code}`)} as GardenPromptConfig;
 api.on("before_prompt_build",createPromptHandler(config,fetch));
}});
