# Project vision

This document preserves Sentinel’s design goals and longer-term ideas. Lists of planned
telemetry, detections, screens and technologies describe intended scope, not a claim that
every item is implemented. Consult the [development roadmap](ROADMAP.md) and linked
phase documents for current capabilities and completion evidence.

## Why Sentinel exists

Modern defensive security teams rarely work from one log source or one simple alert stream. Analysts must reason across identity events, process activity, network flows, service behaviour, endpoint telemetry, configuration changes, threat intelligence, and contextual information about assets.

Industrial and critical-infrastructure environments add another layer of difficulty:

- business IT and operational technology have different availability and safety constraints;
- legacy systems may not support conventional endpoint tooling;
- telemetry may be incomplete or inconsistent;
- normal traffic can include unusual protocols and long-lived connections;
- false positives can be costly;
- a defensive action that is reasonable on a laptop may be unacceptable on an industrial controller.

Sentinel exists to model these problems safely using **synthetic data and simulated assets**.

It is **not** intended to interact with real industrial equipment, bypass protections, deliver payloads, or provide offensive capability against third-party systems.

## Core objectives

Sentinel should demonstrate the following capabilities:

- secure ingestion of high-volume security telemetry;
- normalisation of heterogeneous event formats;
- deterministic detection rules;
- behavioural and anomaly-based detections;
- analyst-driven threat hunting;
- event correlation across multiple sources;
- MITRE ATT&CK and MITRE ATT&CK for ICS mappings;
- asset and identity context;
- investigation timelines;
- finding triage and evidence handling;
- detection quality metrics;
- telemetry-gap identification;
- threat-intelligence enrichment;
- auditable analyst actions;
- reproducible simulation scenarios;
- resilient distributed-service design;
- secure-by-default engineering practices;
- observability for the platform itself.

## Simulated asset model

The initial lab can model the following fictional environment:

```text
Northstar Energy Facility
│
├── Corporate IT
│   ├── employee-workstation-01
│   ├── employee-workstation-02
│   ├── identity-server-01
│   ├── application-server-01
│   └── database-server-01
│
├── Security / DMZ
│   ├── jump-host-01
│   ├── historian-01
│   ├── telemetry-gateway-01
│   └── update-server-01
│
└── OT Lab
    ├── engineering-workstation-01
    ├── hmi-01
    ├── plc-sim-01
    ├── plc-sim-02
    └── sensor-sim-01
```

All systems are synthetic. The initial OT components should be logical simulations rather than real industrial devices.

## Planned telemetry sources

Sentinel should eventually ingest several event families:

### Identity

- successful authentication;
- failed authentication;
- privilege changes;
- MFA events;
- account creation;
- account disablement;
- role assignment;
- service-account use;
- abnormal login time;
- new device association.

### Endpoint

- process start;
- process stop;
- file creation;
- file modification;
- executable hash;
- parent/child process relationship;
- scheduled task simulation;
- service change simulation;
- removable-media event;
- user session event.

### Network

- source and destination;
- port and protocol;
- connection duration;
- bytes transferred;
- DNS query;
- TLS metadata;
- east-west connection;
- IT-to-OT boundary crossing;
- blocked or denied connection.

### OT / ICS simulation

- HMI-to-controller communication;
- engineering workstation access;
- PLC configuration-change event;
- command-frequency changes;
- tag/value change;
- historian interaction;
- device mode change;
- controller restart;
- firmware-update simulation;
- unexpected protocol usage.

### Cloud / platform

- deployment event;
- secret access;
- role assumption;
- container start;
- image digest;
- CI/CD execution;
- infrastructure-change event.

## Detection model

Detections should initially be deterministic and explainable.

A detection should contain:

- stable ID;
- title;
- description;
- severity;
- confidence;
- status;
- author;
- version;
- event selectors;
- conditions;
- time window;
- grouping key;
- suppression logic;
- evidence requirements;
- MITRE mapping;
- ATT&CK for ICS mapping where applicable;
- test fixtures;
- expected positive cases;
- expected negative cases.

Example detection ideas:

- repeated failed authentication followed by success;
- new privileged role assignment;
- privileged account used from an unseen host;
- workstation communicating directly with an OT controller;
- engineering workstation initiating unusual controller interaction;
- unusual process spawning a network-capable child process;
- abnormal service-account use;
- unexpected protocol crossing an IT/OT boundary;
- configuration change outside an approved maintenance window;
- sudden increase in controller-write operations;
- identity activity followed by lateral movement;
- new executable followed by unusual outbound connection.

Every detection should explain **why** it fired.

## Threat hunting

A core differentiator for Sentinel is that it should support analyst-created hunting hypotheses rather than only predefined alerts.

Example hypothesis:

> Identify systems showing lateral movement after suspicious authentication activity.

The hunt engine could:

1. find unusual authentication events;
2. identify affected identities;
3. trace those identities across hosts;
4. identify first-seen network relationships;
5. identify privilege or process activity around the same time;
6. build an evidence timeline;
7. calculate a confidence score;
8. return candidate investigations.

Initial hunts can be implemented as saved queries or code-defined strategies. A later phase may add a query language.

## MITRE ATT&CK integration

Detections and investigations should be mapped to MITRE ATT&CK where appropriate.

Potential enterprise techniques include:

- Valid Accounts;
- Remote Services;
- Command and Scripting Interpreter;
- Account Discovery;
- Network Service Scanning;
- Remote System Discovery;
- Impair Defences.

ICS-specific mappings should be added only where the simulated behaviour actually supports them.

Mappings are context for analysts, not proof of malicious activity.

## Telemetry-gap analysis

Sentinel should eventually answer not only:

> What suspicious behaviour did we detect?

but also:

> What important behaviour would we be unable to detect with our current telemetry?

Examples:

- no process telemetry from an engineering workstation;
- network flow available but identity context absent;
- controller interactions visible but change events unavailable;
- authentication logs arriving late;
- collector stopped sending events;
- event source clock drift.

This feature is especially relevant to detection engineering and critical-infrastructure defence.

## Analyst console

The UI should feel like an operational security product rather than a generic admin dashboard.

Initial screens:

### Overview

- recent findings;
- findings by severity;
- ingestion health;
- event volume;
- active scenarios;
- telemetry coverage;
- recent high-confidence correlations.

### Findings

- sortable finding queue;
- severity and confidence filters;
- detection name;
- asset;
- identity;
- first and last observed time;
- event count;
- status;
- MITRE mapping.

### Investigation

- narrative summary;
- event timeline;
- related assets;
- related identities;
- network relationships;
- detection evidence;
- enrichment;
- analyst notes;
- MITRE techniques;
- raw event inspection.

### Hunts

- saved hunting hypotheses;
- execution history;
- matching entities;
- timeline;
- confidence;
- query details.

### Assets

- hostname;
- zone;
- operating-system family;
- owner / role;
- criticality;
- first seen;
- last seen;
- telemetry sources;
- risk context.

### Detection engineering

- rule list;
- rule version;
- enabled state;
- validation results;
- positive/negative fixtures;
- hit count;
- false-positive rate;
- last triggered.

### Platform health

- collectors;
- queue lag;
- storage health;
- processing latency;
- event rejection rate;
- service versions.

## Proposed technology stack

### Backend

**Go**

Primary responsibilities:

- ingestion APIs;
- event normalisation;
- detection evaluation;
- correlation;
- hunting;
- API services;
- service health;
- background processing.

Why Go:

- predictable concurrency;
- strong standard library;
- low runtime overhead;
- simple deployment;
- good fit for high-throughput event systems;
- relevant to modern infrastructure/security engineering.

### Sensor / collector

**Rust**

Potential responsibilities:

- lightweight local collector;
- safe parsing;
- event batching;
- backpressure handling;
- signed/encrypted transport;
- local buffering;
- integrity metadata.

Rust should only be introduced where it adds value. The project should not become multi-language for appearance alone.

### Frontend

**Next.js + React + TypeScript**

Responsibilities:

- analyst dashboard;
- finding triage;
- investigation timelines;
- threat-hunt interface;
- asset exploration;
- detection management;
- platform-health views.

### Data / state

**PostgreSQL**

Durable storage for:

- events;
- findings;
- assets;
- identities;
- detections;
- investigations;
- analyst actions;
- scenario metadata.

**Redis**

Short-lived state for:

- rule windows;
- deduplication;
- rate limiting;
- temporary correlation state;
- caching.

### Event transport

Start with **NATS JetStream** unless requirements later justify Kafka.

NATS provides a lighter operational footprint for local development while still demonstrating durable event-driven architecture.

Kafka can be introduced later if the project requires partition-heavy streaming semantics or Kafka-specific experience.

### Machine learning

**Python**

Potential later-phase use:

- anomaly scoring;
- baseline modelling;
- feature extraction;
- clustering;
- offline model evaluation.

ML must not replace explainable deterministic detection logic. Models should support analysts, not create opaque security conclusions.

### Infrastructure

- Docker / Docker Compose;
- GitHub Actions;
- OpenTelemetry;
- Prometheus;
- Grafana;
- structured JSON logging;
- optional Kubernetes deployment in a later phase.

## Synthetic scenario framework

Scenarios should be repeatable and deterministic where possible.

A scenario definition may include:

- name;
- description;
- participating assets;
- participating identities;
- start offset;
- normal background events;
- suspicious event sequence;
- expected detections;
- expected non-detections;
- MITRE mappings;
- seed.

Example scenarios:

1. **Credential misuse and lateral movement**
2. **Unexpected engineering workstation access**
3. **IT-to-OT boundary violation**
4. **Abnormal controller write activity**
5. **Service account used interactively**
6. **Suspicious process and outbound connection**
7. **Maintenance-window policy violation**
8. **Telemetry collector loss**
9. **False-positive baseline scenario**
10. **Multi-stage correlated intrusion simulation**

Scenarios must remain non-destructive and lab-safe.

## Data quality

Security analytics are only as useful as the telemetry beneath them.

Sentinel should track:

- missing fields;
- parser failure rate;
- clock skew;
- duplicate events;
- unknown assets;
- unknown identities;
- stale collectors;
- schema versions;
- source reliability;
- event delay;
- dropped events.

Data quality should be visible to analysts rather than silently ignored.

## Privacy and data minimisation

Although the project uses synthetic data, its architecture should model responsible handling of real-world security telemetry.

Avoid unnecessary storage of:

- raw secrets;
- authentication tokens;
- content unrelated to the detection goal;
- excessive personal data.

Retention should eventually be configurable by event class.

## Initial success criteria

The first meaningful demo should be able to show this story:

1. start the Sentinel development environment;
2. launch a named synthetic scenario;
3. watch events enter the platform;
4. see a detection trigger;
5. open the finding;
6. inspect the evidence timeline;
7. view asset and identity context;
8. see MITRE mapping;
9. change the finding status;
10. verify the action appears in the audit log.

That is a small enough target to build properly while still demonstrating the larger vision.

## Example first scenario

### Suspicious engineering workstation access

Normal behaviour:

- engineering workstation communicates with approved OT systems;
- activity occurs during a maintenance window;
- known operator identity is present.

Suspicious behaviour:

- unusual identity logs into the engineering workstation;
- login occurs outside the normal maintenance window;
- workstation initiates a first-seen connection to a controller simulation;
- a controller configuration-change event follows.

Expected Sentinel behaviour:

- identity anomaly contributes context;
- first-seen OT relationship is detected;
- maintenance-window violation is detected;
- events are correlated into a higher-confidence finding;
- analyst receives an ordered evidence timeline;
- relevant ATT&CK / ATT&CK for ICS context is attached.

No real PLC commands are required.

## Portfolio value

Sentinel is intentionally designed to demonstrate experience relevant to roles such as:

- Senior Software Engineer;
- Security Engineer;
- Threat Hunter;
- Detection Engineer;
- Security Platform Engineer;
- DevSecOps Engineer;
- Cloud Security Engineer;
- Security Automation Engineer;
- Backend / Distributed Systems Engineer;
- Critical-Infrastructure Cybersecurity Engineer;
- SOC Engineering;
- Threat Intelligence Engineering.

The project should show engineering depth rather than an excessive number of superficial features.

## Non-goals

Sentinel is not intended to become:

- a commercial SIEM clone;
- an EDR agent competing with mature endpoint vendors;
- a real industrial-control exploitation framework;
- a vulnerability scanner;
- an automated penetration-testing system;
- an AI chatbot wrapped around logs;
- an opaque "AI detects everything" demo.

The value is in a clear, inspectable defensive-security system.

## Current contracts

Use the [versioned telemetry schema](telemetry/EVENT_SCHEMA_V1.md),
[analyst finding workflow](detections/ANALYST_WORKFLOW.md),
[hunt model](hunting/HUNT_MODEL.md) and [event flow](architecture/event-flow.md)
for implemented contracts. The earlier illustrative schema and lifecycle in the root
README are superseded by these documents. Redis, Kafka, object storage, a Rust
collector and Kubernetes remain potential extensions; they are not required by the
current local stack.
