"use client";

import {
  AlertTriangle,
  Binary,
  Database,
  ExternalLink,
  Fingerprint,
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
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import {
  getIntelligenceMetrics,
  listIntelligenceIndicators,
  listIntelligenceMatches,
  listIntelligenceSources,
  setIntelligenceSourceEnabled,
} from "@/lib/sentinel/client";
import type {
  EventEnrichment,
  IntelligenceIndicator,
  IntelligenceMetrics,
  IntelligenceSource,
} from "@/lib/sentinel/types";

function confidenceTone(value: number) {
  if (value >= 85) return "text-emerald-300";
  if (value >= 65) return "text-amber-200";
  return "text-slate-400";
}

function classification(indicator: IntelligenceIndicator) {
  const value = indicator.context.classification;
  return typeof value === "string" ? value : "unclassified";
}

export default function IntelligenceWorkspace() {
  const [metrics, setMetrics] = useState<IntelligenceMetrics | null>(null);
  const [indicators, setIndicators] = useState<IntelligenceIndicator[]>([]);
  const [matches, setMatches] = useState<EventEnrichment[]>([]);
  const [sources, setSources] = useState<IntelligenceSource[]>([]);
  const [loading, setLoading] = useState(true);
  const [busySourceID, setBusySourceID] = useState("");
  const [error, setError] = useState("");

  async function refresh() {
    setLoading(true);
    setError("");
    try {
      const [metricData, indicatorData, matchData, sourceData] =
        await Promise.all([
          getIntelligenceMetrics(),
          listIntelligenceIndicators(),
          listIntelligenceMatches(50),
          listIntelligenceSources(),
        ]);
      setMetrics(metricData);
      setIndicators(indicatorData.indicators);
      setMatches(matchData.matches);
      setSources(sourceData.sources);
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Threat intelligence could not be loaded",
      );
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    const controller = new AbortController();

    void Promise.all([
      getIntelligenceMetrics(controller.signal),
      listIntelligenceIndicators(controller.signal),
      listIntelligenceMatches(50, controller.signal),
      listIntelligenceSources(controller.signal),
    ])
      .then(([metricData, indicatorData, matchData, sourceData]) => {
        setMetrics(metricData);
        setIndicators(indicatorData.indicators);
        setMatches(matchData.matches);
        setSources(sourceData.sources);
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(
          cause instanceof Error
            ? cause.message
            : "Threat intelligence could not be loaded",
        );
      })
      .finally(() => setLoading(false));

    return () => controller.abort();
  }, []);

  const conflictingValues = useMemo(() => {
    const classifications = new Map<string, Set<string>>();
    for (const match of matches) {
      const key = match.indicator_type + ":" + match.observed_value;
      const value =
        typeof match.context.classification === "string"
          ? match.context.classification
          : "unclassified";
      const current = classifications.get(key) ?? new Set<string>();
      current.add(value);
      classifications.set(key, current);
    }

    return new Set(
      Array.from(classifications.entries())
        .filter(([, values]) => values.size > 1)
        .map(([key]) => key),
    );
  }, [matches]);

  async function toggleSource(source: IntelligenceSource) {
    setBusySourceID(source.id);
    setError("");
    try {
      const updated = await setIntelligenceSourceEnabled(
        source.id,
        !source.active,
        "phase6-analyst",
      );
      setSources((current) =>
        current.map((item) => (item.id === updated.id ? updated : item)),
      );
      await refresh();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Intelligence source state could not be updated",
      );
    } finally {
      setBusySourceID("");
    }
  }

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
              <div className="mt-3 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.13em] text-violet-300/75">
                <Binary className="size-3.5" />
                INTELLIGENCE & ENRICHMENT
              </div>
              <h1 className="mt-2 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.035em] text-white sm:text-[1.95rem]">
                Provenance-aware threat intelligence
              </h1>
              <p className="mt-1.5 max-w-3xl text-[0.78rem] leading-6 tracking-[0.025em] text-slate-500">
                Correlate persisted telemetry with local IOC fixtures while
                preserving source provenance, confidence, and analyst-visible context.
              </p>
            </div>

            <Button
              type="button"
              onClick={() => void refresh()}
              variant="outline"
              size="icon"
              aria-label="Refresh threat intelligence"
              className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900"
            >
              <RefreshCcw className={cn("size-4", loading && "animate-spin")} />
            </Button>
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
                    Intelligence service unavailable
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
                className="cursor-pointer border-rose-400/20 bg-rose-400/[0.04] text-rose-200"
              >
                Retry
              </Button>
            </div>
          ) : null}

          <section className="mb-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
            {[
              [metrics?.active_sources ?? 0, "active sources", Database],
              [metrics?.active_indicators ?? 0, "active indicators", Fingerprint],
              [metrics?.enriched_events ?? 0, "enriched events", Binary],
              [metrics?.total_matches ?? 0, "IOC matches", Sparkles],
              [metrics?.high_confidence_hits ?? 0, "high-confidence", ShieldCheck],
            ].map(([value, label, Icon], index) => {
              const IconComponent = Icon as typeof Database;
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

          <section className="grid gap-4 xl:grid-cols-[minmax(0,1.25fr)_minmax(380px,0.8fr)]">
            <div className="grid content-start gap-4">
              <Card className="surface-card rounded-[1.4rem] border-slate-800/85 bg-transparent py-0">
                <CardHeader className="px-5 pb-4 pt-5">
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <h2 className="text-[0.95rem] font-semibold text-slate-100">
                        Indicator catalogue
                      </h2>
                      <p className="mt-1 text-[0.64rem] leading-5 text-slate-600">
                        Local-first fixtures with explicit source and indicator confidence.
                      </p>
                    </div>
                    <Badge
                      variant="outline"
                      className="border-violet-400/20 bg-violet-400/[0.05] text-[0.57rem] text-violet-300"
                    >
                      {indicators.length} IOC{indicators.length === 1 ? "" : "s"}
                    </Badge>
                  </div>
                </CardHeader>
                <Separator className="bg-slate-800/80" />
                <CardContent className="space-y-3 p-4">
                  {indicators.map((indicator, index) => {
                    const effective = Math.round(
                      (indicator.source_confidence * indicator.confidence) / 100,
                    );
                    return (
                      <motion.article
                        key={indicator.id}
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: Math.min(index * 0.04, 0.25) }}
                        className="rounded-2xl border border-slate-800/75 bg-slate-950/30 p-4"
                      >
                        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                          <div className="min-w-0">
                            <div className="flex flex-wrap items-center gap-2">
                              <Badge
                                variant="outline"
                                className="border-sky-400/20 bg-sky-400/[0.05] text-[0.55rem] text-sky-300"
                              >
                                {indicator.indicator_type.toUpperCase()}
                              </Badge>
                              <Badge
                                variant="outline"
                                className="border-slate-700 bg-slate-900/60 text-[0.55rem] text-slate-500"
                              >
                                {classification(indicator).toUpperCase()}
                              </Badge>
                            </div>
                            <div className="mt-3 break-all font-mono text-[0.78rem] font-semibold text-slate-200">
                              {indicator.value}
                            </div>
                            <div className="mt-2 text-[0.62rem] leading-5 text-slate-600">
                              {typeof indicator.context.summary === "string"
                                ? indicator.context.summary
                                : "No additional context."}
                            </div>
                            <div className="mt-3 flex flex-wrap gap-1.5">
                              {indicator.tags.map((tag) => (
                                <span
                                  key={tag}
                                  className="rounded-md border border-slate-800 bg-slate-900/50 px-2 py-1 text-[0.53rem] text-slate-600"
                                >
                                  {tag}
                                </span>
                              ))}
                            </div>
                          </div>

                          <div className="w-full max-w-[220px] rounded-xl border border-slate-800/75 bg-[#080c12] p-3">
                            <div className="flex items-center justify-between text-[0.58rem] text-slate-600">
                              <span>effective confidence</span>
                              <span className={cn("font-semibold", confidenceTone(effective))}>
                                {effective}%
                              </span>
                            </div>
                            <Progress value={effective} className="mt-2 h-1.5 bg-slate-900" />
                            <div className="mt-3 grid grid-cols-2 gap-2 text-[0.55rem] text-slate-700">
                              <span>source {indicator.source_confidence}%</span>
                              <span className="text-right">
                                IOC {indicator.confidence}%
                              </span>
                            </div>
                          </div>
                        </div>
                      </motion.article>
                    );
                  })}
                </CardContent>
              </Card>
            </div>

            <div className="grid content-start gap-4">
              <Card className="overflow-hidden rounded-[1.4rem] border border-violet-400/15 bg-gradient-to-br from-violet-400/[0.07] via-[#0c1119] to-cyan-400/[0.04] py-0">
                <CardHeader className="px-5 pb-4 pt-5">
                  <div className="flex items-center gap-2">
                    <ShieldCheck className="size-4 text-violet-300" />
                    <div>
                      <h2 className="text-[0.9rem] font-semibold text-slate-100">
                        Sources & provenance
                      </h2>
                      <p className="mt-1 text-[0.61rem] leading-5 text-violet-100/40">
                        Runtime source state is audited and immediately invalidates
                        cached intelligence results.
                      </p>
                    </div>
                  </div>
                </CardHeader>
                <Separator className="bg-violet-400/10" />
                <CardContent className="space-y-3 p-5">
                  <div className="text-[0.65rem] leading-5 text-slate-500">
                    Sentinel keeps source confidence independent from indicator
                    confidence and preserves the provenance used when a match is
                    recorded.
                  </div>
                  {sources.map((source) => (
                    <div
                      key={source.id}
                      className="rounded-xl border border-violet-400/10 bg-slate-950/30 p-3"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="text-[0.66rem] font-semibold text-violet-200">
                            {source.name}
                          </div>
                          <div className="mt-1 text-[0.56rem] text-violet-200/40">
                            {source.source_type} · default confidence{" "}
                            {source.default_confidence}%
                          </div>
                        </div>
                        <Switch
                          checked={source.active}
                          disabled={busySourceID === source.id}
                          onCheckedChange={() => void toggleSource(source)}
                          aria-label={
                            (source.active ? "Disable " : "Enable ") + source.name
                          }
                          className="cursor-pointer data-checked:bg-emerald-400"
                        />
                      </div>
                      <div className="mt-2 text-[0.57rem] leading-5 text-slate-600">
                        {source.description}
                      </div>
                    </div>
                  ))}
                </CardContent>
              </Card>

              <Card className="surface-card rounded-[1.4rem] border-slate-800/85 bg-transparent py-0">
                <CardHeader className="px-5 pb-4 pt-5">
                  <h2 className="text-[0.9rem] font-semibold text-slate-100">
                    Recent enrichment matches
                  </h2>
                  <p className="mt-1 text-[0.62rem] leading-5 text-slate-600">
                    IOC context attached to persisted telemetry events.
                  </p>
                </CardHeader>
                <Separator className="bg-slate-800/80" />
                <CardContent className="space-y-2 p-4">
                  {matches.length ? (
                    matches.slice(0, 12).map((match) => (
                      <div
                        key={match.id}
                        className="rounded-xl border border-slate-800/70 bg-slate-950/30 p-3"
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0">
                            <div className="flex flex-wrap items-center gap-2">
                              <div className="truncate font-mono text-[0.64rem] font-semibold text-slate-300">
                                {match.observed_value}
                              </div>
                              {conflictingValues.has(
                                match.indicator_type + ":" + match.observed_value,
                              ) ? (
                                <Badge
                                  variant="outline"
                                  className="border-amber-300/20 bg-amber-300/[0.05] text-[0.5rem] text-amber-200"
                                >
                                  SOURCE DISAGREEMENT
                                </Badge>
                              ) : null}
                            </div>
                            <div className="mt-1 truncate text-[0.57rem] text-slate-700">
                              {match.event_field} · {match.indicator_id}
                            </div>
                          </div>
                          <span
                            className={cn(
                              "text-[0.72rem] font-semibold",
                              confidenceTone(match.effective_confidence),
                            )}
                          >
                            {match.effective_confidence}%
                          </span>
                        </div>
                        <div className="mt-3 flex items-center justify-between gap-3">
                          <span className="text-[0.56rem] text-slate-600">
                            {match.source_name}
                          </span>
                          <span className="font-mono text-[0.52rem] text-slate-700">
                            {new Intl.DateTimeFormat("en-GB", {
                              day: "2-digit",
                              month: "short",
                              hour: "2-digit",
                              minute: "2-digit",
                            }).format(new Date(match.matched_at))}
                          </span>
                        </div>
                      </div>
                    ))
                  ) : (
                    <div className="grid min-h-40 place-items-center text-center">
                      <div>
                        <ExternalLink className="mx-auto size-4 text-slate-700" />
                        <div className="mt-3 text-[0.67rem] text-slate-600">
                          No telemetry has matched an indicator yet.
                        </div>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </section>
        </section>
      </div>
    </main>
  );
}
