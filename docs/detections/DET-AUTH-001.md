# DET-AUTH-001 — Repeated Authentication Failures Followed by Success

## Status

Enabled.

## Purpose

Detect a short burst of failed login attempts followed by a successful login for the same identity and source address.

This pattern can be consistent with credential guessing, password spraying against a single account, or an attacker eventually authenticating with a valid credential.

It is not proof of compromise. The finding is intended to give an analyst an explainable evidence chain for investigation.

## Logic

A finding is generated when all of the following are true:

- event category is `authentication`;
- event action is `login`;
- the current event outcome is `success`;
- the event has an actor identity;
- the event has a source IP;
- at least four failed logins exist for the same identity and source IP;
- those failures occurred during the five minutes before the successful login.

Current threshold:

```text
failed attempts: 4
window:          5 minutes
terminal event:  successful login
```

## Grouping

The correlation key is:

```text
identity_id + network.source_ip
```

This prevents failures from unrelated users or unrelated source addresses from being combined into one finding.

## Severity and confidence

```text
severity:   high
confidence: 85
```

The confidence score represents confidence that the defined behavioural sequence occurred, not an 85% probability that the account is compromised.

## Evidence

A finding records:

- all four correlated failed-event IDs;
- the successful-event ID;
- identity ID;
- source IP;
- failure count;
- correlation window;
- first observed time;
- last observed time.

The related events are also inserted into `finding_events` so an investigation can later retrieve the original evidence directly.

## MITRE ATT&CK context

### T1110 — Brute Force

Repeated failed authentication attempts may be consistent with credential guessing or similar brute-force activity.

### T1078 — Valid Accounts

The final successful authentication may represent use of a legitimate credential after the preceding failures.

These mappings provide analyst context. They do not prove that either technique occurred.

## False-positive examples

Possible benign explanations include:

- a user repeatedly entering an incorrect password before remembering it;
- credential-cache or password-manager issues;
- an application retrying with stale credentials;
- an approved testing exercise;
- an authentication integration temporarily using outdated credentials.

Future contextual enrichment should consider:

- known maintenance/test windows;
- source asset role;
- identity privilege;
- first-seen source address;
- device trust;
- historical authentication baseline;
- geographic or network-zone context.

## Idempotency

Finding creation is deterministic.

The finding and deduplication keys are derived from:

```text
detection ID
identity ID
source IP
successful event ID
```

Repeated processing of the same terminal event therefore does not create a second finding.

## Test coverage

The unit suite verifies:

- four failures followed by success creates a finding;
- fewer than four failures does not create a finding;
- failed terminal events are ignored;
- missing correlation context is ignored;
- repository failures propagate;
- evidence is chronologically ordered.

The Phase 2 smoke test verifies the complete path from HTTP telemetry ingestion through JetStream, PostgreSQL persistence, detection evaluation, finding creation, evidence linking, and findings API retrieval.
