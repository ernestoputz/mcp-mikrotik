# mcp-mikrotik

MCP (Model Context Protocol) server that gives LLMs (Claude, etc.) direct access to MikroTik RouterOS 7.x routers via the REST API.

**23 tools** — diagnostic read-only operations and controlled write actions (WiFi, firewall, QoS, backups). All write operations require explicit user confirmation via `dry_run`.

---

## Setup — MikroTik Router

Run these commands in the **RouterOS terminal** (Winbox > New Terminal, or SSH).  
The script enables HTTPS restricted to your LAN and creates a dedicated API user.

```routeros
# ══════════════════════════════════════════════════════════════════
# Step 1: Generate a self-signed TLS certificate for HTTPS
# ══════════════════════════════════════════════════════════════════
/certificate add \
    name=mcp-ca \
    common-name="MCP CA" \
    key-size=2048 \
    days-valid=3650 \
    key-usage=key-cert-sign,crl-sign
/certificate sign mcp-ca ca-crl-host=10.1.0.1

/certificate add \
    name=mcp-server \
    common-name="mcp-mikrotik" \
    key-size=2048 \
    days-valid=3650 \
    key-usage=digital-signature,key-encipherment,tls-server \
    subject-alt-name="IP:10.1.0.1"
/certificate sign mcp-server ca=mcp-ca

# ══════════════════════════════════════════════════════════════════
# Step 2: Enable HTTPS (www-ssl) only on LAN — disable plain HTTP
# ══════════════════════════════════════════════════════════════════
/ip service set www-ssl \
    certificate=mcp-server \
    address=10.1.0.0/24 \
    port=443 \
    disabled=no

/ip service set www      disabled=yes
/ip service set api      disabled=yes
/ip service set api-ssl  disabled=yes
/ip service set telnet   disabled=yes
/ip service set ftp      disabled=yes

# Keep SSH restricted to LAN for admin access
/ip service set ssh address=10.1.0.0/24

# ══════════════════════════════════════════════════════════════════
# Step 3: Create a user group with minimal required permissions
# ══════════════════════════════════════════════════════════════════
# Allowed: read, write (config changes), rest-api (REST API access)
# Denied:  local console, telnet, ssh, ftp, reboot, password changes,
#          policy changes, sensitive data, packet sniffing, winbox, romon
/user group add name=mcp-group \
    policy=read,write,rest-api,!local,!telnet,!ssh,!ftp,!reboot,!password,!policy,!test,!winbox,!sniff,!sensitive,!romon

# ══════════════════════════════════════════════════════════════════
# Step 4: Create dedicated MCP user
# ══════════════════════════════════════════════════════════════════
# IMPORTANT: replace the password below with a strong random value
# Tip: openssl rand -base64 24
/user add \
    name=mcp-api \
    group=mcp-group \
    password="CHANGE_ME_STRONG_PASSWORD" \
    comment="MCP Server API user - do not use interactively"

# ══════════════════════════════════════════════════════════════════
# Step 5: Verify
# ══════════════════════════════════════════════════════════════════
:put "Setup complete. Test with:"
:put "curl -k -u mcp-api:CHANGE_ME https://10.1.0.1/rest/system/resource"
```

> **Repeat on each CAP router** (escritorio, suite) — same steps, adjusting the IP in `address=` and `subject-alt-name`.

---

## Configuration

Copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
```

Key variables:

| Variable | Description |
|---|---|
| `MIKROTIK_ROUTER_1_HOST` | Primary router IP (CAPsMAN controller) |
| `MIKROTIK_ROUTER_1_USER` | API user created above (`mcp-api`) |
| `MIKROTIK_ROUTER_1_PASS` | API user password |
| `MIKROTIK_ROUTER_1_SCHEME` | `https` (recommended) or `http` |
| `MIKROTIK_ROUTER_1_TLS_SKIP_VERIFY` | `true` for self-signed certs |
| `MIKROTIK_ROUTER_N_*` | Repeat for router 2, 3, … |
| `AWS_S3_BUCKET` | S3 bucket for backup uploads (optional) |
| `MCP_TRANSPORT` | `stdio` (Claude Desktop) or `http` (remote) |

---

## Running

### Claude Desktop (stdio — recommended)

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mikrotik": {
      "command": "/path/to/mcp-mikrotik/bin/mcp-server",
      "env": {
        "MIKROTIK_ROUTER_1_NAME": "tplink",
        "MIKROTIK_ROUTER_1_HOST": "10.1.0.1",
        "MIKROTIK_ROUTER_1_PORT": "443",
        "MIKROTIK_ROUTER_1_SCHEME": "https",
        "MIKROTIK_ROUTER_1_USER": "mcp-api",
        "MIKROTIK_ROUTER_1_PASS": "your-password",
        "MIKROTIK_ROUTER_1_TLS_SKIP_VERIFY": "true"
      }
    }
  }
}
```

Build the binary first:
```bash
make build
```

### Docker Compose (HTTP transport)

```bash
cp .env.example .env   # fill in values
make docker-run        # starts on :8080
make docker-stop       # stop
```

### Local HTTP (for development)

```bash
cp .env.example .env
make run-http          # starts on :8080
```

---

## Tools

### Diagnostic (read-only)

| Tool | What it does |
|---|---|
| `list_routers` | Connectivity check for all configured routers |
| `get_system_info` | Board, version, CPU, memory, uptime |
| `get_interfaces` | Interface list with traffic stats |
| `get_ip_addresses` | IP configuration per interface |
| `get_routing_table` | Active routes |
| `get_arp_table` | ARP entries (IP ↔ MAC) |
| `get_dhcp_leases` | DHCP server leases |
| `get_firewall_rules` | Filter / NAT / mangle rules |
| `get_logs` | System log entries (filterable by topic) |
| `get_wifi_clients` | Connected WiFi clients with signal/band/rates |
| `get_wifi_interfaces` | WiFi radio status (local + CAPsMAN CAPs) |
| `get_wifi_configurations` | CAPsMAN configuration profiles |
| `get_capsman_status` | CAPsMAN controller status and provisioning |
| `get_queue_stats` | QoS queue bandwidth and counters |
| `get_dns_entries` | Static DNS entries |
| `ping_from_router` | Ping from the router to any target |
| `traceroute_from_router` | Traceroute from the router |

### Actions (always `dry_run=true` first)

| Tool | What it does |
|---|---|
| `set_wifi_configuration` | Change SSID, channel, width, TX power on a CAPsMAN profile |
| `restart_interface` | Disable → 2s → enable an interface |
| `add_firewall_rule` | Add a rule to filter, NAT, or mangle |
| `remove_firewall_rule` | Delete a firewall rule by ID |
| `set_queue_limit` | Adjust queue tree or simple queue bandwidth |
| `create_backup` | Create `.backup` on router and upload to S3 |

> All action tools default to `dry_run=true`. Claude will always show you a preview and ask for confirmation before executing any change.

---

## Safety

- **`dry_run=true` by default** — every action tool previews what would happen. Claude only executes after explicit user confirmation.
- **HTTPS + LAN-restricted** — the API is only reachable from your local network.
- **Dedicated user with minimal permissions** — the `mcp-api` user cannot SSH, reboot the router, change passwords, or access sensitive data.
- **No Prometheus, no Grafana, no external dependencies** — pure stdlib Go.
