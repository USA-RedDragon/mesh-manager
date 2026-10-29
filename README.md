# Mesh Manager

> [!WARNING]
> **This project is deprecated and archived. It is no longer maintained.**
>
> Run official, unmodified AREDN® firmware instead, in a virtual machine or on bare metal. AREDN® firmware has WireGuard tunnels, supernode mode, Babel routing and Prometheus metrics built in. For the mesh map, use the `meshmap` AREDN® package from [mesh-walker](https://github.com/USA-RedDragon/mesh-walker).
>
> To move an existing node, follow [docs/MIGRATION.md](docs/MIGRATION.md). The `migrate` subcommand prints a report for your node, with ready-to-run commands that keep your node name, mesh IP, tunnels and WireGuard keys. Remote tunnel partners do not need to change anything.
>
> The rest of this README stays as reference for existing installs.
>
> AREDN® is a registered trademark of Amateur Radio Emergency Data Network, Inc. This project is not affiliated with, endorsed by, or sanctioned by AREDN, Inc.

[![Release](https://github.com/USA-RedDragon/mesh-manager/actions/workflows/release.yaml/badge.svg)](https://github.com/USA-RedDragon/mesh-manager/actions/workflows/release.yaml) [![go.mod version](https://img.shields.io/github/go-mod/go-version/USA-RedDragon/mesh-manager.svg)](https://github.com/USA-RedDragon/mesh-manager) [![coverage](https://raw.githubusercontent.com/USA-RedDragon/mesh-manager/main/.github/badges/coverage.svg)](https://github.com/USA-RedDragon/mesh-manager/actions) [![License](https://badgen.net/github/license/USA-RedDragon/mesh-manager)](https://github.com/USA-RedDragon/mesh-manager/blob/main/LICENSE) [![Release](https://img.shields.io/github/release/USA-RedDragon/mesh-manager.svg)](https://github.com/USA-RedDragon/mesh-manager/releases/)

This project is a glorified configuration generator with an API and web interface.

## Configuration

Configuration is provided via environment variables, CLI flags, or a `config.yaml` file. Environment variables use `_` as a separator for nested fields (e.g., `POSTGRES_HOST`, `BABEL_ENABLED`).

<!-- configulator:begin -->

| Key                           | Type           | Default         | Environment                   | Flag                            | Description                                                           |
|-------------------------------|----------------|-----------------|-------------------------------|---------------------------------|-----------------------------------------------------------------------|
| `log-level`                   | string         | `info`          | `LOG_LEVEL`                   | `--log-level`                   | Logging level for the application. One of debug, info, warn, or error |
| `port`                        | integer        | `3333`          | `PORT`                        | `--port`                        | Port to listen on for HTTP requests                                   |
| `password-salt`               | string         |                 | `PASSWORD_SALT`               | `--password-salt`               | Salt used for password hashing (secret)                               |
| `pprof.enabled`               | boolean        | `false`         | `PPROF_ENABLED`               | `--pprof.enabled`               | Enable pprof debugging                                                |
| `postgres.host`               | string         |                 | `POSTGRES_HOST`               | `--postgres.host`               | PostgreSQL host                                                       |
| `postgres.port`               | integer        | `5432`          | `POSTGRES_PORT`               | `--postgres.port`               | PostgreSQL port                                                       |
| `postgres.user`               | string         |                 | `POSTGRES_USER`               | `--postgres.user`               | PostgreSQL user                                                       |
| `postgres.password`           | string         |                 | `POSTGRES_PASSWORD`           | `--postgres.password`           | PostgreSQL password (secret)                                          |
| `postgres.database`           | string         |                 | `POSTGRES_DATABASE`           | `--postgres.database`           | PostgreSQL database                                                   |
| `initial-admin-user-password` | string         |                 | `INITIAL_ADMIN_USER_PASSWORD` | `--initial-admin-user-password` | Initial password for the admin user (secret)                          |
| `babel.enabled`               | boolean        | `false`         | `BABEL_ENABLED`               | `--babel.enabled`               | Enable Babel routing                                                  |
| `babel.router-id`             | string         |                 | `BABEL_ROUTER_ID`             | `--babel.router-id`             | Babel router ID                                                       |
| `olsr`                        | boolean        | `true`          | `OLSR`                        | `--olsr`                        | Enable OLSR routing                                                   |
| `cors-hosts`                  | list of string |                 | `CORS_HOSTS`                  | `--cors-hosts`                  | CORS hosts for the API                                                |
| `trusted-proxies`             | list of string |                 | `TRUSTED_PROXIES`             | `--trusted-proxies`             | Trusted proxies for the API                                           |
| `hibp-api-key`                | string         |                 | `HIBP_API_KEY`                | `--hibp-api-key`                | Have I Been Pwned API key (secret)                                    |
| `server-name`                 | string         |                 | `SERVER_NAME`                 | `--server-name`                 | Server name                                                           |
| `supernode`                   | boolean        |                 | `SUPERNODE`                   | `--supernode`                   | Enable supernode mode                                                 |
| `node-ip`                     | string         |                 | `NODE_IP`                     | `--node-ip`                     | Node IP address                                                       |
| `latitude`                    | number         |                 | `LATITUDE`                    | `--latitude`                    | Server latitude                                                       |
| `longitude`                   | number         |                 | `LONGITUDE`                   | `--longitude`                   | Server longitude                                                      |
| `gridsquare`                  | string         |                 | `GRIDSQUARE`                  | `--gridsquare`                  | Server gridsquare                                                     |
| `metrics.enabled`             | boolean        |                 | `METRICS_ENABLED`             | `--metrics.enabled`             | Enable Prometheus metrics                                             |
| `metrics.node-exporter-host`  | string         | `node-exporter` | `METRICS_NODE_EXPORTER_HOST`  | `--metrics.node-exporter-host`  | Node exporter host for Prometheus metrics                             |
| `metrics.port`                | integer        | `9100`          | `METRICS_PORT`                | `--metrics.port`                | Port for Prometheus metrics                                           |
| `wireguard.starting-address`  | string         |                 | `WIREGUARD_STARTING_ADDRESS`  | `--wireguard.starting-address`  | Starting address for Wireguard                                        |
| `wireguard.starting-port`     | integer        | `5527`          | `WIREGUARD_STARTING_PORT`     | `--wireguard.starting-port`     | Starting port for Wireguard                                           |
| `session-secret`              | string         |                 | `SESSION_SECRET`              | `--session-secret`              | Session secret (secret)                                               |
| `lqm.enabled`                 | boolean        | `true`          | `LQM_ENABLED`                 | `--lqm.enabled`                 | Enable Link Quality Monitoring                                        |
| `walker`                      | boolean        | `false`         | `WALKER`                      | `--walker`                      | Enable periodic mesh walking to update meshmap                        |

<!-- configulator:end -->

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
