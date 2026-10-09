# ADR 0007 — Signed collector requests and durable access decisions

- Status: Accepted for the simulated lab
- Date: 2026-10-09

## Decision

Required-mode collector ingestion combines the existing expiring bearer credential
with an operator-registered Ed25519 public key. The collector signs exact request
bytes, method, path, timestamp and a cryptographic nonce. A PostgreSQL transaction
reserves the nonce before ingestion; duplicate reservations fail, including across
process restarts. Public keys never grant permission without the bearer principal.

Signature input is UTF-8, joined by newlines without a trailing newline:

```text
sentinel-collector-v1
POST
/api/v1/telemetry
<canonical decimal Unix seconds>
<32 lowercase hexadecimal nonce characters>
<64 lowercase hexadecimal SHA-256 of exact body bytes>
```

Headers are `X-Sentinel-Timestamp`, `X-Sentinel-Nonce`, and `X-Sentinel-Signature`.
The signature and manifest `collector_public_key` use canonical unpadded standard
base64. Accept timestamps within 30 seconds of gateway time; reject query strings,
duplicate headers, invalid signatures and repeated nonces. Keep nonce state for two
minutes, capped at 100,000 live entries globally. Retry a failed ingestion with the
same event ID and a fresh nonce/signature. Existing event idempotency is retained.

Persist verified collector subject, public-key digest and body digest in the reserved
`sentinel.collector.*` label namespace. Remove producer-supplied reserved labels first.
Reported source metadata remains available but does not establish collector trust.
Signatures prove possession of the registered key; they do not establish that a
reported event happened or that a compromised collector is honest.

Persist access decisions in migration 0017 using generated trace context, registered
route, method, verified principal, outcome and status. Omit tokens, signing material,
request bodies, query strings and peer addresses. Record receipt and verified allowance
before dispatch; audit-storage failure blocks protected dispatch. Completion persistence
failure is logged and counted; it cannot undo an already completed domain transaction.
Existing domain audits remain the authoritative transactional mutation history.

The administrative retention endpoint deletes at most 1,000 access records older
than 90 days per explicit call. It never deletes domain evidence or domain audits.
Operators must schedule retention and protect backups. Peer-level rate denials are
counted/logged without additional database writes to avoid amplifying a flood.

## Consequences

Use bounded process-local token buckets: 4096 peer/principal entries, expired idle
entries reclaimed after five minutes, and fail closed on capacity exhaustion. Forwarded
IP/role headers are untrusted. Principal limits apply across keys within one gateway;
horizontal replicas would need shared rate-limit state or an upstream limiter.

Existing required-mode collector manifests need a public key. Provisioning creates
private signing seeds outside the repository. Registry replacement still requires a
restart and invalidates existing credentials/sessions. A compromised operator, signing
key or bearer credential remains a residual risk; central identity and hardware key
management are outside this lab milestone.
