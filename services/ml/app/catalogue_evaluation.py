"""Per-profile synthetic coverage; never presented as production detection accuracy."""
from __future__ import annotations

import json
from statistics import mean

from .catalogue import CATALOGUE_VERSION, DATASET_NAME, PROFILES, contrast_cases
from .model import BehaviourModel, MODEL_VERSION


def evaluate_catalogue(threshold: int = 65, model: BehaviourModel | None = None) -> dict[str, object]:
    if isinstance(threshold, bool) or not isinstance(threshold, int) or not 1 <= threshold <= 99:
        raise ValueError("threshold must be an integer between 1 and 99")
    model = model or BehaviourModel()
    profiles: list[dict[str, object]] = []
    for profile in PROFILES:
        reference = [model.score(row.model_copy(update={"threshold": threshold}))
                     for row in contrast_cases(profile.id, False)]
        changed = [model.score(row.model_copy(update={"threshold": threshold}))
                   for row in contrast_cases(profile.id, True)]
        reference_flagged = sum(score.anomalous for score in reference)
        changed_flagged = sum(score.anomalous for score in changed)
        profiles.append({
            "profile_id": profile.id,
            "reference_count": len(reference), "changed_count": len(changed),
            "reference_flagged": reference_flagged, "changed_flagged": changed_flagged,
            "reference_flag_rate": reference_flagged / len(reference),
            "changed_flag_rate": changed_flagged / len(changed),
            "reference_mean_score": round(mean(score.anomaly_score for score in reference), 3),
            "changed_mean_score": round(mean(score.anomaly_score for score in changed), 3),
            "changed_explanation_features": sorted({item.feature for score in changed for item in score.explanations}),
            "interpretation": "Synthetic context-change sensitivity only; labels do not denote compromise.",
        })
    return {"catalogue_version": CATALOGUE_VERSION, "model_version": MODEL_VERSION,
            "dataset_name": DATASET_NAME, "threshold": threshold, "profiles": profiles}


if __name__ == "__main__":
    import os
    print(json.dumps(evaluate_catalogue(int(os.getenv("SENTINEL_BEHAVIOUR_THRESHOLD", "65"))), sort_keys=True))
