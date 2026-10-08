# Phase Eight — Platform hardening

## Status

In progress. The first slice establishes API credential authentication, explicit RBAC
and verified audit attribution. Phase Seven remains complete within its simulated scope.

## API access boundary

`SENTINEL_AUTH_MODE=required` enables authentication. It is the default whenever
`SENTINEL_ENV` is not `development`. Set `SENTINEL_AUTH_CREDENTIALS_FILE` to a private
regular JSON file (mode 0600 or stricter). Missing, oversized, malformed, unknown-field,
duplicate-digest, expired, overlong-lifetime or unsupported-role manifests fail startup.
Symlinks and files readable by group/others are rejected. The manifest contains an array:

```json
[
  {
    "subject": "lab-analyst",
    "role": "analyst",
    "token_sha256": "<64 lowercase hex characters: SHA-256 of the encoded token>",
    "expires_at": "<UTC RFC3339 timestamp within the next 24 hours>"
  }
]
```

Generate tokens using a cryptographically secure generator: 32 random bytes, unpadded
base64url encoding (43 characters). Hash the encoded token, not its decoded bytes.
Keep the token in a private secret store/file outside the repository and supply it as
`Authorization: Bearer <token>`. Never paste it into documentation, source, command
history or logs. The runtime smoke demonstrates ephemeral provisioning without printing
credentials. This file format is a lab bootstrap contract, not an identity-provider API.

| Role | Permissions |
| --- | --- |
| Collector | POST telemetry; read its authenticated principal |
| Analyst | Read analytical APIs; create/edit/run hunts; create/update investigations and findings; read its principal |
| Administrator | Analyst permissions plus detection state, intelligence-source state, anomaly threshold and evaluation-history creation |

Administrator permissions exclude telemetry ingestion. Read access is currently
platform-wide, with no tenancy or per-investigation ownership restrictions. Only GET
`/health` and `/ready` are public. Unknown routes, unregistered methods and routes lacking
permissions are denied in required mode. Invalid credentials return 401 with a Bearer
challenge; insufficient permission returns 403. Protected responses use `no-store`.
`GET /api/v1/session` returns verified subject, role and expiry, never a token.

Audit actors come from verified request context. Caller-supplied `X-Sentinel-Actor`
and `X-Sentinel-Role` cannot spoof that context. Existing domain audit trails retain
verified subjects. Denied access and mutation outcomes are structured log events; these
logs are not yet a separately durable database access-audit ledger. Behavioural settings
retain their existing latest-updater record rather than gaining a new history table.

## Development compatibility

In development, unset `SENTINEL_AUTH_MODE` retains the legacy local workflow. The explicit
`development` mode is rejected in any other environment and refuses an unexpected
credential-file setting. API requests in this mode require a loopback socket peer; forwarded
headers do not establish trust. The default gateway address is `127.0.0.1:8080`; console
dev/start commands bind to `127.0.0.1`. Health/readiness probes remain public.

Any application on the host or proxy forwarding into loopback can use this development
bypass. Do not expose it through a tunnel, remote console bind or reverse proxy. This is
compatibility for synthetic local development, not a production authentication mode.

The console proxy forwards an explicit bearer credential without adding an ambient
shared credential. Interactive console components currently send no bearer token; they
remain usable in development mode and receive API authentication errors in required mode.
Interactive sign-in and role-aware controls are the next slice. No credentials are placed
in a client bundle, browser local storage or UI state by this implementation.

## Verification

From `/Users/richy/Documents/Github/Sentinel`:

```bash
make smoke-phase8
```

This starts an isolated gateway and NATS broker, uses the local development PostgreSQL
instance, provisions one-hour ephemeral credentials in a private temporary directory,
checks role separation and persisted investigation audit identities, changes/restores the
anomaly threshold, and removes its gateway/broker/private files. It retains the generated
synthetic investigation and its audit evidence. Other development processes can remain
running because the smoke uses its own broker. The ports 18098 and 14328 must be free.

Go tests cover expired/missing/malformed credentials, private-file checks, strict manifest
validation, duplicate authorisation headers, role/actor spoofing, route permissions,
unknown-route denial and loopback enforcement. Console tests cover explicit bearer
forwarding, rejection propagation and absence of ambient credentials. Go vet/race tests,
PostgreSQL monitor integration, console lint, TypeScript checking and production build
passed for this slice, alongside the isolated runtime smoke and six console client/proxy
tests. The complete Phase Seven gate was also rerun successfully after these changes,
including 22 Python tests and the inherited Phase One–Six regression baseline.

## Remaining work

- Interactive console sign-in, bounded server sessions, logout/revocation and CSRF protection.
- Role-aware console controls and end-to-end authenticated analyst workflows.
- Rate limiting and a durable access-audit retention policy.
- Signed collector provenance and additional replay controls.
- Dependency/container/code/secret scanning, SBOM and deployment security controls.

[ADR 0005](../adr/0005-api-authentication.md) records the trust assumptions and trade-offs.
No real scanning, controller writes or disruptive remediation is introduced.
