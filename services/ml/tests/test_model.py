from datetime import datetime, timezone

from app.contracts import BehaviourScoreRequest
from app.model import DEFAULT_ANOMALY_THRESHOLD, BehaviourModel


def request_for(**overrides: object) -> BehaviourScoreRequest:
    values: dict[str, object] = {
        "event_id": "evt-test",
        "entity_id": "asset-test",
        "timestamp": datetime(2026, 1, 7, 11, 30, tzinfo=timezone.utc),
        "category": "network",
        "action": "connection",
        "outcome": "success",
        "source_zone": "corporate",
        "destination_zone": "corporate",
        "destination_port": 443,
        "actor_type": "user",
    }
    values.update(overrides)
    return BehaviourScoreRequest.model_validate(values)


def test_normal_activity_scores_below_threshold() -> None:
    model = BehaviourModel()

    result = model.score(request_for())

    assert result.anomaly_score < DEFAULT_ANOMALY_THRESHOLD
    assert result.anomalous is False
    assert result.severity == "low"


def test_unusual_ot_weekend_activity_is_explainable() -> None:
    model = BehaviourModel()

    result = model.score(
        request_for(
            timestamp=datetime(2026, 1, 10, 2, 15, tzinfo=timezone.utc),
            category="authentication",
            outcome="failure",
            source_zone="corporate",
            destination_zone="ot",
            destination_port=502,
            actor_type="service_account",
        )
    )

    assert result.anomaly_score >= DEFAULT_ANOMALY_THRESHOLD
    assert result.anomalous is True
    assert result.severity in {"medium", "high"}
    features = {item.feature for item in result.explanations}
    assert {"weekend", "cross_zone", "ot_activity"} & features


def test_scoring_is_deterministic() -> None:
    left = BehaviourModel().score(request_for())
    right = BehaviourModel().score(request_for())

    assert left.anomaly_score == right.anomaly_score
    assert left.explanations == right.explanations


def test_entity_rolling_context_affects_score_and_explanation() -> None:
    model = BehaviourModel()

    calm = model.score(
        request_for(
            baseline={
                "prior_events_60m": 6,
                "prior_events_24h": 40,
                "unique_destination_ips_24h": 3,
                "unique_destination_ports_24h": 2,
                "auth_failures_60m": 0,
                "ot_events_24h": 0,
                "event_rate_60m": 6.0,
                "destination_diversity_24h": 0.075,
                "auth_failure_rate_60m": 0.0,
                "ot_activity_rate_24h": 0.0,
            }
        )
    )
    burst = model.score(
        request_for(
            baseline={
                "prior_events_60m": 60,
                "prior_events_24h": 80,
                "unique_destination_ips_24h": 60,
                "unique_destination_ports_24h": 20,
                "auth_failures_60m": 30,
                "ot_events_24h": 30,
                "event_rate_60m": 60.0,
                "destination_diversity_24h": 0.75,
                "auth_failure_rate_60m": 0.5,
                "ot_activity_rate_24h": 0.375,
            }
        )
    )

    assert burst.anomaly_score > calm.anomaly_score
    features = {item.feature for item in burst.explanations}
    assert features & {
        "event_rate_60m",
        "destination_diversity_24h",
        "auth_failure_rate_60m",
        "ot_activity_rate_24h",
    }
