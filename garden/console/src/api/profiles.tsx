import { createContext, useContext, useEffect, useState } from "react";
import { get, post, setActiveProfile } from "./client";

export interface Profile { profile_id: string; display_name: string }
interface Profiles { profiles: Profile[]; selected: string; select: (id: string) => void; reload: () => void }
const Context = createContext<Profiles>({ profiles: [], selected: "", select: () => {}, reload: () => {} });
export const useProfiles = () => useContext(Context);

export function ProfileProvider({ children }: { children: React.ReactNode }) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selected, setSelected] = useState("");
  const [tick, setTick] = useState(0);
  const select = (id: string) => { setActiveProfile(id); localStorage.setItem("garden.profile", id); setSelected(id); };
  useEffect(() => {
    let active = true;
    get<Profile[]>("/v2/profiles").then((rows) => {
      if (!active) return;
      const list = rows ?? []; setProfiles(list);
      const saved = localStorage.getItem("garden.profile") ?? "";
      select(list.some((row) => row.profile_id === saved) ? saved : list[0]?.profile_id ?? "");
    }).catch(() => { if (active) { setProfiles([]); select(""); } });
    return () => { active = false; };
  }, [tick]);
  return <Context.Provider value={{ profiles, selected, select, reload: () => setTick((value) => value + 1) }}>{children}</Context.Provider>;
}

export function OwnerGate({ children }: { children: React.ReactNode }) {
  const [paired, setPaired] = useState<boolean | null>(null);
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => { get<{ paired: boolean }>("/v2/session").then((v) => setPaired(v.paired)).catch((e) => setError(String(e))); }, []);
  if (paired) return <ProfileProvider>{children}</ProfileProvider>;
  return <main className="page"><h1>Garden</h1><p>使用本机 Garden home 中 pairing-code 文件的一次性代码连接控制台。</p>
    <form onSubmit={async (event) => { event.preventDefault(); setBusy(true); setError(""); try { await post("/v2/pairing", { code }); setCode(""); setPaired(true); } catch (e) { setError(e instanceof Error ? e.message : String(e)); } finally { setBusy(false); } }}>
      <label>配对代码 <input className="modules-input" type="password" autoComplete="off" value={code} onChange={(e) => setCode(e.target.value)} /></label>
      <button className="btn-primary" disabled={paired === null || busy || !code.trim()}>连接</button>
    </form>{error && <p role="alert">{error}</p>}</main>;
}
