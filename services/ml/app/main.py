from contextlib import asynccontextmanager
from collections.abc import AsyncIterator, Awaitable, Callable
from fastapi import FastAPI, Request
from starlette.responses import Response
from opentelemetry.propagate import extract
from opentelemetry.trace import SpanKind, Status, StatusCode
from .tracing import configure_tracing

from .catalogue import catalogue
from .catalogue_evaluation import evaluate_catalogue
from .contracts import BehaviourScoreRequest, BehaviourScoreResponse
from .model import BehaviourModel, MODEL_KIND, MODEL_VERSION


provider = configure_tracing()
tracer = provider.get_tracer("sentinel.ml")


@asynccontextmanager
async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
    yield
    provider.force_flush(timeout_millis=3000)
    provider.shutdown()


app = FastAPI(
    lifespan=lifespan,
    title="Sentinel Behavioural Analytics",
    version="0.1.0",
    docs_url="/docs",
    redoc_url=None,
)


@app.middleware("http")
async def trace_request(request: Request, call_next: Callable[[Request], Awaitable[Response]]) -> Response:
    with tracer.start_as_current_span("unmatched", context=extract(dict(request.headers)),
                                     kind=SpanKind.SERVER, record_exception=False,
                                     set_status_on_exception=False) as span:
        try:
            response = await call_next(request)
        except Exception:
            span.set_status(Status(StatusCode.ERROR, "request failed"))
            raise
        route = getattr(request.scope.get("route"), "path", "unmatched")
        span.update_name(request.method + " " + route)
        span.set_attribute("http.request.method", request.method)
        span.set_attribute("http.route", route)
        span.set_attribute("http.response.status_code", response.status_code)
        if response.status_code >= 500:
            span.set_status(Status(StatusCode.ERROR, "request failed"))
        return response


model = BehaviourModel()
catalogue_report = {**catalogue(), "validation": evaluate_catalogue(model=model)}


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


@app.get("/v1/catalogue")
def behavioural_catalogue() -> dict[str, object]:
    return catalogue_report
