"use client";

import {
  Activity,
  AlertTriangle,
  Bell,
  Braces,
  ChevronRight,
  CircleDot,
  Clock3,
  Crosshair,
  Database,
  FilePlus2,
  Fingerprint,
  FlaskConical,
  Link2,
  ListFilter,
  Network,
  Play,
  Plus,
  Radar,
  RefreshCcw,
  Save,
  ShieldCheck,
  Sparkles,
  Trash2,
  Waypoints,
} from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import SidebarDrawer from "@/components/sentinel/sidebar-drawer";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import {
  attachHuntRunToInvestigation,
  createHunt,
  createInvestigation,
  listHunts,
  listInvestigations,
  runHunt,
} from "@/lib/sentinel/client";
import type {
  EvidenceEvent,
  HuntDefinition,
  HuntQuery,
  HuntRunResult,
  InvestigationSummary,
} from "@/lib/sentinel/types";

type ClauseField =
  | "category"
  | "action"
  | "outcome"
  | "identity_id"
  | "asset_id"
  | "source_ip"
  | "destination_ip"
  | "source_zone"
  | "destination_zone"
  | "destination_port"
  | "last_minutes";

type Clause = {
  id: string;
  field: ClauseField;
  value: string;
};

const fieldMeta: Record<
  ClauseField,
  {
    label: string;
    placeholder: string;
    group: "event" | "identity" | "network" | "time";
    icon: typeof Activity;
  }
> = {
  category: {
    label: "Event category",
    placeholder: "authentication",
    group: "event",
    icon: Activity,
  },
  action: {
    label: "Action",
    placeholder: "login",
    group: "event",
    icon: Braces,
  },
  outcome: {
    label: "Outcome",
    placeholder: "success",
    group: "event",
    icon: ShieldCheck,
  },
  identity_id: {
    label: "Identity",
    placeholder: "svc-backup",
    group: "identity",
    icon: Fingerprint,
  },
  asset_id: {
    label: "Asset",
    placeholder: "engineering-ws-01",
    group: "identity",
    icon: Database,
  },
  source_ip: {
    label: "Source IP",
    placeholder: "10.10.20.15",
    group: "network",
    icon: Network,
  },
  destination_ip: {
    label: "Destination IP",
    placeholder: "10.30.0.40",
    group: "network",
    icon: Network,
  },
  source_zone: {
    label: "Source zone",
    placeholder: "corporate",
    group: "network",
    icon: Waypoints,
  },
  destination_zone: {
    label: "Destination zone",
    placeholder: "ot",
    group: "network",
    icon: Waypoints,
  },
  destination_port: {
    label: "Destination port",
    placeholder: "502",
    group: "network",
    icon: Network,
  },
  last_minutes: {
    label: "Time window",
    placeholder: "60",
    group: "time",
    icon: Clock3,
  },
};

const fieldOptions = Object.entries(fieldMeta) as [
  ClauseField,
  (typeof fieldMeta)[ClauseField],
][];

const groupStyles = {
  event: {
    badge: "border-sky-400/25 bg-sky-400/10 text-sky-300",
    line: "from-sky-400 to-cyan-300",
  },
  identity: {
    badge: "border-violet-400/25 bg-violet-400/10 text-violet-300",
    line: "from-violet-400 to-fuchsia-300",
  },
  network: {
    badge: "border-amber-300/25 bg-amber-300/10 text-amber-200",
    line: "from-amber-300 to-orange-300",
  },
  time: {
    badge: "border-emerald-400/25 bg-emerald-400/10 text-emerald-300",
    line: "from-emerald-400 to-cyan-300",
  },
};

function newClause(field: ClauseField = "category"): Clause {
  return {
    id: crypto.randomUUID(),
    field,
    value: "",
  };
}

function valuesFor(clauses: Clause[], field: ClauseField) {
  return clauses
    .filter((clause) => clause.field === field)
    .map((clause) => clause.value.trim())
    .filter(Boolean);
}

function buildQuery(clauses: Clause[]): HuntQuery {
  const query: HuntQuery = { limit: 100 };

  const categories = valuesFor(clauses, "category");
  const actions = valuesFor(clauses, "action");
  const outcomes = valuesFor(clauses, "outcome");
  const sourceZones = valuesFor(clauses, "source_zone");
  const destinationZones = valuesFor(clauses, "destination_zone");
  const ports = valuesFor(clauses, "destination_port")
    .map(Number)
    .filter((port) => Number.isInteger(port) && port > 0 && port <= 65535);

  if (categories.length === 1) query.category = categories[0];
  if (categories.length > 1) query.categories = categories;

  if (actions.length === 1) query.action = actions[0];
  if (actions.length > 1) query.actions = actions;

  if (outcomes.length === 1) query.outcome = outcomes[0];
  if (outcomes.length > 1) query.outcomes = outcomes;

  const identity = valuesFor(clauses, "identity_id")[0];
  if (identity) query.identity_id = identity;

  const asset = valuesFor(clauses, "asset_id")[0];
  if (asset) query.asset_id = asset;

  const sourceIP = valuesFor(clauses, "source_ip")[0];
  if (sourceIP) query.source_ip = sourceIP;

  const destinationIP = valuesFor(clauses, "destination_ip")[0];
  if (destinationIP) query.destination_ip = destinationIP;

  if (sourceZones.length) query.source_zones = sourceZones;
  if (destinationZones.length === 1) {
    query.destination_zone = destinationZones[0];
  } else if (destinationZones.length > 1) {
    query.destination_zones = destinationZones;
  }

  if (ports.length) query.destination_ports = ports;

  const lastMinutes = Number(valuesFor(clauses, "last_minutes")[0]);
  if (Number.isInteger(lastMinutes) && lastMinutes > 0) {
    query.last_minutes = lastMinutes;
  }

  return query;
}

function eventSummary(event: EvidenceEvent) {
  const actor = event.payload.actor?.name || event.identity_id || "unknown identity";
  const asset = event.payload.asset?.hostname || event.asset_id || "unknown asset";
  const destination = event.payload.network?.destination_ip;
  const port = event.payload.network?.destination_port;

  return {
    actor,
    asset,
    network:
      destination && port
        ? `${destination}:${port}`
        : destination || event.payload.network?.source_ip || "no network endpoint",
  };
}

function ClauseBlock({
  clause,
  index,
  onChange,
  onRemove,
}: {
  clause: Clause;
  index: number;
  onChange: (clause: Clause) => void;
  onRemove: () => void;
}) {
  const meta = fieldMeta[clause.field];
  const style = groupStyles[meta.group];
  const Icon = meta.icon;

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 12, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, x: -18, scale: 0.98 }}
      transition={{ duration: 0.24 }}
      className="relative"
    >
      {index > 0 ? (
        <div className="absolute -top-5 left-8 flex h-5 items-center">
          <span className="h-full w-px bg-slate-800" />
          <span className="ml-3 rounded-md border border-slate-800 bg-[#090e15] px-2 py-0.5 text-[0.55rem] font-bold tracking-[0.1em] text-slate-600">
            AND
          </span>
        </div>
      ) : null}

      <div className="overflow-hidden rounded-2xl border border-slate-800/85 bg-[#0b1119]/92">
        <div className={cn("h-[2px] bg-gradient-to-r", style.line)} />
        <div className="grid gap-3 p-3.5 lg:grid-cols-[170px_minmax(0,1fr)_44px] lg:items-center">
          <Select
            value={clause.field}
            onValueChange={(value) =>
              onChange({ ...clause, field: value as ClauseField, value: "" })
            }
          >
            <SelectTrigger className="h-10 w-full cursor-pointer border-slate-800 bg-slate-950/50 px-3 text-[0.7rem] text-slate-300">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="border-slate-800 bg-[#0c1119] text-slate-300">
              {fieldOptions.map(([field, option]) => (
                <SelectItem
                  key={field}
                  value={field}
                  className="cursor-pointer text-[0.7rem]"
                >
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="grid gap-2 sm:grid-cols-[92px_minmax(0,1fr)] sm:items-center">
            <Badge
              variant="outline"
              className={cn(
                "w-fit rounded-md px-2 py-1 text-[0.57rem] font-bold tracking-[0.08em]",
                style.badge,
              )}
            >
              <Icon className="mr-1.5 size-3" />
              {clause.field === "last_minutes" ? "WITHIN" : "IS"}
            </Badge>

            <Input
              value={clause.value}
              onChange={(event) =>
                onChange({ ...clause, value: event.target.value })
              }
              placeholder={meta.placeholder}
              inputMode={
                clause.field === "destination_port" ||
                clause.field === "last_minutes"
                  ? "numeric"
                  : undefined
              }
              className="h-10 border-slate-800 bg-slate-950/50 text-[0.73rem] tracking-[0.02em] text-slate-200 placeholder:text-slate-700"
            />
          </div>

          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={onRemove}
            aria-label="Remove condition"
            className="cursor-pointer text-slate-700 hover:bg-rose-400/10 hover:text-rose-300"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>
    </motion.div>
  );
}

function HuntResultCard({
  event,
  index,
  selected,
  onToggle,
}: {
  event: EvidenceEvent;
  index: number;
  selected: boolean;
  onToggle: () => void;
}) {
  const summary = eventSummary(event);

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.28, delay: Math.min(index * 0.035, 0.3) }}
      className={cn(
        "rounded-2xl border bg-slate-950/35 p-4 transition",
        selected
          ? "border-cyan-400/35 shadow-[0_0_34px_rgba(34,211,238,0.08)]"
          : "border-slate-800/80",
      )}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <Checkbox
            checked={selected}
            onCheckedChange={onToggle}
            aria-label={`Select event ${event.id}`}
            className="mt-0.5 cursor-pointer border-slate-700 data-checked:border-cyan-300 data-checked:bg-cyan-300 data-checked:text-slate-950"
          />
          <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <Badge
              variant="outline"
              className="border-sky-400/25 bg-sky-400/10 text-[0.58rem] font-bold tracking-[0.08em] text-sky-300"
            >
              {event.category.toUpperCase()}
            </Badge>
            <span className="text-[0.73rem] font-semibold text-slate-200">
              {event.action}
            </span>
          </div>
          <div className="mt-2 truncate font-mono text-[0.59rem] text-slate-700">
            {event.id}
          </div>
          </div>
        </div>
        <span className="font-mono text-[0.61rem] text-slate-600">
          {new Date(event.source_timestamp).toLocaleTimeString("en-GB")}
        </span>
      </div>

      <div className="mt-4 grid gap-2 sm:grid-cols-3">
        {[
          ["IDENTITY", summary.actor, Fingerprint],
          ["ASSET", summary.asset, Database],
          ["NETWORK", summary.network, Network],
        ].map(([label, value, Icon]) => {
          const IconComponent = Icon as typeof Fingerprint;
          return (
            <div
              key={String(label)}
              className="rounded-xl border border-slate-800/70 bg-[#0a0f17] p-3"
            >
              <div className="flex items-center gap-2 text-[0.55rem] font-semibold tracking-[0.09em] text-slate-700">
                <IconComponent className="size-3" />
                {String(label)}
              </div>
              <div className="mt-2 truncate text-[0.66rem] font-medium text-slate-400">
                {String(value)}
              </div>
            </div>
          );
        })}
      </div>
    </motion.div>
  );
}

export default function HuntCanvas() {
  const router = useRouter();
  const [clauses, setClauses] = useState<Clause[]>([
    { id: "initial-category", field: "category", value: "authentication" },
    { id: "initial-outcome", field: "outcome", value: "success" },
    { id: "initial-window", field: "last_minutes", value: "60" },
  ]);
  const [name, setName] = useState("Interactive service-account activity");
  const [hypothesis, setHypothesis] = useState(
    "A service identity may be authenticating interactively from a corporate system.",
  );
  const [description, setDescription] = useState(
    "Explore recent successful authentication activity and pivot suspicious results into an investigation.",
  );
  const [hunts, setHunts] = useState<HuntDefinition[]>([]);
  const [investigations, setInvestigations] = useState<InvestigationSummary[]>([]);
  const [result, setResult] = useState<HuntRunResult | null>(null);
  const [selectedEventIDs, setSelectedEventIDs] = useState<Set<string>>(
    new Set(),
  );
  const [running, setRunning] = useState(false);
  const [loadingHunts, setLoadingHunts] = useState(true);
  const [actionBusy, setActionBusy] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [attachDialogOpen, setAttachDialogOpen] = useState(false);
  const [caseTitle, setCaseTitle] = useState("Hunt evidence investigation");
  const [casePriority, setCasePriority] = useState<
    "low" | "medium" | "high" | "critical"
  >("high");
  const [caseOwner, setCaseOwner] = useState("phase4-analyst");
  const [targetInvestigationID, setTargetInvestigationID] = useState("");
  const [error, setError] = useState("");

  const query = useMemo(() => buildQuery(clauses), [clauses]);
  const queryPreview = useMemo(
    () => JSON.stringify(query, null, 2),
    [query],
  );

  useEffect(() => {
    const controller = new AbortController();

    void Promise.all([
      listHunts(controller.signal),
      listInvestigations(controller.signal),
    ])
      .then(([huntResponse, investigationResponse]) => {
        setHunts(huntResponse.hunts);
        setInvestigations(investigationResponse.investigations);
        setTargetInvestigationID(
          investigationResponse.investigations[0]?.id ?? "",
        );
      })
      .catch((cause) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(
          cause instanceof Error
            ? cause.message
            : "Could not load hunt workspace data",
        );
      })
      .finally(() => setLoadingHunts(false));

    return () => controller.abort();
  }, []);

  function updateClause(next: Clause) {
    setClauses((current) =>
      current.map((clause) => (clause.id === next.id ? next : clause)),
    );
  }

  async function saveAndRun() {
    if (!name.trim()) {
      setError("Give the hunt a name before running it.");
      return;
    }

    setRunning(true);
    setError("");

    try {
      const created = await createHunt(
        {
          name: name.trim(),
          description: description.trim(),
          hypothesis: hypothesis.trim(),
          query,
        },
        "phase4-analyst",
      );
      setHunts((current) => [created, ...current]);

      const run = await runHunt(created.id, "phase4-analyst");
      setResult(run);
      setSelectedEventIDs(new Set(run.events.map((event) => event.id)));
      setCaseTitle(`Hunt: ${created.name}`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Hunt failed");
    } finally {
      setRunning(false);
    }
  }

  function toggleEvent(eventID: string) {
    setSelectedEventIDs((current) => {
      const next = new Set(current);
      if (next.has(eventID)) {
        next.delete(eventID);
      } else {
        next.add(eventID);
      }
      return next;
    });
  }

  async function createInvestigationFromSelection() {
    if (!result || selectedEventIDs.size === 0) {
      setError("Select at least one evidence event before creating a case.");
      return;
    }

    setActionBusy(true);
    setError("");

    try {
      const created = await createInvestigation(
        {
          title: caseTitle.trim() || `Hunt investigation ${result.run_id}`,
          description:
            hypothesis.trim() ||
            "Investigation created from selected Sentinel hunt evidence.",
          priority: casePriority,
          owner_id: caseOwner.trim() || undefined,
          event_ids: [...selectedEventIDs],
        },
        "phase4-analyst",
      );

      setInvestigations((current) => [created, ...current]);
      setCreateDialogOpen(false);
      router.push(`/investigations?id=${encodeURIComponent(created.id)}`);
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

  async function attachCurrentRun() {
    if (!result || !targetInvestigationID) {
      setError("Choose an investigation before attaching this hunt run.");
      return;
    }

    setActionBusy(true);
    setError("");

    try {
      const updated = await attachHuntRunToInvestigation(
        targetInvestigationID,
        result.run_id,
        "phase4-analyst",
      );
      setAttachDialogOpen(false);
      router.push(`/investigations?id=${encodeURIComponent(updated.id)}`);
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Hunt run could not be attached",
      );
    } finally {
      setActionBusy(false);
    }
  }

  async function runSavedHunt(hunt: HuntDefinition) {
    setRunning(true);
    setError("");

    try {
      const run = await runHunt(hunt.id, "phase4-analyst");
      setResult(run);
      setSelectedEventIDs(new Set(run.events.map((event) => event.id)));
      setCaseTitle(`Hunt: ${hunt.name}`);
      setName(hunt.name);
      setDescription(hunt.description);
      setHypothesis(hunt.hypothesis);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Hunt failed");
    } finally {
      setRunning(false);
    }
  }

  return (
    <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
      <div className="mx-auto min-h-screen max-w-[1780px]">
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
              <div className="mt-3 flex items-center gap-2 text-[0.67rem] font-semibold tracking-[0.13em] text-sky-300/75">
                <Crosshair className="size-3.5" />
                THREAT HUNT CANVAS
              </div>
              <h1 className="mt-2 text-[1.65rem] font-bold leading-[1.18] tracking-[-0.035em] text-white sm:text-[1.95rem]">
                Build a hypothesis. Test the evidence.
              </h1>
              <p className="mt-1.5 max-w-3xl text-[0.78rem] leading-6 tracking-[0.025em] text-slate-500">
                Compose bounded hunt conditions visually, execute them against
                Sentinel telemetry, and inspect the evidence the backend
                actually returns.
              </p>
            </div>

            <div className="flex items-center gap-2.5">
              <Button
                variant="outline"
                size="icon"
                className="cursor-pointer border-slate-800 bg-slate-950/60 text-slate-400 hover:bg-slate-900"
              >
                <Bell className="size-4" />
              </Button>
              <Avatar className="size-9 border border-sky-400/25">
                <AvatarFallback className="bg-gradient-to-br from-sky-400 to-violet-500 text-[0.72rem] font-bold text-slate-950">
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
                    Hunt workspace needs attention
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

          <section className="grid gap-4 2xl:grid-cols-[minmax(0,1fr)_340px]">
            <div className="grid min-w-0 gap-4">
              <motion.div
                initial={{ opacity: 0, y: 14 }}
                animate={{ opacity: 1, y: 0 }}
                className="overflow-hidden rounded-[1.5rem] border border-slate-800/85 bg-[#080d14]/92"
              >
                <div className="grid gap-4 border-b border-slate-800/80 p-5 lg:grid-cols-[minmax(0,1.4fr)_minmax(280px,0.8fr)]">
                  <div>
                    <div className="flex items-center gap-2 text-[0.65rem] font-bold tracking-[0.12em] text-violet-300/75">
                      <FlaskConical className="size-3.5" />
                      HYPOTHESIS
                    </div>
                    <Textarea
                      value={hypothesis}
                      onChange={(event) => setHypothesis(event.target.value)}
                      className="mt-3 min-h-24 resize-none border-slate-800 bg-slate-950/45 text-[0.8rem] leading-6 tracking-[0.02em] text-slate-200 placeholder:text-slate-700"
                      placeholder="What suspicious behaviour are you trying to prove or disprove?"
                    />
                  </div>

                  <div className="grid gap-3">
                    <div>
                      <Label
                        htmlFor="hunt-name"
                        className="text-[0.62rem] font-semibold tracking-[0.09em] text-slate-600"
                      >
                        HUNT NAME
                      </Label>
                      <Input
                        id="hunt-name"
                        value={name}
                        onChange={(event) => setName(event.target.value)}
                        className="mt-2 border-slate-800 bg-slate-950/45 text-[0.72rem] text-slate-200"
                      />
                    </div>
                    <div>
                      <Label
                        htmlFor="hunt-description"
                        className="text-[0.62rem] font-semibold tracking-[0.09em] text-slate-600"
                      >
                        ANALYST NOTE
                      </Label>
                      <Input
                        id="hunt-description"
                        value={description}
                        onChange={(event) => setDescription(event.target.value)}
                        className="mt-2 border-slate-800 bg-slate-950/45 text-[0.72rem] text-slate-200"
                      />
                    </div>
                  </div>
                </div>

                <div className="relative p-5 sm:p-6">
                  <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(56,189,248,0.05),transparent_28rem),radial-gradient(circle_at_80%_30%,rgba(139,92,246,0.04),transparent_26rem)]" />

                  <div className="relative mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <div className="flex items-center gap-2 text-[0.72rem] font-semibold text-slate-300">
                        <ListFilter className="size-3.5 text-sky-400" />
                        Hunt conditions
                      </div>
                      <div className="mt-1 text-[0.64rem] leading-5 text-slate-600">
                        Conditions are compiled into Sentinel&apos;s typed query
                        model. No raw SQL is exposed.
                      </div>
                    </div>

                    <Button
                      type="button"
                      variant="outline"
                      onClick={() =>
                        setClauses((current) => [...current, newClause()])
                      }
                      className="cursor-pointer border-sky-400/20 bg-sky-400/[0.05] text-[0.68rem] text-sky-300 hover:bg-sky-400/10 hover:text-sky-200"
                    >
                      <Plus className="size-3.5" />
                      Add condition
                    </Button>
                  </div>

                  <div className="relative space-y-5">
                    <AnimatePresence initial={false}>
                      {clauses.map((clause, index) => (
                        <ClauseBlock
                          key={clause.id}
                          clause={clause}
                          index={index}
                          onChange={updateClause}
                          onRemove={() =>
                            setClauses((current) =>
                              current.filter((item) => item.id !== clause.id),
                            )
                          }
                        />
                      ))}
                    </AnimatePresence>

                    {clauses.length === 0 ? (
                      <button
                        type="button"
                        onClick={() => setClauses([newClause()])}
                        className="grid min-h-32 w-full cursor-pointer place-items-center rounded-2xl border border-dashed border-slate-800 bg-slate-950/20 text-center transition hover:border-sky-400/30 hover:bg-sky-400/[0.03]"
                      >
                        <span>
                          <Plus className="mx-auto size-5 text-slate-700" />
                          <span className="mt-2 block text-[0.72rem] font-medium text-slate-500">
                            Add your first hunt condition
                          </span>
                        </span>
                      </button>
                    ) : null}
                  </div>

                  <div className="relative mt-6 grid gap-3 border-t border-slate-800/75 pt-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
                    <div className="rounded-2xl border border-slate-800/75 bg-slate-950/35 p-4">
                      <div className="mb-3 flex items-center justify-between gap-3">
                        <div className="flex items-center gap-2 text-[0.62rem] font-semibold tracking-[0.09em] text-slate-600">
                          <Braces className="size-3.5" />
                          COMPILED QUERY
                        </div>
                        <Badge
                          variant="outline"
                          className="border-emerald-400/20 bg-emerald-400/[0.06] text-[0.56rem] font-semibold tracking-[0.08em] text-emerald-300"
                        >
                          BOUNDED
                        </Badge>
                      </div>
                      <pre className="overflow-x-auto font-mono text-[0.61rem] leading-5 text-slate-500">
                        {queryPreview}
                      </pre>
                    </div>

                    <Button
                      type="button"
                      onClick={() => void saveAndRun()}
                      disabled={running}
                      className="h-12 cursor-pointer bg-gradient-to-r from-sky-400 via-cyan-300 to-violet-400 px-6 text-[0.74rem] font-bold text-slate-950 shadow-[0_16px_50px_rgba(56,189,248,0.12)] hover:from-sky-300 hover:via-cyan-200 hover:to-violet-300 disabled:cursor-not-allowed"
                    >
                      {running ? (
                        <RefreshCcw className="size-4 animate-spin" />
                      ) : (
                        <Play className="size-4" />
                      )}
                      Save & run hunt
                    </Button>
                  </div>
                </div>
              </motion.div>

              <motion.section
                initial={{ opacity: 0, y: 14 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, amount: 0.1 }}
                className="overflow-hidden rounded-[1.5rem] border border-slate-800/85 bg-[#0a0f17]/92"
              >
                <div className="flex flex-col gap-3 border-b border-slate-800/80 p-5 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <div className="flex items-center gap-2 text-[0.75rem] font-semibold text-slate-200">
                      <Radar className="size-4 text-cyan-400" />
                      Hunt evidence
                    </div>
                    <div className="mt-1 text-[0.64rem] leading-5 text-slate-600">
                      Real events returned by the Go threat-hunting engine
                    </div>
                  </div>

                  {result ? (
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge
                        variant="outline"
                        className="border-cyan-400/20 bg-cyan-400/[0.06] text-[0.6rem] font-bold tracking-[0.08em] text-cyan-300"
                      >
                        RUN #{result.run_id}
                      </Badge>
                      <Badge
                        variant="outline"
                        className="border-slate-700 bg-slate-950/50 text-[0.6rem] text-slate-500"
                      >
                        {result.result_count} RESULTS
                      </Badge>
                      <Badge
                        variant="outline"
                        className="border-violet-400/20 bg-violet-400/[0.06] text-[0.6rem] text-violet-300"
                      >
                        {selectedEventIDs.size} SELECTED
                      </Badge>
                    </div>
                  ) : null}
                </div>

                {result && result.events.length > 0 ? (
                  <div className="flex flex-col gap-3 border-b border-slate-800/75 bg-slate-950/20 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex flex-wrap items-center gap-2">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          setSelectedEventIDs(
                            new Set(result.events.map((event) => event.id)),
                          )
                        }
                        className="cursor-pointer text-[0.66rem] text-slate-500 hover:bg-slate-900 hover:text-slate-200"
                      >
                        Select all
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedEventIDs(new Set())}
                        className="cursor-pointer text-[0.66rem] text-slate-500 hover:bg-slate-900 hover:text-slate-200"
                      >
                        Clear
                      </Button>
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => setAttachDialogOpen(true)}
                        className="cursor-pointer border-violet-400/20 bg-violet-400/[0.05] text-[0.67rem] text-violet-300 hover:bg-violet-400/10 hover:text-violet-200"
                      >
                        <Link2 className="size-3.5" />
                        Attach run
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        disabled={selectedEventIDs.size === 0}
                        onClick={() => setCreateDialogOpen(true)}
                        className="cursor-pointer bg-gradient-to-r from-cyan-300 to-sky-400 text-[0.67rem] font-semibold text-slate-950 hover:from-cyan-200 hover:to-sky-300 disabled:cursor-not-allowed"
                      >
                        <FilePlus2 className="size-3.5" />
                        Create investigation
                      </Button>
                    </div>
                  </div>
                ) : null}

                <div className="p-4">
                  {result ? (
                    result.events.length > 0 ? (
                      <div className="grid gap-3 xl:grid-cols-2">
                        {result.events.map((event, index) => (
                          <HuntResultCard
                            key={event.id}
                            event={event}
                            index={index}
                            selected={selectedEventIDs.has(event.id)}
                            onToggle={() => toggleEvent(event.id)}
                          />
                        ))}
                      </div>
                    ) : (
                      <div className="grid min-h-48 place-items-center rounded-2xl border border-dashed border-slate-800 text-center">
                        <div>
                          <CircleDot className="mx-auto size-5 text-slate-700" />
                          <div className="mt-3 text-[0.76rem] font-medium text-slate-400">
                            Hypothesis returned no matching events
                          </div>
                          <div className="mt-1 text-[0.65rem] text-slate-600">
                            Refine the conditions and run the hunt again.
                          </div>
                        </div>
                      </div>
                    )
                  ) : (
                    <div className="grid min-h-48 place-items-center rounded-2xl border border-dashed border-slate-800 text-center">
                      <div>
                        <Sparkles className="mx-auto size-5 text-sky-400/50" />
                        <div className="mt-3 text-[0.76rem] font-medium text-slate-400">
                          Evidence appears here after execution
                        </div>
                        <div className="mt-1 max-w-sm text-[0.65rem] leading-5 text-slate-600">
                          Build a hypothesis above, then save and run it against
                          the persisted telemetry set.
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </motion.section>
            </div>

            <motion.aside
              initial={{ opacity: 0, x: 14 }}
              animate={{ opacity: 1, x: 0 }}
              className="surface-card overflow-hidden rounded-[1.5rem] border border-slate-800/85"
            >
              <div className="flex items-center justify-between gap-3 p-5">
                <div>
                  <div className="flex items-center gap-2 text-[0.76rem] font-semibold text-slate-200">
                    <Save className="size-3.5 text-violet-400" />
                    Saved hunts
                  </div>
                  <div className="mt-1 text-[0.62rem] leading-5 text-slate-600">
                    Reproducible Phase 3 hunt definitions
                  </div>
                </div>
                <Badge
                  variant="outline"
                  className="border-slate-800 bg-slate-950/45 text-[0.58rem] text-slate-600"
                >
                  {hunts.length}
                </Badge>
              </div>
              <Separator className="bg-slate-800/80" />

              <ScrollArea className="h-[780px]">
                <div className="space-y-2 p-3">
                  {loadingHunts ? (
                    <div className="grid h-36 place-items-center text-[0.68rem] text-slate-600">
                      Loading hunts…
                    </div>
                  ) : null}

                  {hunts.map((hunt) => (
                    <button
                      key={hunt.id}
                      type="button"
                      onClick={() => void runSavedHunt(hunt)}
                      disabled={running}
                      className="group w-full cursor-pointer rounded-xl border border-slate-800/70 bg-slate-950/25 p-3.5 text-left transition hover:border-violet-400/25 hover:bg-violet-400/[0.04] disabled:cursor-not-allowed"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <Badge
                          variant="outline"
                          className="border-violet-400/20 bg-violet-400/[0.06] text-[0.55rem] font-bold tracking-[0.08em] text-violet-300"
                        >
                          V{hunt.version}
                        </Badge>
                        <Play className="size-3.5 text-slate-700 transition group-hover:text-violet-300" />
                      </div>
                      <div className="mt-3 line-clamp-2 text-[0.72rem] font-semibold leading-5 text-slate-300">
                        {hunt.name}
                      </div>
                      <div className="mt-2 line-clamp-3 text-[0.62rem] leading-5 text-slate-600">
                        {hunt.hypothesis || "No hypothesis recorded."}
                      </div>
                      <div className="mt-3 flex items-center justify-between gap-2 font-mono text-[0.55rem] text-slate-700">
                        <span>{hunt.id.slice(0, 16)}…</span>
                        <ChevronRight className="size-3" />
                      </div>
                    </button>
                  ))}
                </div>
              </ScrollArea>
            </motion.aside>
          </section>
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogContent className="border-slate-800 bg-[#0b1018] p-0 text-slate-100 sm:max-w-xl">
              <DialogHeader className="border-b border-slate-800/80 p-5">
                <div className="flex items-center gap-2 text-[0.62rem] font-bold tracking-[0.11em] text-cyan-300/75">
                  <FilePlus2 className="size-3.5" />
                  HUNT → INVESTIGATION
                </div>
                <DialogTitle className="mt-2 text-[1.05rem] font-semibold text-white">
                  Create investigation from selected evidence
                </DialogTitle>
                <DialogDescription className="text-[0.68rem] leading-5 text-slate-500">
                  Sentinel will preserve the selected event IDs in a new case and
                  open that case directly in the Investigation Graph.
                </DialogDescription>
              </DialogHeader>

              <div className="grid gap-4 p-5">
                <div>
                  <Label
                    htmlFor="case-title"
                    className="text-[0.61rem] font-semibold tracking-[0.09em] text-slate-600"
                  >
                    CASE TITLE
                  </Label>
                  <Input
                    id="case-title"
                    value={caseTitle}
                    onChange={(event) => setCaseTitle(event.target.value)}
                    className="mt-2 border-slate-800 bg-slate-950/45 text-[0.72rem] text-slate-200"
                  />
                </div>

                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <Label className="text-[0.61rem] font-semibold tracking-[0.09em] text-slate-600">
                      PRIORITY
                    </Label>
                    <Select
                      value={casePriority}
                      onValueChange={(value) =>
                        setCasePriority(
                          value as "low" | "medium" | "high" | "critical",
                        )
                      }
                    >
                      <SelectTrigger className="mt-2 h-10 w-full cursor-pointer border-slate-800 bg-slate-950/45 text-[0.7rem] text-slate-300">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="border-slate-800 bg-[#0c1119] text-slate-300">
                        {["low", "medium", "high", "critical"].map((priority) => (
                          <SelectItem
                            key={priority}
                            value={priority}
                            className="cursor-pointer text-[0.7rem]"
                          >
                            {priority.charAt(0).toUpperCase() + priority.slice(1)}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div>
                    <Label
                      htmlFor="case-owner"
                      className="text-[0.61rem] font-semibold tracking-[0.09em] text-slate-600"
                    >
                      OWNER
                    </Label>
                    <Input
                      id="case-owner"
                      value={caseOwner}
                      onChange={(event) => setCaseOwner(event.target.value)}
                      className="mt-2 border-slate-800 bg-slate-950/45 text-[0.72rem] text-slate-200"
                    />
                  </div>
                </div>

                <div className="rounded-2xl border border-cyan-400/15 bg-cyan-400/[0.04] p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="text-[0.61rem] font-semibold tracking-[0.09em] text-cyan-300/65">
                        SELECTED EVIDENCE
                      </div>
                      <div className="mt-1 text-[0.7rem] text-slate-400">
                        {selectedEventIDs.size} event
                        {selectedEventIDs.size === 1 ? "" : "s"} will be attached
                        to the new investigation.
                      </div>
                    </div>
                    <Badge
                      variant="outline"
                      className="border-cyan-400/25 bg-cyan-400/10 text-cyan-300"
                    >
                      {selectedEventIDs.size}
                    </Badge>
                  </div>
                </div>
              </div>

              <DialogFooter className="border-slate-800 bg-slate-950/35">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setCreateDialogOpen(false)}
                  className="cursor-pointer border-slate-800 bg-slate-950/45 text-slate-400 hover:bg-slate-900"
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  onClick={() => void createInvestigationFromSelection()}
                  disabled={actionBusy || selectedEventIDs.size === 0}
                  className="cursor-pointer bg-gradient-to-r from-cyan-300 to-sky-400 font-semibold text-slate-950 hover:from-cyan-200 hover:to-sky-300 disabled:cursor-not-allowed"
                >
                  {actionBusy ? (
                    <RefreshCcw className="size-4 animate-spin" />
                  ) : (
                    <FilePlus2 className="size-4" />
                  )}
                  Create & open graph
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Dialog open={attachDialogOpen} onOpenChange={setAttachDialogOpen}>
            <DialogContent className="border-slate-800 bg-[#0b1018] p-0 text-slate-100 sm:max-w-xl">
              <DialogHeader className="border-b border-slate-800/80 p-5">
                <div className="flex items-center gap-2 text-[0.62rem] font-bold tracking-[0.11em] text-violet-300/75">
                  <Link2 className="size-3.5" />
                  ATTACH DURABLE HUNT RUN
                </div>
                <DialogTitle className="mt-2 text-[1.05rem] font-semibold text-white">
                  Add this run to an existing investigation
                </DialogTitle>
                <DialogDescription className="text-[0.68rem] leading-5 text-slate-500">
                  This attaches the complete persisted hunt run, preserving the
                  exact result set and audit trail before opening the case graph.
                </DialogDescription>
              </DialogHeader>

              <div className="grid gap-4 p-5">
                <div>
                  <Label className="text-[0.61rem] font-semibold tracking-[0.09em] text-slate-600">
                    TARGET INVESTIGATION
                  </Label>
                  <Select
                    value={targetInvestigationID}
                    onValueChange={(value) =>
                      setTargetInvestigationID(value ?? "")
                    }
                  >
                    <SelectTrigger className="mt-2 h-11 w-full cursor-pointer border-slate-800 bg-slate-950/45 text-left text-[0.7rem] text-slate-300">
                      <SelectValue placeholder="Choose an investigation" />
                    </SelectTrigger>
                    <SelectContent className="border-slate-800 bg-[#0c1119] text-slate-300">
                      {investigations.map((investigation) => (
                        <SelectItem
                          key={investigation.id}
                          value={investigation.id}
                          className="cursor-pointer text-[0.7rem]"
                        >
                          {investigation.title}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {result ? (
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      [result.run_id, "run"],
                      [result.result_count, "events"],
                      [result.hunt_id.slice(0, 10), "hunt"],
                    ].map(([value, label]) => (
                      <div
                        key={String(label)}
                        className="rounded-xl border border-slate-800/75 bg-slate-950/35 p-3 text-center"
                      >
                        <div className="truncate text-[0.76rem] font-semibold text-slate-300">
                          {String(value)}
                        </div>
                        <div className="mt-1 text-[0.54rem] font-semibold tracking-[0.08em] text-slate-700">
                          {String(label).toUpperCase()}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : null}
              </div>

              <DialogFooter className="border-slate-800 bg-slate-950/35">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setAttachDialogOpen(false)}
                  className="cursor-pointer border-slate-800 bg-slate-950/45 text-slate-400 hover:bg-slate-900"
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  onClick={() => void attachCurrentRun()}
                  disabled={actionBusy || !targetInvestigationID}
                  className="cursor-pointer bg-gradient-to-r from-violet-400 to-sky-400 font-semibold text-slate-950 hover:from-violet-300 hover:to-sky-300 disabled:cursor-not-allowed"
                >
                  {actionBusy ? (
                    <RefreshCcw className="size-4 animate-spin" />
                  ) : (
                    <Link2 className="size-4" />
                  )}
                  Attach & open graph
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

        </section>
      </div>
    </main>
  );
}
