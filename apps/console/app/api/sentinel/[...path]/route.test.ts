import assert from "node:assert/strict";
import { test } from "node:test";
import { NextRequest } from "next/server";
import { GET } from "./route";

test("proxy forwards explicit credentials without accepting client role claims", async () => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async (_input, options) => {
      const headers = new Headers(options?.headers);
      assert.equal(headers.get("authorization"), "Bearer synthetic-test-credential");
      assert.equal(headers.get("x-sentinel-role"), null);
      return Response.json({ subject: "verified-analyst", role: "analyst" });
    };
    const request = new NextRequest("http://localhost/api/sentinel/api/v1/session", {
      headers: { Authorization: "Bearer synthetic-test-credential", "X-Sentinel-Role": "administrator" },
    });
    const response = await GET(request, { params: Promise.resolve({ path: ["api", "v1", "session"] }) });
    assert.equal(response.status, 200);
    assert.equal(response.headers.get("cache-control"), "no-store");
  } finally { globalThis.fetch = original; }
});

test("proxy does not supply an ambient credential to unauthenticated browsers", async () => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async (_input, options) => {
      assert.equal(new Headers(options?.headers).get("authorization"), null);
      return Response.json({ error: { code: "unauthenticated", message: "valid bearer credential required" } }, { status: 401 });
    };
    const response = await GET(new NextRequest("http://localhost/api/sentinel/api/v1/events"), {
      params: Promise.resolve({ path: ["api", "v1", "events"] }),
    });
    assert.equal(response.status, 401);
  } finally { globalThis.fetch = original; }
});
