#!/usr/bin/env python3
"""Exercise real console cookies, CSRF, sessions and analyst workflows on isolated local ports."""
import argparse
from datetime import datetime, timedelta, timezone
import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import Request, build_opener, HTTPCookieProcessor, urlopen

ROOT = Path(__file__).resolve().parents[1]
CONSOLE = "http://127.0.0.1:13005"
GATEWAY = "http://127.0.0.1:18099"
BROKER = "sentinel-phase8-session-smoke"


def run(*args: str, cwd: Path = ROOT) -> None:
    subprocess.run(args, cwd=cwd, check=True, stdout=subprocess.DEVNULL)


def wait_ready(url: str, process: subprocess.Popen) -> None:
    for _ in range(60):
        if process.poll() is not None:
            raise RuntimeError("isolated service stopped before readiness")
        try:
            with urlopen(url, timeout=3) as response:
                if response.status == 200:
                    return
        except (HTTPError, URLError, TimeoutError):
            pass
        time.sleep(0.5)
    raise RuntimeError("isolated service did not become ready")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--browser-fixture", action="store_true", help="enable a deterministic synthetic analyst credential for browser QA")
    parser.add_argument("--keep-running", action="store_true", help="keep isolated services for browser QA until interrupted")
    args = parser.parse_args()
    run("docker", "compose", "up", "-d", "postgres")
    with tempfile.TemporaryDirectory(prefix="sentinel-console-session-") as directory:
        private = Path(directory)
        binary = private / "gateway"
        run("go", "build", "-o", str(binary), "./cmd/gateway", cwd=ROOT / "services/gateway")
        tokens = {role: secrets.token_urlsafe(32) for role in ("analyst", "administrator", "collector")}
        if args.browser_fixture:
            tokens["analyst"] = "A" * 43  # Deliberately synthetic, isolated, temporary QA credential.
            tokens["administrator"] = "B" * 42 + "A"
        prefix = "session-smoke-" + secrets.token_hex(8)
        expires = (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat()
        manifest = [{"subject": f"{prefix}-{role}", "role": role,
                     "token_sha256": hashlib.sha256(token.encode()).hexdigest(), "expires_at": expires}
                    for role, token in tokens.items()]
        file = private / "credentials.json"
        with os.fdopen(os.open(file, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), "w") as output:
            json.dump(manifest, output)
        env = {**os.environ, "SENTINEL_ENV": "test", "SENTINEL_AUTH_MODE": "required",
               "SENTINEL_AUTH_CREDENTIALS_FILE": str(file), "SENTINEL_CONSOLE_ORIGIN": CONSOLE,
               "SENTINEL_HTTP_ADDR": "127.0.0.1:18099", "SENTINEL_BEHAVIOUR_ENABLED": "false",
               "DATABASE_URL": "postgres://sentinel:sentinel_dev_only@127.0.0.1:55432/sentinel?sslmode=disable",
               "NATS_URL": "nats://127.0.0.1:14329"}
        gateway = console = None
        broker_started = False
        jar = http.cookiejar.CookieJar()
        browser = build_opener(HTTPCookieProcessor(jar))

        def request(path: str, method: str = "GET", payload: object = None, csrf: str = "", origin: str = CONSOLE, opener=browser):
            headers = {"Content-Type": "application/json", "X-Sentinel-Actor": "spoofed-admin"}
            if method != "GET" and origin:
                headers["Origin"] = origin
            if csrf:
                headers["X-Sentinel-CSRF"] = csrf
            req = Request(CONSOLE + "/api/sentinel/api/v1" + path, method=method, headers=headers,
                          data=json.dumps(payload).encode() if payload is not None else None)
            try:
                with opener.open(req, timeout=10) as response:
                    return response.status, json.load(response), response.headers
            except HTTPError as error:
                return error.code, json.load(error), error.headers

        def expect(code: int, path: str, **kwargs):
            actual, body, headers = request(path, **kwargs)
            if actual != code:
                raise AssertionError(f"{path}: expected {code}, got {actual}")
            return body, headers

        def login(role="analyst"):
            body, headers = expect(201, "/auth/login", method="POST", payload={"token": tokens[role]})
            cookie = headers.get("Set-Cookie", "")
            assert "HttpOnly" in cookie and "SameSite=Strict" in cookie and "Path=/" in cookie
            assert body["subject"] == prefix + "-" + role and body["role"] == role
            assert tokens[role] not in json.dumps(body)
            return body

        try:
            run("docker", "run", "-d", "--rm", "--name", BROKER, "-p", "127.0.0.1:14329:4222", "nats:2.10-alpine", "-js")
            broker_started = True
            with (private / "gateway.log").open("w") as log, (private / "console.log").open("w") as console_log:
                gateway = subprocess.Popen([str(binary)], env=env, stdout=log, stderr=log)
                wait_ready(GATEWAY + "/ready", gateway)
                console_env = {**os.environ, "SENTINEL_GATEWAY_URL": GATEWAY, "SENTINEL_CONSOLE_AUTH_MODE": "required"}
                console = subprocess.Popen(["bun", "run", "start", "--", "-p", "13005"], cwd=ROOT / "apps/console", env=console_env, stdout=console_log, stderr=console_log)
                wait_ready(CONSOLE + "/sign-in", console)
                expect(401, "/events")
                expect(403, "/auth/login", method="POST", payload={"token": tokens["analyst"]}, origin="https://attacker.invalid")
                expect(401, "/auth/login", method="POST", payload={"token": "invalid"})
                expect(403, "/auth/login", method="POST", payload={"token": tokens["collector"]})
                session = login()
                old_cookie = next(cookie.value for cookie in jar if cookie.name == "sentinel_session")
                expect(200, "/events")
                expect(403, "/investigations", method="POST", payload={"title": "Synthetic session investigation"})
                expect(403, "/investigations", method="POST", csrf=session["csrf_token"], origin="https://attacker.invalid", payload={"title": "Synthetic session investigation"})
                investigation, _ = expect(201, "/investigations", method="POST", csrf=session["csrf_token"], payload={"title": "Synthetic console session investigation", "priority": "low"})
                assert investigation["created_by"] == prefix + "-analyst"
                expect(403, "/behaviour/settings", method="PATCH", csrf=session["csrf_token"], payload={"anomaly_threshold": 71})
                expect(403, "/detections/DET-AUTH-001", method="PATCH", csrf=session["csrf_token"], payload={"enabled": False})
                expect(403, "/intelligence/sources/intel-local-review", method="PATCH", csrf=session["csrf_token"], payload={"enabled": False})
                # Seed one uniquely labelled synthetic event through the collector boundary.
                event_id = "evt_session_workflow_" + secrets.token_hex(12)
                event = {"event_id": event_id, "timestamp": datetime.now(timezone.utc).isoformat(),
                         "source": {"type": "network", "collector": prefix + "-collector"},
                         "event": {"category": "network", "action": "connection", "outcome": "success"},
                         "labels": {"environment": "lab", "scenario": prefix}}
                req = Request(GATEWAY + "/api/v1/telemetry", method="POST", data=json.dumps(event).encode(),
                              headers={"Content-Type": "application/json", "Authorization": "Bearer " + tokens["collector"]})
                with urlopen(req, timeout=10) as response:
                    assert response.status == 202
                hunt, _ = expect(201, "/hunts", method="POST", csrf=session["csrf_token"], payload={
                    "name": prefix + " authenticated hunt", "hypothesis": "Review the isolated synthetic session event",
                    "query": {"labels": {"scenario": prefix}, "limit": 5}})
                assert hunt["created_by"] == prefix + "-analyst"
                for _ in range(30):
                    result, _ = expect(200, "/hunts/" + hunt["id"] + "/run", method="POST", csrf=session["csrf_token"], payload={"override": {}})
                    if result["result_count"] == 1:
                        break
                    time.sleep(0.2)
                else:
                    raise AssertionError("authenticated hunt did not find its synthetic event")
                assert result["events"][0]["id"] == event_id
                run_history, _ = expect(200, "/hunts/" + hunt["id"] + "/runs")
                assert any(item["actor_id"] == prefix + "-analyst" and item["id"] == result["run_id"] for item in run_history["runs"])
                case = "/investigations/" + investigation["id"]
                expect(200, case + "/hunt-runs/" + str(result["run_id"]), method="POST", csrf=session["csrf_token"])
                note, _ = expect(201, case + "/notes", method="POST", csrf=session["csrf_token"], payload={"body": "Reviewed isolated synthetic hunt evidence."})
                assert note["notes"][0]["actor_id"] == prefix + "-analyst"
                expect(200, case + "/status", method="PATCH", csrf=session["csrf_token"], payload={"status": "investigating"})
                detail, _ = expect(200, case)
                assert detail["status"] == "investigating" and detail["events"][0]["id"] == event_id
                assert {"event", "note", "audit"}.issubset({item["type"] for item in detail["timeline"]})
                assert all(item["actor_id"] == prefix + "-analyst" for item in detail["audit"])
                with browser.open(CONSOLE + "/investigations", timeout=10) as page:
                    assert (prefix + "-analyst") in page.read().decode()
                # Restarting a gateway must preserve active database sessions and re-check its registry.
                gateway.terminate(); gateway.wait(timeout=15)
                gateway = subprocess.Popen([str(binary)], env=env, stdout=log, stderr=log)
                wait_ready(GATEWAY + "/ready", gateway)
                expect(200, "/session")
                session = login()
                replay = build_opener()
                req = Request(CONSOLE + "/api/sentinel/api/v1/session", headers={"Cookie": "sentinel_session=" + old_cookie})
                try:
                    replay.open(req, timeout=5)
                    raise AssertionError("rotated cookie replay succeeded")
                except HTTPError as error:
                    assert error.code == 401
                expect(200, "/auth/logout", method="POST", csrf=session["csrf_token"])
                expect(401, "/session")
                session = login()
                run("docker", "compose", "exec", "-T", "postgres", "psql", "-U", "sentinel", "-d", "sentinel", "-c",
                    f"UPDATE console_sessions SET last_seen_at=NOW()-INTERVAL '31 minutes', created_at=NOW()-INTERVAL '1 hour' WHERE subject='{prefix}-analyst'")
                expect(401, "/session")
                # Privileged cookies carry the same role limits as bearer requests.
                admin = login("administrator")
                expect(400, "/behaviour/settings", method="PATCH", csrf=admin["csrf_token"], payload={"anomaly_threshold": 0})
                # Exercise a valid privileged mutation without changing the shared lab threshold.
                settings, _ = expect(200, "/behaviour/settings")
                settings, _ = expect(200, "/behaviour/settings", method="PATCH", csrf=admin["csrf_token"], payload={"anomaly_threshold": settings["anomaly_threshold"]})
                assert settings["updated_by"] == prefix + "-administrator"
                expect(200, "/auth/logout", method="POST", csrf=admin["csrf_token"])
                assert not any(token in (private / "gateway.log").read_text() for token in tokens.values())
                print("Console session smoke passed: sign-in, role checks, authenticated hunt/evidence/note/timeline workflows, verified audit actors, administrator mutation, CSRF, gateway restart, rotation, logout replay and idle expiry.", flush=True)
                if args.keep_running:
                    print("Browser QA fixture: " + CONSOLE + "/sign-in", flush=True)
                    while True:
                        time.sleep(1)
        finally:
            for process in (console, gateway):
                if process is not None and process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=15)
                    except subprocess.TimeoutExpired:
                        process.kill(); process.wait(timeout=5)
            if broker_started:
                run("docker", "rm", "-f", BROKER)
            run("docker", "compose", "exec", "-T", "postgres", "psql", "-U", "sentinel", "-d", "sentinel", "-c",
                f"DELETE FROM console_sessions WHERE subject LIKE '{prefix}-%'")


def interrupted(_signal: int, _frame: object) -> None:
    raise KeyboardInterrupt


if __name__ == "__main__":
    signal.signal(signal.SIGTERM, interrupted)
    try:
        main()
    except KeyboardInterrupt:
        raise SystemExit(130)
