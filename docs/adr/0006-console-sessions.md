# ADR 0006 — Server-side console sessions and CSRF

- Status: Accepted for the simulated lab
- Date: 2026-10-08

## Context

API credentials establish verified principals, but an interactive browser should not
persist a bearer credential in local storage, JavaScript state or a client bundle.
Logout must revoke authority on the server, and cookie-based requests introduce CSRF.

## Decision

Exchange an existing short-lived analyst/administrator lab credential for a random
256-bit session identifier. Store only its SHA-256 digest in PostgreSQL, alongside the
credential digest, verified subject/role, creation, activity and absolute expiry. The
browser receives the identifier in a host-only HttpOnly, SameSite=Strict cookie with
Path=/. HTTPS origins use Secure and the `__Host-` cookie prefix. Plain HTTP origins
are permitted only for loopback lab development.

Sessions expire after at most eight hours or the underlying credential expiry, whichever
comes first. Thirty minutes without a valid API request causes idle expiry. Background
API polling counts as activity; absolute expiry cannot be extended. Session lookup/touch
is atomic. Each request also checks the configured credential registry and its role and
subject: removing or changing a credential on gateway restart invalidates its sessions.

Login rotates an existing browser session in the same transaction as creating the new
one. Logout deletes its session record and expires its cookie. Restarting a gateway
with the same credential registry preserves valid database sessions. Session creation
serialises its capacity check and admits at most eight sessions per subject and 10,000
globally; expired state is cleaned during creation. Session state is disposable authority,
not historical security evidence. Domain audit evidence is retained.

Login requires the exact configured Origin to prevent login CSRF. Cookie-authenticated
mutations additionally require a domain-separated SHA-256 token derived from the random
session identifier, delivered through authenticated session JSON. A malicious cross-origin
page cannot read this token; possession of it alone grants no session authority. Bearer-only
API requests remain independent; mixing bearer and session credentials is rejected.

The console proxy forwards only recognised session cookies, Origin and CSRF headers,
propagates cookie issuance/clearing, rejects traversal and bounds request bodies. Server
layouts validate sessions before rendering protected console content; every data request
still goes through gateway authorisation. Client expiry handling hides the workspace and
returns to sign-in. No third-party capture script runs on credential-entry pages.

## Consequences and residual risks

This uses existing PostgreSQL and standard libraries, with no new runtime dependency.
Login still consumes an operator-provisioned lab token; federation, MFA and account
management are not implemented. Manifest revocation requires a gateway restart. A stolen
cookie remains a bearer secret until revocation/expiry, and XSS can act with a user's
session despite HttpOnly. CSP, rate limiting and durable access-log retention remain
hardening work. TLS must protect non-local browser and service traffic. The host operator,
credential file, configured gateway and database remain trusted.

Development compatibility is explicit in local production-preview smoke tests. Production
console mode defaults to required; invalid mode configuration fails rather than bypassing
sign-in. Console UI permissions remain a usability follow-up; the gateway is authoritative.
