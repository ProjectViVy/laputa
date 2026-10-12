import { useEffect, useRef, useState } from "react";
import { post, put, remove } from "../api/client";
import { useApi } from "../api/hooks";
import { useProfiles } from "../api/profiles";
import type { HostBinding, HostControls, HostRetryPreview, HostStatus, NativeBindingConfig } from "../api/types";

const versions: Record<string, string> = { openclaw: "2026.10.2", hermes: "0.21.7", "codex-cli": "0.162.1", "claude-code": "2.1.296" };
const defaultKinds = ["identity", "relationship", "redline", "user"];
const explain = (e: unknown) => e instanceof Error ? e.message : String(e);

export default function HostBindings() {
  const { selected: profile } = useProfiles();
  const bindings = useApi<HostBinding[]>("/v2/host-bindings");
  const [selected, setSelected] = useState("");
  const [agent, setAgent] = useState("");
  const [platform, setPlatform] = useState("openclaw");
  const [workspace, setWorkspace] = useState("");
  const [credential, setCredential] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const rows = (bindings.data ?? []).filter((b) => b.profile_id === profile);
  const binding = rows.find((b) => b.binding_id === selected);
  return <section className="panel panel-pad"><h2>宿主集成</h2><p>为当前人格登记已有宿主。三个开关独立，创建后默认关闭；此页不安装插件或修改宿主文件。</p>
    <form onSubmit={async (event) => { event.preventDefault(); setBusy(true); setMessage(""); try { const result = await post<{ binding: HostBinding; credential: string }>("/v2/host-bindings", { profile_id: profile, agent_id: agent.trim(), platform, workspace_id: workspace.trim() }); setCredential(result.credential); setSelected(result.binding.binding_id); bindings.reload(); } catch (e) { setMessage(explain(e)); } finally { setBusy(false); } }}>
      <label>宿主 Agent ID<input className="modules-input" aria-label="宿主 Agent ID" value={agent} onChange={(e) => setAgent(e.target.value)} /></label>
      <label>宿主<select aria-label="宿主类型" value={platform} onChange={(e) => setPlatform(e.target.value)}>{Object.keys(versions).map((id) => <option key={id}>{id}</option>)}</select></label>
      <label>Garden workspace ID（可选）<input className="modules-input" aria-label="Garden workspace ID" value={workspace} onChange={(e) => setWorkspace(e.target.value)} /></label>
      <button className="btn-ghost" disabled={busy || !profile || !agent.trim()}>登记宿主</button>
    </form>
    {credential && <div role="status"><p>凭据仅显示此次。请保存到宿主插件指定的私有凭据文件。</p><input aria-label="新宿主凭据" type="password" readOnly value={credential} /><button className="btn-ghost" onClick={() => setCredential("")}>已保存，隐藏凭据</button></div>}
    <label>已有绑定<select aria-label="已有宿主绑定" value={selected} onChange={(e) => { setSelected(e.target.value); setCredential(""); }}>{<option value="">选择宿主</option>}{rows.map((b) => <option key={b.binding_id} value={b.binding_id}>{b.agent_id} · {b.platform}{b.revoked ? " · 已撤销" : ""}</option>)}</select></label>
    {binding && <HostBindingPanel key={binding.binding_id} binding={binding} changed={bindings.reload} />}
    {(message || bindings.error) && <p role="alert">{message || bindings.error}</p>}
  </section>;
}
function HostBindingPanel({ binding, changed }: { binding: HostBinding; changed: () => void }) {
  const path = `/v2/host-bindings/${encodeURIComponent(binding.binding_id)}`;
  const status = useApi<HostStatus>(path + "/projection", { poll: 2000 });
  const [controls, setControls] = useState(binding.controls);
  useEffect(() => setControls(binding.controls), [binding.controls]);
  const [adapter, setAdapter] = useState(versions[binding.platform] ? binding.platform : "openclaw");
  const [instance, setInstance] = useState("");
  const [scope, setScope] = useState("");
  const [kinds, setKinds] = useState(defaultKinds);
  const [preview, setPreview] = useState<HostRetryPreview | null>(null);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const act = async (work: () => Promise<void>) => { setBusy(true); setMessage(""); try { await work(); status.reload(); changed(); } catch (e) { setMessage(explain(e)); } finally { setBusy(false); } };
  const control = (key: keyof HostControls, enabled: boolean) => {
    const previous = controls, next = { ...controls, [key]: enabled }; setControls(next);
    void act(async () => { try { await put(path + "/controls", next); } catch (e) { setControls(previous); throw e; } if (key === "persona_sync" && !enabled) setPreview(null); });
  };
  const saved = status.data;
  const hydrated = useRef(false);
  useEffect(() => {
    if (saved?.native && !hydrated.current) { hydrated.current = true; setAdapter(saved.native.adapter_id); setInstance(saved.native.host_instance); setKinds(saved.native.kinds); }
  }, [saved?.native]);
  return <div><p>人格 <code>{binding.profile_id}</code> · Agent <code>{binding.agent_id}</code> · workspace <code>{binding.workspace_id || "personal"}</code></p>
    <fieldset disabled={busy || binding.revoked || binding.worker}><legend>独立开关</legend>{([ ["persona_sync", "人格自动投影"], ["recall", "记忆召回"], ["capture", "经验采集"] ] as const).map(([key, label]) => <label key={key}><input type="checkbox" checked={controls[key]} onChange={(e) => control(key, e.target.checked)} />{label}</label>)}</fieldset>
    {saved && <div role="status"><p>状态：{saved.state} · {saved.mode || "manual_file"}</p><p>需要重试：{saved.pending_retry ? saved.pending_reason || "是" : "否"} · 等待宿主：{saved.waiting_host ? "是" : "否"}</p><p>已保存：{JSON.stringify(saved.saved_revisions ?? {})}</p><p>文件写入：{saved.mode === "native_prompt" ? "不适用" : saved.written_hash || "尚未写入"}</p><p>已提供给宿主钩子：{saved.offered_hash ? JSON.stringify(saved.offered_revisions) : "尚未提供"} · 宿主已加载：{saved.loaded_hash || "未知"}</p><p>最近提供：{saved.last_host_seen_at || "无记录"}</p>{saved.native && <p className="mono">{saved.native.adapter_id} {saved.native.host_version} · {saved.native.host_instance} · scope={saved.native.scope_fingerprint} · sources={saved.native.kinds.join(", ")}</p>}{saved.manual_action && <p>手动清理：{saved.manual_action} · {saved.manual_path}</p>}</div>}
    <form onSubmit={(event) => { event.preventDefault(); void act(async () => { const bytes = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(scope)); const fingerprint = Array.from(new Uint8Array(bytes), (v) => v.toString(16).padStart(2, "0")).join(""); const cfg: NativeBindingConfig = { adapter_id: adapter, adapter_version: "garden-native-prompt/v1", host_version: versions[adapter], host_instance: instance.trim(), scope_fingerprint: fingerprint, kinds }; await put(path + "/projection/native-config", cfg); setPreview(null); }); }}>
      <fieldset disabled={busy || !saved || binding.revoked || binding.worker}><legend>原生人格投影（Linux x86_64）</legend><label>宿主版本<select aria-label="原生宿主版本" value={adapter} onChange={(e) => setAdapter(e.target.value)}>{Object.entries(versions).map(([id, version]) => <option key={id} value={id}>{id} {version}</option>)}</select></label>
        <label>已有宿主安装标识<input className="modules-input" aria-label="宿主安装标识" value={instance} onChange={(e) => setInstance(e.target.value)} /></label>
        <label>已确认的原生范围<textarea className="modules-input" aria-label="原生范围" value={scope} onChange={(e) => setScope(e.target.value)} /></label><p>OpenClaw：Agent ID、换行、真实 workspace 绝对路径。Hermes：已有 home 的真实绝对路径。CLI：当前工作目录的真实绝对路径。插件必须使用完全相同的范围。</p>
        {defaultKinds.concat(["dream", "dark", "mission"]).map((kind) => <label key={kind}><input type="checkbox" checked={kinds.includes(kind)} onChange={(e) => setKinds((old) => e.target.checked ? [...old, kind] : old.filter((item) => item !== kind))} />{kind}</label>)}
        <button className="btn-ghost" disabled={!instance.trim() || !scope.trim() || !kinds.length}>配置原生投影</button>
      </fieldset>
    </form>
    {saved?.mode === "native_prompt" && <div><button className="btn-ghost" disabled={busy || binding.revoked} onClick={() => void act(async () => { setPreview(await post<HostRetryPreview>(path + "/projection/retry-preview", {})); })}>读取新鲜预览</button>{preview && <section aria-label="人格投影预览"><p>来源版本：{JSON.stringify(preview.projection.source_revisions)} · 有效至 {preview.expires_at}</p><pre className="persona-content mono">{preview.projection.markdown}</pre><button className="btn-primary" disabled={busy || !controls.persona_sync || binding.revoked} onClick={() => void act(async () => { await post(path + "/projection/retry-confirm", { preview_id: preview.preview_id }); setPreview(null); })}>确认此次预览，等待宿主</button><p>先显式启用“人格自动投影”。确认后由下一次宿主钩子读取；离线宿主不会被后台写入。</p></section>}</div>}
    <button className="btn-ghost" disabled={busy || binding.revoked} onClick={() => { if (window.confirm("撤销此宿主凭据并停止三个开关？已有宿主文件可能需要手动清理。")) void act(async () => { await remove(path); setPreview(null); }); }}>撤销宿主</button>
    {(message || status.error) && <p role="alert">{message || status.error}</p>}
  </div>;
}
