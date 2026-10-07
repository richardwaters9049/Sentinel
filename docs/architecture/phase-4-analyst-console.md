# Phase 4 — Analyst Console

## Status

In progress on:

```text
feat/phase-4-analyst-console
```

## Goal

Phase 4 turns Sentinel's tested backend capabilities into a usable analyst-facing application.

The console is built with:

- Next.js 16;
- React 19;
- TypeScript;
- Tailwind CSS v4;
- shadcn components;
- Framer Motion;
- Bun.

## Design direction

The analyst console uses a dark, modern defensive-security visual system with:

- deep near-black surfaces;
- restrained cyan, violet, emerald, amber, and rose accents;
- subtle gradient lighting;
- strong text-size hierarchy;
- slightly increased tracking and line height;
- responsive CSS grid layouts;
- visible hover/focus states;
- pointer cursors for interactive controls;
- compact but readable analyst-density information;
- motion on page load, tab changes, and scroll entry.

The design avoids generic admin-dashboard styling. The goal is to feel like a purpose-built threat hunting and investigation product.

## Figma

The Phase 4 design file is:

https://www.figma.com/design/8NCzHYqHVSmzzAaXFUojY1

The file currently contains:

- an editable desktop analyst-console concept;
- an editable mobile concept;
- a captured implementation frame from the running Next.js application.

## First implemented screen

The first screen is the security overview dashboard.

It currently includes:

- responsive sidebar/navigation;
- mobile navigation sheet;
- live defensive posture summary;
- risk index;
- findings/investigations/hunts/coverage metric cards;
- priority findings;
- threat activity;
- active investigations;
- detection coverage;
- page-load and scroll-entry animation;
- a tabbed findings surface.

The first visual slice uses representative data shaped around the real Sentinel backend model.

Live API integration follows once the visual system and application shell are stable.

## Verification

From:

```text
/Users/richy/Documents/Github/Sentinel/apps/console
```

run:

```bash
bun run lint
bun run build
```

Both currently pass.

## Product direction

Phase 4 is deliberately moving beyond a conventional dashboard.

The dashboard remains the entry point, but Sentinel's defining analyst experiences are now:

1. **Environment Graph** — a live logical view of Corporate IT, Security/DMZ, and OT assets.
2. **Investigation Graph** — evidence relationships between identities, assets, events, findings, and detections.
3. **Threat Hunt Canvas** — a visual way to build and execute the typed hunt model from Phase 3.
4. **Incident Replay** — chronological reconstruction of suspicious activity using persisted evidence.

These surfaces are intended to make Sentinel feel like a cyber-investigation instrument rather than a generic admin application.

## Environment workspace

The first signature workspace is now implemented at:

```text
/environment
```

It includes:

- Corporate IT, DMZ, and OT security zones;
- interactive asset nodes;
- node state and finding indicators;
- selected-asset inspection;
- identity association;
- visual path highlighting;
- an explainability panel showing why Sentinel flagged the activity;
- incident replay controls;
- step-by-step evidence progression;
- animated attack-path movement using Framer Motion;
- responsive behaviour for smaller screens.

The running implementation has also been captured back into the Phase 4 Figma file as an editable reference frame.

## Real gateway integration

The console now has a same-origin Next.js proxy:

```text
/api/sentinel/*
```

It forwards bounded analyst requests to the Go gateway configured by:

```text
SENTINEL_GATEWAY_URL
```

The default local value is:

```text
http://127.0.0.1:8080
```

Browser code therefore does not need direct cross-origin access to the Go service.

The proxy currently forwards the analyst methods required by the console:

- `GET`;
- `POST`;
- `PATCH`.

Only the development analyst headers used by the current backend contract are forwarded.

Gateway failures are returned to the UI as explicit `503 gateway_unavailable` responses rather than being silently replaced with fake data.

## Investigation Graph

The second signature workspace is now implemented at:

```text
/investigations
```

Unlike the first overview screen, this workspace is wired to the real Phase 3 backend.

It loads:

- the persisted investigation list;
- selected investigation details;
- linked findings;
- linked evidence events;
- investigation audit history;
- the persisted investigation timeline.

The UI derives a relationship graph from those persisted records.

Current graph node classes are:

- identities;
- assets;
- network endpoints;
- findings;
- detections.

Edges are derived from evidence relationships such as:

- identity → asset activity;
- asset → source IP;
- source IP → destination IP;
- evidence → finding;
- finding → detection.

Suspicious evidence relationships animate across the graph using Framer Motion.

The workspace also includes:

- a real investigation selector;
- priority and status context;
- a node inspector;
- a chronological evidence timeline;
- explicit gateway-unavailable and retry states;
- responsive horizontal graph exploration for smaller displays.

This is the first Phase 4 screen that directly renders live persisted Sentinel data rather than representative presentation data.

The running screen has been captured back into the Phase 4 Figma file as an editable implementation reference.

## Next frontend work

1. connect the Environment workspace to real asset, identity, finding, and event APIs;
2. [x] build the Investigation Graph on top of Phase 3 investigations and timelines;
3. build the Threat Hunt Canvas against the typed saved-hunt API;
4. connect overview metrics and findings to real gateway endpoints;
5. build findings list/detail and evidence timeline;
6. extend investigation case-management actions from the graph workspace;
7. add detection-management controls;
8. expand loading, empty, failure, and degraded-backend states;
9. complete responsive and accessibility testing;
10. add end-to-end analyst workflow tests.
