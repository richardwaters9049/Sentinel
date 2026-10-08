import assert from "node:assert/strict";
import { test } from "node:test";
import { parseConsoleSession, sentinelFetch } from "./session";
const session = { subject: "lab-analyst", role: "analyst", expires_at: new Date(Date.now()+3600_000).toISOString(), csrf_token: "a".repeat(64) };
test("session contract rejects collectors, malformed context and expired sessions", () => {
  assert.ok(parseConsoleSession(session));
  assert.deepEqual(parseConsoleSession({ ...session, untrusted_secret: "must not be projected" }),session);
  for (const value of [null, {}, {...session,role:"collector"},{...session,subject:"spoof\nadmin"}, {...session,expires_at:new Date(Date.now()+9*3600_000).toISOString()}, {...session,expires_at:"invalid"},{...session,expires_at:new Date(0).toISOString()}, {...session,csrf_token:"invalid"}]) assert.equal(parseConsoleSession(value),null);
});
test("mutations obtain current server CSRF context and preserve caller options", async () => {
  const original=globalThis.fetch; const controller=new AbortController(); let calls=0;
  try {
    globalThis.fetch=async (input,init)=>{
      calls++;
      if(calls===1){assert.equal(input,"/api/sentinel/api/v1/session");assert.equal(init?.signal,controller.signal);return Response.json(session);}
      assert.equal(input,"/api/sentinel/api/v1/hunts");assert.equal(new Headers(init?.headers).get("x-sentinel-csrf"),session.csrf_token);
      assert.equal(new Headers(init?.headers).get("authorization"),null);assert.equal(init?.body,"{}");return Response.json({id:"hunt"},{status:201});
    };
    assert.equal((await sentinelFetch("/api/sentinel/api/v1/hunts",{method:"POST",body:"{}",signal:controller.signal})).status,201);assert.equal(calls,2);
  } finally {globalThis.fetch=original;}
});
test("expired session or verification outage prevents mutation forwarding",async()=>{
 const original=globalThis.fetch;
 try{for(const status of [401,503]){let calls=0;globalThis.fetch=async()=>{calls++;return Response.json({error:{code:"unavailable"}},{status});};assert.equal((await sentinelFetch("/api/sentinel/api/v1/hunts",{method:"POST"})).status,status);assert.equal(calls,1);}}
 finally{globalThis.fetch=original;}
});
