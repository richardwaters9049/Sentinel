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

test("proxy forwards only session cookies and CSRF origin, preserving HttpOnly cookie response", async () => {
  const original=globalThis.fetch;
  try{
    globalThis.fetch=async(_input,options)=>{
      const headers=new Headers(options?.headers);
      assert.equal(headers.get("cookie"),"sentinel_session="+"a".repeat(43));
      assert.equal(headers.get("origin"),"http://localhost");assert.equal(headers.get("x-sentinel-csrf"),"b".repeat(64));
      assert.equal(options?.redirect,"error");
      return Response.json({signed_out:true},{headers:{"Set-Cookie":"sentinel_session=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0"}});
    };
    const {POST}=await import("./route");
    const request=new NextRequest("http://localhost/api/sentinel/api/v1/auth/logout",{method:"POST",headers:{cookie:"unrelated=secret; sentinel_session="+"a".repeat(43),Origin:"http://localhost","X-Sentinel-CSRF":"b".repeat(64)}});
    const response=await POST(request,{params:Promise.resolve({path:["api","v1","auth","logout"]})});assert.match(response.headers.get("set-cookie")??"",/HttpOnly/);
  }finally{globalThis.fetch=original;}
});
test("proxy rejects path traversal, invalid encodings and oversized login bodies before forwarding",async()=>{
 const original=globalThis.fetch;
 try{
  globalThis.fetch=async()=>{throw new Error("must not forward");};
  for(const segment of ["..","%2F","%5C","%invalid"]){const response=await GET(new NextRequest("http://localhost/api/sentinel/api/v1/events"),{params:Promise.resolve({path:["api",segment]})});assert.equal(response.status,400);}
  const {POST}=await import("./route");
  const response=await POST(new NextRequest("http://localhost/api/sentinel/api/v1/auth/login",{method:"POST",body:"x".repeat(2049)}),{params:Promise.resolve({path:["api","v1","auth","login"]})});assert.equal(response.status,413);
 }finally{globalThis.fetch=original;}
});
