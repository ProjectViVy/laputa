import { useState } from "react";
import { post, put, remove } from "../api/client";
import { useApi } from "../api/hooks";
import { useProfiles } from "../api/profiles";
import HostBindings from "./HostBindings";
import PageHeader from "../components/PageHeader";
interface Space { id: string; name: string }
interface Member { space_id: string; profile_id: string; access: "read" | "read_write" }
const message = (e: unknown) => e instanceof Error ? e.message : String(e);

export default function AccessPage() {
  const { profiles, reload: reloadProfiles } = useProfiles();
  const spaces = useApi<Space[]>("/v2/shared-spaces");
  const [name, setName] = useState(""); const [profileName, setProfileName] = useState(""); const [error, setError] = useState("");
  return <div className="page"><PageHeader title="共享空间" lede="私有人格与记忆按 ProfileID 分离。共享空间只授予读取或读写权限。" />
    <section className="panel panel-pad"><h2>人格</h2><p>{profiles.length} 个已登记人格</p>
      <form onSubmit={async (e) => { e.preventDefault(); try { await post("/v2/profiles", { display_name: profileName.trim(), create: true }); setProfileName(""); reloadProfiles(); } catch (err) { setError(message(err)); } }}>
        <input className="modules-input" aria-label="人格名称" value={profileName} onChange={(e) => setProfileName(e.target.value)} placeholder="新人格名称" /><button className="btn-ghost" disabled={!profileName.trim()}>创建人格</button>
      </form>
    </section>
    <section className="panel panel-pad"><h2>命名共享空间</h2><form onSubmit={async (e) => { e.preventDefault(); try { await post("/v2/shared-spaces", { name }); setName(""); spaces.reload(); } catch (err) { setError(message(err)); } }}>
      <input className="modules-input" aria-label="共享空间名称" value={name} onChange={(e) => setName(e.target.value)} placeholder="空间名称" /><button className="btn-primary" disabled={!name.trim()}>创建空间</button>
    </form></section>
    {error && <p role="alert">{error}</p>}{spaces.error && <p role="alert">{spaces.error}</p>}
    <HostBindings />
    {(spaces.data ?? []).map((space) => <SpacePanel key={space.id} space={space} />)}
  </div>;
}
function SpacePanel({ space }: { space: Space }) {
  const { profiles, selected } = useProfiles(); const members = useApi<Member[]>(`/v2/shared-spaces/${space.id}/members`);
  const [participant, setParticipant] = useState(""); const [access, setAccess] = useState("read"); const [content, setContent] = useState(""); const [status, setStatus] = useState(""); const [busy, setBusy] = useState(false);
  const canWrite = members.data?.some((m) => m.profile_id === selected && m.access === "read_write");
  const change = async (id: string, right: string) => { try { if (right === "remove") await remove(`/v2/shared-spaces/${space.id}/members/${encodeURIComponent(id)}`); else await put(`/v2/shared-spaces/${space.id}/members/${encodeURIComponent(id)}`, { access: right }); members.reload(); setStatus(""); } catch (e) { setStatus(message(e)); } };
  return <section className="panel panel-pad"><h2>{space.name}</h2><p className="mono">shared:{space.id}</p>
    {(members.data ?? []).map((member) => <p key={member.profile_id}>{profiles.find((p) => p.profile_id === member.profile_id)?.display_name || member.profile_id} · <code>{member.profile_id}</code> <select aria-label={`${member.profile_id} 权限`} value={member.access} onChange={(e) => void change(member.profile_id, e.target.value)}><option value="read">读取</option><option value="read_write">读写</option></select> <button className="btn-ghost" onClick={() => void change(member.profile_id, "remove")}>移除</button></p>)}
    <form onSubmit={(e) => { e.preventDefault(); if (participant) void change(participant, access); }}><select aria-label="参与人格" value={participant} onChange={(e) => setParticipant(e.target.value)}><option value="">选择人格</option>{profiles.map((p) => <option key={p.profile_id} value={p.profile_id}>{p.display_name || p.profile_id}</option>)}</select> <select aria-label="授予权限" value={access} onChange={(e) => setAccess(e.target.value)}><option value="read">读取</option><option value="read_write">读写</option></select> <button className="btn-ghost" disabled={!participant}>添加参与者</button></form>
    {canWrite && <form onSubmit={async (e) => { e.preventDefault(); setBusy(true); try { const result = await post<{ canonical_status: string; index_status: string; id: string }>("/v2/memories", { content, kind: "fact", shared_space_id: space.id }); setContent(""); setStatus(`已保存 ${result.id} · canonical=${result.canonical_status} · index=${result.index_status}`); } catch (err) { setStatus(message(err)); } finally { setBusy(false); } }}><label>以当前人格在此共享空间新建记忆<textarea className="modules-input" value={content} onChange={(e) => setContent(e.target.value)} /></label><button className="btn-primary" disabled={busy || !content.trim()}>保存到 {space.name}</button></form>}
    {members.error && <p role="alert">{members.error}</p>}{status && <p role="status">{status}</p>}
  </section>;
}
