#!/usr/bin/env python3
"""Provision expiring local lab credentials into new private files, never stdout."""
from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
from pathlib import Path
import secrets


def main() -> None:
    base = Path.home() / ".local" / "share" / "sentinel" / "auth"
    base.mkdir(parents=True, exist_ok=True, mode=0o700)
    directory = base / secrets.token_hex(12)
    directory.mkdir(mode=0o700)
    expires = (datetime.now(timezone.utc) + timedelta(hours=8)).isoformat()
    tokens = {role: secrets.token_urlsafe(32) for role in ("analyst", "administrator", "collector")}
    manifest = [{"subject": "lab-" + role, "role": role,
                 "token_sha256": hashlib.sha256(token.encode()).hexdigest(), "expires_at": expires}
                for role, token in tokens.items()]
    for name, value in (("credentials.json", manifest), ("lab-tokens.json", {"expires_at": expires, "tokens": tokens})):
        descriptor = os.open(directory / name, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, "w") as output:
            json.dump(value, output, indent=2)
            output.write("\n")
    print(json.dumps({"manifest_path": str(directory / "credentials.json"), "token_file_path": str(directory / "lab-tokens.json"), "expires_at": expires}))


if __name__ == "__main__":
    main()
