# Local development

Sentinel is a local synthetic lab. Use the canonical console origin
`http://127.0.0.1:3000`; `localhost` is a different origin for sign-in checks.
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
