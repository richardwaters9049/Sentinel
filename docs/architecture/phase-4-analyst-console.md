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

## Next frontend work

1. extract the dashboard into reusable application-shell components;
2. connect overview metrics and findings to real gateway endpoints;
3. build findings list/detail and evidence timeline;
4. build hunts and hunt-builder screens;
5. build investigations and timeline screens;
6. add asset/identity pivot screens;
7. add detection-management controls;
8. add loading, empty, failure, and degraded-backend states;
9. complete responsive and accessibility testing;
10. add end-to-end analyst workflow tests.
