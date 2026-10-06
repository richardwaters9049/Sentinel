# DET-AUTH-002 — Interactive Login Using a Service Account

## Status

Enabled.

## Purpose

Detect a successful interactive login where the actor is classified as a service account.

Service accounts are normally intended for application-to-application or automated workloads rather than human interactive use. An interactive login can therefore be a useful signal for credential misuse, operational drift, or an incorrectly configured account.

This finding is investigative context, not proof of compromise.

## Logic

The rule fires when all of the following are true:

- event category is `authentication`;
- event action is `login`;
- event outcome is `success`;
- actor context exists;
- actor type is `service_account`.

## Severity and confidence

```text
severity:   high
confidence: 90
```

Confidence represents confidence that the defined event condition occurred, not the probability that malicious activity occurred.

## Evidence

The finding records:

- terminal event ID;
- identity ID;
- actor type;
- source IP when available;
- first/last observed timestamp.

The original telemetry event is linked through `finding_events`.

## MITRE ATT&CK context

### T1078 — Valid Accounts

Successful use of a service-account credential can be relevant to Valid Accounts when the credential is used outside its intended context.

The mapping is contextual and should not be interpreted as confirmation of adversary behaviour.

## False-positive examples

Potential benign explanations include:

- an administrator troubleshooting with an approved service identity;
- legacy automation that performs interactive-style authentication;
- temporary migration activity;
- incorrectly classified account type;
- approved break-glass procedures.

Future enrichment should consider account ownership, approved login methods, source asset, maintenance windows, and historical usage.

## Idempotency

The finding key is derived from:

```text
detection ID
identity ID
event ID
```

Reprocessing the same event does not create a duplicate finding.
