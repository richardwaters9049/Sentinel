"use client";
import { useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { parseConsoleSession } from "@/lib/sentinel/session";

export default function SignInForm() {
  const credential = useRef<HTMLInputElement>(null);
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setPending(true); setError("");
    const token = credential.current?.value.trim() ?? "";
    try {
      const response = await fetch("/api/sentinel/api/v1/auth/login", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token }), cache: "no-store",
      });
      const payload: unknown = await response.json();
      if (!response.ok || !parseConsoleSession(payload)) {
        setError(response.status === 401 ? "Credential is invalid or expired." : response.status === 403 ? "This credential or origin cannot sign in to the console." : "Sign-in is unavailable. Try again.");
        return;
      }
      router.replace("/"); router.refresh();
    } catch { setError("Sign-in is unavailable. Try again."); }
    finally { if (credential.current) credential.current.value = ""; setPending(false); }
  }
  return <form onSubmit={submit} className="mt-8 space-y-4">
    <label htmlFor="console-credential" className="block text-sm text-slate-200">Short-lived console credential</label>
    <input id="console-credential" ref={credential} type="password" required autoComplete="off" maxLength={43} minLength={43} spellCheck={false} className="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 text-slate-100 focus:border-sky-400 focus:outline-2 focus:outline-sky-400" aria-describedby="credential-guidance" disabled={pending} />
    <p id="credential-guidance" className="text-xs leading-5 text-slate-400">Use an analyst or administrator credential issued for this simulated lab.</p>
    {error && <p role="alert" className="text-sm text-amber-300">{error}</p>}
    <button type="submit" disabled={pending} className="w-full rounded-lg bg-sky-300 px-4 py-3 font-semibold text-slate-950 hover:bg-sky-200 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-sky-300 disabled:opacity-50">{pending ? "Signing in…" : "Sign in"}</button>
  </form>;
}
