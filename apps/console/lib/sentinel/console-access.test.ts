import assert from "node:assert/strict";
import { test } from "node:test";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { AdministrationNotice, ConsoleAccessProvider, useCanAdminister, useConsoleSession } from "../../components/sentinel/console-access";
import type { ConsoleSession } from "./session";

function Probe() {
  const canAdminister = useCanAdminister();
  const session = useConsoleSession();
  return createElement("section", null,
    createElement("span", null, session?.subject ?? "No session"),
    createElement("button", { disabled: !canAdminister }, "Change configuration"),
    createElement(AdministrationNotice, { id: "access", action: "change configuration" }));
}

function session(role: ConsoleSession["role"], expired = false): ConsoleSession {
  return { subject: "verified-lab-user", role, expires_at: new Date(Date.now() + (expired ? -1000 : 3600_000)).toISOString(), csrf_token: "a".repeat(64) };
}

test("console access denies missing context and missing, expired or analyst sessions", () => {
  const missing = renderToStaticMarkup(createElement(Probe));
  assert.match(missing, /disabled=""/);
  assert.match(missing, /Administrator access is required/);
  for (const principal of [null, session("analyst"), session("administrator", true), { ...session("administrator"), expires_at: "invalid" }]) {
    const html = renderToStaticMarkup(createElement(ConsoleAccessProvider, { session: principal }, createElement(Probe)));
    assert.match(html, /disabled=""/);
    assert.match(html, /id="access"/);
  }
});

test("verified administrator can change configuration; compatibility must be explicit", () => {
  for (const props of [{ session: session("administrator") }, { session: null, development: true }]) {
    const html = renderToStaticMarkup(createElement(ConsoleAccessProvider, props, createElement(Probe)));
    assert.doesNotMatch(html, /disabled=""/);
    assert.doesNotMatch(html, /Administrator access is required/);
  }
  const analyst = renderToStaticMarkup(createElement(ConsoleAccessProvider, { session: session("analyst"), development: true }, createElement(Probe)));
  assert.match(analyst, /verified-lab-user/);
  assert.match(analyst, /disabled=""/);
});
