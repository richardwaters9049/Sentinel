"use client";

import {
  Activity,
  AlertTriangle,
  BrainCircuit,
  Clock3,
  RefreshCcw,
  ShieldCheck,
  Sparkles,
} from "lucide-react";
import { motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

import SidebarDrawer from "@/components/sentinel/sidebar-drawer";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import {
  getBehaviourMetrics,
  listBehaviourScores,
} from "@/lib/sentinel/client";
import type {
  BehaviourMetrics,
  BehaviourScore,
} from "@/lib/sentinel/types";

function scoreTone(score: number) {
  if (score >= 85) return "text-rose-300";
  if (score >= 65) return "text-amber-200";
  return "text-emerald-300";
}

function severityClass(severity: BehaviourScore["severity"]) {
  if (severity === "high") {
    return "border-rose-400/25 bg-rose-400/[0.07] text-rose-300";
  }
  if (severity === "medium") {
    return "border-amber-300/25 bg-amber-300/[0.07] text-amber-200";
  }
  return "border-emerald-400/20 bg-emerald-400/[0.05] text-emerald-300";
}

export default function BehaviourWorkspace() {
  const [metrics, setMetrics] = useState<BehaviourMetrics | null>(null);
  const [scores, setScores] = useState<BehaviourScore[]>([]);
  const [selectedID, setSelectedID] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function refresh() {
    setLoading(true);
    setError("");
    try {
      const [metricData, scoreData] = await Promise.all([
        getBehaviourMetrics(),
        listBehaviourScores(100),
      ]);
      setMetrics(metricData);
      setScores(scoreData.scores);
      setSelectedID((current) => current || scoreData.scores[0]?.event_id || "");
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Behavioural analytics could not be loaded",
      );
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    const controller = new AbortController();
    void Promise.all([
      getBehaviourMetrics(controller.signal),
      listBehaviourScores(100, controller.signal),
    ])
      .then(([metricData, scoreData]) => {
        setMetrics(metricData);
        setScores(scoreData.scores);
        setSelectedID(scoreData.scores[0]?.event_id ?? "");
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(
          cause instanceof Error
            ? cause.message
            : "Behavioural analytics could not be loaded",
        );
      })
      .finally(() => setLoading(false));

    return () => controller.abort();
  }, []);

  const selected =
    scores.find((score) => score.event_id === selectedID) ?? scores[0];

  const anomalyRate = useMemo(() => {
    if (!metrics?.total_scores) return 0;
    return Math.round(
      (metrics.anomalous_scores / metrics.total_scores) * 100,
    );
  }, [metrics]);

  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto min-h-screen max-w-[1760px]">
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
                <BrainCircuit className="size-3.5" />
                BEHAVIOURAL ANALYTICS
              </div>
              <h1 className="mt-2 text-[1.7rem] font-bold leading-[1.18] tracking-[-0.035em] text-white sm:text-[2rem]">
                Baseline deviation explorer
              </h1>
              <p className="mt-1.5 max-w-3xl text-[0.79rem] leading-6 tracking-[0.025em] text-slate-500">
                Compare observed activity with Sentinel&apos;s synthetic Northstar
                baseline. Model scores remain supporting evidence, not security verdicts.
              </p>
            </div>

            <Button
              type="button"
              onClick={() => void refresh()}
              variant="outline"
              size="icon"
              aria-label="Refresh behavioural analytics"
              className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900"
            >
              <RefreshCcw className={cn("size-4", loading && "animate-spin")} />
            </Button>
          </motion.header>

          {error ? (
            <div className="mb-4 flex items-center gap-3 rounded-2xl border border-rose-400/20 bg-rose-400/[0.06] p-4">
              <AlertTriangle className="size-4 text-rose-300" />
              <span className="text-[0.68rem] text-rose-200/70">{error}</span>
            </div>
          ) : null}

          <section className="mb-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {[
              [metrics?.total_scores ?? 0, "events scored", Activity],
              [metrics?.anomalous_scores ?? 0, "baseline deviations", Sparkles],
              [metrics?.high_severity_scores ?? 0, "high-score events", AlertTriangle],
              [anomalyRate + "%", "deviation rate", BrainCircuit],
            ].map(([value, label, Icon], index) => {
              const IconComponent = Icon as typeof Activity;
              return (
                <motion.div
                  key={String(label)}
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: index * 0.045 }}
                  className="surface-card rounded-2xl border border-slate-800/85 p-4"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-[1.45rem] font-bold tracking-[-0.03em] text-white">
                        {String(value)}
                      </div>
                      <div className="mt-2 text-[0.57rem] font-semibold tracking-[0.1em] text-slate-600">
                        {String(label).toUpperCase()}
                      </div>
                    </div>
                    <IconComponent className="size-4 text-slate-700" />
                  </div>
                </motion.div>
              );
            })}
          </section>

          <section className="grid gap-4 xl:grid-cols-[minmax(0,0.95fr)_minmax(430px,1.05fr)]">
            <Card className="surface-card rounded-[1.4rem] border-slate-800/85 bg-transparent py-0">
              <CardHeader className="px-5 pb-4 pt-5">
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <h2 className="text-[0.95rem] font-semibold text-slate-100">
                      Recent behavioural scores
                    </h2>
                    <p className="mt-1 text-[0.64rem] leading-5 text-slate-600">
                      Ranked observations from the Python Isolation Forest service.
                    </p>
                  </div>
                  <Badge
                    variant="outline"
                    className="border-cyan-400/20 bg-cyan-400/[0.05] text-[0.56rem] text-cyan-300"
                  >
                    {scores.length} SCORES
                  </Badge>
                </div>
              </CardHeader>
              <Separator className="bg-slate-800/80" />
              <CardContent className="space-y-2 p-4">
                {scores.map((score, index) => (
                  <motion.button
                    key={score.event_id}
                    type="button"
                    initial={{ opacity: 0, x: -8 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: Math.min(index * 0.025, 0.25) }}
                    onClick={() => setSelectedID(score.event_id)}
                    className={cn(
                      "grid w-full cursor-pointer gap-3 rounded-xl border p-3 text-left transition sm:grid-cols-[70px_minmax(0,1fr)_96px] sm:items-center",
                      selected?.event_id === score.event_id
                        ? "border-cyan-400/25 bg-cyan-400/[0.05]"
                        : "border-slate-800/70 bg-slate-950/25 hover:border-slate-700 hover:bg-slate-900/45",
                    )}
                  >
                    <div className={cn("text-[1rem] font-bold", scoreTone(score.anomaly_score))}>
                      {score.anomaly_score}
                    </div>
                    <div className="min-w-0">
                      <div className="truncate text-[0.7rem] font-semibold text-slate-300">
                        {score.entity_id}
                      </div>
                      <div className="mt-1 flex flex-wrap items-center gap-2">
                        <span className="truncate font-mono text-[0.56rem] text-slate-700">
                          {score.event_id}
                        </span>
                        <span className="text-[0.5rem] font-semibold tracking-[0.08em] text-slate-700">
                          {score.entity_type.toUpperCase()}
                        </span>
                      </div>
                    </div>
                    <Badge
                      variant="outline"
                      className={cn(
                        "w-fit justify-self-start text-[0.53rem] sm:justify-self-end",
                        severityClass(score.severity),
                      )}
                    >
                      {score.severity.toUpperCase()}
                    </Badge>
                  </motion.button>
                ))}
                {!loading && scores.length === 0 ? (
                  <div className="grid min-h-48 place-items-center text-center text-[0.68rem] leading-5 text-slate-600">
                    No behavioural scores have been persisted yet.
                  </div>
                ) : null}
              </CardContent>
            </Card>

            <div className="grid content-start gap-4">
              <Card className="overflow-hidden rounded-[1.4rem] border border-cyan-400/15 bg-gradient-to-br from-cyan-400/[0.07] via-[#0c1119] to-violet-400/[0.05] py-0">
                <CardHeader className="px-5 pb-4 pt-5">
                  <div className="flex items-center gap-2">
                    <ShieldCheck className="size-4 text-cyan-300" />
                    <div>
                      <h2 className="text-[0.9rem] font-semibold text-slate-100">
                        Model contract
                      </h2>
                      <p className="mt-1 text-[0.61rem] leading-5 text-cyan-100/40">
                        Explainable anomaly scoring with a deterministic synthetic baseline.
                      </p>
                    </div>
                  </div>
                </CardHeader>
                <Separator className="bg-cyan-400/10" />
                <CardContent className="grid gap-3 p-5 sm:grid-cols-3">
                  <div className="rounded-xl border border-cyan-400/10 bg-slate-950/30 p-3">
                    <div className="text-[0.56rem] font-semibold tracking-[0.08em] text-slate-700">
                      MODEL
                    </div>
                    <div className="mt-2 text-[0.68rem] font-semibold text-cyan-200">
                      {selected?.model_kind ?? "IsolationForest"}
                    </div>
                  </div>
                  <div className="rounded-xl border border-cyan-400/10 bg-slate-950/30 p-3">
                    <div className="text-[0.56rem] font-semibold tracking-[0.08em] text-slate-700">
                      THRESHOLD
                    </div>
                    <div className="mt-2 text-[0.68rem] font-semibold text-cyan-200">
                      {selected?.threshold ?? 65}
                    </div>
                  </div>
                  <div className="rounded-xl border border-cyan-400/10 bg-slate-950/30 p-3">
                    <div className="text-[0.56rem] font-semibold tracking-[0.08em] text-slate-700">
                      ROLE
                    </div>
                    <div className="mt-2 text-[0.68rem] font-semibold text-cyan-200">
                      Supporting evidence
                    </div>
                  </div>
                </CardContent>
              </Card>


              {selected ? (
                <motion.div
                  key={selected.event_id + "-baseline"}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                >
                  <Card className="overflow-hidden rounded-[1.4rem] border border-violet-400/15 bg-gradient-to-br from-violet-400/[0.06] via-[#0c1119] to-cyan-400/[0.03] py-0">
                    <CardHeader className="px-5 pb-4 pt-5">
                      <div className="flex items-center justify-between gap-4">
                        <div>
                          <div className="text-[0.58rem] font-semibold tracking-[0.1em] text-violet-300/65">
                            ENTITY BASELINE
                          </div>
                          <h2 className="mt-2 text-[0.95rem] font-semibold text-slate-100">
                            Rolling context for {selected.entity_id}
                          </h2>
                          <p className="mt-1 text-[0.61rem] leading-5 text-slate-600">
                            Historical behaviour observed before this event was scored.
                          </p>
                        </div>
                        <Badge
                          variant="outline"
                          className="border-violet-400/20 bg-violet-400/[0.05] text-[0.54rem] text-violet-300"
                        >
                          {selected.entity_type.toUpperCase()}
                        </Badge>
                      </div>
                    </CardHeader>
                    <Separator className="bg-violet-400/10" />
                    <CardContent className="grid gap-2 p-4 sm:grid-cols-2">
                      {[
                        [
                          selected.baseline.prior_events_60m,
                          "events / 60m",
                        ],
                        [
                          selected.baseline.prior_events_24h,
                          "events / 24h",
                        ],
                        [
                          selected.baseline.unique_destination_ips_24h,
                          "destinations / 24h",
                        ],
                        [
                          selected.baseline.auth_failures_60m,
                          "auth failures / 60m",
                        ],
                        [
                          Math.round(
                            selected.baseline.destination_diversity_24h * 100,
                          ) + "%",
                          "destination diversity",
                        ],
                        [
                          Math.round(
                            selected.baseline.ot_activity_rate_24h * 100,
                          ) + "%",
                          "OT activity rate",
                        ],
                      ].map(([value, label]) => (
                        <div
                          key={String(label)}
                          className="rounded-xl border border-violet-400/10 bg-slate-950/30 p-3"
                        >
                          <div className="text-[0.86rem] font-semibold text-violet-100">
                            {String(value)}
                          </div>
                          <div className="mt-1 text-[0.52rem] font-semibold tracking-[0.08em] text-slate-700">
                            {String(label).toUpperCase()}
                          </div>
                        </div>
                      ))}
                    </CardContent>
                  </Card>
                </motion.div>
              ) : null}

              {selected ? (
                <motion.div
                  key={selected.event_id}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                >
                  <Card className="surface-card rounded-[1.4rem] border-slate-800/85 bg-transparent py-0">
                    <CardHeader className="px-5 pb-4 pt-5">
                      <div className="flex items-start justify-between gap-4">
                        <div>
                          <div className="text-[0.58rem] font-semibold tracking-[0.1em] text-slate-700">
                            EXPLAINABILITY
                          </div>
                          <h2 className="mt-2 text-[0.95rem] font-semibold text-slate-100">
                            Why this behaviour differs
                          </h2>
                        </div>
                        <div className={cn("text-[1.45rem] font-bold", scoreTone(selected.anomaly_score))}>
                          {selected.anomaly_score}
                        </div>
                      </div>
                      <Progress
                        value={selected.anomaly_score}
                        className="mt-3 h-1.5 bg-slate-900"
                      />
                    </CardHeader>
                    <Separator className="bg-slate-800/80" />
                    <CardContent className="space-y-2 p-4">
                      {selected.explanations.map((item) => (
                        <div
                          key={item.feature}
                          className="rounded-xl border border-slate-800/70 bg-slate-950/30 p-3"
                        >
                          <div className="flex items-center justify-between gap-3">
                            <span className="text-[0.65rem] font-semibold text-slate-300">
                              {item.feature.replaceAll("_", " ")}
                            </span>
                            <span className="text-[0.58rem] font-semibold text-violet-300">
                              {item.deviation.toFixed(2)}σ
                            </span>
                          </div>
                          <p className="mt-2 text-[0.62rem] leading-5 text-slate-600">
                            {item.message}
                          </p>
                          <div className="mt-2 flex gap-4 font-mono text-[0.54rem] text-slate-700">
                            <span>observed {item.observed}</span>
                            <span>baseline {item.baseline}</span>
                          </div>
                        </div>
                      ))}
                      <div className="mt-3 flex items-center gap-2 rounded-xl border border-slate-800/70 bg-slate-950/20 p-3 text-[0.58rem] leading-5 text-slate-600">
                        <Clock3 className="size-3.5 shrink-0" />
                        {selected.model_version}
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              ) : null}
            </div>
          </section>
        </section>
      </div>
    </main>
  );
}
