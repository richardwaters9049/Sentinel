from datetime import datetime
from typing import Literal

from pydantic import BaseModel, Field


class BaselineContext(BaseModel):
    prior_events_60m: int = Field(default=0, ge=0)
    prior_events_24h: int = Field(default=0, ge=0)
    unique_destination_ips_24h: int = Field(default=0, ge=0)
    unique_destination_ports_24h: int = Field(default=0, ge=0)
    auth_failures_60m: int = Field(default=0, ge=0)
    ot_events_24h: int = Field(default=0, ge=0)
    event_rate_60m: float = Field(default=0.0, ge=0)
    destination_diversity_24h: float = Field(default=0.0, ge=0)
    auth_failure_rate_60m: float = Field(default=0.0, ge=0, le=1)
    ot_activity_rate_24h: float = Field(default=0.0, ge=0, le=1)


class BehaviourScoreRequest(BaseModel):
    event_id: str = Field(min_length=1, max_length=128)
    entity_id: str = Field(min_length=1, max_length=128)
    entity_type: Literal["identity", "asset", "collector"] = "collector"
    timestamp: datetime
    category: str = Field(min_length=1, max_length=64)
    action: str = Field(min_length=1, max_length=128)
    outcome: str = Field(default="", max_length=64)
    source_zone: str = Field(default="", max_length=64)
    destination_zone: str = Field(default="", max_length=64)
    destination_port: int = Field(default=0, ge=0, le=65535)
    actor_type: str = Field(default="", max_length=64)
    baseline: BaselineContext = Field(default_factory=BaselineContext)


class FeatureExplanation(BaseModel):
    feature: str
    observed: float
    baseline: float
    deviation: float
    message: str


class BehaviourScoreResponse(BaseModel):
    event_id: str
    entity_id: str
    entity_type: Literal["identity", "asset", "collector"]
    baseline: BaselineContext
    model_version: str
    model_kind: str
    anomaly_score: int = Field(ge=0, le=100)
    severity: Literal["low", "medium", "high"]
    anomalous: bool
    threshold: int
    explanations: list[FeatureExplanation]
