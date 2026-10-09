#!/usr/bin/env python3
"""Create private deployment files; stdout contains paths, never credentials."""
from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
from pathlib import Path
import secrets
from lab_signing import keypair


def main() -> None:
    directory = Path.home() / '.local/share/sentinel/deploy' / secrets.token_hex(12)
    directory.mkdir(parents=True, mode=0o700)
    expires = (datetime.now(timezone.utc) + timedelta(hours=8)).isoformat()
    tokens = {role: secrets.token_urlsafe(32) for role in ('analyst', 'administrator', 'collector')}
    seed, public = keypair()
    manifest = [{'subject': 'deployment-' + role, 'role': role, 'expires_at': expires,
                 'token_sha256': hashlib.sha256(token.encode()).hexdigest()} for role, token in tokens.items()]
    manifest[2]['collector_public_key'] = public
    password = secrets.token_urlsafe(32)
    files = {'credentials.json': json.dumps(manifest), 'postgres-password': password,
             'database-url': f'postgres://sentinel:{password}@postgres:5432/sentinel?sslmode=disable',
             'metrics-token': tokens['administrator'], 'grafana-password': secrets.token_urlsafe(32),
             'lab-tokens.json': json.dumps({'tokens': tokens, 'collector_private_seed': seed, 'expires_at': expires})}
    for name, content in files.items():
        with os.fdopen(os.open(directory / name, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'w') as output:
            output.write(content + '\n')
    print(json.dumps({'secrets_dir': str(directory), 'expires_at': expires}))


if __name__ == '__main__':
    main()
