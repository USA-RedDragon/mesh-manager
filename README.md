# Mesh Manager

[![Release](https://github.com/USA-RedDragon/mesh-manager/actions/workflows/release.yaml/badge.svg)](https://github.com/USA-RedDragon/mesh-manager/actions/workflows/release.yaml) [![go.mod version](https://img.shields.io/github/go-mod/go-version/USA-RedDragon/mesh-manager.svg)](https://github.com/USA-RedDragon/mesh-manager) [![GoReportCard](https://goreportcard.com/badge/github.com/USA-RedDragon/mesh-manager)](https://goreportcard.com/report/github.com/USA-RedDragon/mesh-manager) [![License](https://badgen.net/github/license/USA-RedDragon/mesh-manager)](https://github.com/USA-RedDragon/mesh-manager/blob/main/LICENSE) [![Release](https://img.shields.io/github/release/USA-RedDragon/mesh-manager.svg)](https://github.com/USA-RedDragon/mesh-manager/releases/)

This project is a glorified configuration generator with an API and web interface.

## Configuration

Configuration is provided via environment variables, CLI flags, or a `config.yaml` file. Environment variables use `_` as a separator for nested fields (e.g., `POSTGRES_HOST`, `BABEL_ENABLED`).

### Required Settings

| Variable | Description |
|---|---|
| `SERVER_NAME` | Node hostname (e.g., `KI5VMF-TEST`) |
| `NODE_IP` | Node IP address (must be in `10.0.0.0/8`) |
| `PASSWORD_SALT` | Salt used for password hashing |
| `SESSION_SECRET` | Session secret |
| `WIREGUARD_STARTING_ADDRESS` | Starting IP for WireGuard interfaces |
| `POSTGRES_HOST` | PostgreSQL host |
| `POSTGRES_USER` | PostgreSQL user |
| `POSTGRES_PASSWORD` | PostgreSQL password |
| `POSTGRES_DATABASE` | PostgreSQL database |

### Optional Settings

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3333` | HTTP listen port |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `WIREGUARD_STARTING_PORT` | `5527` | Starting port for WireGuard |
| `TRUSTED_PROXIES` | | Trusted proxy IPs (comma-separated) |
| `CORS_HOSTS` | | CORS allowed hosts (comma-separated) |
| `INITIAL_ADMIN_USER_PASSWORD` | | Initial admin password |
| `HIBP_API_KEY` | | Have I Been Pwned API key |
| `LATITUDE` | | Server latitude |
| `LONGITUDE` | | Server longitude |
| `GRIDSQUARE` | | Server gridsquare identifier |

### Feature Flags

| Variable | Default | Description |
|---|---|---|
| `SUPERNODE` | `false` | Enable supernode mode |
| `WALKER` | `false` | Enable periodic mesh walking to update meshmap |
| `OLSR` | `true` | Enable OLSR routing |
| `BABEL_ENABLED` | `false` | Enable Babel routing (requires `BABEL_ROUTER_ID`) |
| `LQM_ENABLED` | `true` | Enable Link Quality Monitoring |
| `METRICS_ENABLED` | `false` | Enable Prometheus metrics |
| `RAVEN_ENABLED` | `false` | Enable [Raven](https://github.com/kn6plv/Raven) mesh chat (see below) |
| `PPROF_ENABLED` | `false` | Enable pprof debugging |

### Raven Mesh Chat

[Raven](https://github.com/kn6plv/Raven) is an optional mesh chat service that can be enabled with `RAVEN_ENABLED=true`. When enabled, Raven runs as an s6 service on port 4404 and is proxied through nginx at `/raven/`.

Raven uses the following environment variables for its platform configuration:

| Variable | Description |
|---|---|
| `SERVER_NAME` | Node hostname (shared with mesh-manager) |
| `NODE_IP` | Node IP address (shared with mesh-manager) |
| `LATITUDE` | Node latitude for location features |
| `LONGITUDE` | Node longitude for location features |
| `GRIDSQUARE` | Node gridsquare for location features |
| `SUPERNODE` | Indicates if this node is a supernode |

When Raven is disabled, the `/raven/` nginx locations return 503.

## Development

This project has a Golang backend and embeds a Vue frontend.
