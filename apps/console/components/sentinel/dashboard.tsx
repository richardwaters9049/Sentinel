"use client";

import {
  AlertTriangle,
  Bell,
  Binary,
  CheckCircle2,
  ChevronRight,
  Crosshair,
  Database,
  FileSearch,
  Gauge,
  Network,
  Radar,
  RefreshCcw,
  ShieldCheck,
  Sparkles,
  Target,
} from "lucide-react";
import { motion } from "framer-motion";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

import SidebarDrawer from "@/components/sentinel/sidebar-drawer";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import {
  getDetectionMetrics,
  listDetections,
  listEvents,
  listFindings,
  listHunts,
  listInvestigations,
} from "@/lib/sentinel/client";
import type {
  DetectionMetric,
  DetectionRecord,
  Finding,
  HuntDefinition,
  InvestigationSummary,
  TelemetryEvent,
} from "@/lib/sentinel/types";

const navItems = [
  { label: "Overview", icon: Gauge, href: "/" },
  { label: "Environment", icon: Network, href: "/environment" },
  { label: "Findings", icon: AlertTriangle, href: "/findings" },
  { label: "Hunts", icon: Crosshair, href: "/hunts" },
  { label: "Investigations", icon: FileSearch, href: "/investigations" },
  { label: "Detections", icon: Radar, href: "/detections" },
  { label: "Intelligence", icon: Binary, href: "/intelligence" },
];

const reveal = {
  hidden: { opacity: 0, y: 18 },
  visible: { opacity: 1, y: 0 },
};

function severityClass(severity: string) {
  switch (severity.toLowerCase()) {
    case "critical":
      return "border-rose-400/35 bg-rose-400/10 text-rose-300";
    case "high":
      return "border-orange-300/35 bg-orange-300/10 text-orange-200";
    case "medium":
      return "border-amber-300/35 bg-amber-300/10 text-amber-200";
    default:
      return "border-sky-400/30 bg-sky-400/10 text-sky-300";
  }
}

export function SidebarContent() {
  const pathname = usePathname();

  return (
    <div className="flex h-full flex-col">
      <div className="mb-7 flex items-center gap-3">
        <div className="grid size-9 place-items-center rounded-xl bg-gradient-to-br from-sky-400 via-cyan-300 to-violet-500 shadow-[0_0_40px_rgba(56,189,248,0.18)]">
          <ShieldCheck className="size-5 text-slate-950" />
        </div>
        <div>
          <div className="text-[0.98rem] font-bold tracking-[0.12em] text-white">
            SENTINEL
          </div>
          <div className="mt-0.5 text-[0.62rem] font-semibold tracking-[0.15em] text-slate-600">
            ANALYST CONSOLE
          </div>
        </div>
      </div>

      <div className="mb-3 text-[0.64rem] font-semibold tracking-[0.15em] text-slate-600">
        DEFENSIVE OPERATIONS
      </div>

      <nav aria-label="Primary navigation" className="space-y-1.5">
        {navItems.map((item) => {
          const Icon = item.icon;
          const active =
            item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);

          return (
            <Link
              key={item.label}
              href={item.href}
              aria-current={active ? "page" : undefined}
              className={cn(
                "group flex w-full cursor-pointer items-center gap-3 rounded-xl border px-3.5 py-3 text-left text-[0.88rem] font-medium tracking-[0.015em] transition duration-200",
                active
                  ? "border-sky-400/20 bg-gradient-to-r from-sky-500/20 to-violet-500/15 text-slate-50"
                  : "border-transparent text-slate-400 hover:border-slate-800 hover:bg-slate-900/55 hover:text-slate-100",
              )}
            >
              <Icon
                className={cn(
                  "size-[1.05rem] transition",
                  active
                    ? "text-sky-300"
                    : "text-slate-600 group-hover:text-slate-300",
                )}
              />
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="mt-auto rounded-2xl border border-slate-800/90 bg-slate-950/50 p-4">
        <div className="text-[0.64rem] font-semibold tracking-[0.13em] text-slate-500">
          NORTHSTAR ENERGY
        </div>
        <div className="mt-2 text-[0.82rem] font-semibold text-slate-200">
          Synthetic critical-infrastructure lab
        </div>
        <div className="mt-3 flex items-center gap-2 text-[0.72rem] font-medium text-emerald-300/85">
          <span className="size-2 rounded-full bg-emerald-400 shadow-[0_0_14px_rgba(52,211,153,0.72)]" />
          Defensive simulation
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  label,
  value,
  detail,
  icon: Icon,
  accent,
  index,
}: {
  label: string;
  value: string;
  detail: string;
  icon: typeof AlertTriangle;
  accent: string;
  index: number;
}) {
  return (
    <motion.div
      variants={reveal}
      initial="hidden"
      animate="visible"
      transition={{ duration: 0.45, delay: 0.1 + index * 0.06 }}
    >
      <Card className="surface-card h-full overflow-hidden rounded-2xl border-slate-800/85 bg-transparent py-0">
        <CardContent className="p-5">
          <div className="flex items-center justify-between gap-3">
            <span className="text-[0.65rem] font-semibold tracking-[0.11em] text-slate-500">
              {label.toUpperCase()}
            </span>
            <Icon className="size-4 text-slate-700" />
          </div>
          <div className="mt-5 text-[1.85rem] font-bold leading-none tracking-[-0.035em] text-white">
            {value}
          </div>
          <div className="mt-2 text-[0.7rem] leading-5 text-slate-500">
            {detail}
          </div>
          <div className={cn("mt-5 h-[3px] rounded-full bg-gradient-to-r", accent)} />
        </CardContent>
      </Card>
    </motion.div>
  );
}

export default function SentinelDashboard() {
  const [findings, setFindings] = useState<Finding[]>([]);
  const [investigations, setInvestigations] = useState<InvestigationSummary[]>([]);
  const [hunts, setHunts] = useState<HuntDefinition[]>([]);
  const [events, setEvents] = useState<TelemetryEvent[]>([]);
  const [detections, setDetections] = useState<DetectionRecord[]>([]);
  const [detectionMetrics, setDetectionMetrics] = useState<DetectionMetric[]>([]);
  const [loading, setLoading] = useState(true);
  const [degraded, setDegraded] = useState<string[]>([]);

  async function refresh() {
    setLoading(true);

    const requests = await Promise.allSettled([
      listFindings(100),
      listInvestigations(),
      listHunts(),
      listEvents(40),
      listDetections(),
      getDetectionMetrics(),
    ]);

    const failures: string[] = [];

    if (requests[0].status === "fulfilled") {
      setFindings(requests[0].value.findings);
    } else failures.push("findings");

    if (requests[1].status === "fulfilled") {
      setInvestigations(requests[1].value.investigations);
    } else failures.push("investigations");

    if (requests[2].status === "fulfilled") {
      setHunts(requests[2].value.hunts);
    } else failures.push("hunts");

    if (requests[3].status === "fulfilled") {
      setEvents(requests[3].value.events);
    } else failures.push("telemetry");

    if (requests[4].status === "fulfilled") {
      setDetections(requests[4].value.detections);
    } else failures.push("detections");

    if (requests[5].status === "fulfilled") {
      setDetectionMetrics(requests[5].value.metrics);
    } else failures.push("detection metrics");

    setDegraded(failures);
    setLoading(false);
  }

  useEffect(() => {
    let active = true;

    void Promise.allSettled([
      listFindings(100),
      listInvestigations(),
      listHunts(),
      listEvents(40),
      listDetections(),
      getDetectionMetrics(),
    ]).then((requests) => {
      if (!active) return;
      const failures: string[] = [];

      if (requests[0].status === "fulfilled") setFindings(requests[0].value.findings);
      else failures.push("findings");
      if (requests[1].status === "fulfilled")
        setInvestigations(requests[1].value.investigations);
      else failures.push("investigations");
      if (requests[2].status === "fulfilled") setHunts(requests[2].value.hunts);
      else failures.push("hunts");
      if (requests[3].status === "fulfilled") setEvents(requests[3].value.events);
      else failures.push("telemetry");
      if (requests[4].status === "fulfilled")
        setDetections(requests[4].value.detections);
      else failures.push("detections");
      if (requests[5].status === "fulfilled")
        setDetectionMetrics(requests[5].value.metrics);
      else failures.push("detection metrics");

      setDegraded(failures);
      setLoading(false);
    });

    return () => {
      active = false;
    };
  }, []);

  const openFindings = findings.filter((finding) =>
    ["new", "triaged", "investigating", "confirmed", "contained"].includes(
      finding.status,
    ),
  );
  const highFindings = openFindings.filter((finding) =>
    ["high", "critical"].includes(finding.severity.toLowerCase()),
  );
  const activeInvestigations = investigations.filter(
    (item) => item.status !== "closed",
  );
  const enabledCount = detections.filter((item) => item.enabled).length;
  const coverage = detections.length
    ? Math.round((enabledCount / detections.length) * 100)
    : 0;
  const riskIndex = Math.min(
    100,
    highFindings.length * 18 +
      openFindings.length * 3 +
      activeInvestigations.filter((item) => item.priority === "critical").length * 12,
  );

  const priorityFindings = [...findings]
    .sort(
      (a, b) =>
        b.confidence - a.confidence ||
        new Date(b.last_observed_at).getTime() -
          new Date(a.last_observed_at).getTime(),
    )
    .slice(0, 5);

  const recentEvents = [...events]
    .sort(
      (a, b) =>
        new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
    )
    .slice(0, 5);

  const metrics = [
    {
      label: "Open findings",
      value: String(openFindings.length),
      detail: highFindings.length + " high/critical need review",
      icon: AlertTriangle,
      accent: "from-rose-400 to-orange-300",
    },
    {
      label: "Investigations",
      value: String(activeInvestigations.length),
      detail:
        activeInvestigations.filter((item) => item.priority === "critical").length +
        " critical priority",
      icon: FileSearch,
      accent: "from-violet-400 to-fuchsia-300",
    },
    {
      label: "Saved hunts",
      value: String(hunts.length),
      detail: "reproducible threat-hunting definitions",
      icon: Crosshair,
      accent: "from-sky-400 to-indigo-400",
    },
    {
      label: "Rule coverage",
      value: coverage + "%",
      detail: enabledCount + " of " + detections.length + " rules enabled",
      icon: ShieldCheck,
      accent: "from-emerald-400 to-cyan-300",
    },
  ];

  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto min-h-screen max-w-[1720px]">
        <section className="min-w-0 px-4 py-5 sm:px-6 lg:px-8 xl:px-10">
          <motion.header
            variants={reveal}
            initial="hidden"
            animate="visible"
            className="mb-6 flex items-center justify-between gap-4"
          >
            <div>
              <div className="flex items-center gap-3">
                <SidebarDrawer />
                <span className="text-[0.72rem] font-bold tracking-[0.15em] text-slate-500">
                  SENTINEL
                </span>
              </div>
              <h1 className="mt-3 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.03em] text-white sm:text-[1.9rem]">
                Security overview
              </h1>
              <p className="mt-1.5 text-[0.78rem] font-medium leading-5 tracking-[0.025em] text-slate-500 sm:text-[0.82rem]">
                Northstar Energy Facility · live persisted defensive data
              </p>
            </div>

            <div className="flex items-center gap-2.5">
              <Button
                type="button"
                onClick={() => void refresh()}
                variant="outline"
                size="icon"
                aria-label="Refresh security overview"
                className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900"
              >
                <RefreshCcw className={cn("size-4", loading && "animate-spin")} />
              </Button>
              <Button
                variant="outline"
                size="icon"
                aria-label="Notifications"
                className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900"
              >
                <Bell className="size-4" />
              </Button>
              <Avatar className="size-9 border border-sky-400/25">
                <AvatarFallback className="bg-gradient-to-br from-sky-400/90 to-violet-500/90 text-[0.72rem] font-bold text-slate-950">
                  RW
                </AvatarFallback>
              </Avatar>
            </div>
          </motion.header>

          {degraded.length > 0 ? (
            <div
              role="alert"
              className="mb-4 flex flex-col gap-3 rounded-2xl border border-amber-300/20 bg-amber-300/[0.05] p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div className="flex items-start gap-3">
                <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-200" />
                <div>
                  <div className="text-[0.75rem] font-semibold text-amber-100">
                    Sentinel is operating in degraded view
                  </div>
                  <div className="mt-1 text-[0.65rem] leading-5 text-amber-100/55">
                    Unavailable: {degraded.join(", ")}. Available datasets remain visible.
                  </div>
                </div>
              </div>
              <Button
                type="button"
                onClick={() => void refresh()}
                variant="outline"
                size="sm"
                className="cursor-pointer border-amber-300/20 bg-amber-300/[0.04] text-amber-100 hover:bg-amber-300/10"
              >
                Retry unavailable data
              </Button>
            </div>
          ) : null}

          <motion.section
            variants={reveal}
            initial="hidden"
            animate="visible"
            transition={{ duration: 0.5, delay: 0.05 }}
            className="relative mb-4 overflow-hidden rounded-[1.35rem] border border-slate-800/80 bg-gradient-to-r from-[#0d1722]/95 via-[#101521]/95 to-[#1a1026]/95 p-5 sm:p-6"
          >
            <div className="absolute -right-24 -top-28 size-72 rounded-full bg-sky-500/10 blur-3xl" />
            <div className="relative grid gap-5 lg:grid-cols-[minmax(0,1fr)_180px] lg:items-start">
              <div>
                <div className="mb-2 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.14em] text-sky-300/80">
                  <Sparkles className="size-3.5" />
                  LIVE DEFENSIVE POSTURE
                </div>
                <h2 className="text-[1.2rem] font-semibold leading-7 tracking-[-0.015em] text-white sm:text-[1.35rem]">
                  {highFindings.length > 0
                    ? "Threat posture requires analyst attention"
                    : "Threat posture is stable"}
                </h2>
                <p className="mt-2 max-w-3xl text-[0.78rem] leading-6 tracking-[0.02em] text-slate-400">
                  {highFindings.length} high-confidence finding
                  {highFindings.length === 1 ? "" : "s"} currently open across{" "}
                  {activeInvestigations.length} active investigation
                  {activeInvestigations.length === 1 ? "" : "s"}.
                </p>
              </div>
              <div className="rounded-2xl border border-amber-300/15 bg-amber-300/[0.035] p-4 lg:text-right">
                <div className="text-[2rem] font-bold leading-none tracking-[-0.045em] text-amber-200">
                  {riskIndex}
                </div>
                <div className="mt-2 text-[0.62rem] font-semibold tracking-[0.12em] text-amber-200/45">
                  RISK INDEX / 100
                </div>
              </div>
            </div>
          </motion.section>

          <section
            aria-label="Live security metrics"
            className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
          >
            {metrics.map((metric, index) => (
              <MetricCard key={metric.label} {...metric} index={index} />
            ))}
          </section>

          <section className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,1.65fr)_minmax(320px,0.9fr)]">
            <div className="grid gap-4">
              <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                <CardHeader className="flex-row items-start justify-between gap-4 px-5 pb-4 pt-5">
                  <div>
                    <h3 className="text-[1rem] font-semibold text-slate-100">
                      Priority findings
                    </h3>
                    <p className="mt-1 text-[0.7rem] text-slate-500">
                      Live findings ordered by confidence and recency
                    </p>
                  </div>
                  <Button
                    render={<Link href="/findings" />}
                    variant="ghost"
                    size="sm"
                    className="cursor-pointer text-[0.7rem] text-sky-300 hover:bg-sky-400/10"
                  >
                    View all
                    <ChevronRight className="size-3.5" />
                  </Button>
                </CardHeader>
                <Separator className="bg-slate-800/80" />
                <CardContent className="p-3">
                  {priorityFindings.length > 0 ? (
                    <div className="divide-y divide-slate-800/65">
                      {priorityFindings.map((finding) => (
                        <Link
                          key={finding.id}
                          href="/findings"
                          className="grid cursor-pointer gap-3 rounded-xl px-3 py-4 transition hover:bg-slate-900/60 sm:grid-cols-[82px_minmax(0,1fr)_92px] sm:items-center"
                        >
                          <Badge
                            variant="outline"
                            className={cn(
                              "w-fit rounded-md text-[0.56rem] font-bold tracking-[0.08em]",
                              severityClass(finding.severity),
                            )}
                          >
                            {finding.severity.toUpperCase()}
                          </Badge>
                          <div className="min-w-0">
                            <div className="truncate text-[0.8rem] font-semibold text-slate-200">
                              {finding.title}
                            </div>
                            <div className="mt-1 truncate font-mono text-[0.61rem] text-slate-600">
                              {finding.detection_id} · {finding.status}
                            </div>
                          </div>
                          <div className="sm:text-right">
                            <div className="text-[0.8rem] font-semibold text-slate-300">
                              {finding.confidence}%
                            </div>
                            <div className="mt-1 text-[0.58rem] text-slate-600">
                              CONFIDENCE
                            </div>
                          </div>
                        </Link>
                      ))}
                    </div>
                  ) : (
                    <div className="grid min-h-40 place-items-center text-center">
                      <div>
                        <CheckCircle2 className="mx-auto size-5 text-emerald-400" />
                        <div className="mt-3 text-[0.75rem] text-slate-400">
                          No findings currently available.
                        </div>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>

              <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                <CardHeader className="px-5 pb-4 pt-5">
                  <h3 className="text-[1rem] font-semibold text-slate-100">
                    Threat activity
                  </h3>
                  <p className="text-[0.7rem] text-slate-500">
                    Latest persisted telemetry
                  </p>
                </CardHeader>
                <Separator className="bg-slate-800/80" />
                <CardContent className="space-y-1 p-3">
                  {recentEvents.map((event) => (
                    <div
                      key={event.event_id}
                      className="grid gap-2 rounded-xl px-3 py-3 sm:grid-cols-[76px_78px_minmax(0,1fr)] sm:items-center"
                    >
                      <span className="font-mono text-[0.61rem] text-slate-600">
                        {new Date(event.timestamp).toLocaleTimeString("en-GB")}
                      </span>
                      <Badge
                        variant="outline"
                        className="w-fit border-sky-400/25 bg-sky-400/[0.05] text-[0.56rem] text-sky-300"
                      >
                        {event.event.category.toUpperCase()}
                      </Badge>
                      <div className="min-w-0">
                        <div className="truncate text-[0.74rem] font-medium text-slate-300">
                          {event.event.action}
                        </div>
                        <div className="mt-1 truncate font-mono text-[0.58rem] text-slate-600">
                          {event.actor?.name ||
                            event.asset?.hostname ||
                            event.network?.source_ip ||
                            event.event_id}
                        </div>
                      </div>
                    </div>
                  ))}
                </CardContent>
              </Card>
            </div>

            <div className="grid content-start gap-4">
              <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                <CardHeader className="px-5 pb-4 pt-5">
                  <h3 className="text-[1rem] font-semibold text-slate-100">
                    Active investigations
                  </h3>
                  <p className="text-[0.7rem] text-slate-500">
                    Cases currently in analyst workflow
                  </p>
                </CardHeader>
                <Separator className="bg-slate-800/80" />
                <CardContent className="space-y-2.5 p-3">
                  {activeInvestigations.slice(0, 4).map((item) => (
                    <Link
                      key={item.id}
                      href={"/investigations?id=" + encodeURIComponent(item.id)}
                      className="block cursor-pointer rounded-xl border border-slate-800/75 bg-slate-950/35 p-3.5 transition hover:border-slate-700 hover:bg-slate-900/60"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <span className="truncate text-[0.77rem] font-semibold text-slate-200">
                          {item.title}
                        </span>
                        <Badge
                          variant="outline"
                          className="border-violet-400/20 bg-violet-400/[0.05] text-[0.54rem] text-violet-300"
                        >
                          {item.priority.toUpperCase()}
                        </Badge>
                      </div>
                      <div className="mt-2 flex items-center justify-between text-[0.6rem] text-slate-600">
                        <span>{item.owner_id ?? "unassigned"}</span>
                        <span>{item.status}</span>
                      </div>
                    </Link>
                  ))}
                </CardContent>
              </Card>

              <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                <CardHeader className="flex-row items-center justify-between px-5 pb-4 pt-5">
                  <div>
                    <h3 className="text-[1rem] font-semibold text-slate-100">
                      Detection health
                    </h3>
                    <p className="mt-1 text-[0.7rem] text-slate-500">
                      Runtime catalogue and observed hits
                    </p>
                  </div>
                  <Button
                    render={<Link href="/detections" />}
                    variant="ghost"
                    size="icon"
                    aria-label="Open detection management"
                    className="cursor-pointer text-cyan-300 hover:bg-cyan-400/10"
                  >
                    <Radar className="size-4" />
                  </Button>
                </CardHeader>
                <Separator className="bg-slate-800/80" />
                <CardContent className="space-y-4 p-5">
                  {detections.map((detection) => {
                    const metric = detectionMetrics.find(
                      (item) => item.detection_id === detection.id,
                    );
                    const score =
                      metric && metric.hit_count > 0
                        ? Math.round((1 - metric.false_positive_rate) * 100)
                        : 100;
                    return (
                      <div key={detection.id}>
                        <div className="mb-2 flex items-center justify-between gap-3">
                          <div className="min-w-0">
                            <div className="truncate font-mono text-[0.63rem] font-semibold text-slate-400">
                              {detection.id}
                            </div>
                            <div className="mt-0.5 truncate text-[0.64rem] text-slate-600">
                              {detection.title}
                            </div>
                          </div>
                          <span
                            className={cn(
                              "size-2 rounded-full",
                              detection.enabled ? "bg-emerald-400" : "bg-slate-700",
                            )}
                          />
                        </div>
                        <Progress value={score} className="h-1.5 bg-slate-900" />
                      </div>
                    );
                  })}

                  <div className="grid grid-cols-2 gap-2 pt-1">
                    <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3 text-center">
                      <Target className="mx-auto size-4 text-sky-400" />
                      <div className="mt-2 text-[0.65rem] font-semibold text-slate-300">
                        {detectionMetrics.reduce(
                          (total, item) => total + item.hit_count,
                          0,
                        )}{" "}
                        hits
                      </div>
                    </div>
                    <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3 text-center">
                      <CheckCircle2 className="mx-auto size-4 text-emerald-400" />
                      <div className="mt-2 text-[0.65rem] font-semibold text-slate-300">
                        {enabledCount}/{detections.length} enabled
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </section>

          <footer className="mt-5 flex flex-col gap-2 border-t border-slate-900 py-5 text-[0.63rem] tracking-[0.04em] text-slate-700 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2">
              <ShieldCheck className="size-3 text-emerald-500" />
              Live values are read from the Sentinel gateway
            </div>
            <div className="flex items-center gap-2">
              <Database className="size-3" />
              PostgreSQL · NATS JetStream · Northstar synthetic lab
            </div>
          </footer>
        </section>
      </div>
    </main>
  );
}
