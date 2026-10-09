#!/usr/bin/env python3
"""Fail on reachable Go issues, runtime HIGH/CRITICAL vulnerabilities, secrets and policy failures."""
import argparse
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    args.artifacts.mkdir(parents=True, exist_ok=True)
    for directory in ('services/gateway', 'simulator'):
        subprocess.run(['govulncheck', './...'], cwd=ROOT / directory, check=True)
    subprocess.run(['trivy', 'fs', '--scanners', 'vuln,secret,misconfig', '--severity', 'HIGH,CRITICAL',
                    '--exit-code', '1', '--skip-dirs', 'apps/console/node_modules,apps/console/.next', '.'], cwd=ROOT, check=True)
    subprocess.run(['trivy', 'config', '--severity', 'HIGH,CRITICAL', '--exit-code', '1', 'infra/kubernetes'], cwd=ROOT, check=True)
    for service in ('gateway', 'console', 'ml', 'nats', 'postgres', 'otel', 'prometheus', 'grafana'):
        image = 'sentinel-' + service + ':phase9'
        subprocess.run(['trivy', 'image', '--image-src', 'docker', '--scanners', 'vuln,secret', '--severity', 'HIGH,CRITICAL', '--exit-code', '1', image], cwd=ROOT, check=True)
        subprocess.run(['trivy', 'image', '--image-src', 'docker', '--format', 'cyclonedx', '--output',
                        str(args.artifacts / (service + '.cdx.json')), image], cwd=ROOT, check=True)
    print('Security gate passed; runtime SBOMs written to ' + str(args.artifacts))


if __name__ == '__main__':
    main()
