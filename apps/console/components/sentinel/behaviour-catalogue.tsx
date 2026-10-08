"use client";

import { useEffect, useState } from "react";

import { getBehaviourCatalogue } from "@/lib/sentinel/client";
import type { BehaviourCatalogue, BehaviourScore } from "@/lib/sentinel/types";

export default function BehaviourCataloguePanel({
  revision,
  selected,
  threshold,
}: {
  revision: number;
  selected?: BehaviourScore;
  threshold?: number;
}) {
  const [result, setResult] = useState<{
    revision: number;
    data?: BehaviourCatalogue;
    error?: string;
  }>();

  useEffect(() => {
    const controller = new AbortController();
    void getBehaviourCatalogue(controller.signal)
      .then((data) => {
        if (!controller.signal.aborted) setResult({ revision, data });
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setResult({ revision, error: cause instanceof Error ? cause.message : "Catalogue unavailable" });
        }
      });
    return () => controller.abort();
  }, [revision]);

  const loading = result?.revision !== revision;
  const catalogue = loading ? undefined : result?.data;
  const error = loading ? undefined : result?.error;
  const modelMatches = selected?.model_version === catalogue?.model_version;

  return (
    <section aria-label="Behavioural catalogue" aria-busy={loading} className="mb-4 rounded-2xl border border-slate-800 bg-slate-950/40 p-4">
      <h2 className="text-sm font-semibold text-slate-200">Behavioural catalogue</h2>
      <p className="mt-2 text-xs leading-relaxed text-slate-400">
        Analytical profiles describe feature coverage and review steps. They do not create findings or establish compromise.
      </p>
      {loading ? <p role="status" className="mt-3 text-sm text-slate-400">Loading catalogue…</p> : null}
      {error ? <p role="alert" className="mt-3 text-sm text-amber-200">Catalogue unavailable: {error}</p> : null}
      {catalogue ? <>
        <p className="mt-3 break-all text-xs text-slate-400">
          {catalogue.catalogue_version} · Model {catalogue.model_version} · Dataset {catalogue.dataset_name}
        </p>
        <p className="mt-2 text-xs leading-relaxed text-slate-400">
          Fixed synthetic validation at threshold {catalogue.validation.threshold}; 12 reference and 12 changed-context cases per profile.
          {threshold !== undefined && threshold !== catalogue.validation.threshold
            ? ` Your operational threshold is ${threshold}; these results do not evaluate that policy.` : ""}
          {selected && !modelMatches ? " The selected score uses a different model; profile links are withheld." : ""}
        </p>
        <div className="mt-3 grid items-start gap-3 lg:grid-cols-2">
          {catalogue.profiles.map((profile) => {
            const validation = catalogue.validation.profiles.find((entry) => entry.profile_id === profile.id);
            const sharedFeatures = modelMatches
              ? profile.features.filter((feature) => selected?.explanations.some((entry) => entry.feature === feature))
              : [];
            return (
              <details key={profile.id} className="min-w-0 rounded-xl border border-slate-800 p-3">
                <summary className="cursor-pointer text-sm font-semibold text-slate-200">
                  {profile.id} · {profile.title}
                </summary>
                <p className="mt-3 text-xs leading-relaxed text-slate-400">{profile.description}</p>
                {sharedFeatures.length ? <p className="mt-2 text-xs text-cyan-200">
                  Shares selected explanation features: {sharedFeatures.join(", ")}. This is a feature link, not a rule match.
                </p> : null}
                <p className="mt-2 break-words text-xs text-slate-400">Features: {profile.features.join(", ")}</p>
                <p className="mt-2 text-xs text-slate-400">
                  Context: {profile.context_window} · Suggested minimum prior events: {profile.minimum_prior_events} (review guidance, not a scoring gate)
                </p>
                {validation ? <div className="mt-3 rounded-lg border border-slate-800 bg-slate-900/30 p-3 text-xs leading-relaxed text-slate-300">
                  <p>Flagged reference cases: {validation.reference_flagged}/{validation.reference_count} · Changed-context cases: {validation.changed_flagged}/{validation.changed_count}</p>
                  <p className="mt-1">Mean score: {validation.reference_mean_score.toFixed(1)} reference → {validation.changed_mean_score.toFixed(1)} changed</p>
                  {validation.changed_flagged < validation.changed_count ? <p className="mt-2 text-amber-200">Limited threshold coverage: some changed-context fixtures are not flagged.</p> : null}
                  <p className="mt-2 text-slate-400">{validation.interpretation}</p>
                </div> : null}
                {([
                  ["Required telemetry", profile.required_telemetry],
                  ["Known limitations", profile.limitations],
                  ["Analyst review", profile.analyst_actions],
                ] as const).map(([title, items]) => <div key={title} className="mt-3">
                  <h3 className="text-xs font-semibold text-slate-300">{title}</h3>
                  <ul className="mt-2 list-disc space-y-1 pl-4 text-xs leading-relaxed text-slate-400">
                    {items.map((item) => <li key={item}>{item}</li>)}
                  </ul>
                </div>)}
              </details>
            );
          })}
        </div>
      </> : null}
    </section>
  );
}
