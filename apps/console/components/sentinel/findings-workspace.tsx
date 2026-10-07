"use client";

import {
  Activity, AlertTriangle, Bell, Binary, CheckCircle2, ChevronRight, Clock3,
  Cpu, Crosshair, FilePlus2, Fingerprint, GitBranch, Network, Radar,
  RefreshCcw, Search, ShieldAlert, Sparkles,
} from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

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
  createInvestigation,
  getFinding,
  getFindingEvidence,
  listFindings,
  updateFindingStatus,
} from "@/lib/sentinel/client";
import type {
  Finding,
  FindingDetail,
  FindingEvidenceContext,
  FindingStatus,
} from "@/lib/sentinel/types";

function severityClass(severity: string) {
  switch (severity.toLowerCase()) {
    case "critical":
      return "border-rose-400/40 bg-rose-400/10 text-rose-200";
    case "high":
      return "border-orange-300/35 bg-orange-300/10 text-orange-200";
    case "medium":
      return "border-amber-300/35 bg-amber-300/10 text-amber-200";
    default:
      return "border-sky-400/30 bg-sky-400/10 text-sky-300";
  }
}

function allowedTransitions(status: FindingStatus): FindingStatus[] {
  switch (status) {
    case "new":
      return ["triaged"];
    case "triaged":
      return ["investigating", "false_positive", "benign_expected", "duplicate"];
    case "investigating":
      return ["confirmed", "false_positive", "benign_expected", "duplicate"];
    case "confirmed":
      return ["contained", "closed"];
    case "contained":
      return ["closed"];
    default:
      return [];
  }
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
}

function explain(detail: FindingDetail) {
  const reasons: string[] = [];
  const evidence = detail.evidence;
  if (typeof evidence.failure_count === "number") {
    reasons.push(String(evidence.failure_count) + " failed authentications observed");
  }
  if (
    typeof evidence.source_zone === "string" &&
    typeof evidence.destination_zone === "string"
  ) {
    reasons.push(
      "Security boundary crossed: " +
        evidence.source_zone +
        " → " +
        evidence.destination_zone,
    );
  }
  if (typeof evidence.identity_type === "string") {
    reasons.push("Identity type observed: " + evidence.identity_type);
  }
  if (typeof evidence.destination_port === "number") {
    reasons.push(
      "Destination port " +
        String(evidence.destination_port) +
        " was part of the evidence",
    );
  }
  if (typeof evidence.ot_device_type === "string") {
    reasons.push("OT device type: " + evidence.ot_device_type.toUpperCase());
  }
  if (typeof evidence.ot_protocol === "string") {
    reasons.push("OT protocol context: " + evidence.ot_protocol);
  }
  if (typeof evidence.ot_operation === "string") {
    reasons.push(
      "Observed OT operation: " + evidence.ot_operation.replaceAll("_", " "),
    );
  }
  if (typeof evidence.ot_authorization === "string") {
    reasons.push(
      "Authorization classification: " + evidence.ot_authorization,
    );
  }
  if (typeof evidence.window_seconds === "number") {
    reasons.push(
      "Correlated over a " +
        String(Math.round(evidence.window_seconds / 60)) +
        "-minute evidence window",
    );
  }
  if (detail.events.length > 0) {
    reasons.push(
      String(detail.events.length) +
        " directly linked evidence event" +
        (detail.events.length === 1 ? "" : "s"),
    );
  }
  if (reasons.length === 0) {
    reasons.push("Finding matched the deterministic detection rule conditions");
    reasons.push("Linked telemetry was preserved as direct evidence");
  }
  return reasons.slice(0, 5);
}

export default function FindingsWorkspace() {
  const router = useRouter();
  const [findings, setFindings] = useState<Finding[]>([]);
  const [selectedID, setSelectedID] = useState("");
  const [detail, setDetail] = useState<FindingDetail | null>(null);
  const [evidence, setEvidence] = useState<FindingEvidenceContext | null>(null);
  const [contextMinutes, setContextMinutes] = useState(5);
  const [search, setSearch] = useState("");
  const [loadingList, setLoadingList] = useState(true);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [actionBusy, setActionBusy] = useState(false);
  const [error, setError] = useState("");

  const filteredFindings = useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) return findings;
    return findings.filter((finding) =>
      [
        finding.title,
        finding.detection_id,
        finding.severity,
        finding.status,
        finding.id,
      ].some((value) => value.toLowerCase().includes(term)),
    );
  }, [findings, search]);

  async function loadFindings() {
    setLoadingList(true);
    setError("");
    try {
      const response = await listFindings(200);
      setFindings(response.findings);
      setSelectedID((current) => current || response.findings[0]?.id || "");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not load findings");
    } finally {
      setLoadingList(false);
    }
  }

  useEffect(() => {
    const controller = new AbortController();

    void listFindings(200, controller.signal)
      .then((response) => {
        setFindings(response.findings);
        const firstID = response.findings[0]?.id ?? "";
        if (firstID) {
          setLoadingDetail(true);
          setSelectedID(firstID);
        }
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(
          cause instanceof Error ? cause.message : "Could not load findings",
        );
      })
      .finally(() => setLoadingList(false));

    return () => controller.abort();
  }, []);

  useEffect(() => {
    if (!selectedID) return;
    const controller = new AbortController();

    void Promise.all([
      getFinding(selectedID, controller.signal),
      getFindingEvidence(selectedID, contextMinutes, controller.signal),
    ])
      .then(([finding, evidenceContext]) => {
        setDetail(finding);
        setEvidence(evidenceContext);
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof Error ? cause.message : "Could not load finding");
      })
      .finally(() => setLoadingDetail(false));

    return () => controller.abort();
  }, [selectedID, contextMinutes]);

  async function transitionFinding(status: FindingStatus) {
    if (!detail) return;
    setActionBusy(true);
    setError("");
    try {
      const updated = await updateFindingStatus(
        detail.id,
        status,
        "phase4-analyst",
      );
      setDetail(updated);
      setFindings((current) =>
        current.map((finding) =>
          finding.id === updated.id ? updated : finding,
        ),
      );
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Finding could not be updated",
      );
    } finally {
      setActionBusy(false);
    }
  }

  async function startInvestigation() {
    if (!detail) return;
    setActionBusy(true);
    setError("");
    try {
      const created = await createInvestigation(
        {
          title: "Finding: " + detail.title,
          description:
            "Investigation created from " +
            detail.detection_id +
            " with preserved finding and evidence links.",
          priority:
            detail.severity === "critical"
              ? "critical"
              : detail.severity === "high"
                ? "high"
                : "medium",
          owner_id: "phase4-analyst",
          finding_ids: [detail.id],
          event_ids: detail.events.map((event) => event.id),
        },
        "phase4-analyst",
      );
      router.push("/investigations?id=" + encodeURIComponent(created.id));
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Investigation could not be created",
      );
    } finally {
      setActionBusy(false);
    }
  }

  function pivotToHunt(identityID?: string, assetID?: string) {
    if (!detail) return;
    const firstEvent = detail.events[0];
    const params = new URLSearchParams({
      from_finding: detail.id,
      detection: detail.detection_id,
      title: detail.title,
    });
    if (firstEvent?.category) params.set("category", firstEvent.category);
    const identity = identityID || firstEvent?.identity_id;
    const asset = assetID || firstEvent?.asset_id;
    if (identity) params.set("identity_id", identity);
    if (asset) params.set("asset_id", asset);
    router.push("/hunts?" + params.toString());
  }

  const linkedEvents = evidence?.linked_events ?? detail?.events ?? [];
  const contextEvents = evidence?.context_events ?? [];
  const reasons = detail ? explain(detail) : [];

  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto min-h-screen max-w-[1820px]">
        <section className="min-w-0 px-4 py-5 sm:px-6 lg:px-8 xl:px-10">
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
              <div className="mt-3 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.13em] text-rose-300/75">
                <ShieldAlert className="size-3.5" />
                FINDING ANALYSIS
              </div>
              <h1 className="mt-2 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.035em] text-white sm:text-[1.95rem]">
                Explain, validate, and escalate detections
              </h1>
              <p className="mt-1.5 max-w-3xl text-[0.78rem] leading-6 tracking-[0.025em] text-slate-500">
                Review deterministic findings with direct evidence, nearby
                context, workflow state, and pivots into hunting or investigation.
              </p>
            </div>

            <div className="flex items-center gap-2.5">
              <Button
                type="button"
                onClick={() => void loadFindings()}
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
              <Avatar className="size-9 border border-rose-400/25">
                <AvatarFallback className="bg-gradient-to-br from-rose-300 to-violet-500 text-[0.72rem] font-bold text-slate-950">
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
                  <div className="text-[0.75rem] font-semibold text-rose-200">
                    Finding workspace needs attention
                  </div>
                  <div className="mt-1 text-[0.65rem] text-rose-200/55">
                    {error}
                  </div>
                </div>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setError("")}
                className="cursor-pointer text-rose-200 hover:bg-rose-400/10"
              >
                Dismiss
              </Button>
            </div>
          ) : null}

          <section className="grid gap-4 xl:grid-cols-[330px_minmax(0,1fr)]">
            <motion.aside
              initial={{ opacity: 0, x: -14 }}
              animate={{ opacity: 1, x: 0 }}
              className="surface-card overflow-hidden rounded-[1.4rem] border border-slate-800/85"
            >
              <div className="p-4">
                <div className="relative">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-slate-600" />
                  <Input
                    value={search}
                    onChange={(event) => setSearch(event.target.value)}
                    placeholder="Search findings…"
                    className="h-9 border-slate-800 bg-slate-950/55 pl-9 text-[0.7rem] placeholder:text-slate-700"
                  />
                </div>
              </div>
              <Separator className="bg-slate-800/80" />
              <ScrollArea className="h-[790px]">
                <div className="space-y-2 p-3">
                  {filteredFindings.map((finding) => (
                    <button
                      key={finding.id}
                      type="button"
                      onClick={() => {
                        setLoadingDetail(true);
                        setSelectedID(finding.id);
                      }}
                      className={cn(
                        "w-full cursor-pointer rounded-xl border p-3.5 text-left transition",
                        finding.id === selectedID
                          ? "border-rose-400/30 bg-rose-400/[0.06]"
                          : "border-slate-800/70 bg-slate-950/25 hover:border-slate-700 hover:bg-slate-900/55",
                      )}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <Badge
                          variant="outline"
                          className={cn(
                            "rounded-md px-2 py-0.5 text-[0.56rem] font-bold tracking-[0.08em]",
                            severityClass(finding.severity),
                          )}
                        >
                          {finding.severity.toUpperCase()}
                        </Badge>
                        <span className="text-[0.57rem] font-semibold text-slate-700">
                          {finding.status}
                        </span>
                      </div>
                      <div className="mt-3 line-clamp-2 text-[0.73rem] font-semibold leading-5 text-slate-300">
                        {finding.title}
                      </div>
                      <div className="mt-3 flex items-center justify-between gap-2">
                        <span className="font-mono text-[0.57rem] text-slate-700">
                          {finding.detection_id}
                        </span>
                        <span className="text-[0.62rem] font-semibold text-slate-500">
                          {finding.confidence}%
                        </span>
                      </div>
                    </button>
                  ))}

                  {!loadingList && filteredFindings.length === 0 ? (
                    <div className="grid h-40 place-items-center text-center">
                      <div>
                        <Search className="mx-auto size-4 text-slate-700" />
                        <div className="mt-2 text-[0.68rem] text-slate-600">
                          No findings match this search.
                        </div>
                      </div>
                    </div>
                  ) : null}
                </div>
              </ScrollArea>
            </motion.aside>

            <div className="min-w-0">
              <AnimatePresence mode="wait">
                {detail && !loadingDetail ? (
                  <motion.div
                    key={detail.id}
                    initial={{ opacity: 0, y: 12 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -8 }}
                    className="grid gap-4"
                  >
                    <Card className="overflow-hidden rounded-[1.4rem] border border-slate-800/85 bg-gradient-to-br from-[#10151f] via-[#0c1119] to-[#15101f] py-0">
                      <CardHeader className="px-5 pb-5 pt-5 sm:px-6">
                        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_260px] xl:items-start">
                          <div>
                            <div className="flex flex-wrap items-center gap-2">
                              <Badge
                                variant="outline"
                                className={cn(
                                  "rounded-md px-2 py-0.5 text-[0.58rem] font-bold tracking-[0.08em]",
                                  severityClass(detail.severity),
                                )}
                              >
                                {detail.severity.toUpperCase()}
                              </Badge>
                              <Badge
                                variant="outline"
                                className="rounded-md border-slate-700 bg-slate-950/40 px-2 py-0.5 text-[0.58rem] font-semibold tracking-[0.08em] text-slate-400"
                              >
                                {detail.status.toUpperCase()}
                              </Badge>
                              <Badge
                                variant="outline"
                                className="rounded-md border-violet-400/20 bg-violet-400/[0.05] px-2 py-0.5 font-mono text-[0.58rem] text-violet-300"
                              >
                                {detail.detection_id}
                              </Badge>
                            </div>
                            <h2 className="mt-4 text-[1.3rem] font-semibold leading-7 tracking-[-0.02em] text-white">
                              {detail.title}
                            </h2>
                            <p className="mt-2 text-[0.68rem] leading-5 tracking-[0.02em] text-slate-600">
                              First observed {formatDate(detail.first_observed_at)}
                              {" · "}
                              Last observed {formatDate(detail.last_observed_at)}
                            </p>
                          </div>

                          <div className="rounded-2xl border border-cyan-400/15 bg-cyan-400/[0.035] p-4">
                            <div className="text-[2rem] font-bold leading-none tracking-[-0.04em] text-cyan-200">
                              {detail.confidence}%
                            </div>
                            <div className="mt-2 text-[0.59rem] font-semibold tracking-[0.1em] text-cyan-300/45">
                              DETECTION CONFIDENCE
                            </div>
                          </div>
                        </div>

                        <div className="mt-5 flex flex-wrap gap-2">
                          <Button
                            type="button"
                            onClick={() => pivotToHunt()}
                            variant="outline"
                            className="cursor-pointer border-sky-400/20 bg-sky-400/[0.05] text-[0.68rem] text-sky-300 hover:bg-sky-400/10"
                          >
                            <Crosshair className="size-3.5" />
                            Hunt related activity
                          </Button>
                          <Button
                            type="button"
                            onClick={() => void startInvestigation()}
                            disabled={actionBusy}
                            className="cursor-pointer bg-gradient-to-r from-rose-300 via-orange-300 to-amber-300 text-[0.68rem] font-semibold text-slate-950 hover:from-rose-200 hover:to-amber-200 disabled:cursor-not-allowed"
                          >
                            <FilePlus2 className="size-3.5" />
                            Start investigation
                          </Button>
                        </div>
                      </CardHeader>
                    </Card>

                    <div className="grid gap-4 2xl:grid-cols-[minmax(0,1.45fr)_minmax(330px,0.8fr)]">
                      <div className="grid gap-4">
                        {detail.detection_id.startsWith("DET-OT-") ? (
                          <Card className="overflow-hidden rounded-2xl border border-amber-300/20 bg-gradient-to-br from-amber-300/[0.06] via-[#0c1119] to-rose-400/[0.04] py-0">
                            <CardHeader className="px-5 pb-4 pt-5">
                              <div className="flex items-center gap-2">
                                <Cpu className="size-4 text-amber-200" />
                                <div>
                                  <h3 className="text-[0.9rem] font-semibold text-amber-100">
                                    OT operational-safety context
                                  </h3>
                                  <p className="mt-1 text-[0.62rem] leading-5 text-amber-100/45">
                                    Synthetic Northstar telemetry only. This view
                                    describes observed metadata and does not imply
                                    Sentinel can control industrial equipment.
                                  </p>
                                </div>
                              </div>
                            </CardHeader>
                            <Separator className="bg-amber-300/10" />
                            <CardContent className="grid gap-2 p-5 sm:grid-cols-2 xl:grid-cols-4">
                              {[
                                [detail.evidence.ot_device_type, "device"],
                                [detail.evidence.ot_protocol, "protocol"],
                                [detail.evidence.ot_operation, "operation"],
                                [detail.evidence.safety_impact, "safety impact"],
                              ]
                                .filter(
                                  (item): item is [string, string] =>
                                    typeof item[0] === "string" &&
                                    item[0].length > 0,
                                )
                                .map(([value, label]) => (
                                  <div
                                    key={label}
                                    className="rounded-xl border border-amber-300/10 bg-slate-950/30 p-3"
                                  >
                                    <div className="text-[0.55rem] font-semibold tracking-[0.09em] text-amber-200/40">
                                      {label.toUpperCase()}
                                    </div>
                                    <div className="mt-2 break-words text-[0.68rem] font-medium leading-5 text-amber-100/80">
                                      {value.replaceAll("_", " ")}
                                    </div>
                                  </div>
                                ))}
                            </CardContent>
                          </Card>
                        ) : null}

                        <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                          <CardHeader className="px-5 pb-4 pt-5">
                            <div className="flex items-center gap-2">
                              <Sparkles className="size-4 text-amber-300" />
                              <div>
                                <h3 className="text-[0.9rem] font-semibold text-slate-200">
                                  Why Sentinel flagged this
                                </h3>
                                <p className="mt-1 text-[0.62rem] leading-5 text-slate-600">
                                  Explainability derived from persisted finding
                                  evidence and linked telemetry.
                                </p>
                              </div>
                            </div>
                          </CardHeader>
                          <Separator className="bg-slate-800/80" />
                          <CardContent className="grid gap-2 p-5 sm:grid-cols-2">
                            {reasons.map((reason) => (
                              <div
                                key={reason}
                                className="flex items-start gap-2.5 rounded-xl border border-slate-800/75 bg-slate-950/35 p-3"
                              >
                                <CheckCircle2 className="mt-0.5 size-3.5 shrink-0 text-emerald-400" />
                                <span className="text-[0.67rem] leading-5 text-slate-400">
                                  {reason}
                                </span>
                              </div>
                            ))}
                          </CardContent>
                        </Card>

                        {detail.enrichments.length > 0 ? (
                          <Card className="overflow-hidden rounded-2xl border border-violet-400/20 bg-gradient-to-br from-violet-400/[0.06] via-[#0c1119] to-cyan-400/[0.04] py-0">
                            <CardHeader className="px-5 pb-4 pt-5">
                              <div className="flex items-center gap-2">
                                <Binary className="size-4 text-violet-300" />
                                <div>
                                  <h3 className="text-[0.9rem] font-semibold text-violet-100">
                                    Threat-intelligence enrichment
                                  </h3>
                                  <p className="mt-1 text-[0.62rem] leading-5 text-violet-100/45">
                                    IOC matches linked to this finding&apos;s evidence.
                                    Confidence remains source-aware rather than binary.
                                  </p>
                                </div>
                              </div>
                            </CardHeader>
                            <Separator className="bg-violet-400/10" />
                            <CardContent className="space-y-2 p-4">
                              {detail.enrichments.map((match) => (
                                <div
                                  key={match.id}
                                  className="grid gap-3 rounded-xl border border-violet-400/10 bg-slate-950/30 p-3 sm:grid-cols-[minmax(0,1fr)_120px] sm:items-center"
                                >
                                  <div className="min-w-0">
                                    <div className="flex flex-wrap items-center gap-2">
                                      <Badge
                                        variant="outline"
                                        className="border-violet-400/20 bg-violet-400/[0.05] text-[0.54rem] text-violet-300"
                                      >
                                        {match.indicator_type.toUpperCase()}
                                      </Badge>
                                      <span className="truncate font-mono text-[0.65rem] font-semibold text-slate-300">
                                        {match.observed_value}
                                      </span>
                                    </div>
                                    <div className="mt-2 text-[0.58rem] leading-5 text-slate-600">
                                      {match.source_name} · {match.event_field} ·{" "}
                                      {typeof match.context.classification === "string"
                                        ? match.context.classification
                                        : "unclassified"}
                                    </div>
                                  </div>
                                  <div className="sm:text-right">
                                    <div className="text-[0.86rem] font-semibold text-cyan-300">
                                      {match.effective_confidence}%
                                    </div>
                                    <div className="mt-1 text-[0.52rem] font-semibold tracking-[0.08em] text-slate-700">
                                      EFFECTIVE
                                    </div>
                                  </div>
                                </div>
                              ))}
                            </CardContent>
                          </Card>
                        ) : null}

                        <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                          <CardHeader className="px-5 pb-4 pt-5">
                            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                              <div>
                                <div className="flex items-center gap-2">
                                  <Activity className="size-4 text-sky-400" />
                                  <h3 className="text-[0.9rem] font-semibold text-slate-200">
                                    Evidence timeline
                                  </h3>
                                </div>
                                <p className="mt-1 text-[0.62rem] leading-5 text-slate-600">
                                  Direct evidence plus nearby asset/identity context.
                                </p>
                              </div>
                              <div className="flex items-center gap-1 rounded-xl border border-slate-800 bg-slate-950/40 p-1">
                                {[0, 5, 15, 30].map((minutes) => (
                                  <button
                                    key={minutes}
                                    type="button"
                                    onClick={() => {
                                      setLoadingDetail(true);
                                      setContextMinutes(minutes);
                                    }}
                                    className={cn(
                                      "cursor-pointer rounded-lg px-2.5 py-1.5 text-[0.6rem] font-semibold transition",
                                      contextMinutes === minutes
                                        ? "bg-sky-400/15 text-sky-300"
                                        : "text-slate-600 hover:bg-slate-900 hover:text-slate-300",
                                    )}
                                  >
                                    {minutes === 0 ? "Direct" : "±" + String(minutes) + "m"}
                                  </button>
                                ))}
                              </div>
                            </div>
                          </CardHeader>
                          <Separator className="bg-slate-800/80" />
                          <CardContent className="p-3">
                            <div className="space-y-2">
                              {[...linkedEvents, ...contextEvents]
                                .sort(
                                  (a, b) =>
                                    new Date(a.source_timestamp).getTime() -
                                    new Date(b.source_timestamp).getTime(),
                                )
                                .map((event) => {
                                  const direct = linkedEvents.some(
                                    (linked) => linked.id === event.id,
                                  );
                                  return (
                                    <div
                                      key={event.id}
                                      className={cn(
                                        "grid gap-3 rounded-xl border p-3 sm:grid-cols-[90px_84px_minmax(0,1fr)] sm:items-center",
                                        direct
                                          ? "border-rose-400/20 bg-rose-400/[0.035]"
                                          : "border-slate-800/70 bg-slate-950/25",
                                      )}
                                    >
                                      <span className="font-mono text-[0.58rem] text-slate-600">
                                        {formatDate(event.source_timestamp)}
                                      </span>
                                      <Badge
                                        variant="outline"
                                        className={cn(
                                          "w-fit rounded-md px-2 py-0.5 text-[0.54rem] font-bold tracking-[0.07em]",
                                          direct
                                            ? "border-rose-400/25 bg-rose-400/[0.06] text-rose-300"
                                            : "border-slate-700 bg-slate-800/30 text-slate-500",
                                        )}
                                      >
                                        {direct ? "EVIDENCE" : "CONTEXT"}
                                      </Badge>
                                      <div className="min-w-0">
                                        <div className="truncate text-[0.7rem] font-semibold text-slate-300">
                                          {event.category} / {event.action}
                                        </div>
                                        <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 font-mono text-[0.56rem] text-slate-700">
                                          {event.identity_id ? (
                                            <span>{event.identity_id}</span>
                                          ) : null}
                                          {event.asset_id ? (
                                            <span>{event.asset_id}</span>
                                          ) : null}
                                          <span>{event.id}</span>
                                        </div>
                                      </div>
                                    </div>
                                  );
                                })}
                            </div>
                          </CardContent>
                        </Card>
                      </div>

                      <div className="grid content-start gap-4">
                        <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                          <CardHeader className="px-5 pb-4 pt-5">
                            <div className="flex items-center gap-2">
                              <GitBranch className="size-4 text-violet-400" />
                              <h3 className="text-[0.86rem] font-semibold text-slate-200">
                                Finding workflow
                              </h3>
                            </div>
                          </CardHeader>
                          <Separator className="bg-slate-800/80" />
                          <CardContent className="space-y-2 p-4">
                            {allowedTransitions(detail.status).length ? (
                              allowedTransitions(detail.status).map((status) => (
                                <Button
                                  key={status}
                                  type="button"
                                  variant="outline"
                                  disabled={actionBusy}
                                  onClick={() => void transitionFinding(status)}
                                  className="w-full cursor-pointer justify-between border-slate-800 bg-slate-950/35 text-[0.66rem] text-slate-300 hover:border-violet-400/25 hover:bg-violet-400/[0.04] disabled:cursor-not-allowed"
                                >
                                  <span className="capitalize">
                                    Mark {status.replace("_", " ")}
                                  </span>
                                  <ChevronRight className="size-3.5" />
                                </Button>
                              ))
                            ) : (
                              <div className="rounded-xl border border-slate-800/75 bg-slate-950/35 p-3 text-[0.65rem] leading-5 text-slate-600">
                                {detail.status} is a terminal finding state.
                              </div>
                            )}
                          </CardContent>
                        </Card>

                        <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                          <CardHeader className="px-5 pb-4 pt-5">
                            <div className="flex items-center gap-2">
                              <Radar className="size-4 text-cyan-400" />
                              <h3 className="text-[0.86rem] font-semibold text-slate-200">
                                Investigation pivots
                              </h3>
                            </div>
                          </CardHeader>
                          <Separator className="bg-slate-800/80" />
                          <CardContent className="space-y-3 p-4">
                            {Array.from(
                              new Set(
                                linkedEvents
                                  .map((event) => event.identity_id)
                                  .filter((value): value is string => Boolean(value)),
                              ),
                            ).map((identity) => (
                              <button
                                key={identity}
                                type="button"
                                onClick={() => pivotToHunt(identity)}
                                className="flex w-full cursor-pointer items-center justify-between rounded-xl border border-violet-400/15 bg-violet-400/[0.04] p-3 text-left transition hover:border-violet-400/30 hover:bg-violet-400/[0.07]"
                              >
                                <span>
                                  <span className="flex items-center gap-2 text-[0.58rem] font-semibold tracking-[0.08em] text-violet-300/60">
                                    <Fingerprint className="size-3" />
                                    IDENTITY
                                  </span>
                                  <span className="mt-1 block font-mono text-[0.66rem] text-violet-200">
                                    {identity}
                                  </span>
                                </span>
                                <ChevronRight className="size-3.5 text-violet-400/50" />
                              </button>
                            ))}

                            {Array.from(
                              new Set(
                                linkedEvents
                                  .map((event) => event.asset_id)
                                  .filter((value): value is string => Boolean(value)),
                              ),
                            ).map((asset) => (
                              <button
                                key={asset}
                                type="button"
                                onClick={() => pivotToHunt(undefined, asset)}
                                className="flex w-full cursor-pointer items-center justify-between rounded-xl border border-sky-400/15 bg-sky-400/[0.04] p-3 text-left transition hover:border-sky-400/30 hover:bg-sky-400/[0.07]"
                              >
                                <span>
                                  <span className="flex items-center gap-2 text-[0.58rem] font-semibold tracking-[0.08em] text-sky-300/60">
                                    <Network className="size-3" />
                                    ASSET
                                  </span>
                                  <span className="mt-1 block font-mono text-[0.66rem] text-sky-200">
                                    {asset}
                                  </span>
                                </span>
                                <ChevronRight className="size-3.5 text-sky-400/50" />
                              </button>
                            ))}
                          </CardContent>
                        </Card>

                        <Card className="surface-card rounded-2xl border-slate-800/85 bg-transparent py-0">
                          <CardHeader className="px-5 pb-4 pt-5">
                            <div className="flex items-center gap-2">
                              <Clock3 className="size-4 text-slate-500" />
                              <h3 className="text-[0.86rem] font-semibold text-slate-200">
                                Audit
                              </h3>
                            </div>
                          </CardHeader>
                          <Separator className="bg-slate-800/80" />
                          <CardContent className="space-y-2 p-4">
                            {detail.audit
                              .slice(-5)
                              .reverse()
                              .map((audit) => (
                                <div
                                  key={audit.id}
                                  className="rounded-xl border border-slate-800/70 bg-slate-950/25 p-3"
                                >
                                  <div className="text-[0.65rem] font-medium text-slate-400">
                                    {audit.action}
                                  </div>
                                  <div className="mt-1 font-mono text-[0.55rem] text-slate-700">
                                    {formatDate(audit.occurred_at)}
                                  </div>
                                </div>
                              ))}
                          </CardContent>
                        </Card>
                      </div>
                    </div>
                  </motion.div>
                ) : (
                  <div className="grid min-h-[680px] place-items-center rounded-[1.4rem] border border-slate-800/85 bg-[#080d14]/70">
                    <div className="text-center">
                      <ShieldAlert className="mx-auto size-5 animate-pulse text-rose-400" />
                      <div className="mt-3 text-[0.75rem] font-medium text-slate-500">
                        {selectedID ? "Loading finding evidence…" : "Choose a finding"}
                      </div>
                    </div>
                  </div>
                )}
              </AnimatePresence>
            </div>
          </section>
        </section>
      </div>
    </main>
  );
}
