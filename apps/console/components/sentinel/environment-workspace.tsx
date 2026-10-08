"use client";

import {
  Activity,
  AlertTriangle,
  Bell,
  ChevronRight,
  CircleDot,
  CirclePause,
  CirclePlay,
  Clock3,
  Cpu,
  Database,
  Fingerprint,
  GitBranch,
  Laptop,
  Network,
  Radar,
  RefreshCcw,
  RotateCcw,
  Search,
  Server,
  Shield,
  ShieldCheck,
  SquareTerminal,
  Waypoints,
} from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

import SentinelBrand from "@/components/sentinel/brand";
import SidebarDrawer from "@/components/sentinel/sidebar-drawer";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import {
  getAssetPivot,
  listEvents,
  listFindings,
} from "@/lib/sentinel/client";
import type {
  AssetPivot,
  Finding,
  TelemetryEvent,
} from "@/lib/sentinel/types";

type Zone = "corporate" | "dmz" | "ot";
type AssetState = "healthy" | "observed" | "elevated" | "critical";

type EnvironmentNode = {
  id: string;
  assetID?: string;
  name: string;
  subtitle: string;
  zone: Zone;
  state: AssetState;
  ip: string;
  identity?: string;
  identityType?: string;
  lastSeen: string;
  lastSeenAt: string;
  findings: Finding[];
  source: "asset" | "endpoint";
  icon: typeof Laptop;
};

type Flow = {
  id: string;
  timestamp: string;
  sourceNodeID?: string;
  sourceLabel: string;
  destinationNodeID?: string;
  destinationLabel: string;
  category: string;
  action: string;
  port?: number;
  protocol?: string;
  detection?: Finding;
  event: TelemetryEvent;
};

const zoneMeta = {
  corporate: {
    label: "Corporate IT",
    subtitle: "Business identities and workstations",
    accent: "text-sky-300",
    border: "border-sky-400/20",
    bg: "from-sky-500/[0.07] to-transparent",
  },
  dmz: {
    label: "Security / DMZ",
    subtitle: "Controlled transition and telemetry",
    accent: "text-violet-300",
    border: "border-violet-400/20",
    bg: "from-violet-500/[0.07] to-transparent",
  },
  ot: {
    label: "OT Network",
    subtitle: "Engineering and synthetic control systems",
    accent: "text-amber-200",
    border: "border-amber-300/20",
    bg: "from-amber-400/[0.06] to-transparent",
  },
} satisfies Record<
  Zone,
  {
    label: string;
    subtitle: string;
    accent: string;
    border: string;
    bg: string;
  }
>;

function normaliseZone(value?: string): Zone | null {
  const zone = value?.trim().toLowerCase();
  if (!zone) return null;
  if (zone === "corporate" || zone === "corp" || zone === "it") {
    return "corporate";
  }
  if (zone === "dmz" || zone === "security") return "dmz";
  if (zone === "ot" || zone === "ics") return "ot";
  return null;
}

function formatRelative(value: string) {
  const diff = Math.max(0, Date.now() - new Date(value).getTime());
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
  }).format(new Date(value));
}

function findingEventIDs(finding: Finding) {
  const raw = finding.evidence.event_ids;
  return Array.isArray(raw)
    ? raw.filter((value): value is string => typeof value === "string")
    : [];
}

function nodeIcon(event: TelemetryEvent, zone: Zone) {
  if (zone === "ot") {
    return event.asset?.hostname?.toLowerCase().includes("plc") ? Network : Cpu;
  }
  if (event.source.type === "identity") return Fingerprint;
  if (event.source.type === "server") return Server;
  if (event.source.type === "gateway") return Radar;
  if (event.source.type === "database") return Database;
  if (event.asset?.hostname?.toLowerCase().includes("jump")) {
    return SquareTerminal;
  }
  return Laptop;
}

function severityState(findings: Finding[]): AssetState {
  if (findings.some((finding) => finding.severity === "critical")) {
    return "critical";
  }
  if (findings.some((finding) => finding.severity === "high")) {
    return "critical";
  }
  if (findings.some((finding) => finding.severity === "medium")) {
    return "elevated";
  }
  return findings.length > 0 ? "observed" : "healthy";
}

function stateClasses(state: AssetState) {
  switch (state) {
    case "critical":
      return {
        border: "border-rose-400/45",
        glow: "shadow-[0_0_42px_rgba(251,113,133,0.14)]",
        dot: "bg-rose-400",
      };
    case "elevated":
      return {
        border: "border-amber-300/35",
        glow: "shadow-[0_0_36px_rgba(251,191,36,0.10)]",
        dot: "bg-amber-300",
      };
    case "observed":
      return {
        border: "border-sky-400/30",
        glow: "",
        dot: "bg-sky-400",
      };
    default:
      return {
        border: "border-slate-800",
        glow: "",
        dot: "bg-emerald-400",
      };
  }
}

function buildEnvironment(events: TelemetryEvent[], findings: Finding[]) {
  const findingByEvent = new Map<string, Finding[]>();
  for (const finding of findings) {
    for (const eventID of findingEventIDs(finding)) {
      const current = findingByEvent.get(eventID) ?? [];
      current.push(finding);
      findingByEvent.set(eventID, current);
    }
  }

  const nodes = new Map<string, EnvironmentNode>();
  const eventNode = new Map<string, string>();

  for (const event of events) {
    if (event.asset) {
      const zone = normaliseZone(event.asset.zone);
      if (zone) {
        const id = `asset:${event.asset.id}`;
        const relatedFindings = findingByEvent.get(event.event_id) ?? [];
        const previous = nodes.get(id);
        const mergedFindings = [
          ...(previous?.findings ?? []),
          ...relatedFindings.filter(
            (candidate) =>
              !(previous?.findings ?? []).some(
                (existing) => existing.id === candidate.id,
              ),
          ),
        ];
        const sourceIP =
          event.network?.source_ip || previous?.ip || "No source IP";
        nodes.set(id, {
          id,
          assetID: event.asset.id,
          name: event.asset.hostname || event.asset.id,
          subtitle:
            zone === "ot"
              ? "Observed OT asset"
              : zone === "dmz"
                ? "Observed DMZ asset"
                : "Observed corporate asset",
          zone,
          state: severityState(mergedFindings),
          ip: sourceIP,
          identity: event.actor?.name || previous?.identity,
          identityType: event.actor?.type || previous?.identityType,
          lastSeen: formatRelative(event.timestamp),
          lastSeenAt: event.timestamp,
          findings: mergedFindings,
          source: "asset",
          icon: nodeIcon(event, zone),
        });
        eventNode.set(event.event_id, id);
      }
    }

    const destinationZone = normaliseZone(event.network?.destination_zone);
    const destinationIP = event.network?.destination_ip;
    if (destinationZone && destinationIP) {
      const id = `endpoint:${destinationIP}`;
      const relatedFindings = findingByEvent.get(event.event_id) ?? [];
      const previous = nodes.get(id);
      const mergedFindings = [
        ...(previous?.findings ?? []),
        ...relatedFindings.filter(
          (candidate) =>
            !(previous?.findings ?? []).some(
              (existing) => existing.id === candidate.id,
            ),
        ),
      ];
      nodes.set(id, {
        id,
        name:
          destinationZone === "ot"
            ? `OT endpoint ${destinationIP}`
            : `Network endpoint ${destinationIP}`,
        subtitle: `Observed destination · ${destinationZone.toUpperCase()}`,
        zone: destinationZone,
        state: severityState(mergedFindings),
        ip: destinationIP,
        lastSeen: formatRelative(event.timestamp),
        lastSeenAt: event.timestamp,
        findings: mergedFindings,
        source: "endpoint",
        icon: destinationZone === "ot" ? Network : Server,
      });
    }
  }

  const orderedNodes = [...nodes.values()]
    .sort(
      (a, b) =>
        new Date(b.lastSeenAt).getTime() - new Date(a.lastSeenAt).getTime(),
    )
    .slice(0, 18);

  const visibleIDs = new Set(orderedNodes.map((node) => node.id));
  const flows: Flow[] = [];

  for (const event of [...events].reverse()) {
    const sourceID = eventNode.get(event.event_id);
    const destinationIP = event.network?.destination_ip;
    const destinationID = destinationIP
      ? `endpoint:${destinationIP}`
      : undefined;
    const detection = (findingByEvent.get(event.event_id) ?? [])[0];

    if (!sourceID && !destinationID && !detection) continue;
    if (
      sourceID &&
      !visibleIDs.has(sourceID) &&
      destinationID &&
      !visibleIDs.has(destinationID)
    ) {
      continue;
    }

    flows.push({
      id: event.event_id,
      timestamp: event.timestamp,
      sourceNodeID: sourceID,
      sourceLabel:
        event.asset?.hostname ||
        event.actor?.name ||
        event.network?.source_ip ||
        "Unknown source",
      destinationNodeID: destinationID,
      destinationLabel:
        destinationIP ||
        event.actor?.name ||
        event.event.action ||
        "Observed activity",
      category: event.event.category,
      action: event.event.action,
      port: event.network?.destination_port,
      protocol: event.network?.protocol,
      detection,
      event,
    });
  }

  return {
    nodes: orderedNodes,
    flows: flows.slice(-12),
  };
}

function AssetNode({
  node,
  selected,
  active,
  onSelect,
}: {
  node: EnvironmentNode;
  selected: boolean;
  active: boolean;
  onSelect: () => void;
}) {
  const Icon = node.icon;
  const classes = stateClasses(node.state);

  return (
    <motion.button
      type="button"
      onClick={onSelect}
      animate={{
        scale: active ? 1.022 : 1,
        y: active ? -2 : 0,
      }}
      transition={{ type: "spring", stiffness: 300, damping: 24 }}
      className={cn(
        "group relative w-full cursor-pointer rounded-2xl border bg-[#0c1119]/92 p-3.5 text-left transition duration-200 hover:-translate-y-0.5 hover:border-slate-600 hover:bg-[#111824]",
        classes.border,
        classes.glow,
        selected && "ring-1 ring-sky-300/45",
      )}
    >
      {active ? (
        <motion.span
          layoutId="active-node-ring"
          className="pointer-events-none absolute -inset-px rounded-2xl border border-cyan-300/70 shadow-[0_0_34px_rgba(34,211,238,0.26)]"
        />
      ) : null}

      <div className="flex items-start justify-between gap-3">
        <div
          className={cn(
            "grid size-9 shrink-0 place-items-center rounded-xl border bg-slate-950/70",
            classes.border,
          )}
        >
          <Icon className="size-4 text-slate-300" />
        </div>
        <div className="flex items-center gap-1.5 text-[0.6rem] font-semibold tracking-[0.08em] text-slate-600">
          <span className={cn("size-1.5 rounded-full", classes.dot)} />
          {node.state.toUpperCase()}
        </div>
      </div>

      <div className="mt-3 truncate text-[0.78rem] font-semibold tracking-[0.01em] text-slate-200">
        {node.name}
      </div>
      <div className="mt-1 truncate text-[0.65rem] leading-5 tracking-[0.02em] text-slate-600">
        {node.subtitle}
      </div>

      <div className="mt-3 flex items-center justify-between gap-2 font-mono text-[0.6rem] text-slate-600">
        <span className="truncate">{node.ip}</span>
        {node.findings.length > 0 ? (
          <span className="shrink-0 rounded-md bg-rose-400/10 px-1.5 py-0.5 text-rose-300">
            {node.findings.length} finding
            {node.findings.length === 1 ? "" : "s"}
          </span>
        ) : (
          <span className="shrink-0">{node.lastSeen}</span>
        )}
      </div>
    </motion.button>
  );
}

function ZoneColumn({
  zone,
  nodes,
  selectedID,
  activeIDs,
  onSelect,
}: {
  zone: Zone;
  nodes: EnvironmentNode[];
  selectedID: string;
  activeIDs: Set<string>;
  onSelect: (id: string) => void;
}) {
  const meta = zoneMeta[zone];

  return (
    <div
      className={cn(
        "relative min-h-[250px] rounded-[1.4rem] border bg-gradient-to-b p-3.5",
        meta.border,
        meta.bg,
      )}
    >
      <div className="mb-4 border-b border-slate-800/65 pb-3">
        <div
          className={cn(
            "text-[0.72rem] font-bold tracking-[0.12em]",
            meta.accent,
          )}
        >
          {meta.label.toUpperCase()}
        </div>
        <div className="mt-1 text-[0.64rem] leading-5 tracking-[0.02em] text-slate-600">
          {meta.subtitle}
        </div>
      </div>

      <div className="grid gap-3">
        {nodes.length > 0 ? (
          nodes.map((node) => (
            <AssetNode
              key={node.id}
              node={node}
              selected={node.id === selectedID}
              active={activeIDs.has(node.id)}
              onSelect={() => onSelect(node.id)}
            />
          ))
        ) : (
          <div className="grid min-h-36 place-items-center rounded-xl border border-dashed border-slate-800/75 text-center">
            <div>
              <CircleDot className="mx-auto size-4 text-slate-700" />
              <div className="mt-2 text-[0.65rem] text-slate-600">
                No recent telemetry in this zone
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default function EnvironmentWorkspace() {
  const [events, setEvents] = useState<TelemetryEvent[]>([]);
  const [findings, setFindings] = useState<Finding[]>([]);
  const [selectedID, setSelectedID] = useState("");
  const [pivot, setPivot] = useState<AssetPivot | null>(null);
  const [step, setStep] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");

  const environment = useMemo(
    () => buildEnvironment(events, findings),
    [events, findings],
  );

  const filteredNodes = useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) return environment.nodes;
    return environment.nodes.filter((node) =>
      [
        node.name,
        node.ip,
        node.identity,
        node.identityType,
        node.zone,
        node.assetID,
      ]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(term)),
    );
  }, [environment.nodes, search]);

  const selected =
    environment.nodes.find((node) => node.id === selectedID) ??
    environment.nodes[0];

  const replayFlows = environment.flows;
  const safeStep = Math.min(step, Math.max(0, replayFlows.length - 1));
  const currentFlow = replayFlows[safeStep];
  const selectedPivot =
    selected?.assetID && pivot?.id === selected.assetID ? pivot : null;

  const activeIDs = useMemo(() => {
    const ids = new Set<string>();
    for (let index = 0; index <= safeStep; index += 1) {
      const flow = replayFlows[index];
      if (!flow) continue;
      if (flow.sourceNodeID) ids.add(flow.sourceNodeID);
      if (flow.destinationNodeID) ids.add(flow.destinationNodeID);
    }
    return ids;
  }, [replayFlows, safeStep]);

  async function refreshEnvironment() {
    setLoading(true);
    setError("");

    const controller = new AbortController();
    try {
      const [eventResponse, findingResponse] = await Promise.all([
        listEvents(160, controller.signal),
        listFindings(160, controller.signal),
      ]);
      setEvents(eventResponse.events);
      setFindings(findingResponse.findings);
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Could not load the Northstar environment",
      );
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    const controller = new AbortController();

    void Promise.all([
      listEvents(160, controller.signal),
      listFindings(160, controller.signal),
    ])
      .then(([eventResponse, findingResponse]) => {
        setEvents(eventResponse.events);
        setFindings(findingResponse.findings);
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(
          cause instanceof Error
            ? cause.message
            : "Could not load the Northstar environment",
        );
      })
      .finally(() => setLoading(false));

    const timer = window.setInterval(() => {
      void Promise.all([listEvents(160), listFindings(160)])
        .then(([eventResponse, findingResponse]) => {
          setEvents(eventResponse.events);
          setFindings(findingResponse.findings);
        })
        .catch(() => {});
    }, 15_000);

    return () => {
      controller.abort();
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    if (!selected?.assetID) return;

    const controller = new AbortController();
    void getAssetPivot(selected.assetID, 30, controller.signal)
      .then(setPivot)
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
      });

    return () => controller.abort();
  }, [selected?.assetID]);

  useEffect(() => {
    if (!playing || replayFlows.length === 0) return;

    const id = window.setInterval(() => {
      setStep((current) => {
        if (current >= replayFlows.length - 1) {
          setPlaying(false);
          return current;
        }
        return current + 1;
      });
    }, 1450);

    return () => window.clearInterval(id);
  }, [playing, replayFlows.length]);

  function resetReplay() {
    setPlaying(false);
    setStep(0);
  }

  const detectionCount = new Set(
    findings.map((finding) => finding.detection_id),
  ).size;

  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto min-h-screen max-w-[1780px]">
        <section className="min-w-0 px-4 py-5 sm:px-6 lg:px-8 xl:px-10">
          <motion.header
            initial={{ opacity: 0, y: -12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.42 }}
            className="mb-5 flex items-center justify-between gap-4"
          >
            <div className="min-w-0">
              <div className="flex items-center gap-3">
                <SidebarDrawer />
                <SentinelBrand />
              </div>

              <div className="mt-3 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.13em] text-cyan-300/70">
                <Waypoints className="size-3.5" />
                LIVE ENVIRONMENT
              </div>
              <h1 className="mt-2 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.035em] text-white sm:text-[1.95rem]">
                Northstar attack surface
              </h1>
              <p className="mt-1.5 max-w-3xl text-[0.78rem] leading-6 tracking-[0.025em] text-slate-500">
                Explore real persisted asset relationships, follow identity
                movement, and replay the evidence chain that produced Sentinel
                findings.
              </p>
            </div>

            <div className="flex items-center gap-2.5">
              <div className="relative hidden w-[17rem] xl:block">
                <Search className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-600" />
                <Input
                  aria-label="Search environment"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="Find asset, identity or IP…"
                  className="h-10 border-slate-800 bg-slate-950/60 pl-10 text-[0.78rem] tracking-[0.02em] text-slate-200 placeholder:text-slate-600"
                />
              </div>
              <Button
                type="button"
                onClick={() => void refreshEnvironment()}
                variant="outline"
                size="icon"
                className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900 hover:text-slate-100"
              >
                <RefreshCcw className={cn("size-4", loading && "animate-spin")} />
              </Button>
              <Button
                variant="outline"
                size="icon"
                className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900 hover:text-slate-100"
              >
                <Bell className="size-4" />
              </Button>
              <Avatar className="size-9 border border-cyan-400/25">
                <AvatarFallback className="bg-gradient-to-br from-cyan-300 to-violet-500 text-[0.72rem] font-bold text-slate-950">
                  RW
                </AvatarFallback>
              </Avatar>
            </div>
          </motion.header>

          {error ? (
            <div className="mb-4 flex items-center justify-between gap-4 rounded-2xl border border-rose-400/20 bg-rose-400/[0.06] p-4">
              <div className="flex items-center gap-3">
                <AlertTriangle className="size-4 text-rose-300" />
                <div>
                  <div className="text-[0.76rem] font-semibold text-rose-200">
                    Environment telemetry unavailable
                  </div>
                  <div className="mt-1 text-[0.65rem] text-rose-200/55">
                    {error}
                  </div>
                </div>
              </div>
              <Button
                type="button"
                onClick={() => void refreshEnvironment()}
                variant="outline"
                size="sm"
                className="cursor-pointer border-rose-400/20 bg-rose-400/[0.05] text-rose-200 hover:bg-rose-400/10"
              >
                Retry
              </Button>
            </div>
          ) : null}

          <motion.section
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.48, delay: 0.05 }}
            className="mb-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
          >
            {[
              [environment.nodes.length, "visible nodes", "text-sky-300", Shield],
              [
                new Set(environment.nodes.map((node) => node.zone)).size,
                "active zones",
                "text-violet-300",
                GitBranch,
              ],
              [replayFlows.length, "replay events", "text-amber-200", Clock3],
              [detectionCount, "active detections", "text-rose-300", AlertTriangle],
            ].map(([value, label, color, Icon]) => {
              const IconComponent = Icon as typeof Shield;
              return (
                <div
                  key={String(label)}
                  className="surface-card rounded-2xl border border-slate-800/80 p-4"
                >
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div
                        className={cn(
                          "text-[1.5rem] font-bold leading-none tracking-[-0.035em]",
                          color,
                        )}
                      >
                        {String(value)}
                      </div>
                      <div className="mt-2 text-[0.66rem] font-semibold tracking-[0.08em] text-slate-600">
                        {String(label).toUpperCase()}
                      </div>
                    </div>
                    <IconComponent className="size-4 text-slate-700" />
                  </div>
                </div>
              );
            })}
          </motion.section>

          <div className="grid gap-4 2xl:grid-cols-[minmax(0,1fr)_360px]">
            <motion.section
              initial={{ opacity: 0, scale: 0.992 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.5, delay: 0.09 }}
              className="relative overflow-hidden rounded-[1.6rem] border border-slate-800/85 bg-[#080d14]/90 p-3.5 shadow-[0_34px_120px_rgba(0,0,0,0.28)] sm:p-4"
            >
              <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_48%_0%,rgba(56,189,248,0.08),transparent_34%),radial-gradient(circle_at_87%_32%,rgba(139,92,246,0.07),transparent_30%)]" />

              <div className="relative mb-4 flex flex-col gap-3 border-b border-slate-800/75 pb-4 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <div className="text-[0.76rem] font-semibold tracking-[0.04em] text-slate-300">
                    Environment graph
                  </div>
                  <div className="mt-1 text-[0.66rem] leading-5 tracking-[0.02em] text-slate-600">
                    Derived from the latest persisted telemetry · click a node to
                    query its live pivot
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  <Badge
                    variant="outline"
                    className="border-emerald-400/25 bg-emerald-400/10 text-[0.62rem] tracking-[0.08em] text-emerald-300"
                  >
                    <span className="mr-1.5 size-1.5 rounded-full bg-emerald-400" />
                    LIVE DATA
                  </Badge>
                  <Badge
                    variant="outline"
                    className="border-slate-700 bg-slate-950/65 text-[0.62rem] tracking-[0.06em] text-slate-500"
                  >
                    15s refresh
                  </Badge>
                </div>
              </div>

              <div className="relative grid gap-3 xl:grid-cols-3">
                {(["corporate", "dmz", "ot"] as Zone[]).map((zone) => (
                  <ZoneColumn
                    key={zone}
                    zone={zone}
                    nodes={filteredNodes.filter((node) => node.zone === zone)}
                    selectedID={selected?.id ?? ""}
                    activeIDs={activeIDs}
                    onSelect={(id) => {
                      setPivot(null);
                      setSelectedID(id);
                    }}
                  />
                ))}
              </div>

              <div className="relative mt-4 rounded-2xl border border-slate-800/75 bg-slate-950/25 p-4">
                <div className="mb-3 flex items-center justify-between gap-3">
                  <div>
                    <div className="text-[0.7rem] font-semibold text-slate-300">
                      Observed flow ledger
                    </div>
                    <div className="mt-1 text-[0.62rem] text-slate-600">
                      Recent relationships reconstructed from telemetry
                    </div>
                  </div>
                  <Network className="size-4 text-slate-700" />
                </div>

                <div className="grid gap-2 lg:grid-cols-2">
                  {replayFlows.slice(-6).map((flow) => (
                    <div
                      key={flow.id}
                      className="rounded-xl border border-slate-800/70 bg-[#0a0f17] p-3"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <span className="font-mono text-[0.58rem] text-slate-700">
                          {new Date(flow.timestamp).toLocaleTimeString("en-GB")}
                        </span>
                        {flow.detection ? (
                          <Badge
                            variant="outline"
                            className="border-rose-400/25 bg-rose-400/[0.07] text-[0.54rem] font-bold text-rose-300"
                          >
                            {flow.detection.detection_id}
                          </Badge>
                        ) : null}
                      </div>
                      <div className="mt-2 flex items-center gap-2 text-[0.66rem] font-medium text-slate-400">
                        <span className="truncate">{flow.sourceLabel}</span>
                        <ChevronRight className="size-3 shrink-0 text-slate-700" />
                        <span className="truncate">{flow.destinationLabel}</span>
                      </div>
                      <div className="mt-2 font-mono text-[0.56rem] text-slate-700">
                        {flow.category}/{flow.action}
                        {flow.port
                          ? ` · ${flow.protocol ?? "tcp"}:${flow.port}`
                          : ""}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </motion.section>

            <div className="grid content-start gap-4">
              <motion.div
                initial={{ opacity: 0, x: 16 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.45, delay: 0.14 }}
              >
                <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                  <CardHeader className="px-5 pb-4 pt-5">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <div className="text-[0.64rem] font-semibold tracking-[0.11em] text-slate-600">
                          SELECTED NODE
                        </div>
                        <h2 className="mt-2 text-[1rem] font-semibold tracking-[-0.01em] text-white">
                          {selected?.name ?? "No recent node"}
                        </h2>
                      </div>
                      {selected ? (
                        <div
                          className={cn(
                            "size-2.5 rounded-full",
                            stateClasses(selected.state).dot,
                          )}
                        />
                      ) : null}
                    </div>
                  </CardHeader>

                  <Separator className="bg-slate-800/80" />

                  <CardContent className="space-y-4 p-5">
                    {selected ? (
                      <>
                        <div className="grid grid-cols-2 gap-2">
                          <div className="rounded-xl border border-slate-800/75 bg-slate-950/35 p-3">
                            <div className="text-[0.6rem] font-semibold tracking-[0.08em] text-slate-600">
                              IP ADDRESS
                            </div>
                            <div className="mt-2 truncate font-mono text-[0.7rem] text-slate-300">
                              {selected.ip}
                            </div>
                          </div>
                          <div className="rounded-xl border border-slate-800/75 bg-slate-950/35 p-3">
                            <div className="text-[0.6rem] font-semibold tracking-[0.08em] text-slate-600">
                              LAST SEEN
                            </div>
                            <div className="mt-2 text-[0.7rem] font-medium text-slate-300">
                              {selected.lastSeen}
                            </div>
                          </div>
                        </div>

                        {selected.identity ? (
                          <div className="rounded-xl border border-violet-400/15 bg-violet-400/[0.05] p-3">
                            <div className="text-[0.6rem] font-semibold tracking-[0.08em] text-violet-300/60">
                              ASSOCIATED IDENTITY
                            </div>
                            <div className="mt-1 text-[0.76rem] font-semibold text-violet-200">
                              {selected.identity}
                            </div>
                            <div className="mt-1 text-[0.6rem] text-violet-300/45">
                              {selected.identityType ?? "unknown identity type"}
                            </div>
                          </div>
                        ) : null}

                        <div className="grid grid-cols-3 gap-2">
                          {[
                            [
                              selectedPivot?.recent_events.length ?? 0,
                              "events",
                              Activity,
                            ],
                            [
                              selectedPivot?.findings.length ??
                                selected.findings.length,
                              "findings",
                              AlertTriangle,
                            ],
                            [selected.zone.toUpperCase(), "zone", GitBranch],
                          ].map(([value, label, Icon]) => {
                            const IconComponent = Icon as typeof Activity;
                            return (
                              <div
                                key={String(label)}
                                className="rounded-xl border border-slate-800/75 bg-slate-950/35 p-3 text-center"
                              >
                                <IconComponent className="mx-auto size-3.5 text-slate-600" />
                                <div className="mt-2 text-[0.76rem] font-semibold text-slate-300">
                                  {String(value)}
                                </div>
                                <div className="mt-1 text-[0.54rem] font-semibold tracking-[0.08em] text-slate-700">
                                  {String(label).toUpperCase()}
                                </div>
                              </div>
                            );
                          })}
                        </div>

                        {selectedPivot?.findings[0] ? (
                          <div className="rounded-xl border border-rose-400/20 bg-rose-400/[0.05] p-3">
                            <div className="text-[0.58rem] font-semibold tracking-[0.09em] text-rose-300/65">
                              LATEST LINKED FINDING
                            </div>
                            <div className="mt-2 text-[0.72rem] font-semibold leading-5 text-rose-100">
                              {selectedPivot.findings[0].title}
                            </div>
                            <div className="mt-2 font-mono text-[0.58rem] text-rose-300/55">
                              {selectedPivot.findings[0].detection_id} ·{" "}
                              {selectedPivot.findings[0].confidence}% confidence
                            </div>
                          </div>
                        ) : null}
                      </>
                    ) : (
                      <div className="py-8 text-center text-[0.7rem] text-slate-600">
                        No recent environment data.
                      </div>
                    )}
                  </CardContent>
                </Card>
              </motion.div>

              <motion.div
                initial={{ opacity: 0, x: 16 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.45, delay: 0.2 }}
              >
                <Card className="overflow-hidden rounded-2xl border-rose-400/20 bg-gradient-to-b from-rose-500/[0.08] to-[#0c1119] py-0">
                  <CardHeader className="px-5 pb-4 pt-5">
                    <div className="flex items-center gap-2">
                      <ShieldCheck className="size-4 text-rose-300" />
                      <span className="text-[0.67rem] font-bold tracking-[0.11em] text-rose-300">
                        LIVE DETECTION CONTEXT
                      </span>
                    </div>
                  </CardHeader>
                  <Separator className="bg-rose-400/10" />
                  <CardContent className="space-y-3 p-5">
                    {selected?.findings.length ? (
                      selected.findings.slice(0, 3).map((finding) => (
                        <div key={finding.id}>
                          <div className="flex items-start gap-2.5 text-[0.7rem] leading-5 tracking-[0.02em] text-slate-400">
                            <ShieldCheck className="mt-0.5 size-3.5 shrink-0 text-emerald-400" />
                            <span>
                              {finding.title}
                              <span className="mt-1 block font-mono text-[0.58rem] text-slate-700">
                                {finding.detection_id} · {finding.severity} ·{" "}
                                {finding.confidence}%
                              </span>
                            </span>
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="text-[0.68rem] leading-5 text-slate-600">
                        No findings are currently linked to this node in the
                        recent evidence window.
                      </div>
                    )}
                  </CardContent>
                </Card>
              </motion.div>
            </div>
          </div>

          <motion.section
            initial={{ opacity: 0, y: 18 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.18 }}
            transition={{ duration: 0.5 }}
            className="mt-4 overflow-hidden rounded-[1.5rem] border border-slate-800/85 bg-[#0b1018]/92"
          >
            <div className="grid gap-0 xl:grid-cols-[280px_minmax(0,1fr)_330px]">
              <div className="border-b border-slate-800/80 p-5 xl:border-b-0 xl:border-r">
                <div className="flex items-center gap-2 text-[0.68rem] font-bold tracking-[0.12em] text-amber-200">
                  <Clock3 className="size-3.5" />
                  INCIDENT REPLAY
                </div>
                <h2 className="mt-3 text-[1.1rem] font-semibold tracking-[-0.015em] text-white">
                  Reconstruct persisted activity
                </h2>
                <p className="mt-2 text-[0.7rem] leading-5 tracking-[0.025em] text-slate-500">
                  Replay the most recent telemetry relationships in chronological
                  order. Detection context is attached when the underlying event
                  belongs to a finding.
                </p>

                <div className="mt-5 flex items-center gap-2">
                  <Button
                    type="button"
                    onClick={() => setPlaying((value) => !value)}
                    disabled={replayFlows.length === 0}
                    className="cursor-pointer bg-gradient-to-r from-amber-300 to-orange-300 text-[0.72rem] font-semibold text-slate-950 hover:from-amber-200 hover:to-orange-200 disabled:cursor-not-allowed"
                  >
                    {playing ? (
                      <CirclePause className="size-4" />
                    ) : (
                      <CirclePlay className="size-4" />
                    )}
                    {playing ? "Pause" : "Replay"}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    onClick={resetReplay}
                    className="cursor-pointer border-slate-800 bg-slate-950/45 text-slate-400 hover:bg-slate-900"
                  >
                    <RotateCcw className="size-3.5" />
                  </Button>
                </div>

                <div className="mt-5">
                  <div className="mb-2 flex items-center justify-between text-[0.61rem] font-semibold tracking-[0.07em] text-slate-600">
                    <span>EVENT {replayFlows.length ? step + 1 : 0}</span>
                    <span>{replayFlows.length}</span>
                  </div>
                  <div className="h-1.5 overflow-hidden rounded-full bg-slate-900">
                    <motion.div
                      animate={{
                        width: replayFlows.length
                          ? `${((step + 1) / replayFlows.length) * 100}%`
                          : "0%",
                      }}
                      transition={{ duration: 0.35 }}
                      className="h-full rounded-full bg-gradient-to-r from-amber-300 via-orange-300 to-rose-400"
                    />
                  </div>
                </div>
              </div>

              <div className="min-w-0 border-b border-slate-800/80 p-4 xl:border-b-0 xl:border-r">
                <ScrollArea className="h-[430px]">
                  <div className="space-y-1">
                    {replayFlows.map((flow, index) => {
                      const past = index <= safeStep;
                      const current = index === safeStep;

                      return (
                        <button
                          key={flow.id}
                          type="button"
                          onClick={() => {
                            setPlaying(false);
                            setStep(index);
                          }}
                          className={cn(
                            "grid w-full cursor-pointer grid-cols-[66px_18px_minmax(0,1fr)] gap-2 rounded-xl px-3 py-3 text-left transition",
                            current
                              ? "bg-amber-300/[0.07]"
                              : "hover:bg-slate-900/55",
                            !past && "opacity-40",
                          )}
                        >
                          <span className="font-mono text-[0.61rem] text-slate-600">
                            {new Date(flow.timestamp).toLocaleTimeString("en-GB")}
                          </span>
                          <span className="relative mt-1.5 flex justify-center">
                            <span
                              className={cn(
                                "z-10 size-2 rounded-full",
                                current
                                  ? "bg-amber-300 shadow-[0_0_16px_rgba(251,191,36,0.8)]"
                                  : flow.detection
                                    ? "bg-rose-400"
                                    : past
                                      ? "bg-sky-400"
                                      : "bg-slate-700",
                              )}
                            />
                            {index < replayFlows.length - 1 ? (
                              <span className="absolute top-2 h-10 w-px bg-slate-800" />
                            ) : null}
                          </span>
                          <span>
                            <span
                              className={cn(
                                "block text-[0.75rem] font-semibold tracking-[0.01em]",
                                current
                                  ? "text-amber-100"
                                  : "text-slate-300",
                              )}
                            >
                              {flow.sourceLabel} → {flow.destinationLabel}
                            </span>
                            <span className="mt-1 block text-[0.65rem] leading-5 tracking-[0.02em] text-slate-600">
                              {flow.category}/{flow.action}
                              {flow.detection
                                ? ` · ${flow.detection.detection_id}`
                                : ""}
                            </span>
                          </span>
                        </button>
                      );
                    })}
                  </div>
                </ScrollArea>
              </div>

              <div className="p-5">
                <AnimatePresence mode="wait">
                  {currentFlow ? (
                    <motion.div
                      key={currentFlow.id}
                      initial={{ opacity: 0, x: 12 }}
                      animate={{ opacity: 1, x: 0 }}
                      exit={{ opacity: 0, x: -12 }}
                      transition={{ duration: 0.25 }}
                    >
                      <div className="text-[0.62rem] font-semibold tracking-[0.1em] text-slate-600">
                        CURRENT EVIDENCE
                      </div>
                      <div className="mt-3 text-[0.88rem] font-semibold leading-6 text-white">
                        {currentFlow.event.event.category} /{" "}
                        {currentFlow.event.event.action}
                      </div>
                      <div className="mt-2 text-[0.68rem] leading-5 tracking-[0.02em] text-slate-500">
                        {currentFlow.sourceLabel} communicated with or produced
                        activity toward {currentFlow.destinationLabel}.
                      </div>

                      <div className="mt-4 rounded-xl border border-slate-800/80 bg-slate-950/45 p-3">
                        <div className="flex items-center gap-2 font-mono text-[0.64rem] text-slate-400">
                          <span className="truncate">
                            {currentFlow.sourceLabel}
                          </span>
                          <ChevronRight className="size-3 shrink-0 text-slate-700" />
                          <span className="truncate">
                            {currentFlow.destinationLabel}
                          </span>
                        </div>
                      </div>

                      {currentFlow.detection ? (
                        <div className="mt-3 rounded-xl border border-rose-400/20 bg-rose-400/[0.05] p-3">
                          <div className="text-[0.6rem] font-semibold tracking-[0.09em] text-rose-300/70">
                            DETECTION
                          </div>
                          <div className="mt-1.5 font-mono text-[0.72rem] font-semibold text-rose-200">
                            {currentFlow.detection.detection_id}
                          </div>
                          <div className="mt-1.5 text-[0.63rem] leading-5 text-rose-200/60">
                            {currentFlow.detection.title}
                          </div>
                        </div>
                      ) : null}
                    </motion.div>
                  ) : (
                    <div className="grid min-h-56 place-items-center text-center">
                      <div>
                        <Activity className="mx-auto size-5 text-slate-700" />
                        <div className="mt-3 text-[0.72rem] text-slate-600">
                          No replayable activity in the current telemetry window.
                        </div>
                      </div>
                    </div>
                  )}
                </AnimatePresence>
              </div>
            </div>
          </motion.section>
        </section>
      </div>
    </main>
  );
}
