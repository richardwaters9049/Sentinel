import assert from "node:assert/strict";
import { test } from "node:test";

import { getBehaviourMonitor } from "./client";

test("monitor client encodes entity namespace and forwards cancellation", async () => {
  const original = globalThis.fetch;
  const controller = new AbortController();
  try {
    globalThis.fetch = async (input, options) => {
      const url = new URL(String(input), "http://localhost");
      assert.equal(url.pathname, "/api/sentinel/api/v1/behaviour/monitor");
      assert.equal(url.searchParams.get("entity_type"), "identity");
      assert.equal(url.searchParams.get("entity_id"), "svc+worker&ops");
      assert.equal(options?.signal, controller.signal);
      assert.equal(options?.cache, "no-store");
      return Response.json({ model_version: "v1" });
    };
    const result = await getBehaviourMonitor(controller.signal, {
      entity_type: "identity", entity_id: "svc+worker&ops",
    });
    assert.equal(result.model_version, "v1");
  } finally {
    globalThis.fetch = original;
  }
});

test("monitor client reports dependency errors independently", async () => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async (input) => {
      assert.equal(new URL(String(input), "http://localhost").search, "");
      return Response.json({ error: { code: "query_failed", message: "Monitoring unavailable" } }, { status: 503 });
    };
    await assert.rejects(getBehaviourMonitor(), /Monitoring unavailable/);
  } finally {
    globalThis.fetch = original;
  }
});
