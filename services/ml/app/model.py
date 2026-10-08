from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
import math
import random

import numpy as np
from sklearn.ensemble import IsolationForest

from .contracts import (
    BehaviourScoreRequest,
    BehaviourScoreResponse,
    FeatureExplanation,
)
from .features import FEATURE_NAMES, FeatureVector, extract_features


MODEL_VERSION = "sentinel-behaviour-iforest-v1"
MODEL_KIND = "IsolationForest"
RANDOM_SEED = 707
DEFAULT_ANOMALY_THRESHOLD = 65


@dataclass(frozen=True)
class BaselineStats:
    means: np.ndarray
    stddevs: np.ndarray


class BehaviourModel:
    def __init__(self) -> None:
        training_rows = _synthetic_baseline_rows()
        self._baseline = BaselineStats(
            means=training_rows.mean(axis=0),
            stddevs=np.maximum(training_rows.std(axis=0), 0.05),
        )
        self._model = IsolationForest(
            n_estimators=160,
            contamination=0.05,
            random_state=RANDOM_SEED,
            n_jobs=1,
        )
        self._model.fit(training_rows)

        baseline_raw = -self._model.score_samples(training_rows)
        self._score_floor = float(np.quantile(baseline_raw, 0.05))
        self._score_ceiling = float(np.quantile(baseline_raw, 0.995))
        if self._score_ceiling <= self._score_floor:
            self._score_ceiling = self._score_floor + 1.0

    def score(self, request: BehaviourScoreRequest) -> BehaviourScoreResponse:
        vector = extract_features(request)
        raw = float(-self._model.score_samples([vector.values])[0])
        anomaly_score = self._scale_score(raw)
        explanations = self._explain(vector)

        threshold = request.threshold
        if anomaly_score >= 85:
            severity = "high"
        elif anomaly_score >= threshold:
            severity = "medium"
        else:
            severity = "low"

        return BehaviourScoreResponse(
            event_id=request.event_id,
            entity_id=request.entity_id,
            entity_type=request.entity_type,
            baseline=request.baseline,
            model_version=MODEL_VERSION,
            model_kind=MODEL_KIND,
            anomaly_score=anomaly_score,
            severity=severity,
            anomalous=anomaly_score >= threshold,
            threshold=threshold,
            explanations=explanations,
        )

    def _scale_score(self, raw: float) -> int:
        normalized = (raw - self._score_floor) / (
            self._score_ceiling - self._score_floor
        )
        return int(round(max(0.0, min(1.0, normalized)) * 100.0))

    def _explain(self, vector: FeatureVector) -> list[FeatureExplanation]:
        z_scores = np.abs(
            (vector.values - self._baseline.means) / self._baseline.stddevs
        )
        ranked = np.argsort(z_scores)[::-1]

        explanations: list[FeatureExplanation] = []
        for index in ranked[:3]:
            deviation = float(z_scores[index])
            if deviation < 0.75:
                continue

            feature = FEATURE_NAMES[int(index)]
            observed = float(vector.values[index])
            baseline = float(self._baseline.means[index])
            explanations.append(
                FeatureExplanation(
                    feature=feature,
                    observed=round(observed, 4),
                    baseline=round(baseline, 4),
                    deviation=round(deviation, 2),
                    message=_feature_message(feature, observed, baseline),
                )
            )

        if not explanations:
            explanations.append(
                FeatureExplanation(
                    feature="baseline_similarity",
                    observed=0.0,
                    baseline=0.0,
                    deviation=0.0,
                    message="Observed behaviour is close to the synthetic Northstar baseline.",
                )
            )

        return explanations


def _synthetic_baseline_rows(size: int = 720) -> np.ndarray:
    rng = random.Random(RANDOM_SEED)
    rows: list[list[float]] = []
    start = datetime(2026, 1, 5, 7, 0, tzinfo=timezone.utc)

    common_ports = (22, 80, 443, 5432, 8080, 8443)
    for index in range(size):
        day_offset = index // 72
        weekday = day_offset % 5
        hour = 7.0 + rng.random() * 11.5
        minute = int((hour % 1.0) * 60)
        whole_hour = int(hour)
        timestamp = (
            start
            + timedelta(days=weekday)
        ).replace(hour=whole_hour, minute=minute)

        port = rng.choice(common_ports)
        source_zone = "corporate"
        destination_zone = "corporate"
        category = rng.choice(("network", "process", "authentication"))
        outcome = "success"
        actor_type = "user"

        if category == "authentication" and rng.random() < 0.03:
            outcome = "failure"
        if rng.random() < 0.04:
            actor_type = "service_account"
        if rng.random() < 0.025:
            destination_zone = "dmz"

        prior_events_60m = rng.randint(1, 14)
        prior_events_24h = rng.randint(max(prior_events_60m, 12), 90)
        unique_destination_ips = rng.randint(1, min(8, prior_events_24h))
        unique_destination_ports = rng.randint(1, min(5, prior_events_24h))
        auth_failures_60m = 1 if category == "authentication" and outcome == "failure" else 0
        ot_events_24h = 0

        request = BehaviourScoreRequest(
            event_id=f"baseline-{index}",
            entity_id=f"entity-{index % 24}",
            entity_type="identity" if index % 3 else "asset",
            timestamp=timestamp,
            category=category,
            action="baseline",
            outcome=outcome,
            source_zone=source_zone,
            destination_zone=destination_zone,
            destination_port=port,
            actor_type=actor_type,
            baseline={
                "prior_events_60m": prior_events_60m,
                "prior_events_24h": prior_events_24h,
                "unique_destination_ips_24h": unique_destination_ips,
                "unique_destination_ports_24h": unique_destination_ports,
                "auth_failures_60m": auth_failures_60m,
                "ot_events_24h": ot_events_24h,
                "event_rate_60m": float(prior_events_60m),
                "destination_diversity_24h": unique_destination_ips / prior_events_24h,
                "auth_failure_rate_60m": auth_failures_60m / prior_events_60m,
                "ot_activity_rate_24h": 0.0,
            },
        )
        rows.append(extract_features(request).as_list())

    return np.asarray(rows, dtype=float)


def _feature_message(feature: str, observed: float, baseline: float) -> str:
    messages = {
        "weekend": "Weekend activity differs from the weekday-dominant baseline.",
        "destination_port": "Destination-port behaviour differs from the common baseline range.",
        "cross_zone": "Cross-zone activity is uncommon in the learned baseline.",
        "auth_failure": "Authentication failure behaviour is uncommon in the learned baseline.",
        "service_account": "Service-account activity is less common than normal user activity.",
        "ot_activity": "OT-zone activity is uncommon relative to the enterprise baseline.",
        "event_rate_60m": "The entity's recent event rate is unusual relative to the synthetic baseline.",
        "destination_diversity_24h": "The entity is contacting an unusual variety of destinations.",
        "auth_failure_rate_60m": "The entity's recent authentication-failure rate is elevated.",
        "ot_activity_rate_24h": "The entity's recent OT activity rate differs from the normal baseline.",
        "hour_sin": "Event timing differs from the normal daily activity pattern.",
        "hour_cos": "Event timing differs from the normal daily activity pattern.",
    }
    return messages.get(
        feature,
        f"{feature} differs from its baseline value ({baseline:.2f}).",
    )
