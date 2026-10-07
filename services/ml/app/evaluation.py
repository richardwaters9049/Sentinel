from __future__ import annotations

from dataclasses import asdict, dataclass
from datetime import datetime, timedelta, timezone
import json

from .contracts import BehaviourScoreRequest
from .model import BehaviourModel


@dataclass(frozen=True)
class Evaluation:
    normal_count: int
    anomaly_count: int
    true_positive: int
    false_positive: int
    true_negative: int
    false_negative: int
    precision: float
    recall: float
    false_positive_rate: float


def evaluate_model() -> Evaluation:
    model = BehaviourModel()
    normal = _normal_validation_rows()
    anomalies = _anomaly_validation_rows()

    normal_scores = [model.score(row) for row in normal]
    anomaly_scores = [model.score(row) for row in anomalies]

    false_positive = sum(score.anomalous for score in normal_scores)
    true_negative = len(normal_scores) - false_positive
    true_positive = sum(score.anomalous for score in anomaly_scores)
    false_negative = len(anomaly_scores) - true_positive

    precision_denominator = true_positive + false_positive
    precision = (
        true_positive / precision_denominator
        if precision_denominator
        else 0.0
    )
    recall = true_positive / len(anomalies) if anomalies else 0.0
    false_positive_rate = (
        false_positive / len(normal) if normal else 0.0
    )

    return Evaluation(
        normal_count=len(normal),
        anomaly_count=len(anomalies),
        true_positive=true_positive,
        false_positive=false_positive,
        true_negative=true_negative,
        false_negative=false_negative,
        precision=round(precision, 4),
        recall=round(recall, 4),
        false_positive_rate=round(false_positive_rate, 4),
    )


def _normal_validation_rows() -> list[BehaviourScoreRequest]:
    start = datetime(2026, 2, 2, 8, 0, tzinfo=timezone.utc)
    ports = (80, 443, 5432, 8080)
    rows: list[BehaviourScoreRequest] = []

    for index in range(120):
        timestamp = start + timedelta(
            days=index % 5,
            hours=(index * 3) % 10,
            minutes=(index * 7) % 60,
        )
        rows.append(
            BehaviourScoreRequest(
                event_id=f"eval-normal-{index}",
                entity_id=f"eval-user-{index % 20}",
                timestamp=timestamp,
                category="network" if index % 4 else "authentication",
                action="connection" if index % 4 else "login",
                outcome="success",
                source_zone="corporate",
                destination_zone="corporate",
                destination_port=ports[index % len(ports)],
                actor_type="user",
            )
        )
    return rows


def _anomaly_validation_rows() -> list[BehaviourScoreRequest]:
    start = datetime(2026, 2, 7, 1, 0, tzinfo=timezone.utc)
    rows: list[BehaviourScoreRequest] = []

    for index in range(30):
        rows.append(
            BehaviourScoreRequest(
                event_id=f"eval-anomaly-{index}",
                entity_id=f"eval-service-{index % 5}",
                timestamp=start + timedelta(minutes=index * 9),
                category="authentication",
                action="login",
                outcome="failure",
                source_zone="corporate",
                destination_zone="ot",
                destination_port=502,
                actor_type="service_account",
            )
        )
    return rows


if __name__ == "__main__":
    print(json.dumps(asdict(evaluate_model()), sort_keys=True))
