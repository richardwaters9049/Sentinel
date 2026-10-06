# Trust Boundaries

Sentinel is a security product and therefore treats all external input as untrusted.

## Initial boundaries

### 1. User or client to Gateway

Risks include malformed requests, oversized payloads, invalid state changes, and resource exhaustion.

Initial controls:

- request-size limits;
- strict routing;
- safe response headers;
- server timeouts;
- structured logging.

Authentication and RBAC are planned before the system exposes privileged workflows.

### 2. Telemetry producer to Gateway

Future telemetry producers remain untrusted until authenticated and validated.

Required future controls:

- collector identity;
- schema validation;
- replay resistance;
- rate limiting;
- provenance;
- quarantine for malformed events.

### 3. Gateway to Event Bus

Publish rights should eventually be scoped to only the subjects required.

### 4. Consumers to PostgreSQL

Each service should receive only the database permissions it needs. Shared development credentials are acceptable only during the local bootstrap and must not become a deployment pattern.

### 5. Simulation to Host

Simulation code must remain non-destructive and must not require access to real industrial equipment or third-party targets.

## Principle

Crossing a trust boundary requires explicit validation, authentication where appropriate, and observable failure behaviour.
