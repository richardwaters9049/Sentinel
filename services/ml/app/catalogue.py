"""Repository-owned analytical profiles and held-out synthetic contrast cases."""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone

from .contracts import BehaviourScoreRequest
from .model import MODEL_VERSION

CATALOGUE_VERSION = "sentinel-behaviour-catalogue-v1"
DATASET_NAME = "synthetic-behaviour-catalogue-v1"


@dataclass(frozen=True)
class Profile:
    id: str
    title: str
    description: str
    features: tuple[str, ...]
    required_telemetry: tuple[str, ...]
    context_window: str
    minimum_prior_events: int
    limitations: tuple[str, ...]
    analyst_actions: tuple[str, ...]


PROFILES = (
    Profile(
        "BA-001", "Unusual activity timing",
        "Compare UTC event timing with the weekday and daytime synthetic reference.",
        ("hour_sin", "hour_cos", "weekend"), ("timestamp with timezone",),
        "event", 0,
        ("Shift work, maintenance and local timezones can explain deviations.",
         "Producer clock errors and replay timestamps affect timing features.",
         "Isolated timing changes may remain below the default anomaly threshold."),
        ("Check working hours and maintenance schedules.", "Compare neighbouring events and source clock health."),
    ),
    Profile(
        "BA-002", "Authentication failure concentration",
        "Review authentication failures and their proportion of all entity events in 60 minutes.",
        ("auth_failure", "auth_failure_rate_60m"),
        ("event category and outcome", "stable entity identity", "persisted entity events"),
        "60 minutes", 5,
        ("The rate denominator is all entity events, not only authentication attempts.",
         "Retries and expired credentials can explain failures; sparse context inflates proportions."),
        ("Inspect authentication outcomes and identity history.", "Corroborate with deterministic authentication findings."),
    ),
    Profile(
        "BA-003", "Entity activity burst",
        "Review the count of prior entity events in a rolling 60-minute window.",
        ("event_rate_60m",), ("stable entity identity", "persisted entity events"),
        "60 minutes", 5,
        ("Batch jobs, collection changes and replay can raise event volume.",
         "The feature saturates at 60 prior events; it is not a throughput measurement.",
         "Isolated activity bursts may remain below the default anomaly threshold."),
        ("Check collection and replay activity.", "Review event categories and neighbouring entity activity."),
    ),
    Profile(
        "BA-004", "Destination diversity",
        "Review distinct destination IPs divided by prior entity events over 24 hours.",
        ("destination_diversity_24h",),
        ("destination IP", "stable entity identity", "persisted entity events"),
        "24 hours", 5,
        ("Missing destination IPs reduce observed diversity; small samples inflate it.",
         "Legitimate discovery, proxies and dynamic infrastructure can broaden destinations.",
         "Isolated diversity changes may remain below the default anomaly threshold."),
        ("Inspect destination evidence and asset roles.", "Check collection coverage before interpreting diversity."),
    ),
    Profile(
        "BA-005", "Simulated enterprise-to-OT context",
        "Review cross-zone service-account activity and the entity's recent OT activity proportion.",
        ("cross_zone", "service_account", "ot_activity", "ot_activity_rate_24h"),
        ("asset and destination zones", "actor type", "stable entity identity", "persisted entity events"),
        "event and 24 hours", 5,
        ("The reference is enterprise-dominant, not a learned OT process baseline.",
         "Routine authorised OT activity can score highly; scores do not establish controller manipulation."),
        ("Check simulated asset roles and authorised OT workflows.", "Corroborate with passive evidence; do not disrupt controllers."),
    ),
)


def catalogue() -> dict[str, object]:
    from dataclasses import asdict

    return {
        "catalogue_version": CATALOGUE_VERSION,
        "model_version": MODEL_VERSION,
        "dataset_name": DATASET_NAME,
        "profiles": [asdict(profile) for profile in PROFILES],
    }


def contrast_cases(profile_id: str, changed: bool) -> list[BehaviourScoreRequest]:
    """Twelve held-out cases per cohort; labels mean context change, not maliciousness."""
    if profile_id not in {profile.id for profile in PROFILES}:
        raise ValueError("unknown behavioural profile")
    rows: list[BehaviourScoreRequest] = []
    for index in range(12):
        baseline = {
            "prior_events_60m": 6,
            "prior_events_24h": 40,
            "unique_destination_ips_24h": 3,
            "unique_destination_ports_24h": 2,
            "event_rate_60m": 6.0,
            "destination_diversity_24h": 0.075,
        }
        values: dict[str, object] = {
            "event_id": f"catalogue-{profile_id}-{'changed' if changed else 'reference'}-{index}",
            "entity_id": f"catalogue-identity-{index}",
            "entity_type": "identity",
            "timestamp": datetime(2026, 3, 2 + index % 5, 9 + index % 8, index * 3, tzinfo=timezone.utc),
            "category": "network", "action": "connection", "outcome": "success",
            "source_zone": "corporate", "destination_zone": "corporate",
            "destination_port": 443 if index % 2 else 80,
            "actor_type": "user", "baseline": baseline,
        }
        if changed:
            if profile_id == "BA-001":
                values["timestamp"] = datetime(2026, 3, 7 + index % 2, 1 + index % 3, index * 3, tzinfo=timezone.utc)
            elif profile_id == "BA-002":
                values.update(category="authentication", action="login", outcome="failure")
                baseline.update(auth_failures_60m=4, auth_failure_rate_60m=4 / 6)
            elif profile_id == "BA-003":
                baseline.update(prior_events_60m=60, prior_events_24h=90, event_rate_60m=60.0,
                                destination_diversity_24h=3 / 90)
            elif profile_id == "BA-004":
                baseline.update(unique_destination_ips_24h=30, destination_diversity_24h=0.75)
            elif profile_id == "BA-005":
                values.update(destination_zone="ot", destination_port=502, actor_type="service_account")
                baseline.update(ot_events_24h=30, ot_activity_rate_24h=0.75)
        rows.append(BehaviourScoreRequest.model_validate(values))
    return rows
