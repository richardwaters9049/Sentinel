import assert from "node:assert/strict";
import { test } from "node:test";

import { getBehaviourCatalogue } from "./client";

test("catalogue client uses model-wide endpoint and cancellation", async () => {
  const original = globalThis.fetch;
  const controller = new AbortController();
  try {
    globalThis.fetch = async (input, options) => {
      assert.equal(String(input), "/api/sentinel/api/v1/behaviour/catalogue");
      assert.equal(options?.signal, controller.signal);
      assert.equal(options?.cache, "no-store");
      return Response.json({ catalogue_version: "v1" });
    };
    assert.equal((await getBehaviourCatalogue(controller.signal)).catalogue_version, "v1");
  } finally {
    globalThis.fetch = original;
  }
});

test("catalogue client exposes sanitised upstream failure", async () => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async () => Response.json({ error: { message: "Catalogue unavailable" } }, { status: 503 });
    await assert.rejects(getBehaviourCatalogue(), /Catalogue unavailable/);
  } finally {
    globalThis.fetch = original;
  }
});
