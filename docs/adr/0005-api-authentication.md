# ADR 0005 — Verified API principals before interactive sign-in

- Status: Accepted for the simulated lab
- Date: 2026-10-08

## Context

Existing analyst and administrative mutations accept `X-Sentinel-Actor` as an audit
label. That caller-controlled label supplies no identity verification or authorisation.
Phase Eight needs an enforceable gateway boundary before an interactive identity flow.

## Decision

Introduce operator-provisioned opaque API credentials for the lab. Generate each token
from 32 random bytes encoded as unpadded base64url. The gateway stores only SHA-256
hashes in an operator-controlled private manifest, alongside subject, role and absolute
expiry. The manifest is bounded to 64 KiB and 128 entries; credentials must expire within
24 hours of startup. Credentials are verified on every request and expiry is enforced
at request time. No client-provided role or actor claim grants permission.

Every API route declares permitted roles. Collector credentials can ingest telemetry;
analysts can read evidence and perform investigation/hunting workflows; administrators
can additionally change detection, intelligence-source and behavioural governance
controls. Administrator credentials do not grant collector ingestion rights. A route
without an explicit policy is denied in required mode.

Authenticated identity comes from server request context. Domain audit records use
that identity. Structured access-decision logs record denied requests and mutation
outcomes using method, registered route, subject, role and status, without credentials,
query strings or request payloads.

Only GET health/readiness probes bypass authentication. Development compatibility is
limited to direct loopback clients and `SENTINEL_ENV=development`, with a startup
warning. Outside development, authentication defaults to required and startup fails
if configuration is absent or invalid. Gateway and console default binds are loopback.

## Consequences

This standard-library implementation adds no dependency or database migration. Existing
synthetic smoke workflows continue in local development mode. The console proxy can
forward an explicitly supplied bearer credential, but does not inject a shared server
credential or trust a role header.

These are lab API credentials, not a completed user identity system. Interactive console
sign-in, secure session cookies/CSRF controls, central identity integration and online
revocation are subsequent work. Manifest replacement requires a gateway restart; absolute
expiry limits the remaining exposure. TLS termination is required before using bearer
credentials across non-local networks. Host operators and the manifest remain trusted.
Local proxies can bridge remote traffic into loopback development mode, so development
mode must not be deployed or exposed through a tunnel/proxy. Collector authentication
identifies the API caller; signed event-source provenance remains separate future work.
