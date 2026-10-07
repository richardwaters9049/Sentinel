from fastapi import FastAPI

from .contracts import BehaviourScoreRequest, BehaviourScoreResponse
from .model import BehaviourModel, MODEL_KIND, MODEL_VERSION


app = FastAPI(
    title="Sentinel Behavioural Analytics",
    version="0.1.0",
    docs_url="/docs",
    redoc_url=None,
)

model = BehaviourModel()


@app.get("/health")
def health() -> dict[str, str]:
    return {
        "status": "ok",
        "model_version": MODEL_VERSION,
        "model_kind": MODEL_KIND,
    }


@app.post("/v1/score", response_model=BehaviourScoreResponse)
def score(request: BehaviourScoreRequest) -> BehaviourScoreResponse:
    return model.score(request)
