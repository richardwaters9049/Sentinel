import { NextRequest, NextResponse } from "next/server";

const gatewayBase = process.env.SENTINEL_GATEWAY_URL?.replace(/\/$/, "") ?? "http://127.0.0.1:8080";
const allowedMethods = new Set(["GET", "POST", "PATCH"]);
const sessionCookies = new Set(["sentinel_session", "__Host-sentinel_session"]);

function error(status: number, code: string, message: string) {
  return NextResponse.json({ error: { code, message } }, { status, headers: { "Cache-Control": "no-store" } });
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  if (!allowedMethods.has(request.method)) return error(405, "method_not_allowed", "Method not allowed");
  const { path } = await context.params;
  let safePath: string;
  try {
    const segments = path.map((segment) => decodeURIComponent(segment));
    if (segments.some((segment) => !segment || segment === "." || segment === ".." || /[/\\]/.test(segment)))
      return error(400, "invalid_path", "Invalid API path");
    safePath = segments.map(encodeURIComponent).join("/");
  } catch { return error(400, "invalid_path", "Invalid API path"); }
  const target = new URL(`${gatewayBase}/${safePath}`);
  request.nextUrl.searchParams.forEach((value, key) => target.searchParams.append(key, value));
  const headers = new Headers({ Accept: "application/json" });
  for (const header of ["content-type", "authorization", "x-sentinel-actor", "x-request-id", "origin", "x-sentinel-csrf", "traceparent", "x-sentinel-timestamp", "x-sentinel-nonce", "x-sentinel-signature"]) {
    const value = request.headers.get(header);
    if (value) headers.set(header, value);
  }
  // Preserve duplicate session cookies for the gateway to reject; forward no unrelated cookies.
  const selectedCookies = (request.headers.get("cookie") ?? "").split(";").map((value) => value.trim())
    .filter((value) => sessionCookies.has(value.split("=", 1)[0]));
  if (selectedCookies.length) headers.set("Cookie", selectedCookies.join("; "));
  let body: Uint8Array | undefined;
  if (request.method !== "GET" && request.body) {
    const reader = request.body.getReader();
    const chunks: Uint8Array[] = []; let size = 0;
    const limit = safePath === "api/v1/auth/login" ? 2048 : 1 << 20;
    try {
      while (true) {
        const { done, value } = await reader.read(); if (done) break;
        size += value.byteLength;
        if (size > limit) { await reader.cancel(); return error(413, "body_too_large", "Request body is too large"); }
        chunks.push(value);
      }
      body = new Uint8Array(size); let offset = 0;
      for (const chunk of chunks) { body.set(chunk, offset); offset += chunk.byteLength; }
    } catch { return error(400, "invalid_body", "Request body could not be read"); }
  }
  try {
    const response = await fetch(target, { method: request.method, headers, body: body as BodyInit | undefined,
      cache: "no-store", redirect: "error", signal: AbortSignal.timeout(10_000) });
    const outputHeaders = new Headers({ "Content-Type": response.headers.get("content-type") ?? "application/json", "Cache-Control": "no-store" });
    for (const name of ["traceparent", "x-sentinel-request-id", "retry-after"]) {
      const value = response.headers.get(name);
      if (value) outputHeaders.set(name, value);
    }
    const cookie = response.headers.get("set-cookie");
    if (cookie) outputHeaders.set("Set-Cookie", cookie);
    const challenge = response.headers.get("www-authenticate");
    if (challenge) outputHeaders.set("WWW-Authenticate", challenge);
    return new NextResponse(await response.arrayBuffer(), { status: response.status, headers: outputHeaders });
  } catch { return error(503, "gateway_unavailable", "Sentinel gateway is unavailable"); }
}
export const GET = proxy;
export const POST = proxy;
export const PATCH = proxy;
