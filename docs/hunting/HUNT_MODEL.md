# Sentinel Hunt Model

## Principle

A threat hunt starts with a hypothesis, not with an alert.

Sentinel therefore stores the analyst hypothesis separately from the query used to test it.

## Example

Hypothesis:

```text
A service identity may be used interactively from a corporate asset.
```

Query:

```json
{
  "category": "authentication",
  "action": "login",
  "outcome": "success",
  "identity_id": "svc-backup",
  "limit": 100
}
```

## Query safety

Phase 3 hunt queries are deliberately constrained rather than accepting arbitrary SQL.

The API exposes a typed set of filters over the normalised event model.

This avoids:

- raw SQL injection;
- arbitrary database access;
- unbounded queries;
- coupling analysts to physical database layout.

## Reproducibility

Saved definitions and run history preserve:

- what the analyst was testing;
- what filters were used;
- who ran the hunt;
- when it ran;
- how many events matched.

Later Phase 3 work can build comparison and scheduling features on this history.
