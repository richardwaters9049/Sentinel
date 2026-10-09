"use client";

import { createContext, useContext, useState } from "react";
import type { ConsoleSession } from "@/lib/sentinel/session";

type ConsoleAccess = { session: ConsoleSession | null; canAdminister: boolean };
const AccessContext = createContext<ConsoleAccess>({ session: null, canAdminister: false });

export function ConsoleAccessProvider({ session, development = false, children }: {
  session: ConsoleSession | null;
  development?: boolean;
  children?: React.ReactNode;
}) {
  const [verifiedAt] = useState(() => Date.now());
  // Development access is an explicit server-layout choice, never inferred from a missing session.
  const canAdminister = session
    ? session.role === "administrator" && Date.parse(session.expires_at) > verifiedAt
    : development;
  return <AccessContext value={{ session, canAdminister }}>{children}</AccessContext>;
}

export function useConsoleSession() { return useContext(AccessContext).session; }
export function useCanAdminister() { return useContext(AccessContext).canAdminister; }

export function AdministrationNotice({ id, action }: { id: string; action: string }) {
  const canAdminister = useCanAdminister();
  if (canAdminister) return null;
  return <p id={id} className="mb-3 text-xs leading-5 text-slate-400">
    Administrator access is required to {action}. You can still review the current configuration and evidence.
  </p>;
}
