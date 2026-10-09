"""Bounded OTLP export; telemetry includes route/status, never bodies or headers."""
import os
from urllib.parse import urlparse
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor


def configure_tracing() -> TracerProvider:
    provider = TracerProvider(resource=Resource.create({"service.name": "sentinel-ml"}))
    endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "")
    if endpoint:
        url = urlparse(endpoint)
        if url.scheme not in ("http", "https") or not url.netloc or url.username or url.path or url.query or url.fragment:
            raise ValueError("OTLP endpoint must be an HTTP(S) origin")
        provider.add_span_processor(BatchSpanProcessor(
            OTLPSpanExporter(endpoint=endpoint + "/v1/traces", timeout=2),
            max_queue_size=256, max_export_batch_size=128, schedule_delay_millis=1000,
            export_timeout_millis=2000,
        ))
    return provider
