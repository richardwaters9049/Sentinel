from __future__ import annotations

import math
from dataclasses import dataclass
from datetime import timezone

import numpy as np

from .contracts import BehaviourScoreRequest


FEATURE_NAMES = (
    "hour_sin",
    "hour_cos",
    "weekend",
    "destination_port",
    "cross_zone",
    "auth_failure",
    "service_account",
    "ot_activity",
    "event_rate_60m",
    "destination_diversity_24h",
    "auth_failure_rate_60m",
    "ot_activity_rate_24h",
)


@dataclass(frozen=True)
class FeatureVector:
    values: np.ndarray

    def as_list(self) -> list[float]:
        return [float(value) for value in self.values]


def extract_features(request: BehaviourScoreRequest) -> FeatureVector:
    timestamp = request.timestamp.astimezone(timezone.utc)
    hour = timestamp.hour + timestamp.minute / 60.0
    radians = 2.0 * math.pi * hour / 24.0

    source_zone = request.source_zone.strip().lower()
    destination_zone = request.destination_zone.strip().lower()
    category = request.category.strip().lower()
    outcome = request.outcome.strip().lower()
    actor_type = request.actor_type.strip().lower()

    cross_zone = bool(
        source_zone
        and destination_zone
        and source_zone != destination_zone
    )

    values = np.array(
        [
            math.sin(radians),
            math.cos(radians),
            1.0 if timestamp.weekday() >= 5 else 0.0,
            min(request.destination_port / 65535.0, 1.0),
            1.0 if cross_zone else 0.0,
            1.0 if category == "authentication" and outcome == "failure" else 0.0,
            1.0 if actor_type == "service_account" else 0.0,
            1.0 if source_zone == "ot" or destination_zone == "ot" else 0.0,
            min(request.baseline.event_rate_60m / 60.0, 1.0),
            min(request.baseline.destination_diversity_24h, 1.0),
            request.baseline.auth_failure_rate_60m,
            request.baseline.ot_activity_rate_24h,
        ],
        dtype=float,
    )

    return FeatureVector(values=values)
