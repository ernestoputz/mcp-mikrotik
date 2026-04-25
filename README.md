# mcp-mikrotik

MCP (Model Context Protocol) server that gives LLMs (Claude, etc.) direct access to MikroTik RouterOS 7.x routers via the REST API.

**57 tools** — diagnostic read-only operations and full write control: WiFi/CAPsMAN, firewall, QoS, DHCP, bridges, VLANs, IP addresses, DNS, WireGuard, and backups. All write operations require explicit user confirmation via `dry_run`.

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
    policy=read,write,api,rest-api,!local,!telnet,!ssh,!ftp,!reboot,!password,!policy,!test,!winbox,!sniff,!sensitive,!romon

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

Build the Docker image first:
```bash
make docker-build
```

Then add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mikrotik": {
      "command": "/usr/local/bin/docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "MCP_TRANSPORT=stdio",
        "-e", "LOG_LEVEL=info",
        "-e", "MIKROTIK_ROUTER_1_NAME=tplink",
        "-e", "MIKROTIK_ROUTER_1_HOST=10.1.0.1",
        "-e", "MIKROTIK_ROUTER_1_PORT=443",
        "-e", "MIKROTIK_ROUTER_1_SCHEME=https",
        "-e", "MIKROTIK_ROUTER_1_USER=mcp-api",
        "-e", "MIKROTIK_ROUTER_1_PASS=your-password",
        "-e", "MIKROTIK_ROUTER_1_TLS_SKIP_VERIFY=true",
        "-e", "MIKROTIK_ROUTER_2_NAME=escritorio",
        "-e", "MIKROTIK_ROUTER_2_HOST=10.1.0.X",
        "-e", "MIKROTIK_ROUTER_2_PORT=443",
        "-e", "MIKROTIK_ROUTER_2_SCHEME=https",
        "-e", "MIKROTIK_ROUTER_2_USER=mcp-api",
        "-e", "MIKROTIK_ROUTER_2_PASS=your-password",
        "-e", "MIKROTIK_ROUTER_2_TLS_SKIP_VERIFY=true",
        "-e", "MIKROTIK_ROUTER_3_NAME=suite",
        "-e", "MIKROTIK_ROUTER_3_HOST=10.1.0.3",
        "-e", "MIKROTIK_ROUTER_3_PORT=443",
        "-e", "MIKROTIK_ROUTER_3_SCHEME=https",
        "-e", "MIKROTIK_ROUTER_3_USER=mcp-api",
        "-e", "MIKROTIK_ROUTER_3_PASS=your-password",
        "-e", "MIKROTIK_ROUTER_3_TLS_SKIP_VERIFY=true",
        "-e", "AWS_ACCESS_KEY_ID=your-key",
        "-e", "AWS_SECRET_ACCESS_KEY=your-secret",
        "-e", "AWS_REGION=us-east-1",
        "-e", "AWS_S3_BUCKET=your-bucket",
        "-e", "AWS_S3_PREFIX=mikrotik-backups/",
        "mcp-mikrotik:local"
      ]
    }
  }
}
```

> Remove router blocks (ROUTER_2, ROUTER_3) or AWS vars if not in use. The server ignores routers with empty HOST.

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

All action tools default to `dry_run=true` — Claude always shows a preview and asks for confirmation before executing any change.

#### WiFi / CAPsMAN

| Tool | What it does |
|---|---|
| `set_wifi_configuration` | Change SSID, channel, width, TX power, passphrase, auth, FT on a CAPsMAN profile |
| `create_wifi_network` | Create a new CAPsMAN configuration profile (SSID + band + security) |
| `delete_wifi_network` | Delete a CAPsMAN profile and its provisioning rules |
| `add_wifi_security` | Create a WiFi security profile (passphrase, WPA2/3, FT, WPS) |
| `remove_wifi_security` | Delete a WiFi security profile |
| `add_wifi_datapath` | Create a CAPsMAN datapath profile (bridge, VLAN, client isolation) |
| `remove_wifi_datapath` | Delete a datapath profile |
| `add_wifi_interface` | Create a virtual (slave) WiFi interface on a physical radio |
| `remove_wifi_interface` | Delete a virtual WiFi interface |
| `add_wifi_provisioning` | Add a CAPsMAN provisioning rule (radio-mac → config) |
| `remove_wifi_provisioning` | Remove a provisioning rule by ID |
| `provision_caps` | Force re-provisioning of all managed APs |

#### Firewall

| Tool | What it does |
|---|---|
| `add_firewall_rule` | Add a rule to filter, NAT, or mangle |
| `remove_firewall_rule` | Delete a firewall rule by ID |

#### Bridges / VLANs

| Tool | What it does |
|---|---|
| `add_bridge` | Create a bridge interface |
| `remove_bridge` | Delete a bridge and all its ports |
| `add_bridge_port` | Add an interface to a bridge |
| `remove_bridge_port` | Remove an interface from a bridge |
| `add_vlan` | Create a VLAN sub-interface |
| `remove_vlan` | Delete a VLAN interface |

#### IP Addresses

| Tool | What it does |
|---|---|
| `add_ip_address` | Assign an IP address to an interface |
| `remove_ip_address` | Remove an IP address from an interface |

#### DHCP

| Tool | What it does |
|---|---|
| `add_ip_pool` | Create an IP address pool |
| `remove_ip_pool` | Delete an IP pool |
| `add_dhcp_server` | Create a DHCP server on an interface |
| `remove_dhcp_server` | Delete a DHCP server |
| `add_dhcp_network` | Add a DHCP network (gateway, DNS, NTP options) |
| `remove_dhcp_network` | Remove a DHCP network entry |
| `add_dhcp_lease` | Create a static DHCP lease (MAC → IP) |
| `remove_dhcp_lease` | Remove a static DHCP lease |

#### DNS

| Tool | What it does |
|---|---|
| `add_dns_entry` | Add a static DNS entry (hostname → IP) |
| `remove_dns_entry` | Remove a static DNS entry |

#### WireGuard

| Tool | What it does |
|---|---|
| `add_wireguard_peer` | Generate keys, assign IP, and create a peer config |
| `disable_wireguard_peer` | Revoke VPN access without deleting the peer |
| `enable_wireguard_peer` | Re-enable a disabled peer |
| `remove_wireguard_peer` | Permanently delete a WireGuard peer |

#### Other

| Tool | What it does |
|---|---|
| `set_queue_limit` | Adjust queue tree or simple queue bandwidth |
| `restart_interface` | Disable → 2s → enable an interface |
| `create_backup` | Create `.backup` on router and upload to S3 |

---

## Safety

- **`dry_run=true` by default** — every action tool previews what would happen. Claude only executes after explicit user confirmation.
- **HTTPS + LAN-restricted** — the API is only reachable from your local network.
- **Dedicated user with minimal permissions** — the `mcp-api` user cannot SSH, reboot the router, change passwords, or access sensitive data.
- **No Prometheus, no Grafana, no external dependencies** — pure stdlib Go.
