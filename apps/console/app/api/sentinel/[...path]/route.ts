import { NextRequest, NextResponse } from "next/server";

const gatewayBase =
  process.env.SENTINEL_GATEWAY_URL?.replace(/\/$/, "") ??
  "http://127.0.0.1:8080";

const allowedMethods = new Set(["GET", "POST", "PATCH"]);

async function proxy(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> },
) {
  if (!allowedMethods.has(request.method)) {
    return NextResponse.json(
      { error: { code: "method_not_allowed", message: "Method not allowed" } },
      { status: 405 },
    );
  }

  const { path } = await context.params;
  const safePath = path
    .map((segment) => encodeURIComponent(decodeURIComponent(segment)))
    .join("/");

  const target = new URL(`${gatewayBase}/${safePath}`);
  request.nextUrl.searchParams.forEach((value, key) => {
    target.searchParams.append(key, value);
  });

  const headers = new Headers();
  headers.set("Accept", "application/json");

  const contentType = request.headers.get("content-type");
  if (contentType) headers.set("Content-Type", contentType);

  for (const header of ["authorization", "x-sentinel-actor", "x-request-id"]) {
    const value = request.headers.get(header);
    if (value) headers.set(header, value);
  }

  let body: ArrayBuffer | undefined;
  if (request.method !== "GET") {
    body = await request.arrayBuffer();
  }

  try {
    const response = await fetch(target, {
      method: request.method,
      headers,
      body,
      cache: "no-store",
      signal: AbortSignal.timeout(10_000),
    });

    const responseBody = await response.arrayBuffer();
    return new NextResponse(responseBody, {
      status: response.status,
      headers: {
        "Content-Type":
          response.headers.get("content-type") ?? "application/json",
        "Cache-Control": "no-store",
      },
    });
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "gateway_unavailable",
          message: "Sentinel gateway is unavailable",
        },
      },
      { status: 503 },
    );
  }
}

export const GET = proxy;
export const POST = proxy;
export const PATCH = proxy;
