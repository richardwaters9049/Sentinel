"use client";

import {
  Activity,
  AlertTriangle,
  Bell,
  ChevronRight,
  CirclePause,
  CirclePlay,
  Clock3,
  Cpu,
  Database,
  Fingerprint,
  GitBranch,
  Laptop,
  Menu,
  Network,
  Radar,
  RotateCcw,
  Search,
  Server,
  Shield,
  ShieldCheck,
  SquareTerminal,
  UserRound,
  Waypoints,
} from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

import { SidebarContent } from "@/components/sentinel/dashboard";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";

type Zone = "corporate" | "dmz" | "ot";
type AssetState = "healthy" | "observed" | "elevated" | "critical";

type Asset = {
  id: string;
  name: string;
  subtitle: string;
  zone: Zone;
  state: AssetState;
  icon: typeof Laptop;
  ip: string;
  identity?: string;
  lastSeen: string;
  findings: number;
};

type ReplayEvent = {
  id: string;
  time: string;
  title: string;
  detail: string;
  source: string;
  target: string;
  detection?: string;
  severity?: "medium" | "high";
};

const assets: Asset[] = [
  {
    id: "employee-ws-01",
    name: "Employee WS-01",
    subtitle: "Corporate workstation",
    zone: "corporate",
    state: "elevated",
    icon: Laptop,
    ip: "10.10.10.44",
    identity: "operator-07",
    lastSeen: "8s ago",
    findings: 1,
  },
  {
    id: "identity-01",
    name: "Identity Server",
    subtitle: "Directory services",
    zone: "corporate",
    state: "observed",
    icon: Fingerprint,
    ip: "10.10.10.10",
    lastSeen: "4s ago",
    findings: 1,
  },
  {
    id: "app-01",
    name: "Application Server",
    subtitle: "Internal services",
    zone: "corporate",
    state: "healthy",
    icon: Server,
    ip: "10.10.10.20",
    lastSeen: "11s ago",
    findings: 0,
  },
  {
    id: "jump-host-01",
    name: "Jump Host",
    subtitle: "Approved security path",
    zone: "dmz",
    state: "observed",
    icon: SquareTerminal,
    ip: "10.20.0.10",
    identity: "svc-backup",
    lastSeen: "6s ago",
    findings: 1,
  },
  {
    id: "historian-01",
    name: "Historian",
    subtitle: "Process data archive",
    zone: "dmz",
    state: "healthy",
    icon: Database,
    ip: "10.20.0.20",
    lastSeen: "15s ago",
    findings: 0,
  },
  {
    id: "gateway-01",
    name: "Telemetry Gateway",
    subtitle: "Collector edge",
    zone: "dmz",
    state: "healthy",
    icon: Radar,
    ip: "10.20.0.30",
    lastSeen: "2s ago",
    findings: 0,
  },
  {
    id: "engineering-ws-01",
    name: "Engineering WS",
    subtitle: "OT engineering station",
    zone: "ot",
    state: "critical",
    icon: Cpu,
    ip: "10.30.0.12",
    identity: "svc-backup",
    lastSeen: "3s ago",
    findings: 2,
  },
  {
    id: "hmi-01",
    name: "HMI-01",
    subtitle: "Operator interface",
    zone: "ot",
    state: "observed",
    icon: UserRound,
    ip: "10.30.0.20",
    lastSeen: "5s ago",
    findings: 0,
  },
  {
    id: "plc-01",
    name: "PLC-01",
    subtitle: "Synthetic controller",
    zone: "ot",
    state: "critical",
    icon: Network,
    ip: "10.30.0.40",
    lastSeen: "1s ago",
    findings: 1,
  },
];

const replayEvents: ReplayEvent[] = [
  {
    id: "evt-01",
    time: "21:41:02",
    title: "Authentication failures begin",
    detail: "operator-07 generated repeated failed logins from Employee WS-01.",
    source: "employee-ws-01",
    target: "identity-01",
  },
  {
    id: "evt-02",
    time: "21:41:33",
    title: "Authentication succeeds",
    detail: "Successful authentication follows the failure burst.",
    source: "employee-ws-01",
    target: "identity-01",
    detection: "DET-AUTH-001",
    severity: "medium",
  },
  {
    id: "evt-03",
    time: "21:42:06",
    title: "Service identity appears interactively",
    detail: "svc-backup is observed on the approved jump host.",
    source: "identity-01",
    target: "jump-host-01",
    detection: "DET-AUTH-002",
    severity: "high",
  },
  {
    id: "evt-04",
    time: "21:42:39",
    title: "Corporate path crosses into OT",
    detail: "The service identity reaches the engineering workstation.",
    source: "jump-host-01",
    target: "engineering-ws-01",
    detection: "DET-NET-001",
    severity: "high",
  },
  {
    id: "evt-05",
    time: "21:43:10",
    title: "PLC communication observed",
    detail: "Engineering WS communicates with PLC-01 over TCP/502.",
    source: "engineering-ws-01",
    target: "plc-01",
    detection: "DET-NET-001",
    severity: "high",
  },
];

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
} satisfies Record<Zone, { label: string; subtitle: string; accent: string; border: string; bg: string }>;

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

function AssetNode({
  asset,
  selected,
  active,
  onSelect,
}: {
  asset: Asset;
  selected: boolean;
  active: boolean;
  onSelect: () => void;
}) {
  const Icon = asset.icon;
  const classes = stateClasses(asset.state);

  return (
    <motion.button
      type="button"
      onClick={onSelect}
      animate={{
        scale: active ? 1.025 : 1,
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
          {asset.state.toUpperCase()}
        </div>
      </div>

      <div className="mt-3 text-[0.78rem] font-semibold tracking-[0.01em] text-slate-200">
        {asset.name}
      </div>
      <div className="mt-1 text-[0.65rem] leading-5 tracking-[0.02em] text-slate-600">
        {asset.subtitle}
      </div>

      <div className="mt-3 flex items-center justify-between gap-2 font-mono text-[0.6rem] text-slate-600">
        <span>{asset.ip}</span>
        {asset.findings > 0 ? (
          <span className="rounded-md bg-rose-400/10 px-1.5 py-0.5 text-rose-300">
            {asset.findings} finding{asset.findings === 1 ? "" : "s"}
          </span>
        ) : (
          <span>{asset.lastSeen}</span>
        )}
      </div>
    </motion.button>
  );
}

function ZoneColumn({
  zone,
  assetsInZone,
  selectedId,
  activeIds,
  onSelect,
}: {
  zone: Zone;
  assetsInZone: Asset[];
  selectedId: string;
  activeIds: Set<string>;
  onSelect: (id: string) => void;
}) {
  const meta = zoneMeta[zone];

  return (
    <div
      className={cn(
        "relative rounded-[1.4rem] border bg-gradient-to-b p-3.5",
        meta.border,
        meta.bg,
      )}
    >
      <div className="mb-4 border-b border-slate-800/65 pb-3">
        <div className={cn("text-[0.72rem] font-bold tracking-[0.12em]", meta.accent)}>
          {meta.label.toUpperCase()}
        </div>
        <div className="mt-1 text-[0.64rem] leading-5 tracking-[0.02em] text-slate-600">
          {meta.subtitle}
        </div>
      </div>

      <div className="grid gap-3">
        {assetsInZone.map((asset) => (
          <AssetNode
            key={asset.id}
            asset={asset}
            selected={asset.id === selectedId}
            active={activeIds.has(asset.id)}
            onSelect={() => onSelect(asset.id)}
          />
        ))}
      </div>
    </div>
  );
}

export default function EnvironmentWorkspace() {
  const [selectedId, setSelectedId] = useState("engineering-ws-01");
  const [step, setStep] = useState(4);
  const [playing, setPlaying] = useState(false);

  const selected = assets.find((asset) => asset.id === selectedId) ?? assets[0];
  const currentEvent = replayEvents[Math.min(step, replayEvents.length - 1)];

  const activeIds = useMemo(() => {
    const ids = new Set<string>();
    for (let index = 0; index <= step; index += 1) {
      const event = replayEvents[index];
      if (!event) continue;
      ids.add(event.source);
      ids.add(event.target);
    }
    return ids;
  }, [step]);

  useEffect(() => {
    if (!playing) return;

    const id = window.setInterval(() => {
      setStep((current) => {
        if (current >= replayEvents.length - 1) {
          setPlaying(false);
          return current;
        }
        return current + 1;
      });
    }, 1450);

    return () => window.clearInterval(id);
  }, [playing]);

  function resetReplay() {
    setPlaying(false);
    setStep(0);
  }

  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto grid min-h-screen max-w-[1780px] lg:grid-cols-[242px_minmax(0,1fr)]">
        <aside className="sticky top-0 hidden h-screen border-r border-slate-800/80 bg-[#090d14]/95 px-5 py-6 backdrop-blur-xl lg:block">
          <SidebarContent />
        </aside>

        <section className="min-w-0 px-4 py-5 sm:px-6 lg:px-8 xl:px-10">
          <motion.header
            initial={{ opacity: 0, y: -12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.42 }}
            className="mb-5 flex items-center justify-between gap-4"
          >
            <div className="min-w-0">
              <div className="flex items-center gap-3 lg:hidden">
                <Sheet>
                  <SheetTrigger
                    render={
                      <Button
                        variant="outline"
                        size="icon"
                        className="cursor-pointer border-slate-800 bg-slate-950/70 text-slate-300 hover:bg-slate-900"
                      />
                    }
                  >
                    <Menu className="size-4" />
                  </SheetTrigger>
                  <SheetContent
                    side="left"
                    className="w-[280px] border-slate-800 bg-[#090d14] p-5 text-slate-100"
                  >
                    <SheetTitle className="sr-only">
                      Sentinel navigation
                    </SheetTitle>
                    <SidebarContent />
                  </SheetContent>
                </Sheet>

                <span className="text-sm font-bold tracking-[0.12em] text-white">
                  SENTINEL
                </span>
              </div>

              <div className="mt-3 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.13em] text-cyan-300/70 lg:mt-0">
                <Waypoints className="size-3.5" />
                LIVE ENVIRONMENT
              </div>
              <h1 className="mt-2 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.035em] text-white sm:text-[1.95rem]">
                Northstar attack surface
              </h1>
              <p className="mt-1.5 max-w-3xl text-[0.78rem] leading-6 tracking-[0.025em] text-slate-500">
                Explore asset relationships, follow identity movement, and replay
                the evidence chain that produced Sentinel findings.
              </p>
            </div>

            <div className="flex items-center gap-2.5">
              <div className="relative hidden w-[17rem] xl:block">
                <Search className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-600" />
                <Input
                  aria-label="Search environment"
                  placeholder="Find asset, identity or IP…"
                  className="h-10 border-slate-800 bg-slate-950/60 pl-10 text-[0.78rem] tracking-[0.02em] text-slate-200 placeholder:text-slate-600"
                />
              </div>
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

          <motion.section
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.48, delay: 0.05 }}
            className="mb-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
          >
            {[
              ["9", "visible assets", "text-sky-300", Shield],
              ["3", "security zones", "text-violet-300", GitBranch],
              ["5", "replay events", "text-amber-200", Clock3],
              ["3", "active detections", "text-rose-300", AlertTriangle],
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
                    Synthetic Northstar topology · click any node to pivot
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  <Badge
                    variant="outline"
                    className="border-emerald-400/25 bg-emerald-400/10 text-[0.62rem] tracking-[0.08em] text-emerald-300"
                  >
                    <span className="mr-1.5 size-1.5 rounded-full bg-emerald-400" />
                    LIVE
                  </Badge>
                  <Badge
                    variant="outline"
                    className="border-slate-700 bg-slate-950/65 text-[0.62rem] tracking-[0.06em] text-slate-500"
                  >
                    2s telemetry delay
                  </Badge>
                </div>
              </div>

              <div className="relative grid gap-3 xl:grid-cols-3">
                {(["corporate", "dmz", "ot"] as Zone[]).map((zone) => (
                  <ZoneColumn
                    key={zone}
                    zone={zone}
                    assetsInZone={assets.filter((asset) => asset.zone === zone)}
                    selectedId={selectedId}
                    activeIds={activeIds}
                    onSelect={setSelectedId}
                  />
                ))}

                <div className="pointer-events-none absolute inset-0 hidden xl:block">
                  <svg
                    className="h-full w-full"
                    viewBox="0 0 1200 590"
                    preserveAspectRatio="none"
                    aria-hidden="true"
                  >
                    <defs>
                      <linearGradient id="edgeNormal" x1="0" x2="1">
                        <stop offset="0%" stopColor="#38bdf8" stopOpacity="0.18" />
                        <stop offset="100%" stopColor="#8b5cf6" stopOpacity="0.22" />
                      </linearGradient>
                      <linearGradient id="edgeAlert" x1="0" x2="1">
                        <stop offset="0%" stopColor="#f59e0b" stopOpacity="0.7" />
                        <stop offset="100%" stopColor="#fb7185" stopOpacity="0.9" />
                      </linearGradient>
                    </defs>

                    <path
                      d="M 355 118 C 410 118 430 118 480 118"
                      fill="none"
                      stroke="url(#edgeNormal)"
                      strokeWidth="2"
                      strokeDasharray="7 8"
                    />
                    <path
                      d="M 720 118 C 770 118 795 118 845 118"
                      fill="none"
                      stroke={step >= 3 ? "url(#edgeAlert)" : "url(#edgeNormal)"}
                      strokeWidth={step >= 3 ? 3 : 2}
                      strokeDasharray="7 8"
                    />
                    <path
                      d="M 1012 172 C 1012 215 1012 260 1012 304"
                      fill="none"
                      stroke={step >= 4 ? "url(#edgeAlert)" : "url(#edgeNormal)"}
                      strokeWidth={step >= 4 ? 3 : 2}
                      strokeDasharray="7 8"
                    />

                    {step >= 3 ? (
                      <motion.circle
                        r="5"
                        fill="#fb7185"
                        initial={{ opacity: 0 }}
                        animate={{
                          opacity: [0.3, 1, 0.3],
                          cx: [720, 845],
                          cy: [118, 118],
                        }}
                        transition={{
                          duration: 1.4,
                          repeat: Infinity,
                          ease: "linear",
                        }}
                      />
                    ) : null}

                    {step >= 4 ? (
                      <motion.circle
                        r="5"
                        fill="#f59e0b"
                        initial={{ opacity: 0 }}
                        animate={{
                          opacity: [0.3, 1, 0.3],
                          cx: [1012, 1012],
                          cy: [172, 304],
                        }}
                        transition={{
                          duration: 1.1,
                          repeat: Infinity,
                          ease: "linear",
                        }}
                      />
                    ) : null}
                  </svg>
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
                          SELECTED ASSET
                        </div>
                        <h2 className="mt-2 text-[1rem] font-semibold tracking-[-0.01em] text-white">
                          {selected.name}
                        </h2>
                      </div>
                      <div
                        className={cn(
                          "size-2.5 rounded-full",
                          stateClasses(selected.state).dot,
                        )}
                      />
                    </div>
                  </CardHeader>

                  <Separator className="bg-slate-800/80" />

                  <CardContent className="space-y-4 p-5">
                    <div className="grid grid-cols-2 gap-2">
                      <div className="rounded-xl border border-slate-800/75 bg-slate-950/35 p-3">
                        <div className="text-[0.6rem] font-semibold tracking-[0.08em] text-slate-600">
                          IP ADDRESS
                        </div>
                        <div className="mt-2 font-mono text-[0.7rem] text-slate-300">
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
                      <button
                        type="button"
                        className="flex w-full cursor-pointer items-center justify-between rounded-xl border border-violet-400/15 bg-violet-400/[0.05] p-3 text-left transition hover:border-violet-400/35 hover:bg-violet-400/[0.08]"
                      >
                        <div>
                          <div className="text-[0.6rem] font-semibold tracking-[0.08em] text-violet-300/60">
                            ASSOCIATED IDENTITY
                          </div>
                          <div className="mt-1 text-[0.76rem] font-semibold text-violet-200">
                            {selected.identity}
                          </div>
                        </div>
                        <ChevronRight className="size-4 text-violet-400/60" />
                      </button>
                    ) : null}

                    <div>
                      <div className="mb-2 flex items-center justify-between gap-3">
                        <span className="text-[0.66rem] font-semibold tracking-[0.08em] text-slate-600">
                          FINDINGS
                        </span>
                        <span className="text-[0.72rem] font-semibold text-slate-300">
                          {selected.findings}
                        </span>
                      </div>
                      <div className="h-1.5 overflow-hidden rounded-full bg-slate-900">
                        <motion.div
                          key={selected.id}
                          initial={{ width: 0 }}
                          animate={{
                            width: `${Math.min(100, selected.findings * 38)}%`,
                          }}
                          transition={{ duration: 0.45 }}
                          className="h-full rounded-full bg-gradient-to-r from-amber-300 to-rose-400"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-2">
                      <Button
                        variant="outline"
                        className="cursor-pointer border-slate-800 bg-slate-950/40 text-[0.72rem] text-slate-300 hover:bg-slate-900"
                      >
                        <Activity className="size-3.5" />
                        View events
                      </Button>
                      <Button className="cursor-pointer bg-gradient-to-r from-sky-400 to-cyan-300 text-[0.72rem] font-semibold text-slate-950 hover:from-sky-300 hover:to-cyan-200">
                        <GitBranch className="size-3.5" />
                        Pivot
                      </Button>
                    </div>
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
                        WHY SENTINEL FLAGGED THIS
                      </span>
                    </div>
                  </CardHeader>
                  <Separator className="bg-rose-400/10" />
                  <CardContent className="space-y-3 p-5">
                    {[
                      "Source crossed a security-zone boundary",
                      "Service identity used interactively",
                      "Destination exposed TCP/502",
                      "Relationship not previously observed",
                    ].map((reason, index) => (
                      <motion.div
                        key={reason}
                        initial={{ opacity: 0, x: -8 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.25 + index * 0.06 }}
                        className="flex items-start gap-2.5 text-[0.7rem] leading-5 tracking-[0.02em] text-slate-400"
                      >
                        <ShieldCheck className="mt-0.5 size-3.5 shrink-0 text-emerald-400" />
                        {reason}
                      </motion.div>
                    ))}
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
                  Reconstruct the attack path
                </h2>
                <p className="mt-2 text-[0.7rem] leading-5 tracking-[0.025em] text-slate-500">
                  Replay correlated evidence in order and watch the environment
                  graph illuminate as the incident develops.
                </p>

                <div className="mt-5 flex items-center gap-2">
                  <Button
                    type="button"
                    onClick={() => setPlaying((value) => !value)}
                    className="cursor-pointer bg-gradient-to-r from-amber-300 to-orange-300 text-[0.72rem] font-semibold text-slate-950 hover:from-amber-200 hover:to-orange-200"
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
                    <span>EVENT {step + 1}</span>
                    <span>{replayEvents.length}</span>
                  </div>
                  <div className="h-1.5 overflow-hidden rounded-full bg-slate-900">
                    <motion.div
                      animate={{
                        width: `${((step + 1) / replayEvents.length) * 100}%`,
                      }}
                      transition={{ duration: 0.35 }}
                      className="h-full rounded-full bg-gradient-to-r from-amber-300 via-orange-300 to-rose-400"
                    />
                  </div>
                </div>
              </div>

              <div className="min-w-0 border-b border-slate-800/80 p-4 xl:border-b-0 xl:border-r">
                <div className="space-y-1">
                  {replayEvents.map((event, index) => {
                    const past = index <= step;
                    const current = index === step;

                    return (
                      <button
                        key={event.id}
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
                          {event.time}
                        </span>
                        <span className="relative mt-1.5 flex justify-center">
                          <span
                            className={cn(
                              "z-10 size-2 rounded-full",
                              current
                                ? "bg-amber-300 shadow-[0_0_16px_rgba(251,191,36,0.8)]"
                                : past
                                  ? "bg-sky-400"
                                  : "bg-slate-700",
                            )}
                          />
                          {index < replayEvents.length - 1 ? (
                            <span className="absolute top-2 h-10 w-px bg-slate-800" />
                          ) : null}
                        </span>
                        <span>
                          <span
                            className={cn(
                              "block text-[0.75rem] font-semibold tracking-[0.01em]",
                              current ? "text-amber-100" : "text-slate-300",
                            )}
                          >
                            {event.title}
                          </span>
                          <span className="mt-1 block text-[0.65rem] leading-5 tracking-[0.02em] text-slate-600">
                            {event.detail}
                          </span>
                        </span>
                      </button>
                    );
                  })}
                </div>
              </div>

              <div className="p-5">
                <AnimatePresence mode="wait">
                  <motion.div
                    key={currentEvent.id}
                    initial={{ opacity: 0, x: 12 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -12 }}
                    transition={{ duration: 0.25 }}
                  >
                    <div className="text-[0.62rem] font-semibold tracking-[0.1em] text-slate-600">
                      CURRENT EVIDENCE
                    </div>
                    <div className="mt-3 text-[0.88rem] font-semibold leading-6 text-white">
                      {currentEvent.title}
                    </div>
                    <div className="mt-2 text-[0.68rem] leading-5 tracking-[0.02em] text-slate-500">
                      {currentEvent.detail}
                    </div>

                    <div className="mt-4 rounded-xl border border-slate-800/80 bg-slate-950/45 p-3">
                      <div className="flex items-center gap-2 font-mono text-[0.64rem] text-slate-400">
                        <span>{currentEvent.source}</span>
                        <ChevronRight className="size-3 text-slate-700" />
                        <span>{currentEvent.target}</span>
                      </div>
                    </div>

                    {currentEvent.detection ? (
                      <div className="mt-3 rounded-xl border border-rose-400/20 bg-rose-400/[0.05] p-3">
                        <div className="text-[0.6rem] font-semibold tracking-[0.09em] text-rose-300/70">
                          DETECTION
                        </div>
                        <div className="mt-1.5 font-mono text-[0.72rem] font-semibold text-rose-200">
                          {currentEvent.detection}
                        </div>
                      </div>
                    ) : null}

                    <div className="mt-4 grid grid-cols-2 gap-2">
                      <Button
                        variant="outline"
                        className="cursor-pointer border-slate-800 bg-slate-950/45 text-[0.68rem] text-slate-300 hover:bg-slate-900"
                      >
                        <Network className="size-3.5" />
                        Pivot path
                      </Button>
                      <Button className="cursor-pointer bg-gradient-to-r from-violet-400 to-sky-400 text-[0.68rem] font-semibold text-slate-950 hover:from-violet-300 hover:to-sky-300">
                        <ShieldCheck className="size-3.5" />
                        Investigate
                      </Button>
                    </div>
                  </motion.div>
                </AnimatePresence>
              </div>
            </div>
          </motion.section>
        </section>
      </div>
    </main>
  );
}
