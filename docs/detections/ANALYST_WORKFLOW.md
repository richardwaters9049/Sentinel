# Analyst Finding Workflow

## Purpose

Phase 2 introduces an explicit analyst workflow around security findings.

A finding is not treated as resolved merely because a detection fired. Analysts must be able to inspect evidence, record a decision, progress the case through controlled states, and preserve an audit trail.

## Finding detail

The detail endpoint is:

```http
GET /api/v1/findings/{id}
```

It returns:

- finding metadata;
- detection ID and version;
- severity and confidence;
- current workflow status;
- first and last observed timestamps;
- structured detection evidence;
- the original contributing telemetry events in chronological order;
- audit events associated with the finding.

This endpoint is the backend foundation for the later investigation timeline UI.

## Workflow states

The supported states are:

```text
new
 ↓
triaged
 ├─→ false_positive
 ├─→ benign_expected
 ├─→ duplicate
 └─→ investigating
       ├─→ false_positive
       ├─→ benign_expected
       ├─→ duplicate
       └─→ confirmed
             ├─→ closed
             └─→ contained
                    ↓
                  closed
```

Terminal states are:

- `false_positive`
- `benign_expected`
- `duplicate`
- `closed`

Invalid state jumps return HTTP `409 Conflict`.

## Status mutation

The endpoint is:

```http
PATCH /api/v1/findings/{id}/status
X-Sentinel-Actor: analyst-id
X-Request-ID: optional-request-id
Content-Type: application/json
```

Example body:

```json
{
  "status": "triaged"
}
```

`X-Sentinel-Actor` is mandatory while the project is pre-authentication. It ensures local development mutations still produce attributable audit records.

This header is not intended to replace authenticated identity. When authentication is added, the actor identity must come from validated server-side authentication context rather than a caller-controlled header.

## Audit behaviour

A successful status transition inserts an append-only `audit_events` row in the same transaction as the finding update.

The record contains:

- actor ID;
- action: `finding.status_changed`;
- resource type: `finding`;
- resource ID;
- optional request ID;
- previous status;
- new status;
- server timestamp.

If audit insertion fails, the state transition is rolled back.

## Detection management

Detection metadata is available through:

```http
GET /api/v1/detections
```

The response includes:

- stable detection ID;
- version;
- title and description;
- severity;
- enabled state;
- structured definition;
- MITRE context;
- creation/update timestamps.

Detection state can be changed through:

```http
PATCH /api/v1/detections/{id}
X-Sentinel-Actor: analyst-id
Content-Type: application/json
```

Example:

```json
{
  "enabled": false
}
```

The operation is audited using `detection.enabled_changed`.

## Runtime enforcement

Detection enable/disable state is not cosmetic.

Before evaluating DET-AUTH-001, the detection engine reads the persisted enabled state.

When disabled:

- telemetry continues to be accepted;
- telemetry continues to be persisted;
- the detector does not correlate or create findings for that rule.

When re-enabled, subsequent events are evaluated normally.

## Verified behaviour

The Phase 2 workflow smoke test proves:

- finding detail includes all five contributing events;
- invalid state jumps are rejected;
- a finding can follow the complete valid path through `closed`;
- every successful transition generates an audit record;
- DET-AUTH-001 can be disabled through the API;
- auth-burst telemetry received while disabled does not create a finding;
- re-enabling the rule restores finding generation;
- detection state changes are audited.

## Security note

The actor header is a development bridge only.

Before Sentinel is deployed outside the local development environment, analyst actions require authenticated identities and server-side authorisation. The API must not trust caller-supplied actor identity in a production deployment.
