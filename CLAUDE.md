# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Commands

```bash
make build        # Compile binary to bin/mcp-server (CGO_ENABLED=0)
make run          # Run locally via stdio (requires .env)
make run-http     # Run locally via HTTP on :8080 (requires .env)
make test         # go test ./... -race -count=1
make lint         # golangci-lint run ./...
make tidy         # go mod tidy
make docker-build # Build Docker image
make docker-run   # Start via Docker Compose (requires .env)
make docker-stop  # Stop containers
```

Copy `.env.example` to `.env` and populate before running locally.

To run a single test:
```bash
go test ./internal/mcp/... -run TestFunctionName -v
```

## Architecture

This is a **Model Context Protocol (MCP) server** written in Go that bridges LLMs (Claude, etc.) to MikroTik RouterOS 7.x routers via the REST API.

**Transport layer** (`internal/transport/`): Two transports — stdio (newline-delimited JSON over stdin/stdout, used by Claude Desktop) and HTTP+SSE (for remote clients). Selected via `MCP_TRANSPORT` env var. Default is `stdio`.

**MCP protocol** (`internal/mcp/server.go`): `Server.Handle()` dispatches JSON-RPC 2.0 messages. Handles `initialize`, `tools/list`, and `tools/call`. Tool handlers are registered in `toolHandlers()`.

**Tool implementations** (all in `internal/mcp/`):
- `tools_diagnostic.go` — read-only: list_routers, get_system_info, get_interfaces, get_ip_addresses, get_routing_table, get_arp_table, get_dhcp_leases, get_logs, ping_from_router, traceroute_from_router
- `tools_wifi.go` — WiFi/CAPsMAN: get_wifi_clients, get_wifi_interfaces, get_wifi_configurations, get_capsman_status, set_wifi_configuration
- `tools_firewall.go` — firewall: get_firewall_rules, add_firewall_rule, remove_firewall_rule
- `tools_qos.go` — QoS + misc: get_queue_stats, get_dns_entries, set_queue_limit, restart_interface
- `tools_backup.go` — create_backup (creates .backup on router, downloads, uploads to S3)
- `tools_registry.go` — all Tool definitions with InputSchema (edit here to add/change tools)

**MikroTik client** (`internal/mikrotik/client.go`): Thin HTTP client wrapping the RouterOS 7 REST API (`/rest/…`). Supports GET, POST, PATCH, PUT, DELETE and file download. Uses HTTP Basic Auth.

**AWS S3** (`internal/aws/s3.go`): Minimal S3 PutObject implementation using AWS Signature Version 4, no external dependencies.

**Config** (`internal/mcp/config.go`): All config from env vars. Routers are discovered by scanning `MIKROTIK_ROUTER_1_*`, `MIKROTIK_ROUTER_2_*`, … up to N.

## Key Design Constraints

- **Zero external dependencies** — stdlib only (`net/http`, `encoding/json`, `log/slog`, `crypto/*`).
- **Stateless** — no persistent state; multiple instances can run safely.
- **Confirmation required for actions** — all write tools default to `dry_run=true`. The LLM must show the preview to the user and only call with `dry_run=false` after explicit confirmation.
- **RouterOS 7.x REST API** — path prefix `/rest/`. WiFi uses `/rest/interface/wifi/` (new unified WiFi package, not `caps-man` or `wifiwave2`).
- **Single controller for WiFi** — all CAPsMAN WiFi configuration goes through the primary router (CAPsMAN controller). CAP routers (escritorio, suite) are reached directly only for system-level operations.

## Adding a new tool

1. Add a `Tool` definition to `buildToolRegistry()` in `tools_registry.go`
2. Add the handler method to the appropriate `tools_*.go` file
3. Register it in `toolHandlers()` in `server.go`
4. If it's a write operation, include a `dry_run` parameter (default: true)

## RouterOS REST API notes

- WiFi interfaces: `GET /rest/interface/wifi` — returns both local (wifi1/wifi2) and CAP-managed (cap-wifi1..4)
- WiFi clients: `GET /rest/interface/wifi/registration-table`
- CAPsMAN configs: `GET /rest/interface/wifi/configuration`
- CAPsMAN controller: `GET /rest/interface/wifi/capsman`
- Provisioning rules: `GET /rest/interface/wifi/provisioning`
- Logs: `GET /rest/log` — returns all entries; filter client-side
- Ping: `POST /rest/ping` with `{"address": "…", "count": "N"}` — returns incremental summaries, use the last entry
- Traceroute: `POST /rest/tool/traceroute`
- Backup: `POST /rest/system/backup/save` → `GET /rest/file` → `GET /{filename}.backup`
