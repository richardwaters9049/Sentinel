# Local development

Sentinel is a local synthetic lab. Use the exact console origin printed by the launcher
(or `http://127.0.0.1:3000` for manual startup); `localhost` is a different origin for sign-in checks.
The gateway and console bind to loopback by default. Compose publishes database,
NATS and ML ports, so run the lab on a controlled host/network and do not expose it
through public tunnels or proxies.

## Prerequisites

- Go 1.27.1, matching the Go modules.
- Bun 1.3.14, matching the console package manager.
- Docker with Compose, with its daemon running.
- Python 3 for provisioning and Phase Eight smoke scripts.

Python 3.12 and the pinned ML dependencies are supplied by the ML container.
Configuration placeholders live in [the root example](../.env.example) and
[the console example](../apps/console/.env.example). The Go process reads environment
variables; it does not automatically load the root example file.

## Start everything with one command

From `/Users/richy/Documents/Github/Sentinel`:

```bash
cd /Users/richy/Documents/Github/Sentinel
./sent-start
```

The root launcher opens Docker Desktop on macOS if needed, starts Compose dependencies,
installs locked frontend dependencies, builds the gateway, provisions private eight-hour
lab credentials, and starts the authenticated gateway and console. It waits for readiness
before opening the sign-in page and prints the credential-file path, never the credential.

Keep the terminal open. Ctrl-C stops only the gateway and console started by this launcher;
Docker dependencies and the PostgreSQL volume remain available. The launcher prefers
3000 and 8080, skips occupied ports, and prints the selected console and gateway URLs.
It passes the matching gateway address and console origin to both services for sign-in.
It never terminates existing listeners. Logs and the gateway binary are stored in a
private run directory outside the repository.

Existing Docker dependencies belonging to this checkout retain their ports and database.
New dependencies use Docker-assigned free ports bound to loopback. The launcher passes
their addresses to the gateway and uses them for readiness checks. Separate checkouts
get distinct Compose project names, container names and persistent database volumes.
Manual Compose commands still use the fixed ports in compose.yaml; to manage launcher
dependencies, use the printed Docker project name with Docker Desktop or
`docker compose -p PROJECT -f /printed/run/directory/compose.json ps`.

Ports are selected shortly before the app processes start. An unrelated process can
still claim an app port in that short interval; rerun the launcher if this occurs.

Optional flags: `--no-browser`, `--copy-token` (copy the analyst credential to the macOS
clipboard), `--check` (prerequisites and ports only), `--console-port PORT`, and
`--gateway-port PORT` (preferred starting ports, also skip busy ports). Each launch
provisions a new credential registry; use the newly printed credential file for that run. The manual startup steps below remain available.

## Install and start infrastructure

From `/Users/richy/Documents/Github/Sentinel`:

```bash
cd /Users/richy/Documents/Github/Sentinel
(cd apps/console && bun install --frozen-lockfile)
make dev-up
make provision-lab-auth
```

`dev-up` starts PostgreSQL, NATS and the Python ML service. Provisioning prints paths
and expiry only. It creates a new private directory outside the repository containing
`credentials.json` (hash manifest) and `lab-tokens.json` (raw lab credentials), both mode
0600. Credentials last eight hours. Do not paste raw credentials into source, logs,
command history or documentation.

## Start the gateway with authentication

From `/Users/richy/Documents/Github/Sentinel`, replace the example manifest path below
with the `manifest_path` printed by provisioning:

```bash
cd /Users/richy/Documents/Github/Sentinel
SENTINEL_AUTH_MODE=required \
SENTINEL_AUTH_CREDENTIALS_FILE="/absolute/path/from/provisioning/credentials.json" \
SENTINEL_CONSOLE_ORIGIN=http://127.0.0.1:3000 \
SENTINEL_BEHAVIOUR_ENABLED=true \
make gateway-run
```

The gateway uses local PostgreSQL at port 55432, NATS at 4222 and ML at 8090 unless
configured otherwise. Startup validates dependencies and applies ordered migrations.
Health is available at `http://127.0.0.1:8080/health`; dependency readiness is at `/ready`.
Protected API requests require an appropriate credential or console session.

## Start the console

In another terminal, from `/Users/richy/Documents/Github/Sentinel`:

```bash
cd /Users/richy/Documents/Github/Sentinel
SENTINEL_CONSOLE_AUTH_MODE=required \
SENTINEL_GATEWAY_URL=http://127.0.0.1:8080 \
make console-dev
```

Open `http://127.0.0.1:3000/sign-in` and use the analyst credential from the private
token file. Administrator credentials allow privileged controls; collector credentials
cannot sign in to the console. Sessions have an eight-hour maximum lifetime, are capped
by credential expiry, and expire after 30 minutes without API activity.

See [Phase Eight](architecture/phase-8-platform-hardening.md) for registry rotation,
CSRF, role boundaries, logout semantics and HTTPS cookie behaviour. Changing the
manifest requires restarting the gateway. Provisioning alone does not switch registries
or revoke existing credentials.

## Synthetic data and compatibility mode

The existing simulator and inherited smoke gates model safe events and fixtures.
`make simulator-run` runs the default `auth-burst` scenario against the default gateway;
its current CLI does not supply bearer credentials, so use it only with the explicit
local development compatibility workflow described in
[Phase Eight](architecture/phase-8-platform-hardening.md#development-compatibility).
The isolated Phase Eight smokes provision their own credentials and synthetic evidence.
Do not weaken an authenticated lab just to run a legacy example.

## Checks

From `/Users/richy/Documents/Github/Sentinel`:

```bash
cd /Users/richy/Documents/Github/Sentinel
python3 scripts/test_sent_start.py
make check
make console-lint
make console-build
(cd apps/console && bun test lib/sentinel/*.test.ts 'app/api/sentinel/[...path]/route.test.ts')
(cd apps/console && bunx tsc --noEmit)
make ml-test
```

`make check` covers Go formatting, vet, unit tests and race detection. Database
integration tests require `SENTINEL_TEST_DATABASE_URL`; see the phase documentation
for real PostgreSQL gates. Console and ML checks are separate from the root `make test`.
The [roadmap](ROADMAP.md#verification-gates) lists full regression and session gates.
The Phase Seven gate requires exclusive gateway/NATS workers and cycles dependencies;
stop your development gateway before running it. Its console test port must also be free.

## Stop services

Stop foreground gateway/console processes with Ctrl-C. From
`/Users/richy/Documents/Github/Sentinel`, `make dev-down` stops Compose services while
preserving the named PostgreSQL volume. Do not remove that volume casually: it contains
synthetic events, findings, investigations and audit evidence.
