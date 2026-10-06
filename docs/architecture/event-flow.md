# Event Flow

## Target flow

```text
Synthetic producer
      |
      v
Gateway ingestion
      |
validation + normalisation
      |
      v
NATS JetStream
      |
      +-------------------+
      |                   |
      v                   v
Persistence         Detection engine
      |                   |
      v                   v
PostgreSQL           Findings
                          |
                          v
                     PostgreSQL
                          |
                          v
                    Analyst API/UI
```

## Invariants

- Every accepted event receives or preserves a stable event ID.
- Every event records both source time and receive time.
- Invalid events are rejected or quarantined, never silently accepted.
- Consumers must tolerate duplicate delivery.
- Event processing should be idempotent where practical.
- Event schema versions must be explicit.
- Internal timestamps use UTC.
- The source clock is never blindly trusted.
- Detection findings preserve enough evidence to explain why they were generated.

## Phase 0

Phase 0 establishes the transport and service foundations but does not yet publish security events.

The first production-like event path is introduced in Phase 1.
