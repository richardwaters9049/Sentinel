#!/usr/bin/env python3
"""Isolated synthetic deployment/load/outage/restore gate. Removes only its own volumes."""
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
import argparse
import base64
import json
from http.client import RemoteDisconnected
import os
from pathlib import Path
import secrets
import shutil
import statistics
import subprocess
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen
from lab_signing import signature_headers

ROOT = Path(__file__).resolve().parents[1]
BASE = 'http://127.0.0.1:13006/api/sentinel'


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument('--report', type=Path)
    args = parser.parse_args()
    metadata = json.loads(subprocess.check_output(['python3', 'scripts/provision-deployment.py'], cwd=ROOT))
    directory = Path(metadata['secrets_dir'])
    credentials = json.loads((directory / 'lab-tokens.json').read_text())
    project = 'sentinel-phase9-' + secrets.token_hex(6)
    environment = {**os.environ, 'SENTINEL_DEPLOY_SECRETS_DIR': str(directory),
                   'SENTINEL_BUILD_VERSION': subprocess.check_output(['git', 'rev-parse', '--short', 'HEAD'], cwd=ROOT, text=True).strip() + '-working'}
    command = ['docker', 'compose', '-p', project, '-f', 'infra/docker/compose.production.yaml']

    def compose(*parts: str, data: bytes | None = None) -> bytes:
        return subprocess.check_output(command + list(parts), cwd=ROOT, env=environment, input=data)

    def request(path: str, role: str = '', method: str = 'GET', payload: object = None,
                signed: dict[str, str] | None = None, body: bytes | None = None) -> tuple[int, bytes, dict[str, str]]:
        if body is None and payload is not None:
            body = json.dumps(payload).encode()
        headers = {'Content-Type': 'application/json'}
        if role:
            headers['Authorization'] = 'Bearer ' + credentials['tokens'][role]
        if role == 'collector' and method == 'POST':
            headers.update(signed if signed is not None else signature_headers(credentials['collector_private_seed'], method, path, body or b''))
        try:
            with urlopen(Request(BASE + path, data=body, method=method, headers=headers), timeout=15) as response:
                return response.status, response.read(), dict(response.headers)
        except HTTPError as error:
            return error.code, error.read(), dict(error.headers)

    def expect(code: int, path: str, **kwargs: object) -> bytes:
        actual, body, _ = request(path, **kwargs)
        if actual != code:
            raise AssertionError(f'{path}: expected {code}, got {actual}')
        return body

    def ready() -> None:
        for _ in range(90):
            try:
                if request('/ready')[0] == 200:
                    return
            except (URLError, TimeoutError, RemoteDisconnected, ConnectionResetError):
                pass
            time.sleep(1)
        raise AssertionError('deployment did not become ready')

    report: dict[str, object] = {'scope': 'isolated synthetic local deployment', 'utc': datetime.now(timezone.utc).isoformat(), 'events': 64, 'concurrency': 4}
    try:
        compose('config', '--quiet')
        compose('up', '-d', '--build')
        ready()
        expect(401, '/api/v1/events')
        expect(200, '/api/v1/behaviour/catalogue', role='administrator')
        prefix = 'evt_phase9_' + secrets.token_hex(8)
        def event(index: int) -> dict[str, object]:
            return {'event_id': prefix + '_' + str(index), 'timestamp': datetime.now(timezone.utc).isoformat(),
                    'source': {'type': 'network', 'collector': 'producer-claimed-name'},
                    'event': {'category': 'network', 'action': 'connection', 'outcome': 'success'},
                    'labels': {'environment': 'lab', 'scenario': 'phase9-deployment', 'sentinel.collector.subject': 'spoofed'}}
        def ingest(index: int) -> float:
            started = time.perf_counter()
            expect(202, '/api/v1/telemetry', role='collector', method='POST', payload=event(index))
            return (time.perf_counter() - started) * 1000
        started = time.perf_counter()
        with ThreadPoolExecutor(max_workers=4) as pool:
            latency = sorted(pool.map(ingest, range(64)))
        elapsed = time.perf_counter() - started
        report['ingestion'] = {'seconds': round(elapsed, 3), 'events_per_second': round(64 / elapsed, 2),
                               'p50_ms': round(statistics.median(latency), 2), 'p95_ms': round(latency[int(len(latency)*.95)-1], 2), 'p99_ms': round(latency[-1], 2)}
        for _ in range(60):
            rows = json.loads(expect(200, '/api/v1/events?limit=200', role='analyst'))['events']
            matching = [row for row in rows if row['event_id'].startswith(prefix)]
            if len(matching) == 64:
                break
            time.sleep(.25)
        else:
            raise AssertionError('accepted telemetry did not persist')
        assert all(row['labels']['sentinel.collector.subject'] == 'deployment-collector' for row in matching)
        replay_body = json.dumps(event(0)).encode()
        signed = signature_headers(credentials['collector_private_seed'], 'POST', '/api/v1/telemetry', replay_body)
        expect(401, '/api/v1/telemetry', role='collector', method='POST', signed=signed, body=replay_body+b' ')
        expect(202, '/api/v1/telemetry', role='collector', method='POST', signed=signed, body=replay_body)
        expect(409, '/api/v1/telemetry', role='collector', method='POST', signed=signed, body=replay_body)
        compose('restart', 'gateway')
        ready()
        expect(409, '/api/v1/telemetry', role='collector', method='POST', signed=signed, body=replay_body)
        report['replay_survives_restart'] = True
        with ThreadPoolExecutor(max_workers=24) as pool:
            statuses = list(pool.map(lambda _: request('/api/v1/session', role='analyst')[0], range(180)))
        assert 429 in statuses and all(code in (200,429) for code in statuses), 'rate limit did not engage safely'
        report['rate_limit'] = {'accepted': statuses.count(200), 'denied': statuses.count(429)}
        time.sleep(2)
        compose('stop', 'nats')
        expect(503, '/ready')
        compose('start', 'nats')
        ready()
        compose('stop', 'postgres')
        expect(503, '/api/v1/session', role='administrator')
        compose('start', 'postgres')
        ready()
        report['dependency_outage_and_recovery'] = True
        compose('stop', 'otel')
        expect(200, '/api/v1/session', role='administrator')
        time.sleep(3)
        metrics = expect(200, '/api/v1/platform/metrics', role='administrator').decode()
        failures = next(float(line.split()[1]) for line in metrics.splitlines() if line.startswith('sentinel_trace_failures_total '))
        assert failures > 0, 'trace outage was hidden'
        compose('start', 'otel')
        report['trace_outage_does_not_block_api'] = True
        dump = compose('exec','-T','postgres','pg_dump','-U','sentinel','-d','sentinel','-Fc')
        compose('exec','-T','postgres','createdb','-U','sentinel','sentinel_recovery')
        compose('exec','-T','postgres','pg_restore','-U','sentinel','-d','sentinel_recovery','--exit-on-error',data=dump)
        restored = compose('exec','-T','postgres','psql','-U','sentinel','-d','sentinel_recovery','-Atc','SELECT COUNT(*) FROM events').decode().strip()
        assert int(restored) == 64, 'backup restore lost or amplified events'
        report['backup_restore_events'] = int(restored)
        for _ in range(10):
            with urlopen('http://127.0.0.1:13090/api/v1/targets',timeout=5) as response:
                targets = json.load(response)['data']['activeTargets']
            if any(target['health']=='up' for target in targets):break
            time.sleep(2)
        else:raise AssertionError('authenticated Prometheus scrape failed')
        with urlopen('http://127.0.0.1:13000/api/health',timeout=5) as response:assert response.status==200
        grafana_password = (directory / 'grafana-password').read_text().strip()
        authorization = base64.b64encode(('admin:' + grafana_password).encode()).decode()
        health = Request('http://127.0.0.1:13000/api/datasources/uid/sentinel-prometheus/health', headers={'Authorization': 'Basic ' + authorization})
        with urlopen(health, timeout=10) as response:
            assert json.load(response)['status'] == 'OK', 'Grafana datasource query failed'
        expect(200,'/api/v1/security/audit?limit=200',role='administrator')
        time.sleep(3)
        with tempfile.TemporaryDirectory(prefix='sentinel-trace-check-') as temporary:
            trace_file=Path(temporary)/'traces.json'
            container=compose('ps','-q','otel').decode().strip()
            subprocess.run(['docker','cp',container+':/var/lib/otel/traces.json',str(trace_file)],env=environment,check=True,stdout=subprocess.DEVNULL)
            trace_data = trace_file.read_text()
            gateway_spans: set[tuple[str, str]] = set()
            ml_parents: set[tuple[str, str]] = set()
            for line in trace_data.splitlines():
                for resource in json.loads(line).get('resourceSpans', []):
                    attributes = resource.get('resource', {}).get('attributes', [])
                    service_name = next((item['value'].get('stringValue') for item in attributes if item['key'] == 'service.name'), '')
                    for scope in resource.get('scopeSpans', []):
                        for span in scope.get('spans', []):
                            trace_id = span.get('traceId', '').lower()
                            if service_name == 'sentinel-gateway':
                                gateway_spans.add((trace_id, span.get('spanId', '').lower()))
                            elif service_name == 'sentinel-ml':
                                ml_parents.add((trace_id, span.get('parentSpanId', '').lower()))
            assert gateway_spans & ml_parents, 'gateway and ML traces were not linked'
            assert not any(token in trace_data for token in credentials['tokens'].values())
        report['monitoring_and_otlp_delivery'] = True
        for service in ('gateway', 'console', 'ml', 'nats', 'postgres', 'otel', 'prometheus', 'grafana'):
            container=compose('ps','-q',service).decode().strip()
            raw=subprocess.check_output(['docker','inspect',container],env=environment)
            spec=json.loads(raw)[0]
            expected_user = '70:70' if service == 'postgres' else '10001:10001'
            assert spec['Config']['User'] == expected_user and spec['HostConfig']['ReadonlyRootfs']
            assert 'ALL' in spec['HostConfig']['CapDrop']
        report['runtime_security_controls'] = True
        logs=compose('logs','--no-color','gateway','console').decode()
        assert not any(token in logs for token in credentials['tokens'].values())
        assert credentials['collector_private_seed'] not in logs
        print(json.dumps(report,indent=2))
        if args.report:
            args.report.parent.mkdir(parents=True,exist_ok=True)
            args.report.write_text(json.dumps(report,indent=2)+'\n')
    except Exception:
        diagnostic = Path('/tmp/sentinel-phase9-failure.log')
        descriptor = os.open(diagnostic, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
        with os.fdopen(descriptor, 'wb') as output:
            output.write(compose('logs', '--no-color', 'gateway', 'console', 'ml', 'otel', 'postgres', 'grafana', 'prometheus'))
        raise
    finally:
        try:
            compose('down','--volumes','--remove-orphans')
        finally:
            shutil.rmtree(directory)


if __name__ == '__main__':
    main()
