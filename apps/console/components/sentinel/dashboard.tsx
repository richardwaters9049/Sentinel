"use client";

import {
  Activity,
  AlertTriangle,
  Bell,
  Binary,
  CheckCircle2,
  ChevronRight,
  Crosshair,
  Database,
  FileSearch,
  Fingerprint,
  Gauge,
  Menu,
  Radar,
  Search,
  ShieldCheck,
  Sparkles,
  Target,
  Workflow,
} from "lucide-react";
import { motion } from "framer-motion";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

const navItems = [
  { label: "Overview", icon: Gauge, active: true },
  { label: "Findings", icon: AlertTriangle },
  { label: "Hunts", icon: Crosshair },
  { label: "Investigations", icon: FileSearch },
  { label: "Assets & identities", icon: Fingerprint },
  { label: "Detections", icon: Radar },
  { label: "Telemetry", icon: Activity },
];

const metrics = [
  {
    label: "Open findings",
    value: "14",
    detail: "+3 since last hour",
    icon: AlertTriangle,
    accent: "from-rose-400 to-orange-300",
    dot: "bg-rose-400",
  },
  {
    label: "Investigations",
    value: "7",
    detail: "2 critical priority",
    icon: FileSearch,
    accent: "from-violet-400 to-fuchsia-300",
    dot: "bg-violet-400",
  },
  {
    label: "Hunts today",
    value: "18",
    detail: "42 events surfaced",
    icon: Crosshair,
    accent: "from-sky-400 to-indigo-400",
    dot: "bg-sky-400",
  },
  {
    label: "Coverage",
    value: "96%",
    detail: "3 rules enabled",
    icon: ShieldCheck,
    accent: "from-emerald-400 to-cyan-300",
    dot: "bg-emerald-400",
  },
];

const findings = [
  {
    severity: "HIGH",
    title: "Service account interactive login",
    detection: "DET-AUTH-002",
    subject: "svc-backup · 10.10.20.15",
    confidence: 90,
    tone: "rose",
  },
  {
    severity: "HIGH",
    title: "Corporate → OT network connection",
    detection: "DET-NET-001",
    subject: "employee-ws-01 · PLC segment",
    confidence: 95,
    tone: "rose",
  },
  {
    severity: "MEDIUM",
    title: "Authentication burst followed by success",
    detection: "DET-AUTH-001",
    subject: "operator-07 · 10.10.10.44",
    confidence: 85,
    tone: "amber",
  },
];

const activity = [
  {
    time: "21:43:08",
    type: "AUTH",
    title: "Service account login accepted",
    subject: "svc-backup",
    color: "violet",
  },
  {
    time: "21:42:31",
    type: "NET",
    title: "Corporate host reached OT zone",
    subject: "10.30.0.10:502",
    color: "sky",
  },
  {
    time: "21:41:54",
    type: "PROC",
    title: "Unsigned process started",
    subject: "engineering-ws-02",
    color: "amber",
  },
];

const investigations = [
  {
    priority: "CRITICAL",
    title: "OT boundary access",
    owner: "senior-analyst",
    status: "investigating",
    tone: "rose",
  },
  {
    priority: "HIGH",
    title: "Service identity misuse",
    owner: "r.waters",
    status: "open",
    tone: "amber",
  },
  {
    priority: "MEDIUM",
    title: "Authentication anomaly",
    owner: "unassigned",
    status: "contained",
    tone: "violet",
  },
];

const detections = [
  { id: "DET-AUTH-001", title: "Auth burst", score: 85, color: "bg-violet-400" },
  { id: "DET-AUTH-002", title: "Service login", score: 90, color: "bg-sky-400" },
  { id: "DET-NET-001", title: "IT → OT", score: 95, color: "bg-emerald-400" },
];

const reveal = {
  hidden: { opacity: 0, y: 18 },
  visible: { opacity: 1, y: 0 },
};

function SeverityBadge({
  label,
  tone,
}: {
  label: string;
  tone: string;
}) {
  return (
    <Badge
      variant="outline"
      className={cn(
        "rounded-lg px-2.5 py-1 text-[0.66rem] font-bold tracking-[0.09em]",
        tone === "rose" &&
          "border-rose-400/35 bg-rose-400/10 text-rose-300",
        tone === "amber" &&
          "border-amber-300/35 bg-amber-300/10 text-amber-200",
        tone === "violet" &&
          "border-violet-400/35 bg-violet-400/10 text-violet-300",
      )}
    >
      {label}
    </Badge>
  );
}

function SidebarContent() {
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

      <nav className="space-y-1.5">
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <button
              key={item.label}
              type="button"
              className={cn(
                "group flex w-full cursor-pointer items-center gap-3 rounded-xl border px-3.5 py-3 text-left text-[0.88rem] font-medium tracking-[0.015em] transition duration-200",
                item.active
                  ? "border-sky-400/20 bg-gradient-to-r from-sky-500/20 to-violet-500/15 text-slate-50"
                  : "border-transparent text-slate-400 hover:border-slate-800 hover:bg-slate-900/55 hover:text-slate-100",
              )}
            >
              <Icon
                className={cn(
                  "size-[1.05rem] transition",
                  item.active
                    ? "text-sky-300"
                    : "text-slate-600 group-hover:text-slate-300",
                )}
              />
              {item.label}
            </button>
          );
        })}
      </nav>

      <div className="mt-auto rounded-2xl border border-slate-800/90 bg-slate-950/50 p-4">
        <div className="text-[0.64rem] font-semibold tracking-[0.13em] text-slate-500">
          NORTHSTAR ENERGY
        </div>
        <div className="mt-2 text-[0.82rem] font-semibold text-slate-200">
          Lab environment healthy
        </div>
        <div className="mt-3 flex items-center gap-2 text-[0.72rem] font-medium text-emerald-300/85">
          <span className="size-2 rounded-full bg-emerald-400 shadow-[0_0_14px_rgba(52,211,153,0.72)]" />
          All collectors online
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  metric,
  index,
}: {
  metric: (typeof metrics)[number];
  index: number;
}) {
  const Icon = metric.icon;
  return (
    <motion.div
      variants={reveal}
      initial="hidden"
      animate="visible"
      transition={{ duration: 0.45, delay: 0.12 + index * 0.07 }}
    >
      <Card className="surface-card h-full overflow-hidden rounded-2xl border-slate-800/85 bg-transparent py-0">
        <CardContent className="flex h-full flex-col justify-between p-5">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className={cn("size-2 rounded-full", metric.dot)} />
              <span className="text-[0.68rem] font-semibold tracking-[0.12em] text-slate-500">
                {metric.label.toUpperCase()}
              </span>
            </div>
            <Icon className="size-4 text-slate-700" />
          </div>
          <div className="mt-5">
            <div className="text-[1.85rem] font-bold leading-none tracking-[-0.035em] text-white">
              {metric.value}
            </div>
            <div className="mt-2 text-[0.75rem] font-medium tracking-[0.02em] text-slate-500">
              {metric.detail}
            </div>
          </div>
          <div
            className={cn(
              "mt-5 h-[3px] rounded-full bg-gradient-to-r",
              metric.accent,
            )}
          />
        </CardContent>
      </Card>
    </motion.div>
  );
}

export default function SentinelDashboard() {
  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto grid min-h-screen max-w-[1720px] lg:grid-cols-[242px_minmax(0,1fr)]">
        <aside className="sticky top-0 hidden h-screen border-r border-slate-800/80 bg-[#090d14]/95 px-5 py-6 backdrop-blur-xl lg:block">
          <SidebarContent />
        </aside>

        <section className="min-w-0 px-4 py-5 sm:px-6 lg:px-8 xl:px-10">
          <motion.header
            variants={reveal}
            initial="hidden"
            animate="visible"
            transition={{ duration: 0.45 }}
            className="mb-6 flex items-center justify-between gap-4"
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

              <h1 className="mt-3 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.03em] text-white sm:text-[1.9rem] lg:mt-0">
                Security overview
              </h1>
              <p className="mt-1.5 text-[0.78rem] font-medium leading-5 tracking-[0.025em] text-slate-500 sm:text-[0.82rem]">
                Northstar Energy Facility · live defensive telemetry
              </p>
            </div>

            <div className="flex items-center gap-2.5">
              <div className="relative hidden w-[18rem] xl:block">
                <Search className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-slate-600" />
                <Input
                  aria-label="Search Sentinel"
                  placeholder="Search findings, assets, identities…"
                  className="h-10 border-slate-800 bg-slate-950/60 pl-10 text-[0.78rem] tracking-[0.02em] text-slate-200 placeholder:text-slate-600 focus-visible:ring-sky-400/35"
                />
              </div>
              <Button
                variant="outline"
                size="icon"
                className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900 hover:text-slate-100"
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

          <motion.section
            variants={reveal}
            initial="hidden"
            animate="visible"
            transition={{ duration: 0.5, delay: 0.07 }}
            className="relative mb-4 overflow-hidden rounded-[1.35rem] border border-slate-800/80 bg-gradient-to-r from-[#0d1722]/95 via-[#101521]/95 to-[#1a1026]/95 p-5 shadow-[0_28px_100px_rgba(0,0,0,0.24)] sm:p-6"
          >
            <div className="absolute -right-24 -top-28 size-72 rounded-full bg-sky-500/10 blur-3xl" />
            <div className="absolute right-10 top-0 size-64 rounded-full bg-violet-500/10 blur-3xl" />
            <div className="relative grid gap-5 lg:grid-cols-[minmax(0,1fr)_180px] lg:items-start">
              <div>
                <div className="mb-2 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.14em] text-sky-300/80">
                  <Sparkles className="size-3.5" />
                  LIVE DEFENSIVE POSTURE
                </div>
                <h2 className="text-[1.2rem] font-semibold leading-7 tracking-[-0.015em] text-white sm:text-[1.35rem]">
                  Threat posture is elevated
                </h2>
                <p className="mt-2 max-w-3xl text-[0.78rem] leading-6 tracking-[0.02em] text-slate-400 sm:text-[0.82rem]">
                  3 high-confidence findings need analyst review. No collector or
                  queue degradation detected.
                </p>
              </div>
              <div className="rounded-2xl border border-amber-300/15 bg-amber-300/[0.035] p-4 lg:text-right">
                <div className="text-[2rem] font-bold leading-none tracking-[-0.045em] text-amber-200">
                  72
                </div>
                <div className="mt-2 text-[0.62rem] font-semibold tracking-[0.12em] text-amber-200/45">
                  RISK INDEX / 100
                </div>
              </div>
            </div>

            <div className="relative mt-5 flex flex-wrap gap-2">
              <SeverityBadge label="3 HIGH FINDINGS" tone="rose" />
              <SeverityBadge label="7 OPEN CASES" tone="violet" />
              <Badge
                variant="outline"
                className="rounded-lg border-sky-400/35 bg-sky-400/10 px-2.5 py-1 text-[0.66rem] font-bold tracking-[0.09em] text-sky-300"
              >
                18 HUNTS / 24H
              </Badge>
              <Badge
                variant="outline"
                className="rounded-lg border-emerald-400/35 bg-emerald-400/10 px-2.5 py-1 text-[0.66rem] font-bold tracking-[0.09em] text-emerald-300"
              >
                PIPELINE HEALTHY
              </Badge>
            </div>
          </motion.section>

          <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {metrics.map((metric, index) => (
              <MetricCard key={metric.label} metric={metric} index={index} />
            ))}
          </section>

          <section className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,1.7fr)_minmax(320px,0.95fr)]">
            <div className="grid gap-4">
              <motion.div
                variants={reveal}
                initial="hidden"
                whileInView="visible"
                viewport={{ once: true, amount: 0.15 }}
                transition={{ duration: 0.48 }}
              >
                <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                  <CardHeader className="flex-row items-start justify-between gap-4 px-5 pb-4 pt-5 sm:px-6">
                    <div>
                      <h3 className="text-[1.02rem] font-semibold tracking-[-0.01em] text-slate-100">
                        Priority findings
                      </h3>
                      <p className="mt-1 text-[0.72rem] leading-5 tracking-[0.025em] text-slate-500">
                        Sorted by confidence and recency
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="cursor-pointer text-[0.72rem] font-semibold tracking-[0.03em] text-sky-300 hover:bg-sky-400/10 hover:text-sky-200"
                    >
                      View all
                      <ChevronRight className="size-3.5" />
                    </Button>
                  </CardHeader>
                  <Separator className="bg-slate-800/80" />
                  <CardContent className="p-2 sm:p-3">
                    <Tabs defaultValue="priority">
                      <TabsList className="mb-2 h-9 bg-slate-950/55">
                        <TabsTrigger
                          value="priority"
                          className="cursor-pointer text-[0.72rem]"
                        >
                          Priority
                        </TabsTrigger>
                        <TabsTrigger
                          value="recent"
                          className="cursor-pointer text-[0.72rem]"
                        >
                          Recent
                        </TabsTrigger>
                      </TabsList>

                      <TabsContent value="priority" className="mt-0">
                        <div className="divide-y divide-slate-800/65">
                          {findings.map((finding, index) => (
                            <motion.button
                              key={finding.detection}
                              type="button"
                              initial={{ opacity: 0, x: -10 }}
                              animate={{ opacity: 1, x: 0 }}
                              transition={{
                                duration: 0.35,
                                delay: 0.12 + index * 0.06,
                              }}
                              className="grid w-full cursor-pointer gap-3 rounded-xl px-3 py-4 text-left transition hover:bg-slate-900/60 sm:grid-cols-[82px_minmax(0,1fr)_92px] sm:items-center"
                            >
                              <SeverityBadge
                                label={finding.severity}
                                tone={finding.tone}
                              />
                              <div className="min-w-0">
                                <div className="truncate text-[0.82rem] font-semibold tracking-[0.01em] text-slate-200">
                                  {finding.title}
                                </div>
                                <div className="mt-1 truncate font-mono text-[0.64rem] tracking-[0.02em] text-slate-600">
                                  {finding.detection} · {finding.subject}
                                </div>
                              </div>
                              <div className="sm:text-right">
                                <div className="text-[0.8rem] font-semibold text-slate-300">
                                  {finding.confidence}%
                                </div>
                                <div className="mt-1 text-[0.61rem] tracking-[0.05em] text-slate-600">
                                  CONFIDENCE
                                </div>
                              </div>
                            </motion.button>
                          ))}
                        </div>
                      </TabsContent>

                      <TabsContent value="recent" className="mt-0">
                        <div className="grid min-h-48 place-items-center rounded-xl border border-dashed border-slate-800 text-center">
                          <div>
                            <Activity className="mx-auto size-5 text-slate-600" />
                            <div className="mt-3 text-[0.8rem] font-medium text-slate-400">
                              Recent finding stream
                            </div>
                            <div className="mt-1 text-[0.7rem] text-slate-600">
                              Live API wiring is the next frontend slice.
                            </div>
                          </div>
                        </div>
                      </TabsContent>
                    </Tabs>
                  </CardContent>
                </Card>
              </motion.div>

              <motion.div
                variants={reveal}
                initial="hidden"
                whileInView="visible"
                viewport={{ once: true, amount: 0.15 }}
                transition={{ duration: 0.48, delay: 0.05 }}
              >
                <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                  <CardHeader className="px-5 pb-4 pt-5 sm:px-6">
                    <h3 className="text-[1.02rem] font-semibold tracking-[-0.01em] text-slate-100">
                      Threat activity
                    </h3>
                    <p className="text-[0.72rem] leading-5 tracking-[0.025em] text-slate-500">
                      Latest analyst-relevant events
                    </p>
                  </CardHeader>
                  <Separator className="bg-slate-800/80" />
                  <CardContent className="space-y-1 p-3">
                    {activity.map((event) => (
                      <button
                        key={event.time}
                        type="button"
                        className="grid w-full cursor-pointer grid-cols-[64px_58px_minmax(0,1fr)] items-center gap-2 rounded-xl px-3 py-3 text-left transition hover:bg-slate-900/60"
                      >
                        <span className="font-mono text-[0.64rem] text-slate-600">
                          {event.time}
                        </span>
                        <Badge
                          variant="outline"
                          className={cn(
                            "w-fit rounded-md px-2 py-0.5 text-[0.6rem] font-bold tracking-[0.08em]",
                            event.color === "violet" &&
                              "border-violet-400/30 bg-violet-400/10 text-violet-300",
                            event.color === "sky" &&
                              "border-sky-400/30 bg-sky-400/10 text-sky-300",
                            event.color === "amber" &&
                              "border-amber-300/30 bg-amber-300/10 text-amber-200",
                          )}
                        >
                          {event.type}
                        </Badge>
                        <div className="min-w-0">
                          <div className="truncate text-[0.78rem] font-medium text-slate-300">
                            {event.title}
                          </div>
                          <div className="mt-1 truncate font-mono text-[0.62rem] text-slate-600">
                            {event.subject}
                          </div>
                        </div>
                      </button>
                    ))}
                  </CardContent>
                </Card>
              </motion.div>
            </div>

            <div className="grid content-start gap-4">
              <motion.div
                variants={reveal}
                initial="hidden"
                whileInView="visible"
                viewport={{ once: true, amount: 0.15 }}
                transition={{ duration: 0.48, delay: 0.08 }}
              >
                <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                  <CardHeader className="px-5 pb-4 pt-5">
                    <h3 className="text-[1.02rem] font-semibold tracking-[-0.01em] text-slate-100">
                      Active investigations
                    </h3>
                    <p className="text-[0.72rem] leading-5 tracking-[0.025em] text-slate-500">
                      Cases currently in analyst workflow
                    </p>
                  </CardHeader>
                  <Separator className="bg-slate-800/80" />
                  <CardContent className="space-y-2.5 p-3">
                    {investigations.map((item) => (
                      <button
                        key={item.title}
                        type="button"
                        className="w-full cursor-pointer rounded-xl border border-slate-800/75 bg-slate-950/35 p-3.5 text-left transition hover:border-slate-700 hover:bg-slate-900/60"
                      >
                        <div className="flex flex-wrap items-center gap-2">
                          <SeverityBadge
                            label={item.priority}
                            tone={item.tone}
                          />
                          <span className="text-[0.8rem] font-semibold tracking-[0.01em] text-slate-200">
                            {item.title}
                          </span>
                        </div>
                        <div className="mt-3 flex items-center justify-between gap-3 text-[0.66rem] tracking-[0.025em]">
                          <span className="text-slate-600">{item.owner}</span>
                          <span className="font-medium text-slate-500">
                            {item.status}
                          </span>
                        </div>
                      </button>
                    ))}
                  </CardContent>
                </Card>
              </motion.div>

              <motion.div
                variants={reveal}
                initial="hidden"
                whileInView="visible"
                viewport={{ once: true, amount: 0.15 }}
                transition={{ duration: 0.48, delay: 0.12 }}
              >
                <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                  <CardHeader className="px-5 pb-4 pt-5">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <h3 className="text-[1.02rem] font-semibold tracking-[-0.01em] text-slate-100">
                          Detection coverage
                        </h3>
                        <p className="mt-1 text-[0.72rem] leading-5 tracking-[0.025em] text-slate-500">
                          Enabled deterministic rules
                        </p>
                      </div>
                      <Binary className="size-4 text-slate-700" />
                    </div>
                  </CardHeader>
                  <Separator className="bg-slate-800/80" />
                  <CardContent className="space-y-5 p-5">
                    {detections.map((item) => (
                      <div key={item.id}>
                        <div className="mb-2 flex items-center justify-between gap-3">
                          <div className="flex min-w-0 items-center gap-2.5">
                            <span
                              className={cn("size-2 rounded-full", item.color)}
                            />
                            <div className="min-w-0">
                              <div className="truncate font-mono text-[0.66rem] font-semibold tracking-[0.04em] text-slate-400">
                                {item.id}
                              </div>
                              <div className="mt-0.5 text-[0.68rem] text-slate-600">
                                {item.title}
                              </div>
                            </div>
                          </div>
                          <span className="text-[0.72rem] font-semibold text-slate-400">
                            {item.score}%
                          </span>
                        </div>
                        <Progress
                          value={item.score}
                          className="h-1.5 bg-slate-900"
                        />
                      </div>
                    ))}

                    <div className="grid grid-cols-3 gap-2 pt-1">
                      <div className="rounded-xl border border-slate-800/80 bg-slate-950/40 p-3 text-center">
                        <Target className="mx-auto size-4 text-sky-400" />
                        <div className="mt-2 text-[0.66rem] font-semibold text-slate-300">
                          3 rules
                        </div>
                      </div>
                      <div className="rounded-xl border border-slate-800/80 bg-slate-950/40 p-3 text-center">
                        <Workflow className="mx-auto size-4 text-violet-400" />
                        <div className="mt-2 text-[0.66rem] font-semibold text-slate-300">
                          audited
                        </div>
                      </div>
                      <div className="rounded-xl border border-slate-800/80 bg-slate-950/40 p-3 text-center">
                        <CheckCircle2 className="mx-auto size-4 text-emerald-400" />
                        <div className="mt-2 text-[0.66rem] font-semibold text-slate-300">
                          healthy
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            </div>
          </section>

          <motion.footer
            variants={reveal}
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, amount: 0.3 }}
            transition={{ duration: 0.4 }}
            className="mt-5 flex flex-col gap-2 border-t border-slate-900 py-5 text-[0.65rem] tracking-[0.04em] text-slate-700 sm:flex-row sm:items-center sm:justify-between"
          >
            <div className="flex items-center gap-2">
              <ShieldCheck className="size-3 text-emerald-500" />
              Sentinel gateway · PostgreSQL · NATS JetStream
            </div>
            <div className="flex items-center gap-2">
              <Database className="size-3" />
              Synthetic Northstar Energy environment
            </div>
          </motion.footer>
        </section>
      </div>
    </main>
  );
}
