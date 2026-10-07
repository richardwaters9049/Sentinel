"use client";

import {
  AlertTriangle,
  BadgeCheck,
  Bell,
  ChevronRight,
  CircleDot,
  Clock3,
  Fingerprint,
  GitBranch,
  Link2,
  Network,
  RefreshCcw,
  Search,
  ShieldAlert,
  SquareArrowOutUpRight,
  UserRound,
} from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";

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
  getInvestigation,
  listInvestigations,
} from "@/lib/sentinel/client";
import type {
  EvidenceEvent,
  InvestigationDetail,
  InvestigationSummary,
  TimelineEntry,
} from "@/lib/sentinel/types";

type GraphNodeKind = "identity" | "asset" | "network" | "finding" | "detection";

type GraphNode = {
  id: string;
  label: string;
  detail: string;
  kind: GraphNodeKind;
  x: number;
  y: number;
};

type GraphEdge = {
  id: string;
  source: string;
  target: string;
  label: string;
  suspicious?: boolean;
};

const kindMeta = {
  identity: {
    label: "Identity",
    icon: Fingerprint,
    border: "border-violet-400/35",
    glow: "shadow-[0_0_36px_rgba(139,92,246,0.13)]",
    iconClass: "text-violet-300",
    fill: "#a78bfa",
  },
  asset: {
    label: "Asset",
    icon: UserRound,
    border: "border-sky-400/35",
    glow: "shadow-[0_0_36px_rgba(56,189,248,0.12)]",
    iconClass: "text-sky-300",
    fill: "#38bdf8",
  },
  network: {
    label: "Network",
    icon: Network,
    border: "border-cyan-400/35",
    glow: "",
    iconClass: "text-cyan-300",
    fill: "#22d3ee",
  },
  finding: {
    label: "Finding",
    icon: ShieldAlert,
    border: "border-rose-400/40",
    glow: "shadow-[0_0_42px_rgba(251,113,133,0.12)]",
    iconClass: "text-rose-300",
    fill: "#fb7185",
  },
  detection: {
    label: "Detection",
    icon: BadgeCheck,
    border: "border-amber-300/35",
    glow: "",
    iconClass: "text-amber-200",
    fill: "#fcd34d",
  },
} satisfies Record<
  GraphNodeKind,
  {
    label: string;
    icon: typeof Fingerprint;
    border: string;
    glow: string;
    iconClass: string;
    fill: string;
  }
>;

function formatTime(value: string) {
  return new Intl.DateTimeFormat("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
}

function priorityClass(priority: InvestigationSummary["priority"]) {
  switch (priority) {
    case "critical":
      return "border-rose-400/35 bg-rose-400/10 text-rose-300";
    case "high":
      return "border-amber-300/35 bg-amber-300/10 text-amber-200";
    case "medium":
      return "border-violet-400/35 bg-violet-400/10 text-violet-300";
    default:
      return "border-slate-700 bg-slate-800/40 text-slate-400";
  }
}

function graphFromInvestigation(detail: InvestigationDetail) {
  const nodes: GraphNode[] = [];
  const edges: GraphEdge[] = [];
  const seen = new Set<string>();

  const addNode = (node: GraphNode) => {
    if (seen.has(node.id)) return;
    seen.add(node.id);
    nodes.push(node);
  };

  const identities = new Map<string, EvidenceEvent>();
  const assets = new Map<string, EvidenceEvent>();
  const networkValues = new Map<string, string>();

  for (const event of detail.events) {
    if (event.identity_id) identities.set(event.identity_id, event);
    if (event.asset_id) assets.set(event.asset_id, event);

    const sourceIP = event.payload.network?.source_ip;
    if (sourceIP) networkValues.set(`ip:${sourceIP}`, sourceIP);

    const destinationIP = event.payload.network?.destination_ip;
    if (destinationIP) networkValues.set(`ip:${destinationIP}`, destinationIP);
  }

  const identityList = [...identities.entries()];
  const assetList = [...assets.entries()];
  const networkList = [...networkValues.entries()];
  const findingList = detail.findings;

  identityList.forEach(([id, event], index) => {
    addNode({
      id: `identity:${id}`,
      label: event.payload.actor?.name || id,
      detail: event.payload.actor?.type || id,
      kind: "identity",
      x: 50,
      y: 100 + index * 145,
    });
  });

  assetList.forEach(([id, event], index) => {
    addNode({
      id: `asset:${id}`,
      label: event.payload.asset?.hostname || id,
      detail: event.payload.asset?.zone
        ? `${event.payload.asset.zone} zone`
        : id,
      kind: "asset",
      x: 300,
      y: 80 + index * 145,
    });
  });

  networkList.forEach(([id, ip], index) => {
    addNode({
      id,
      label: ip,
      detail: "network endpoint",
      kind: "network",
      x: 550,
      y: 80 + index * 118,
    });
  });

  findingList.forEach((finding, index) => {
    addNode({
      id: `finding:${finding.id}`,
      label: finding.title,
      detail: `${finding.severity.toUpperCase()} · ${finding.confidence}% confidence`,
      kind: "finding",
      x: 805,
      y: 75 + index * 165,
    });

    addNode({
      id: `detection:${finding.detection_id}`,
      label: finding.detection_id,
      detail: `version ${finding.detection_version}`,
      kind: "detection",
      x: 1070,
      y: 95 + index * 165,
    });

    edges.push({
      id: `edge-detection-${finding.id}`,
      source: `finding:${finding.id}`,
      target: `detection:${finding.detection_id}`,
      label: "triggered by",
      suspicious: true,
    });
  });

  for (const event of detail.events) {
    if (event.identity_id && event.asset_id) {
      edges.push({
        id: `edge-identity-asset-${event.id}`,
        source: `identity:${event.identity_id}`,
        target: `asset:${event.asset_id}`,
        label: event.action || event.category,
      });
    }

    const sourceIP = event.payload.network?.source_ip;
    if (event.asset_id && sourceIP) {
      edges.push({
        id: `edge-asset-source-${event.id}`,
        source: `asset:${event.asset_id}`,
        target: `ip:${sourceIP}`,
        label: "source IP",
      });
    }

    const destinationIP = event.payload.network?.destination_ip;
    if (sourceIP && destinationIP) {
      edges.push({
        id: `edge-network-${event.id}`,
        source: `ip:${sourceIP}`,
        target: `ip:${destinationIP}`,
        label: event.payload.network?.destination_port
          ? `${event.payload.network.protocol ?? "tcp"}:${event.payload.network.destination_port}`
          : event.category,
        suspicious: true,
      });
    }
  }

  for (const finding of detail.findings) {
    const evidenceEventIDs = Array.isArray(finding.evidence.event_ids)
      ? finding.evidence.event_ids.filter(
          (item): item is string => typeof item === "string",
        )
      : [];

    for (const eventID of evidenceEventIDs) {
      const event = detail.events.find((candidate) => candidate.id === eventID);
      if (!event) continue;

      const destinationIP = event.payload.network?.destination_ip;
      const sourceIP = event.payload.network?.source_ip;

      if (destinationIP && seen.has(`ip:${destinationIP}`)) {
        edges.push({
          id: `edge-evidence-dest-${finding.id}-${eventID}`,
          source: `ip:${destinationIP}`,
          target: `finding:${finding.id}`,
          label: "evidence",
          suspicious: true,
        });
      } else if (sourceIP && seen.has(`ip:${sourceIP}`)) {
        edges.push({
          id: `edge-evidence-source-${finding.id}-${eventID}`,
          source: `ip:${sourceIP}`,
          target: `finding:${finding.id}`,
          label: "evidence",
          suspicious: true,
        });
      } else if (event.asset_id && seen.has(`asset:${event.asset_id}`)) {
        edges.push({
          id: `edge-evidence-asset-${finding.id}-${eventID}`,
          source: `asset:${event.asset_id}`,
          target: `finding:${finding.id}`,
          label: "evidence",
          suspicious: true,
        });
      }
    }
  }

  return { nodes, edges };
}

function NodeCard({
  node,
  selected,
  onSelect,
}: {
  node: GraphNode;
  selected: boolean;
  onSelect: () => void;
}) {
  const meta = kindMeta[node.kind];
  const Icon = meta.icon;

  return (
    <motion.button
      type="button"
      onClick={onSelect}
      initial={{ opacity: 0, scale: 0.94 }}
      animate={{ opacity: 1, scale: selected ? 1.025 : 1 }}
      whileHover={{ y: -2 }}
      transition={{ type: "spring", stiffness: 280, damping: 24 }}
      style={{ left: node.x, top: node.y }}
      className={cn(
        "absolute z-10 w-[205px] cursor-pointer rounded-2xl border bg-[#0b1119]/96 p-3.5 text-left transition",
        meta.border,
        meta.glow,
        selected && "ring-1 ring-white/20",
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div
          className={cn(
            "grid size-8 place-items-center rounded-xl border bg-slate-950/70",
            meta.border,
          )}
        >
          <Icon className={cn("size-4", meta.iconClass)} />
        </div>
        <span className="text-[0.58rem] font-semibold tracking-[0.1em] text-slate-600">
          {meta.label.toUpperCase()}
        </span>
      </div>
      <div className="mt-3 truncate text-[0.74rem] font-semibold tracking-[0.01em] text-slate-200">
        {node.label}
      </div>
      <div className="mt-1 truncate text-[0.62rem] leading-5 tracking-[0.02em] text-slate-600">
        {node.detail}
      </div>
    </motion.button>
  );
}

function EvidenceTimeline({ timeline }: { timeline: TimelineEntry[] }) {
  return (
    <div className="space-y-1">
      {timeline.map((item, index) => (
        <motion.div
          key={item.id}
          initial={{ opacity: 0, x: 10 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: index * 0.025 }}
          className="grid grid-cols-[62px_18px_minmax(0,1fr)] gap-2 rounded-xl px-2 py-2.5 hover:bg-slate-900/45"
        >
          <span className="font-mono text-[0.58rem] text-slate-600">
            {formatTime(item.timestamp)}
          </span>
          <span className="relative mt-1.5 flex justify-center">
            <span
              className={cn(
                "z-10 size-2 rounded-full",
                item.type === "finding"
                  ? "bg-rose-400"
                  : item.type === "event"
                    ? "bg-sky-400"
                    : item.type === "note"
                      ? "bg-violet-400"
                      : "bg-slate-600",
              )}
            />
            {index < timeline.length - 1 ? (
              <span className="absolute top-2 h-9 w-px bg-slate-800" />
            ) : null}
          </span>
          <span>
            <span className="block text-[0.7rem] font-medium text-slate-300">
              {item.summary}
            </span>
            <span className="mt-0.5 block text-[0.58rem] font-semibold tracking-[0.08em] text-slate-700">
              {item.type.toUpperCase()}
            </span>
          </span>
        </motion.div>
      ))}
    </div>
  );
}

export default function InvestigationWorkspace() {
  const searchParams = useSearchParams();
  const requestedInvestigationID = searchParams.get("id") ?? "";
  const [investigations, setInvestigations] = useState<InvestigationSummary[]>(
    [],
  );
  const [selectedID, setSelectedID] = useState("");
  const [detail, setDetail] = useState<InvestigationDetail | null>(null);
  const [selectedNodeID, setSelectedNodeID] = useState("");
  const [loadingList, setLoadingList] = useState(true);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState("");

  async function loadList() {
    setLoadingList(true);
    setError("");

    try {
      const response = await listInvestigations();
      setInvestigations(response.investigations);
      setSelectedID((current) => current || response.investigations[0]?.id || "");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not load investigations");
    } finally {
      setLoadingList(false);
    }
  }

  useEffect(() => {
    const controller = new AbortController();

    void listInvestigations(controller.signal)
      .then((response) => {
        setInvestigations(response.investigations);
        const requestedExists = response.investigations.some(
          (item) => item.id === requestedInvestigationID,
        );
        const firstID = requestedExists
          ? requestedInvestigationID
          : response.investigations[0]?.id ?? "";
        if (firstID) {
          setLoadingDetail(true);
          setSelectedID(firstID);
        }
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(
          cause instanceof Error
            ? cause.message
            : "Could not load investigations",
        );
      })
      .finally(() => setLoadingList(false));

    return () => controller.abort();
  }, [requestedInvestigationID]);

  useEffect(() => {
    if (!selectedID) return;

    const controller = new AbortController();

    void getInvestigation(selectedID, controller.signal)
      .then((response) => {
        setDetail(response);
        setSelectedNodeID("");
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(
          cause instanceof Error ? cause.message : "Could not load investigation",
        );
      })
      .finally(() => setLoadingDetail(false));

    return () => controller.abort();
  }, [selectedID]);

  const graph = useMemo(
    () => (detail ? graphFromInvestigation(detail) : { nodes: [], edges: [] }),
    [detail],
  );

  const nodeByID = useMemo(
    () => new Map(graph.nodes.map((node) => [node.id, node])),
    [graph.nodes],
  );

  const selectedNode = nodeByID.get(selectedNodeID) ?? graph.nodes[0];

  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto min-h-screen max-w-[1820px]">
        <section className="min-w-0 px-4 py-5 sm:px-6 lg:px-8 xl:px-10">
          <motion.header
            initial={{ opacity: 0, y: -12 }}
            animate={{ opacity: 1, y: 0 }}
            className="mb-5 flex items-center justify-between gap-4"
          >
            <div className="min-w-0">
              <div className="flex items-center gap-3">
                <SidebarDrawer />
                <span className="text-[0.72rem] font-bold tracking-[0.15em] text-slate-500">
                  SENTINEL
                </span>
              </div>

              <div className="mt-3 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.13em] text-violet-300/75 ">
                <GitBranch className="size-3.5" />
                INVESTIGATION GRAPH
              </div>
              <h1 className="mt-2 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.035em] text-white sm:text-[1.95rem]">
                Evidence relationship explorer
              </h1>
              <p className="mt-1.5 max-w-3xl text-[0.78rem] leading-6 tracking-[0.025em] text-slate-500">
                Follow identities, assets, network endpoints, findings and
                detections as one connected investigation.
              </p>
            </div>

            <div className="flex items-center gap-2.5">
              <Button
                type="button"
                onClick={() => void loadList()}
                variant="outline"
                size="icon"
                className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900"
              >
                <RefreshCcw className={cn("size-4", loadingList && "animate-spin")} />
              </Button>
              <Button
                variant="outline"
                size="icon"
                className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900"
              >
                <Bell className="size-4" />
              </Button>
              <Avatar className="size-9 border border-violet-400/25">
                <AvatarFallback className="bg-gradient-to-br from-violet-400 to-sky-400 text-[0.72rem] font-bold text-slate-950">
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
                    Sentinel gateway data unavailable
                  </div>
                  <div className="mt-1 text-[0.66rem] text-rose-200/55">
                    {error}
                  </div>
                </div>
              </div>
              <Button
                type="button"
                onClick={() => void loadList()}
                variant="outline"
                size="sm"
                className="cursor-pointer border-rose-400/20 bg-rose-400/[0.05] text-rose-200 hover:bg-rose-400/10"
              >
                Retry
              </Button>
            </div>
          ) : null}

          <section className="grid gap-4 xl:grid-cols-[310px_minmax(0,1fr)]">
            <motion.div
              initial={{ opacity: 0, x: -14 }}
              animate={{ opacity: 1, x: 0 }}
              className="surface-card overflow-hidden rounded-[1.4rem] border border-slate-800/85"
            >
              <div className="p-4">
                <div className="relative">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-slate-600" />
                  <Input
                    placeholder="Search cases…"
                    className="h-9 border-slate-800 bg-slate-950/55 pl-9 text-[0.7rem] placeholder:text-slate-700"
                  />
                </div>
              </div>
              <Separator className="bg-slate-800/80" />
              <ScrollArea className="h-[720px]">
                <div className="space-y-2 p-3">
                  {loadingList && investigations.length === 0 ? (
                    <div className="grid h-40 place-items-center text-[0.7rem] text-slate-600">
                      Loading investigations…
                    </div>
                  ) : null}

                  {investigations.map((item) => (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => {
                        setLoadingDetail(true);
                        setSelectedID(item.id);
                      }}
                      className={cn(
                        "w-full cursor-pointer rounded-xl border p-3.5 text-left transition",
                        item.id === selectedID
                          ? "border-violet-400/30 bg-violet-400/[0.07]"
                          : "border-slate-800/70 bg-slate-950/25 hover:border-slate-700 hover:bg-slate-900/55",
                      )}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <Badge
                          variant="outline"
                          className={cn(
                            "rounded-md px-2 py-0.5 text-[0.57rem] font-bold tracking-[0.08em]",
                            priorityClass(item.priority),
                          )}
                        >
                          {item.priority.toUpperCase()}
                        </Badge>
                        <span className="text-[0.58rem] font-medium text-slate-700">
                          {item.status}
                        </span>
                      </div>
                      <div className="mt-3 line-clamp-2 text-[0.72rem] font-semibold leading-5 text-slate-300">
                        {item.title}
                      </div>
                      <div className="mt-3 flex items-center justify-between gap-2 text-[0.6rem] text-slate-600">
                        <span>{item.owner_id ?? "unassigned"}</span>
                        <ChevronRight className="size-3" />
                      </div>
                    </button>
                  ))}
                </div>
              </ScrollArea>
            </motion.div>

            <div className="min-w-0">
              <AnimatePresence mode="wait">
                {detail && !loadingDetail ? (
                  <motion.div
                    key={detail.id}
                    initial={{ opacity: 0, y: 12 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -8 }}
                    transition={{ duration: 0.28 }}
                    className="grid gap-4"
                  >
                    <Card className="surface-card rounded-[1.4rem] border-slate-800/85 bg-transparent py-0">
                      <CardHeader className="px-5 pb-4 pt-5">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                          <div>
                            <div className="flex flex-wrap items-center gap-2">
                              <Badge
                                variant="outline"
                                className={cn(
                                  "rounded-md px-2 py-0.5 text-[0.58rem] font-bold tracking-[0.08em]",
                                  priorityClass(detail.priority),
                                )}
                              >
                                {detail.priority.toUpperCase()}
                              </Badge>
                              <Badge
                                variant="outline"
                                className="rounded-md border-slate-700 bg-slate-800/35 px-2 py-0.5 text-[0.58rem] font-semibold tracking-[0.08em] text-slate-400"
                              >
                                {detail.status.toUpperCase()}
                              </Badge>
                            </div>
                            <h2 className="mt-3 text-[1.15rem] font-semibold tracking-[-0.015em] text-white">
                              {detail.title}
                            </h2>
                            <p className="mt-2 max-w-3xl text-[0.7rem] leading-5 tracking-[0.02em] text-slate-500">
                              {detail.description ||
                                "Investigation assembled from Sentinel findings and evidence."}
                            </p>
                          </div>
                          <div className="grid grid-cols-3 gap-2 text-center">
                            {[
                              [detail.findings.length, "findings"],
                              [detail.events.length, "events"],
                              [detail.timeline.length, "timeline"],
                            ].map(([value, label]) => (
                              <div
                                key={String(label)}
                                className="rounded-xl border border-slate-800/80 bg-slate-950/35 px-3 py-2.5"
                              >
                                <div className="text-[0.92rem] font-bold text-slate-200">
                                  {value}
                                </div>
                                <div className="mt-1 text-[0.55rem] font-semibold tracking-[0.08em] text-slate-700">
                                  {String(label).toUpperCase()}
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      </CardHeader>
                    </Card>

                    <div className="grid gap-4 2xl:grid-cols-[minmax(0,1fr)_330px]">
                      <section className="overflow-hidden rounded-[1.4rem] border border-slate-800/85 bg-[#080d14]/92">
                        <div className="flex flex-col gap-3 border-b border-slate-800/80 p-4 sm:flex-row sm:items-center sm:justify-between">
                          <div>
                            <div className="text-[0.74rem] font-semibold text-slate-300">
                              Relationship graph
                            </div>
                            <div className="mt-1 text-[0.63rem] leading-5 text-slate-600">
                              Click a node to inspect its role in this investigation
                            </div>
                          </div>
                          <div className="flex flex-wrap gap-1.5">
                            {(
                              Object.entries(kindMeta) as [
                                GraphNodeKind,
                                (typeof kindMeta)[GraphNodeKind],
                              ][]
                            ).map(([kind, meta]) => (
                              <Badge
                                key={kind}
                                variant="outline"
                                className={cn(
                                  "rounded-md bg-slate-950/45 px-2 py-0.5 text-[0.55rem] tracking-[0.06em] text-slate-500",
                                  meta.border,
                                )}
                              >
                                {meta.label}
                              </Badge>
                            ))}
                          </div>
                        </div>

                        <div className="overflow-x-auto">
                          <div className="relative h-[620px] min-w-[1320px]">
                            <svg
                              className="pointer-events-none absolute inset-0 h-full w-full"
                              viewBox="0 0 1320 620"
                              aria-hidden="true"
                            >
                              <defs>
                                <linearGradient id="graphEdge" x1="0" x2="1">
                                  <stop
                                    offset="0%"
                                    stopColor="#334155"
                                    stopOpacity="0.55"
                                  />
                                  <stop
                                    offset="100%"
                                    stopColor="#64748b"
                                    stopOpacity="0.3"
                                  />
                                </linearGradient>
                                <linearGradient id="graphAlert" x1="0" x2="1">
                                  <stop
                                    offset="0%"
                                    stopColor="#f59e0b"
                                    stopOpacity="0.72"
                                  />
                                  <stop
                                    offset="100%"
                                    stopColor="#fb7185"
                                    stopOpacity="0.9"
                                  />
                                </linearGradient>
                              </defs>

                              {graph.edges.map((edge) => {
                                const source = nodeByID.get(edge.source);
                                const target = nodeByID.get(edge.target);
                                if (!source || !target) return null;

                                const x1 = source.x + 205;
                                const y1 = source.y + 48;
                                const x2 = target.x;
                                const y2 = target.y + 48;
                                const mid = (x1 + x2) / 2;

                                return (
                                  <g key={edge.id}>
                                    <path
                                      d={`M ${x1} ${y1} C ${mid} ${y1}, ${mid} ${y2}, ${x2} ${y2}`}
                                      fill="none"
                                      stroke={
                                        edge.suspicious
                                          ? "url(#graphAlert)"
                                          : "url(#graphEdge)"
                                      }
                                      strokeWidth={edge.suspicious ? 2.2 : 1.4}
                                      strokeDasharray={
                                        edge.suspicious ? "8 7" : "5 7"
                                      }
                                    />
                                    {edge.suspicious ? (
                                      <motion.circle
                                        r="4"
                                        fill="#fb7185"
                                        animate={{
                                          cx: [x1, x2],
                                          cy: [y1, y2],
                                          opacity: [0, 1, 0],
                                        }}
                                        transition={{
                                          duration: 1.8,
                                          repeat: Infinity,
                                          ease: "linear",
                                        }}
                                      />
                                    ) : null}
                                  </g>
                                );
                              })}
                            </svg>

                            {graph.nodes.map((node) => (
                              <NodeCard
                                key={node.id}
                                node={node}
                                selected={selectedNode?.id === node.id}
                                onSelect={() => setSelectedNodeID(node.id)}
                              />
                            ))}

                            {graph.nodes.length === 0 ? (
                              <div className="absolute inset-0 grid place-items-center">
                                <div className="text-center">
                                  <Link2 className="mx-auto size-5 text-slate-700" />
                                  <div className="mt-3 text-[0.76rem] font-medium text-slate-500">
                                    No linked evidence yet
                                  </div>
                                </div>
                              </div>
                            ) : null}
                          </div>
                        </div>
                      </section>

                      <div className="grid content-start gap-4">
                        <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                          <CardHeader className="px-5 pb-4 pt-5">
                            <div className="text-[0.62rem] font-semibold tracking-[0.1em] text-slate-600">
                              NODE INSPECTOR
                            </div>
                            <h3 className="mt-2 text-[0.96rem] font-semibold text-white">
                              {selectedNode?.label ?? "Select a node"}
                            </h3>
                          </CardHeader>
                          <Separator className="bg-slate-800/80" />
                          <CardContent className="space-y-3 p-5">
                            {selectedNode ? (
                              <>
                                <div className="rounded-xl border border-slate-800/75 bg-slate-950/40 p-3">
                                  <div className="text-[0.58rem] font-semibold tracking-[0.09em] text-slate-700">
                                    TYPE
                                  </div>
                                  <div className="mt-1.5 text-[0.7rem] font-medium text-slate-300">
                                    {kindMeta[selectedNode.kind].label}
                                  </div>
                                </div>
                                <div className="rounded-xl border border-slate-800/75 bg-slate-950/40 p-3">
                                  <div className="text-[0.58rem] font-semibold tracking-[0.09em] text-slate-700">
                                    CONTEXT
                                  </div>
                                  <div className="mt-1.5 text-[0.68rem] leading-5 text-slate-400">
                                    {selectedNode.detail}
                                  </div>
                                </div>
                                <Button className="w-full cursor-pointer bg-gradient-to-r from-violet-400 to-sky-400 text-[0.7rem] font-semibold text-slate-950 hover:from-violet-300 hover:to-sky-300">
                                  <SquareArrowOutUpRight className="size-3.5" />
                                  Pivot from node
                                </Button>
                              </>
                            ) : (
                              <div className="text-[0.68rem] text-slate-600">
                                Select any graph node to inspect it.
                              </div>
                            )}
                          </CardContent>
                        </Card>

                        <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                          <CardHeader className="px-5 pb-4 pt-5">
                            <div className="flex items-center gap-2">
                              <Clock3 className="size-3.5 text-sky-400" />
                              <h3 className="text-[0.86rem] font-semibold text-slate-200">
                                Evidence timeline
                              </h3>
                            </div>
                          </CardHeader>
                          <Separator className="bg-slate-800/80" />
                          <CardContent className="p-3">
                            <ScrollArea className="h-[330px]">
                              <EvidenceTimeline timeline={detail.timeline} />
                            </ScrollArea>
                          </CardContent>
                        </Card>
                      </div>
                    </div>
                  </motion.div>
                ) : (
                  <motion.div
                    key="loading"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="grid min-h-[640px] place-items-center rounded-[1.4rem] border border-slate-800/85 bg-[#080d14]/70"
                  >
                    <div className="text-center">
                      <CircleDot className="mx-auto size-5 animate-pulse text-violet-400" />
                      <div className="mt-3 text-[0.75rem] font-medium text-slate-500">
                        {selectedID
                          ? "Loading investigation graph…"
                          : "Choose an investigation"}
                      </div>
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </section>
        </section>
      </div>
    </main>
  );
}
