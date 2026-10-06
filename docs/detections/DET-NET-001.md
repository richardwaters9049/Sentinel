# DET-NET-001 — Unexpected Corporate-to-OT Network Connection

## Status

Enabled.

## Purpose

Detect a network connection where the observed source asset is in the corporate zone and the destination is explicitly classified as OT.

The rule models a simple but important critical-infrastructure boundary: ordinary corporate endpoints should not directly communicate with operational-technology systems unless the architecture and operating procedure explicitly allow it.

## Logic

The rule fires when all of the following are true:

- event category is `network`;
- event action is `connection`;
- source asset context exists;
- source asset zone is `corporate`;
- network destination zone is `ot`.

## Severity and confidence

```text
severity:   high
confidence: 95
```

The high confidence reflects that the zone-crossing condition is directly represented in the normalised event.

It does not mean the connection is necessarily malicious.

## Evidence

The finding records:

- terminal event ID;
- source IP;
- source zone;
- destination IP;
- destination zone;
- first/last observed timestamp.

The original event is linked through `finding_events`.

## ATT&CK mapping

No ATT&CK technique is assigned in the initial version.

A bare corporate-to-OT connection is a network-policy signal, not enough evidence by itself to assert a particular adversary technique. Future rules may add ATT&CK or ATT&CK for ICS context when richer behaviour justifies it.

## False-positive examples

Potential benign explanations include:

- an approved jump-host path represented incorrectly in telemetry;
- monitoring or backup infrastructure;
- a temporary maintenance exception;
- an asset assigned to the wrong zone;
- lab or simulation traffic;
- explicitly approved engineering access.

Future enrichment should compare the connection against network policy, maintenance windows, asset role, protocol expectations, and approved paths.

## Telemetry requirement

The event schema now supports:

```json
{
  "network": {
    "destination_zone": "ot"
  }
}
```

This optional field is normalised to lowercase and bounded to 64 characters.
