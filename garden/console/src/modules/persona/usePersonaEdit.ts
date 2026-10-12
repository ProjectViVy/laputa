import { useCallback, useEffect, useRef, useState } from "react";
import { useBlocker } from "react-router-dom";
import { ApiError, sendForProfile } from "../../api/client";
import { useProfiles } from "../../api/profiles";
import type { EditLease, PersonaDocument, PersonaHistoryRevision } from "./types";

type Mode = "viewing" | "acquiring" | "editing" | "saving" | "recovering";
interface Editor { mode: Mode; document: PersonaDocument | null; base: PersonaDocument | null; lease: EditLease | null; draft: string; message: string; latest: PersonaDocument | null }
const initial: Editor = { mode: "viewing", document: null, base: null, lease: null, draft: "", message: "", latest: null };
const reasonOf = (e: unknown) => e instanceof Error ? e.message : String(e);
const normalized = (s: string) => s.replace(/\r\n?/g, "\n").normalize("NFC").replace(/^[\u0009-\u000d\u0020\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+|[\u0009-\u000d\u0020\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+$/g, "");
async function contentHash(s: string) {
  const hash = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(normalized(s)));
  return "sha256:" + Array.from(new Uint8Array(hash), (v) => v.toString(16).padStart(2, "0")).join("");
}

// Every request is pinned to this mount's profile/kind and editor attempt.
export function usePersonaEdit(kind: string) {
  const { selected: profile, registerNavigationGuard } = useProfiles();
  const path = `/v2/persona/documents/${encodeURIComponent(kind)}`;
  const [state, setState] = useState<Editor>(initial);
  const current = useRef(state), attempt = useRef(0), readSequence = useRef(0), mounted = useRef(false);
  const update = useCallback((patch: Partial<Editor>) => { current.current = { ...current.current, ...patch }; setState(current.current); }, []);
  const request = useCallback(<T,>(suffix = "", method = "GET", body?: unknown) => sendForProfile<T>(profile, path + suffix, { method, headers: body === undefined ? undefined : { "Content-Type": "application/json" }, body: body === undefined ? undefined : JSON.stringify(body) }), [profile, path]);
  const release = useCallback((lease: EditLease) => request("/edit-lease", "DELETE", { token: lease.token, generation: lease.generation }).catch(() => {}), [request]);
  const valid = (id: number) => mounted.current && attempt.current === id;
  const canNavigate = useCallback(() => {
    const s = current.current;
    if (s.mode === "saving") { window.alert("Wait for the save result before leaving this document."); return false; }
    return !s.base || s.draft === s.base.content || window.confirm("Discard this unsaved draft?");
  }, []);
  useEffect(() => registerNavigationGuard(canNavigate), [registerNavigationGuard, canNavigate]);
  const blocker = useBlocker(({ currentLocation, nextLocation }) => currentLocation.pathname !== nextLocation.pathname && (state.mode === "saving" || !!state.base && state.draft !== state.base.content));
  useEffect(() => { if (blocker.state === "blocked") { if (canNavigate()) blocker.proceed(); else blocker.reset(); } }, [blocker, canNavigate]);
  useEffect(() => {
    mounted.current = true;
    const unload = (event: BeforeUnloadEvent) => { const s = current.current; if (s.mode === "saving" || s.base && s.draft !== s.base.content) event.preventDefault(); };
    window.addEventListener("beforeunload", unload);
    return () => { mounted.current = false; attempt.current++; const lease = current.current.lease; if (lease) void release(lease); window.removeEventListener("beforeunload", unload); };
  }, [release]);
  const refresh = useCallback(async () => {
    if (current.current.mode !== "viewing" || document.visibilityState !== "visible") return;
    const id = attempt.current, sequence = ++readSequence.current;
    try { const document = await request<PersonaDocument>(); if (valid(id) && sequence === readSequence.current && current.current.mode === "viewing") update({ document, message: "" }); }
    catch (e) { if (valid(id) && sequence === readSequence.current && current.current.mode === "viewing") update({ message: reasonOf(e) }); }
  }, [request, update]);
  useEffect(() => {
    if (state.mode !== "viewing") return;
    void refresh(); const timer = setInterval(() => void refresh(), 2000);
    const focus = () => void refresh(); window.addEventListener("focus", focus); document.addEventListener("visibilitychange", focus);
    return () => { clearInterval(timer); window.removeEventListener("focus", focus); document.removeEventListener("visibilitychange", focus); };
  }, [state.mode, refresh]);
  useEffect(() => {
    if (state.mode !== "editing" || !state.lease) return;
    const id = attempt.current, lease = state.lease;
    const timer = setInterval(async () => {
      try { const renewed = await request<EditLease>("/edit-lease", "PATCH", { token: lease.token, generation: lease.generation }); if (valid(id) && current.current.mode === "editing") update({ lease: renewed }); }
      catch (e) { if (valid(id) && current.current.mode === "editing") update({ mode: "recovering", latest: null, message: `Editing permission was lost. Your draft is preserved. ${reasonOf(e)}` }); }
    }, 30000);
    return () => clearInterval(timer);
  }, [state.mode, state.lease, request, update]);
  const acquire = async (base: PersonaDocument, reconcile: boolean) => {
    const id = ++attempt.current, draft = reconcile ? current.current.draft : base.content;
    update({ mode: "acquiring", message: "" });
    try {
      if (reconcile && current.current.lease) await release(current.current.lease);
      if (!valid(id)) return;
      const lease = await request<EditLease>("/edit-lease", "POST", { base_revision: base.revision, base_hash: base.content_hash, request_id: crypto.randomUUID() });
      if (!valid(id)) { void release(lease); return; }
      update({ mode: "editing", base, document: base, lease, draft, latest: null });
    } catch (e) { if (valid(id)) update({ mode: reconcile ? "recovering" : "viewing", message: reasonOf(e) }); }
  };
  const cancel = () => { const s = current.current; if (!canNavigate()) return; attempt.current++; if (s.lease) void release(s.lease); update({ mode: "viewing", base: null, lease: null, draft: "", latest: null, message: "" }); };
  const save = async (reason: string) => {
    const s = current.current;
    if (s.mode !== "editing" || !s.base || !s.lease) return;
    if (Date.parse(s.lease.expires_at) <= Date.now()) { update({ mode: "recovering", message: "Editing permission expired. Your draft is preserved.", latest: null }); return; }
    const id = ++attempt.current;
    update({ mode: "saving", message: "" });
    const finish = (document: PersonaDocument) => { if (valid(id)) update({ mode: "viewing", document, base: null, lease: null, draft: "", latest: null, message: "Saved in Garden." }); };
    try {
      await request("", "PUT", { base_revision: s.base.revision, base_hash: s.base.content_hash, edit_proof: { token: s.lease.token, generation: s.lease.generation }, content: s.draft, scope: "document", reason: reason.trim() || "User-authorized Persona document edit" });
      const document = await request<PersonaDocument>(); finish(document);
    } catch (e) {
      if (!valid(id)) return;
      if (e instanceof ApiError && (e.status === 400 || e.status === 413 || e.code === "persona_world_protected_claim")) { update({ mode: "editing", message: reasonOf(e) }); return; }
      // An interrupted response may follow a completed commit: inspect authority and history, never resend blindly.
      if (!(e instanceof ApiError)) {
        try {
          const document = await request<PersonaDocument>();
          const hash = await contentHash(s.draft);
          const history = await sendForProfile<PersonaHistoryRevision>(profile, `/v2/persona/history/${encodeURIComponent(kind)}/${document.revision}`, {});
          if (document.content_hash === hash && !document.edit_status?.held && history.entry.content_hash === hash && (history.entry.base_revision === s.base.revision || document.revision === s.base.revision)) { finish(document); return; }
        } catch { /* remain in explicit recovery */ }
      }
      if (valid(id)) update({ mode: "recovering", latest: null, message: `Save could not be confirmed. Your draft is preserved. ${reasonOf(e)}` });
    }
  };
  const readLatest = async () => { const id = ++attempt.current; try { const latest = await request<PersonaDocument>(); if (valid(id)) update({ latest, message: "Compare the latest document with your draft before continuing." }); } catch (e) { if (valid(id)) update({ message: reasonOf(e) }); } };
  const reconcile = () => { const s = current.current; if (s.latest && window.confirm("Use this inspected revision as the new base for your reconciled draft?")) void acquire(s.latest, true); };
  return { ...state, begin: () => { if (current.current.document) void acquire(current.current.document, false); }, cancel, save, readLatest, reconcile, setDraft: (draft: string) => update({ draft }), refresh };
}
