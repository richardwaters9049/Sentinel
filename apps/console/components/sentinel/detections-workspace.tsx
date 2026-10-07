"use client";

import {
  AlertTriangle,
  Bell,
  CheckCircle2,
  Gauge,
  Radar,
  RefreshCcw,
  ShieldCheck,
  Target,
  Workflow,
} from "lucide-react";
import { motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

import SidebarDrawer from "@/components/sentinel/sidebar-drawer";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import {
  getDetectionMetrics,
  listDetections,
  setDetectionEnabled,
} from "@/lib/sentinel/client";
import type {
  DetectionMetric,
  DetectionRecord,
} from "@/lib/sentinel/types";

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

function metricFor(
  metrics: DetectionMetric[],
  detectionID: string,
): DetectionMetric | undefined {
  return metrics.find((metric) => metric.detection_id === detectionID);
}

function attackMappings(detection: DetectionRecord) {
  return detection.mitre.flatMap((mapping) => {
    const framework =
      typeof mapping.framework === "string" ? mapping.framework : "";
    const techniqueID =
      typeof mapping.technique_id === "string" ? mapping.technique_id : "";
    const technique =
      typeof mapping.technique === "string" ? mapping.technique : "";

    return framework && techniqueID
      ? [{ framework, techniqueID, technique }]
      : [];
  });
}

function safetyNote(detection: DetectionRecord) {
  const value = detection.definition.safety_note;
  return typeof value === "string" ? value : "";
}

export default function DetectionsWorkspace() {
  const [detections, setDetections] = useState<DetectionRecord[]>([]);
  const [metrics, setMetrics] = useState<DetectionMetric[]>([]);
  const [loading, setLoading] = useState(true);
  const [busyID, setBusyID] = useState("");
  const [error, setError] = useState("");

  async function refresh() {
    setLoading(true);
    setError("");
    try {
      const [catalogue, metricResponse] = await Promise.all([
        listDetections(),
        getDetectionMetrics(),
      ]);
      setDetections(catalogue.detections);
      setMetrics(metricResponse.metrics);
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Could not load detections",
      );
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    const controller = new AbortController();

    void Promise.all([
      listDetections(controller.signal),
      getDetectionMetrics(controller.signal),
    ])
      .then(([catalogue, metricResponse]) => {
        setDetections(catalogue.detections);
        setMetrics(metricResponse.metrics);
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(
          cause instanceof Error ? cause.message : "Could not load detections",
        );
      })
      .finally(() => setLoading(false));

    return () => controller.abort();
  }, []);

  async function toggleDetection(detection: DetectionRecord) {
    setBusyID(detection.id);
    setError("");

    try {
      const updated = await setDetectionEnabled(
        detection.id,
        !detection.enabled,
        "phase4-analyst",
      );
      setDetections((current) =>
        current.map((item) => (item.id === updated.id ? updated : item)),
      );
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Detection state could not be updated",
      );
    } finally {
      setBusyID("");
    }
  }

  const enabledCount = detections.filter((item) => item.enabled).length;
  const totalHits = metrics.reduce((total, item) => total + item.hit_count, 0);
  const openFindings = metrics.reduce((total, item) => total + item.open_count, 0);
  const averageFalsePositiveRate = useMemo(() => {
    if (!metrics.length) return 0;
    return (
      metrics.reduce((total, item) => total + item.false_positive_rate, 0) /
      metrics.length
    );
  }, [metrics]);

  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto min-h-screen max-w-[1700px]">
        <section className="px-4 py-5 sm:px-6 lg:px-8 xl:px-10">
          <motion.header
            initial={{ opacity: 0, y: -12 }}
            animate={{ opacity: 1, y: 0 }}
            className="mb-5 flex items-center justify-between gap-4"
          >
            <div>
              <div className="flex items-center gap-3">
                <SidebarDrawer />
                <span className="text-[0.72rem] font-bold tracking-[0.15em] text-slate-500">
                  SENTINEL
                </span>
              </div>
              <div className="mt-3 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.13em] text-cyan-300/75">
                <Radar className="size-3.5" />
                DETECTION ENGINEERING
              </div>
              <h1 className="mt-2 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.035em] text-white sm:text-[1.95rem]">
                Detection catalogue and runtime state
              </h1>
              <p className="mt-1.5 max-w-3xl text-[0.78rem] leading-6 tracking-[0.025em] text-slate-500">
                Inspect deterministic rules, review their observed performance,
                and enable or disable runtime evaluation through the audited backend.
              </p>
            </div>

            <div className="flex items-center gap-2.5">
              <Button
                type="button"
                onClick={() => void refresh()}
                variant="outline"
                size="icon"
                aria-label="Refresh detections"
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
              <Avatar className="size-9 border border-cyan-400/25">
                <AvatarFallback className="bg-gradient-to-br from-cyan-300 to-violet-500 text-[0.72rem] font-bold text-slate-950">
                  RW
                </AvatarFallback>
              </Avatar>
            </div>
          </motion.header>

          {error ? (
            <div
              role="alert"
              className="mb-4 flex items-center justify-between gap-4 rounded-2xl border border-rose-400/20 bg-rose-400/[0.06] p-4"
            >
              <div className="flex items-center gap-3">
                <AlertTriangle className="size-4 text-rose-300" />
                <div>
                  <div className="text-[0.75rem] font-semibold text-rose-200">
                    Detection service unavailable
                  </div>
                  <div className="mt-1 text-[0.65rem] text-rose-200/55">
                    {error}
                  </div>
                </div>
              </div>
              <Button
                type="button"
                onClick={() => void refresh()}
                variant="outline"
                size="sm"
                className="cursor-pointer border-rose-400/20 bg-rose-400/[0.05] text-rose-200 hover:bg-rose-400/10"
              >
                Retry
              </Button>
            </div>
          ) : null}

          <section
            aria-label="Detection metrics"
            className="mb-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
          >
            {[
              [enabledCount + "/" + detections.length, "enabled rules", ShieldCheck],
              [totalHits, "total hits", Target],
              [openFindings, "open findings", AlertTriangle],
              [
                Math.round(averageFalsePositiveRate * 100) + "%",
                "avg false positive",
                Gauge,
              ],
            ].map(([value, label, Icon], index) => {
              const IconComponent = Icon as typeof ShieldCheck;
              return (
                <motion.div
                  key={String(label)}
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: index * 0.05 }}
                  className="surface-card rounded-2xl border border-slate-800/85 p-4"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-[1.45rem] font-bold tracking-[-0.03em] text-white">
                        {String(value)}
                      </div>
                      <div className="mt-2 text-[0.6rem] font-semibold tracking-[0.1em] text-slate-600">
                        {String(label).toUpperCase()}
                      </div>
                    </div>
                    <IconComponent className="size-4 text-slate-700" />
                  </div>
                </motion.div>
              );
            })}
          </section>

          <section className="grid gap-4">
            {loading && detections.length === 0 ? (
              <div
                aria-live="polite"
                className="grid min-h-64 place-items-center rounded-[1.4rem] border border-slate-800/85 bg-[#0a0f17]/70"
              >
                <div className="text-center">
                  <RefreshCcw className="mx-auto size-5 animate-spin text-cyan-400" />
                  <div className="mt-3 text-[0.72rem] text-slate-500">
                    Loading detection catalogue…
                  </div>
                </div>
              </div>
            ) : null}

            {detections.map((detection, index) => {
              const metric = metricFor(metrics, detection.id);
              const performance =
                metric && metric.hit_count > 0
                  ? Math.max(
                      0,
                      Math.min(
                        100,
                        Math.round((1 - metric.false_positive_rate) * 100),
                      ),
                    )
                  : 100;

              return (
                <motion.article
                  key={detection.id}
                  initial={{ opacity: 0, y: 14 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: Math.min(index * 0.05, 0.3) }}
                  className="overflow-hidden rounded-[1.4rem] border border-slate-800/85 bg-[#0a0f17]/92"
                >
                  <div className="grid gap-0 xl:grid-cols-[minmax(0,1.4fr)_minmax(420px,0.9fr)]">
                    <div className="p-5 sm:p-6">
                      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                        <div>
                          <div className="flex flex-wrap items-center gap-2">
                            <Badge
                              variant="outline"
                              className={cn(
                                "rounded-md text-[0.57rem] font-bold tracking-[0.08em]",
                                severityClass(detection.severity),
                              )}
                            >
                              {detection.severity.toUpperCase()}
                            </Badge>
                            <Badge
                              variant="outline"
                              className="rounded-md border-slate-700 bg-slate-950/45 font-mono text-[0.57rem] text-slate-500"
                            >
                              {detection.id} · v{detection.version}
                            </Badge>
                          </div>
                          <h2 className="mt-3 text-[1rem] font-semibold text-slate-100">
                            {detection.title}
                          </h2>
                          <p className="mt-2 max-w-3xl text-[0.68rem] leading-5 tracking-[0.02em] text-slate-500">
                            {detection.description}
                          </p>

                          {attackMappings(detection).length > 0 ? (
                            <div className="mt-3 flex flex-wrap gap-2">
                              {attackMappings(detection).map((mapping) => (
                                <Badge
                                  key={
                                    mapping.framework +
                                    ":" +
                                    mapping.techniqueID
                                  }
                                  variant="outline"
                                  className="rounded-md border-violet-400/20 bg-violet-400/[0.05] px-2 py-1 text-[0.55rem] text-violet-300"
                                >
                                  {mapping.framework} · {mapping.techniqueID}
                                  {mapping.technique
                                    ? " · " + mapping.technique
                                    : ""}
                                </Badge>
                              ))}
                            </div>
                          ) : null}

                          {safetyNote(detection) ? (
                            <div className="mt-3 max-w-3xl rounded-xl border border-amber-300/15 bg-amber-300/[0.04] px-3 py-2 text-[0.61rem] leading-5 text-amber-100/55">
                              {safetyNote(detection)}
                            </div>
                          ) : null}
                        </div>

                        <div className="flex items-center gap-3 rounded-xl border border-slate-800/80 bg-slate-950/35 px-3 py-2.5">
                          <div>
                            <div className="text-[0.58rem] font-semibold tracking-[0.09em] text-slate-700">
                              RUNTIME
                            </div>
                            <div
                              className={cn(
                                "mt-1 text-[0.66rem] font-semibold",
                                detection.enabled
                                  ? "text-emerald-300"
                                  : "text-slate-500",
                              )}
                            >
                              {detection.enabled ? "Enabled" : "Disabled"}
                            </div>
                          </div>
                          <Switch
                            checked={detection.enabled}
                            disabled={busyID === detection.id}
                            onCheckedChange={() => void toggleDetection(detection)}
                            aria-label={
                              (detection.enabled ? "Disable " : "Enable ") +
                              detection.title
                            }
                            className="cursor-pointer data-checked:bg-emerald-400"
                          />
                        </div>
                      </div>

                      <div className="mt-5 grid gap-2 sm:grid-cols-4">
                        {[
                          [metric?.hit_count ?? 0, "hits"],
                          [metric?.open_count ?? 0, "open"],
                          [metric?.confirmed_count ?? 0, "confirmed"],
                          [metric?.false_positive_count ?? 0, "false positive"],
                        ].map(([value, label]) => (
                          <div
                            key={String(label)}
                            className="rounded-xl border border-slate-800/70 bg-slate-950/30 p-3"
                          >
                            <div className="text-[0.9rem] font-bold text-slate-300">
                              {String(value)}
                            </div>
                            <div className="mt-1 text-[0.54rem] font-semibold tracking-[0.08em] text-slate-700">
                              {String(label).toUpperCase()}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>

                    <div className="border-t border-slate-800/80 p-5 xl:border-l xl:border-t-0">
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <div className="text-[0.64rem] font-semibold text-slate-300">
                            Observed quality
                          </div>
                          <div className="mt-1 text-[0.59rem] text-slate-600">
                            Based on persisted false-positive disposition
                          </div>
                        </div>
                        <span className="text-[0.8rem] font-semibold text-cyan-300">
                          {performance}%
                        </span>
                      </div>
                      <Progress
                        value={performance}
                        className="mt-3 h-1.5 bg-slate-900"
                      />

                      <div className="mt-5 grid gap-3">
                        <div className="rounded-xl border border-slate-800/70 bg-slate-950/30 p-3">
                          <div className="text-[0.56rem] font-semibold tracking-[0.09em] text-slate-700">
                            LAST TRIGGERED
                          </div>
                          <div className="mt-2 text-[0.66rem] text-slate-400">
                            {metric?.last_triggered_at
                              ? new Intl.DateTimeFormat("en-GB", {
                                  day: "2-digit",
                                  month: "short",
                                  hour: "2-digit",
                                  minute: "2-digit",
                                }).format(new Date(metric.last_triggered_at))
                              : "No recorded hit"}
                          </div>
                        </div>

                        <div className="flex items-center gap-2 text-[0.6rem] leading-5 text-slate-600">
                          <Workflow className="size-3.5 shrink-0 text-violet-400" />
                          Runtime changes are persisted and audited by the Go
                          gateway.
                        </div>
                        <div className="flex items-center gap-2 text-[0.6rem] leading-5 text-slate-600">
                          <CheckCircle2 className="size-3.5 shrink-0 text-emerald-400" />
                          Rules remain deterministic; metrics report observed
                          analyst outcomes.
                        </div>
                      </div>
                    </div>
                  </div>
                </motion.article>
              );
            })}
          </section>
        </section>
      </div>
    </main>
  );
}
