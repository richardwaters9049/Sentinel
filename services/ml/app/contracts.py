from datetime import datetime
from typing import Literal

from pydantic import BaseModel, Field


class BehaviourScoreRequest(BaseModel):
    event_id: str = Field(min_length=1, max_length=128)
    entity_id: str = Field(min_length=1, max_length=128)
    timestamp: datetime
    category: str = Field(min_length=1, max_length=64)
    action: str = Field(min_length=1, max_length=128)
    outcome: str = Field(default="", max_length=64)
    source_zone: str = Field(default="", max_length=64)
    destination_zone: str = Field(default="", max_length=64)
    destination_port: int = Field(default=0, ge=0, le=65535)
    actor_type: str = Field(default="", max_length=64)


class FeatureExplanation(BaseModel):
    feature: str
    observed: float
    baseline: float
    deviation: float
    message: str


class BehaviourScoreResponse(BaseModel):
    event_id: str
    entity_id: str
    model_version: str
    model_kind: str
    anomaly_score: int = Field(ge=0, le=100)
    severity: Literal["low", "medium", "high"]
    anomalous: bool
    threshold: int
    explanations: list[FeatureExplanation]
