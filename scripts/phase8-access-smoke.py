#!/usr/bin/env python3
"""Local API authentication/RBAC regression; credentials never leave a private temp directory."""
from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[1]
BASE = "http://127.0.0.1:18098"
CONTAINER = "sentinel-phase8-access-smoke"


def run(*args: str, cwd: Path = ROOT) -> None:
    subprocess.run(args, cwd=cwd, check=True, stdout=subprocess.DEVNULL)


def main() -> None:
    run("docker", "compose", "up", "-d", "postgres")
    with tempfile.TemporaryDirectory(prefix="sentinel-access-") as directory:
        private = Path(directory)
        binary = private / "gateway"
        run("go", "build", "-o", str(binary), "./cmd/gateway", cwd=ROOT / "services/gateway")
        tokens = {role: secrets.token_urlsafe(32) for role in ("analyst", "administrator", "collector")}
        expiry = (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat()
        manifest = [{"subject": f"phase8-{role}", "role": role,
                     "token_sha256": hashlib.sha256(token.encode()).hexdigest(), "expires_at": expiry}
                    for role, token in tokens.items()]
        credential_file = private / "credentials.json"
        descriptor = os.open(credential_file, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, "w") as output:
            json.dump(manifest, output)
        environment = {**os.environ, "SENTINEL_ENV": "test", "SENTINEL_AUTH_MODE": "required",
                       "SENTINEL_AUTH_CREDENTIALS_FILE": str(credential_file),
                       "SENTINEL_HTTP_ADDR": "127.0.0.1:18098", "SENTINEL_BEHAVIOUR_ENABLED": "false",
                       "DATABASE_URL": "postgres://sentinel:sentinel_dev_only@127.0.0.1:55432/sentinel?sslmode=disable",
                       "NATS_URL": "nats://127.0.0.1:14328"}
        gateway = None
        original_threshold = None
        broker_started = False

        def request(path: str, role: str = "", method: str = "GET", payload: object = None) -> tuple[int, dict]:
            headers = {"Content-Type": "application/json", "X-Sentinel-Actor": "spoofed-admin",
                       "X-Sentinel-Role": "administrator"}
            if role:
                headers["Authorization"] = "Bearer " + tokens[role]
            data = json.dumps(payload).encode() if payload is not None else None
            req = Request(BASE + path, data=data, headers=headers, method=method)
            try:
                with urlopen(req, timeout=5) as response:
                    return response.status, json.load(response)
            except HTTPError as error:
                return error.code, json.load(error)

        def expect(code: int, path: str, role: str = "", method: str = "GET", payload: object = None) -> dict:
            actual, body = request(path, role, method, payload)
            if actual != code:
                raise AssertionError(f"{method} {path}: expected {code}, got {actual}")
            return body

        try:
            run("docker", "run", "-d", "--rm", "--name", CONTAINER,
                "-p", "127.0.0.1:14328:4222", "nats:2.10-alpine", "-js")
            broker_started = True
            with (private / "gateway.log").open("w") as log:
                gateway = subprocess.Popen([str(binary)], env=environment, stdout=log, stderr=log)
                for _ in range(60):
                    if gateway.poll() is not None:
                        raise RuntimeError("isolated authentication gateway stopped before readiness")
                    try:
                        code, _ = request("/ready")
                        if code == 200:
                            break
                    except (URLError, TimeoutError):
                        pass
                    time.sleep(0.5)
                else:
                    raise RuntimeError("isolated authentication gateway did not become ready")
                expect(200, "/health")
                expect(401, "/api/v1/events")
                expect(200, "/api/v1/events", "analyst")
                expect(403, "/api/v1/events", "collector")
                expect(403, "/api/v1/telemetry", "analyst", "POST", {})
                expect(403, "/api/v1/telemetry", "administrator", "POST", {})
                expect(400, "/api/v1/telemetry", "collector", "POST", {})
                event_id = "evt_phase8_access_" + secrets.token_hex(12)
                expect(202, "/api/v1/telemetry", "collector", "POST", {
                    "event_id": event_id, "timestamp": datetime.now(timezone.utc).isoformat(),
                    "source": {"type": "network", "collector": "phase8-collector"},
                    "event": {"category": "network", "action": "connection", "outcome": "success"},
                    "labels": {"environment": "lab", "scenario": "phase8-access-control"},
                })
                for _ in range(30):
                    events = expect(200, "/api/v1/events?limit=200", "analyst")["events"]
                    if any(event["event_id"] == event_id for event in events):
                        break
                    time.sleep(0.2)
                else:
                    raise AssertionError("authenticated collector telemetry was not persisted")
                expect(403, "/api/v1/behaviour/settings", "analyst", "PATCH", {"anomaly_threshold": 71})
                expect(403, "/api/v1/detections/DET-AUTH-001", "analyst", "PATCH", {"enabled": False})
                expect(403, "/api/v1/intelligence/sources/local", "analyst", "PATCH", {"enabled": False})
                expect(403, "/api/v1/unreviewed", "administrator")
                principal = expect(200, "/api/v1/session", "analyst")
                assert principal["subject"] == "phase8-analyst" and principal["role"] == "analyst"
                original_threshold = expect(200, "/api/v1/behaviour/settings", "administrator")["anomaly_threshold"]
                settings = expect(200, "/api/v1/behaviour/settings", "administrator", "PATCH", {"anomaly_threshold": 71})
                assert settings["updated_by"] == "phase8-administrator"
                investigation = expect(201, "/api/v1/investigations", "analyst", "POST",
                                       {"title": "Phase Eight synthetic access-control regression", "priority": "low"})
                assert investigation["created_by"] == "phase8-analyst"
                detail = expect(200, "/api/v1/investigations/" + investigation["id"], "analyst")
                assert any(audit["actor_id"] == "phase8-analyst" for audit in detail["audit"])
                logs = (private / "gateway.log").read_text()
                assert '"outcome":"forbidden"' in logs and '"actor_id":"phase8-administrator"' in logs
                assert not any(token in logs for token in tokens.values())
                assert "spoofed-admin" not in logs
        finally:
            try:
                if original_threshold is not None:
                    restored = expect(200, "/api/v1/behaviour/settings", "administrator", "PATCH",
                                      {"anomaly_threshold": original_threshold})
                    assert restored["anomaly_threshold"] == original_threshold
            finally:
                if gateway is not None and gateway.poll() is None:
                    gateway.terminate()
                    try:
                        gateway.wait(timeout=15)
                    except subprocess.TimeoutExpired:
                        gateway.kill()
                        gateway.wait(timeout=5)
                if broker_started:
                    run("docker", "rm", "-f", CONTAINER)
    print("Phase Eight access smoke passed: authentication, role separation, verified audit actors, no credential logging and threshold restoration.")


def interrupted(_signal: int, _frame: object) -> None:
    raise KeyboardInterrupt


if __name__ == "__main__":
    signal.signal(signal.SIGTERM, interrupted)
    main()
