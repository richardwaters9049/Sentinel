export type ConsoleSession = {
  subject: string;
  role: "analyst" | "administrator";
  expires_at: string;
  csrf_token: string;
};

export function parseConsoleSession(value: unknown): ConsoleSession | null {
  if (typeof value !== "object" || value === null) return null;
  const session = value as Record<string, unknown>;
  if (typeof session.subject !== "string" || !/^[A-Za-z0-9][A-Za-z0-9_.:@-]{0,127}$/.test(session.subject) ||
      (session.role !== "analyst" && session.role !== "administrator") ||
      typeof session.expires_at !== "string" || !Number.isFinite(Date.parse(session.expires_at)) ||
      Date.parse(session.expires_at) <= Date.now() || Date.parse(session.expires_at) > Date.now() + 8*3600_000 + 60_000 ||
      typeof session.csrf_token !== "string" || !/^[a-f0-9]{64}$/.test(session.csrf_token)) return null;
  return { subject: session.subject, role: session.role, expires_at: session.expires_at, csrf_token: session.csrf_token };
}

export function notifySessionExpired(): void {
  if (typeof window !== "undefined") window.dispatchEvent(new Event("sentinel-session-expired"));
}

// Retrieve CSRF context at mutation time, including after sign-in rotation in another tab.
export async function sentinelFetch(input: string, init?: RequestInit): Promise<Response> {
  const method = init?.method?.toUpperCase() ?? "GET";
  const headers = new Headers(init?.headers);
  if (method !== "GET" && method !== "HEAD") {
    const context = await fetch("/api/sentinel/api/v1/session", { cache: "no-store", signal: init?.signal });
    const payload: unknown = await context.json();
    const session = parseConsoleSession(payload);
    if (session) headers.set("X-Sentinel-CSRF", session.csrf_token);
    else if (!context.ok && typeof payload === "object" && payload !== null &&
             "error" in payload && typeof payload.error === "object" && payload.error !== null &&
             "code" in payload.error && payload.error.code === "authentication_disabled") {
      // Explicit gateway development compatibility has no cookie session.
    } else {
      if (context.status === 401 || context.ok) notifySessionExpired();
      return context.ok ? Response.json({ error: { code: "unauthenticated", message: "Session has expired" } }, { status: 401 })
                        : Response.json(payload, { status: context.status });
    }
  }
  const response = await fetch(input, { ...init, headers });
  if (response.status === 401) notifySessionExpired();
  return response;
}
