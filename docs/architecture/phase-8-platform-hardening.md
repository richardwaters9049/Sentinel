# Phase Eight — Platform hardening

## Status

Complete within the simulated lab scope on 9 October 2026. API credentials, RBAC,
interactive console sessions, role-aware controls, signed collector provenance, durable
replay protection, bounded rate limits and access-audit retention are implemented.
The final regression and security gates passed; see the
[completion evidence](phase-8-9-completion.md).

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
platform-wide, with no tenancy or per-investigation ownership restrictions. GET `/health` and `/ready` are public; POST `/api/v1/auth/login` is the
origin-checked credential exchange. Unknown routes, unregistered methods and routes lacking
permissions are denied in required mode. Invalid credentials return 401 with a Bearer
challenge; insufficient permission returns 403. Protected responses use `no-store`.
`GET /api/v1/session` returns verified subject, role and expiry, never a token.

Audit actors come from verified request context. Caller-supplied `X-Sentinel-Actor`
and `X-Sentinel-Role` cannot spoof that context. Existing domain audit trails retain
verified subjects. Denied access and mutation outcomes are structured log events; these
logs are now accompanied by the durable access ledger described below. Behavioural settings
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

The console proxy forwards explicit bearer credentials for API clients and recognised
session cookies for interactive use. It never adds a shared server credential. Console
production mode defaults to `required`; local `next dev` defaults to development
compatibility. `SENTINEL_CONSOLE_AUTH_MODE=required` enables sign-in during development.
The separate development mode is for local simulation only.

## Interactive sign-in and sessions

The canonical console origin is configured in the gateway using
`SENTINEL_CONSOLE_ORIGIN` (default `http://127.0.0.1:3000`). It must be an exact origin;
non-loopback HTTP origins are rejected. Open that origin to sign in: localhost and
127.0.0.1 are different origins. HTTPS origins issue `__Host-sentinel_session` with
Secure; loopback HTTP uses `sentinel_session`. Both are HttpOnly, host-only,
SameSite=Strict, Path=/ and expire at the session's absolute deadline.

`POST /api/v1/auth/login` accepts exactly `{ "token": "<lab credential>" }`, requires
the configured Origin, rejects collectors and rotates a previous browser session.
`POST /api/v1/auth/logout` requires the current cookie, configured Origin and
`X-Sentinel-CSRF`, then revokes the record and expires the cookie. `GET /api/v1/session`
returns verified identity, role, absolute expiry and CSRF context for cookie sessions;
it does not return the credential or session identifier in JSON. Bearer-only principal
responses retain their existing contract.

Migration 0016 stores session/credential hashes, principal, creation/activity and expiry
in PostgreSQL. Sessions survive gateway restarts with the same registry, expire after
at most eight hours (capped by credential expiry), and have a 30-minute API inactivity
timeout. API polling counts as activity. Absolute expiry never slides. Removing or
changing the underlying credential on restart denies its existing sessions. Atomic
lookup/touch and transactional creation/rotation enforce eight sessions per principal
and 10,000 globally. Expired session state is removed during creation; analytical
evidence and domain audits are retained. Logout is local-session revocation, not
revocation of the underlying API credential or every session for that identity.

Protected console pages share a server layout that verifies cookies through the gateway.
Outages fail closed. The client obtains fresh CSRF context before mutations, including
when another tab has rotated the cookie, and redirects on expired authentication. It
shows the verified identity/role and sign-out in navigation. Credential inputs are cleared
after submission; no bearer credential is placed in local storage, React state or bundles.
CSRF tokens are supporting request context, not authentication credentials. Login body
limits are 2 KiB; the proxy caps other request bodies at 1 MiB and refuses redirects or
path traversal. Read-only session checks and API responses use no-store. The previous
third-party development capture script was removed from the app shell.

From `/Users/richy/Documents/Github/Sentinel`, create new eight-hour local credentials:

```bash
make provision-lab-auth
```

This creates a new 0700 directory beneath `~/.local/share/sentinel/auth`, with a private
0600 hash manifest and separate 0600 token file. It prints only paths and expiry, never
credentials, and does not overwrite older credentials. Set the gateway's required mode
and manifest path, set the console to required mode, and open the configured origin.
Use the analyst token from the private file to sign in. Credentials are for this fictional
lab only; retire private files after use. The generator does not restart services or
revoke older registries by itself.

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
passed for the first slice, alongside the isolated API smoke and its console tests.
The complete Phase Seven gate was also rerun successfully for that slice,
including 22 Python tests and the inherited Phase One–Six regression baseline.

### Session-slice verification

From `/Users/richy/Documents/Github/Sentinel`:

```bash
make sessions-phase8
```

This builds the console and tests a real production console in required mode against an
isolated gateway/broker on ports 13005, 18099 and 14329. It exercises login failures,
collector rejection, cookie issuance, CSRF/origin rejection, verified investigation
creation, role limits, gateway restart, rotation/replay, logout and idle expiry. It removes
only its own session rows, processes, broker and private test files; synthetic investigation
evidence is retained. Optional browser-fixture mode is for isolated QA only and never
provisions its deterministic test credential into the normal gateway.

Go tests and PostgreSQL integration cover session limits, expiry boundaries, migration
reruns, failed persistence, role/credential changes, rotation, replay, cookie flags and
origin/CSRF validation. The eleven console tests cover session contracts, CSRF acquisition,
mutation blocking on expired/unavailable sessions, cookie proxying, path validation and
request bounds. Browser QA verified analyst sign-in, verified navigation identity/role,
logout, logged-out page redirects and keyboard invalid-credential feedback.

The final session gate passed on 8 October 2026: Go vet and race tests including
PostgreSQL integration, all eleven console tests, lint, TypeScript checking and the
production build; both isolated Phase Eight smokes; and the full Phase Seven final
regression with the inherited Phase One–Six baseline. The local console and gateway
were restarted in required mode with privately provisioned eight-hour lab credentials.

## Role-aware console slice

Detection enable/disable switches, intelligence-source switches and the anomaly-threshold
slider/apply action now use the server-verified session role. Analysts retain read access
and investigation/hunt actions; privileged controls are disabled with accessible explanations.
Administrators can edit configuration. Missing access context denies privileged UI actions;
legacy local compatibility is an explicit server-layout choice. The session expiry boundary
hides the workspace, and current CSRF/session context is obtained before each mutation.

UI capability checks support usability, not authorisation. The browser remains untrusted;
the gateway continues to check every route and ignores client role/actor claims. No new
trust boundary, role permission or database schema is introduced. An already rendered
page can show a stale role after a session is replaced in another tab; current gateway
permissions still apply and refreshing the page obtains the new verified role.

Verification on 9 October 2026 includes thirteen console tests, lint, TypeScript checking
and a production build. The expanded required-mode session smoke seeds a uniquely labelled
collector event, creates/runs a saved hunt, attaches its exact results to an investigation,
adds a note, changes case status and checks the timeline and persisted audit actors. It
also checks analyst denials for all three administrative actions and performs a valid
administrator threshold update using its existing value. Synthetic hunts, events, cases
and audits are retained; only test session records and private/runtime fixtures are cleaned.

Isolated browser QA verified disabled analyst controls in all three workspaces, administrator
controls including keyboard threshold editing, analyst investigation-note persistence,
verified navigation roles and sign-out. Browser QA leaves shared detection/source/threshold
values unchanged; it adds a synthetic investigation note.

## Final hardening slice

Required mode now enforces token buckets before database work: peer requests allow
300/second with a burst of 600; sign-in allows ten/minute with a burst of 20. Verified
analyst/administrator principals allow 30/second with a burst of 60; collectors allow
100/second with a burst of 200. Buckets are process-local, capped at 4096 entries and
reclaim entries idle for five minutes. Capacity exhaustion denies new keys. No forwarded
IP header grants a new peer identity. A shared console proxy intentionally shares its
peer budget; per-principal budgets remain separate. HTTP timeouts/header/body bounds
limit resource use. A 429 response includes Retry-After through the console proxy.

Migration 0017 stores access receipt, verified allowance, denials and completions in
PostgreSQL. Protected dispatch fails closed when receipt/allowance cannot be recorded.
Records omit credentials, body/query data and IP addresses. Completion persistence
failure is counted/logged and cannot roll back a finished domain transaction. Peer-rate
rejections avoid database writes and remain visible in metrics and structured request
logs. Read the ledger through administrator-only `GET /api/v1/security/audit`, using
`limit` 1–200 and an exclusive `before` record-ID cursor. Invalid/duplicate query keys
are rejected. Explicit administrator retention removes at most 1,000 access records
older than 90 days; domain audits/evidence are preserved.

Collector credentials additionally require a canonical unpadded-base64 Ed25519 public
key in `collector_public_key` and a unique collector subject. Ingestion signs exact
bytes, method, path, timestamp and nonce. Requests outside a 30-second allowance or
with changed signatures, duplicate headers/query strings or replayed nonces fail.
Nonce reservations are transactional and durable across restarts. Producer claims in
`sentinel.collector.*` labels are replaced with verified subject, key digest and body
digest. Up to 47 other labels fit the existing 50-label bound. Keep source metadata as
reported context; a signed source can still provide false events if compromised.

Private lab provisioning now creates the collector signing seed, and the simulator
accepts `-credential-file` for authenticated synthetic scenarios. No signing secret is
placed in source, the browser bundle, logs or the runtime principal registry. See
[ADR 0007](../adr/0007-collector-signatures-and-access-ledger.md) for canonical signing
bytes, retry behaviour, retention and trust assumptions.

Security CI runs reachable Go vulnerability checks, Trivy runtime dependency/image,
secret and deployment-policy scans, CycloneDX SBOM generation, and CodeQL for Go,
TypeScript/JavaScript and Python. Security actions and deployment bases are pinned.
Reachable Go standard-library issues were addressed by upgrading to Go 1.27.2;
Starlette/FastAPI were upgraded and runtime images reduced. The shadcn generator is a
build-time dependency, and unused npm tooling is removed from the standalone console
runtime. An unfixed braces advisory remains in development tooling; runtime image and
runtime-dependency gates do not suppress deployed HIGH/CRITICAL findings. GitHub code
scanning execution/branch-protection enforcement depends on repository settings;
local checks do not claim a completed hosted CodeQL run.

The final gate is `make final-phase8`; production-shaped deployment verification and
security scanning are described in [Phase Nine](phase-9-deployment.md).

[ADR 0006](../adr/0006-console-sessions.md) describes interactive session choices.
[ADR 0005](../adr/0005-api-authentication.md) records the trust assumptions and trade-offs.
No real scanning, controller writes or disruptive remediation is introduced.
