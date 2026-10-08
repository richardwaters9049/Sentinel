import { cookies } from "next/headers";
import { parseConsoleSession } from "./session";

export const gatewayBase = process.env.SENTINEL_GATEWAY_URL?.replace(/\/$/, "") ?? "http://127.0.0.1:8080";
export const consoleAuthMode = process.env.SENTINEL_CONSOLE_AUTH_MODE ??
  (process.env.NODE_ENV === "development" ? "development" : "required");
if (consoleAuthMode !== "development" && consoleAuthMode !== "required") throw new Error("Invalid console authentication mode");

export async function readServerSession() {
  const jar = await cookies();
  const candidates = [...jar.getAll("sentinel_session"), ...jar.getAll("__Host-sentinel_session")];
  if (candidates.length !== 1 || !/^[A-Za-z0-9_-]{43}$/.test(candidates[0].value)) return null;
  const response = await fetch(`${gatewayBase}/api/v1/session`, {
    headers: { Cookie: `${candidates[0].name}=${candidates[0].value}` },
    cache: "no-store", redirect: "error", signal: AbortSignal.timeout(5000),
  });
  if (response.status === 401) return null;
  if (!response.ok) throw new Error("Session verification is temporarily unavailable");
  const payload: unknown = await response.json();
  const session = parseConsoleSession(payload);
  if (!session) throw new Error("Session verification returned invalid data");
  return session;
}
