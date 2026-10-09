from fastapi.testclient import TestClient
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter
from app import main


def test_trace_propagation_omits_query_and_headers(monkeypatch):
    exporter = InMemorySpanExporter()
    provider = TracerProvider()
    provider.add_span_processor(SimpleSpanProcessor(exporter))
    monkeypatch.setattr(main, "tracer", provider.get_tracer("test"))
    client = TestClient(main.app)
    response = client.get("/health?secret=never-export", headers={
        "traceparent": "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01",
        "Authorization": "Bearer never-export",
    })
    assert response.status_code == 200
    spans = exporter.get_finished_spans()
    assert len(spans) == 1
    assert spans[0].name == "GET /health"
    assert spans[0].context.trace_id == int("0123456789abcdef0123456789abcdef", 16)
    assert spans[0].parent.span_id == int("0123456789abcdef", 16)
    assert "never-export" not in str(spans[0].attributes)
    provider.shutdown()
