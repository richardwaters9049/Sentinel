"use client";
import { createContext, useContext, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { type ConsoleSession, sentinelFetch } from "@/lib/sentinel/session";

const SessionContext = createContext<ConsoleSession | null>(null);
export function useConsoleSession() { return useContext(SessionContext); }

export default function SessionProvider({ session, children }: { session: ConsoleSession | null; children: React.ReactNode }) {
  const router = useRouter();
  const [expired, setExpired] = useState(false);
  useEffect(() => {
    if (!session) return;
    const expire = () => { setExpired(true); router.replace("/sign-in?reason=expired"); router.refresh(); };
    window.addEventListener("sentinel-session-expired", expire);
    const timer = window.setTimeout(expire, Math.max(0, Date.parse(session.expires_at) - Date.now()));
    return () => { window.removeEventListener("sentinel-session-expired", expire); window.clearTimeout(timer); };
  }, [session, router]);
  if (expired) return <main className="p-8" role="status">Your session has expired. Opening sign-in…</main>;
  return <SessionContext value={session}>{children}</SessionContext>;
}

export function SessionControl() {
  const session = useConsoleSession();
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function signOut() {
    setBusy(true); setError("");
    try {
      const response = await sentinelFetch("/api/sentinel/api/v1/auth/logout", { method: "POST" });
      if (!response.ok) throw new Error("Sign-out failed. Try again.");
      router.replace("/sign-in"); router.refresh();
    } catch { setError("Sign-out failed. Try again."); } finally { setBusy(false); }
  }
  return <div className="mt-4 border-t border-slate-800 pt-4 text-xs text-slate-400">
    {session ? <><p className="break-all text-slate-200">{session.subject}</p><p className="mt-1 capitalize">{session.role}</p><button onClick={signOut} disabled={busy} className="mt-3 rounded border border-slate-700 px-3 py-2 text-slate-200 hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-sky-400 disabled:opacity-50">{busy ? "Signing out…" : "Sign out"}</button></> : <p>Local development mode</p>}
    {error && <p role="alert" className="mt-2 text-amber-300">{error}</p>}
  </div>;
}
