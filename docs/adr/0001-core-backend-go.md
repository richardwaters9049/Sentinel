# ADR-0001: Use Go for Core Backend Services

## Status

Accepted.

## Context

Sentinel requires concurrent event processing, HTTP APIs, background consumers, clear deployment units, and predictable resource use.

## Decision

Use Go for the gateway and the majority of core backend services.

## Consequences

Benefits:

- simple native binaries;
- strong concurrency model;
- mature networking standard library;
- straightforward testing;
- low operational overhead.

Trade-offs:

- some analytics workflows are more natural in Python;
- low-level collector work may benefit from Rust.

Those languages remain available where they provide a concrete advantage.
