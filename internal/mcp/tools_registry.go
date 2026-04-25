package mcp

// buildToolRegistry returns the full map of Tool definitions exposed via MCP.
func (s *Server) buildToolRegistry() map[string]Tool {
	routerNames := s.routerNames()

	routerProp := Property{
		Type:        "string",
		Description: "Router name. Available: " + joinNames(routerNames) + ". Defaults to the first router.",
		Enum:        routerNames,
	}

	tools := []Tool{
		// ── Diagnostic ──────────────────────────────────────────────────────
		{
			Name:        "list_routers",
			Description: "Lists all configured MikroTik routers and checks their connectivity. Returns name, host, and reachability status.",
			InputSchema: InputSchema{Type: "object", Properties: map[string]Property{}},
		},
		{
			Name:        "get_system_info",
			Description: "Returns system resource information for a router: board model, RouterOS version, CPU load, memory usage, disk space, and uptime.",
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{},
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name:        "get_interfaces",
			Description: "Lists network interfaces on a router with their status, traffic counters (rx/tx bytes and packets), MAC address, and MTU.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
					"type": {
						Type:        "string",
						Description: "Filter by interface type: ether, wifi, bridge, vlan, veth, wg (WireGuard), etc. Omit for all.",
					},
				},
			},
		},
		{
			Name:        "get_ip_addresses",
			Description: "Returns all IP addresses configured on the router, showing interface, address/prefix, and whether the address is dynamic.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name:        "get_routing_table",
			Description: "Returns the active IP routing table: destination, gateway, interface, distance, and route type (static, dynamic, OSPF, BGP, etc.).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name:        "get_arp_table",
			Description: "Returns the ARP table: IP addresses mapped to MAC addresses with their interface and status.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name:        "get_dhcp_leases",
			Description: "Returns DHCP server leases. By default returns only active (bound) leases. Shows IP, MAC, hostname, and expiry.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
					"all": {
						Type:        "boolean",
						Description: "Set true to include expired/waiting leases in addition to active ones. Default: false.",
						Default:     false,
					},
				},
			},
		},
		{
			Name:        "get_logs",
			Description: "Returns recent system log entries from the router. Optionally filter by topic (e.g. 'wireless', 'dhcp', 'firewall', 'system', 'error').",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
					"topics": {
						Type:        "string",
						Description: "Comma-separated log topics to filter: wireless, dhcp, firewall, system, info, warning, error, critical. Omit for all.",
					},
					"count": {
						Type:        "integer",
						Description: "Maximum number of log entries to return. Default: 50.",
						Default:     50,
					},
				},
			},
		},
		{
			Name:        "ping_from_router",
			Description: "Runs a ping from the router to a target address. Useful to test connectivity from the router's perspective.",
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"address"},
				Properties: map[string]Property{
					"router": routerProp,
					"address": {
						Type:        "string",
						Description: "Target IP address or hostname to ping.",
					},
					"count": {
						Type:        "integer",
						Description: "Number of ping packets to send. Default: 4.",
						Default:     4,
					},
					"interface": {
						Type:        "string",
						Description: "Source interface name (optional). Useful to test specific paths.",
					},
				},
			},
		},
		{
			Name:        "traceroute_from_router",
			Description: "Runs a traceroute from the router to a target address, showing each hop with latency.",
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"address"},
				Properties: map[string]Property{
					"router": routerProp,
					"address": {
						Type:        "string",
						Description: "Target IP address or hostname.",
					},
				},
			},
		},

		// ── WiFi / CAPsMAN ──────────────────────────────────────────────────
		{
			Name:        "get_wifi_clients",
			Description: "Returns all currently connected WiFi clients: MAC address, interface (AP), signal strength, band, rx/tx rates, and uptime.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
					"interface": {
						Type:        "string",
						Description: "Filter by WiFi interface name (e.g. wifi1, cap-wifi3). Omit for all interfaces.",
					},
				},
			},
		},
		{
			Name:        "get_wifi_interfaces",
			Description: "Lists WiFi radio interfaces on the router, including CAPsMAN-managed CAP interfaces. Shows band, channel, configuration, and running status.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name:        "get_wifi_configurations",
			Description: "Lists CAPsMAN WiFi configuration profiles (e.g. rede5, rede2.4): SSID, band, channel frequencies, width, security, and TX power.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name:        "get_capsman_status",
			Description: "Returns CAPsMAN controller status: enabled state, managed interfaces, and provisioning rules (which AP gets which configuration).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name: "set_wifi_configuration",
			Description: `Modifies a CAPsMAN WiFi configuration profile (e.g. rede5 or rede2.4).
IMPORTANT: Always call first with dry_run=true (default) to preview the change. Only call with dry_run=false after the user explicitly confirms.
Changes apply to all APs using the profile (channel changes require the APs to reconnect briefly).`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"configuration"},
				Properties: map[string]Property{
					"router": routerProp,
					"configuration": {
						Type:        "string",
						Description: "CAPsMAN configuration profile name (e.g. rede5, rede2.4).",
					},
					"ssid": {
						Type:        "string",
						Description: "New SSID (network name). Leave empty to keep current.",
					},
					"channel_frequency": {
						Type:        "string",
						Description: "Comma-separated channel frequencies in MHz (e.g. '5180,5260,5320'). Leave empty to keep current.",
					},
					"channel_width": {
						Type:        "string",
						Description: "Channel width: 20mhz, 20/40mhz, 20/40/80mhz, 20/40/80/160mhz. Leave empty to keep current.",
					},
					"tx_power": {
						Type:        "string",
						Description: "TX power in dBm (e.g. '18'). Leave empty to keep current.",
					},
					"passphrase": {
						Type:        "string",
						Description: "WPA2/WPA3 passphrase (password). Leave empty to keep current.",
					},
					"auth_types": {
						Type:        "string",
						Description: "Comma-separated authentication types: wpa2-psk, wpa3-psk (e.g. 'wpa2-psk,wpa3-psk'). Leave empty to keep current.",
					},
					"ft": {
						Type:        "boolean",
						Description: "Enable (true) or disable (false) 802.11r Fast Transition (fast roaming between APs). Omit to keep current.",
					},
					"wps": {
						Type:        "boolean",
						Description: "Enable (true) or disable (false) WPS. Omit to keep current.",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), returns a preview without making changes. Set to false only after user confirms.",
						Default:     true,
					},
				},
			},
		},
		{
			Name: "create_wifi_network",
			Description: `Creates a new CAPsMAN WiFi configuration profile with the specified SSID, band, and security settings.
After creation, provisioning rules must be added in RouterOS (or via CAPsMAN) to assign the new profile to specific APs.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name", "ssid", "band"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Unique profile name (e.g. rede-guest-5g).",
					},
					"ssid": {
						Type:        "string",
						Description: "WiFi network name (SSID) to broadcast.",
					},
					"band": {
						Type:        "string",
						Description: "Radio band: 2ghz-n, 2ghz-ax, 5ghz-ac, 5ghz-ax, 6ghz-ax.",
						Enum:        []string{"2ghz-n", "2ghz-ax", "5ghz-ac", "5ghz-ax", "6ghz-ax"},
					},
					"channel_frequency": {
						Type:        "string",
						Description: "Comma-separated channel frequencies in MHz (e.g. '5180,5260,5320'). Router chooses automatically if omitted.",
					},
					"channel_width": {
						Type:        "string",
						Description: "Channel width: 20mhz, 20/40mhz, 20/40/80mhz, 20/40/80/160mhz. Omit for router default.",
					},
					"passphrase": {
						Type:        "string",
						Description: "WPA2/WPA3 passphrase. Leave empty for an open (no password) network.",
					},
					"auth_types": {
						Type:        "string",
						Description: "Comma-separated authentication types: wpa2-psk, wpa3-psk. Defaults to 'wpa2-psk,wpa3-psk' when passphrase is set.",
					},
					"ft": {
						Type:        "boolean",
						Description: "Enable 802.11r Fast Transition (fast roaming). Default: false.",
						Default:     false,
					},
					"wps": {
						Type:        "boolean",
						Description: "Enable WPS. Default: false.",
						Default:     false,
					},
					"tx_power": {
						Type:        "string",
						Description: "TX power in dBm (e.g. '18'). Omit for router default.",
					},
					"country": {
						Type:        "string",
						Description: "Regulatory country code (e.g. 'brazil'). Omit to inherit from existing profiles.",
					},
					"datapath": {
						Type:        "string",
						Description: "Name of an existing datapath profile to use (e.g. capdp). Omit to use router default.",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), previews the configuration without creating it.",
						Default:     true,
					},
				},
			},
		},
		{
			Name: "delete_wifi_network",
			Description: `Deletes a CAPsMAN WiFi configuration profile and optionally removes all provisioning rules that reference it.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.
Warning: this action is irreversible.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Configuration profile name to delete (e.g. rede-guest-5g). Use get_wifi_configurations to list profiles.",
					},
					"remove_provisioning": {
						Type:        "boolean",
						Description: "If true (default), also deletes provisioning rules that reference this profile.",
						Default:     true,
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), shows what would be deleted without making changes.",
						Default:     true,
					},
				},
			},
		},

		// ── Firewall ────────────────────────────────────────────────────────
		{
			Name:        "get_firewall_rules",
			Description: "Returns firewall rules from filter, NAT, and/or mangle tables. Shows chain, action, src/dst addresses, protocol, ports, and comment.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
					"table": {
						Type:        "string",
						Description: "Table to query: filter, nat, mangle. Omit for all three.",
						Enum:        []string{"filter", "nat", "mangle"},
					},
					"chain": {
						Type:        "string",
						Description: "Filter by chain name (e.g. forward, input, output, srcnat, dstnat). Omit for all chains.",
					},
				},
			},
		},
		{
			Name: "add_firewall_rule",
			Description: `Adds a new firewall rule to the specified table and chain.
IMPORTANT: Always call first with dry_run=true (default) to preview the rule. Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"table", "chain", "action"},
				Properties: map[string]Property{
					"router": routerProp,
					"table": {
						Type:        "string",
						Description: "Firewall table: filter, nat, mangle.",
						Enum:        []string{"filter", "nat", "mangle"},
					},
					"chain": {
						Type:        "string",
						Description: "Chain name: forward, input, output, srcnat, dstnat, prerouting, postrouting.",
					},
					"action": {
						Type:        "string",
						Description: "Rule action: accept, drop, reject, masquerade, dst-nat, src-nat, log, passthrough, return.",
					},
					"src_address": {
						Type:        "string",
						Description: "Source IP address or CIDR (e.g. 192.168.1.0/24). Optional.",
					},
					"dst_address": {
						Type:        "string",
						Description: "Destination IP address or CIDR. Optional.",
					},
					"protocol": {
						Type:        "string",
						Description: "Protocol: tcp, udp, icmp, etc. Optional.",
					},
					"dst_port": {
						Type:        "string",
						Description: "Destination port or range (e.g. '80', '80-443'). Requires protocol. Optional.",
					},
					"src_port": {
						Type:        "string",
						Description: "Source port or range. Requires protocol. Optional.",
					},
					"in_interface": {
						Type:        "string",
						Description: "Inbound interface name. Optional.",
					},
					"out_interface": {
						Type:        "string",
						Description: "Outbound interface name. Optional.",
					},
					"comment": {
						Type:        "string",
						Description: "Human-readable comment for this rule.",
					},
					"position": {
						Type:        "string",
						Description: "Insert position: 'top' to prepend before all rules, or a rule ID (e.g. '*3') to insert after. Default: append at end.",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), returns a preview without making changes.",
						Default:     true,
					},
				},
			},
		},
		{
			Name: "remove_firewall_rule",
			Description: `Removes a firewall rule by its ID. Use get_firewall_rules to find the rule ID (shown as .id field).
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"rule_id", "table"},
				Properties: map[string]Property{
					"router": routerProp,
					"rule_id": {
						Type:        "string",
						Description: "The rule's .id from get_firewall_rules (e.g. '*11', '*F9').",
					},
					"table": {
						Type:        "string",
						Description: "Firewall table the rule belongs to: filter, nat, mangle.",
						Enum:        []string{"filter", "nat", "mangle"},
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), returns a preview without deleting the rule.",
						Default:     true,
					},
				},
			},
		},

		// ── QoS / DNS / Interface ────────────────────────────────────────────
		{
			Name:        "get_queue_stats",
			Description: "Returns QoS queue statistics: queue names, parent, max bandwidth limit, current rate, and bytes transferred.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name:        "get_dns_entries",
			Description: "Returns static DNS entries configured on the router.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name: "set_queue_limit",
			Description: `Adjusts the maximum bandwidth limit on a QoS queue (tree or simple).
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"queue_name", "max_limit"},
				Properties: map[string]Property{
					"router": routerProp,
					"queue_name": {
						Type:        "string",
						Description: "Queue name as shown in get_queue_stats (e.g. queue-upload, queue-download).",
					},
					"max_limit": {
						Type:        "string",
						Description: "New maximum bandwidth: use M for Mbps, G for Gbps, K for Kbps (e.g. '500M', '1G', '100M'). Use '0' to remove limit.",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), returns a preview without making changes.",
						Default:     true,
					},
				},
			},
		},
		{
			Name: "restart_interface",
			Description: `Restarts a network interface by disabling then re-enabling it (2-second pause between).
Useful to reset a stuck interface without rebooting the router.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.
Warning: restarting an interface may briefly interrupt traffic on that segment.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"interface"},
				Properties: map[string]Property{
					"router": routerProp,
					"interface": {
						Type:        "string",
						Description: "Interface name to restart (e.g. ether1, wifi1, cap-wifi3).",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), describes the action without performing it.",
						Default:     true,
					},
				},
			},
		},

		// ── Backup ──────────────────────────────────────────────────────────
		{
			Name: "create_backup",
			Description: `Creates a RouterOS binary backup (.backup) on the router and optionally uploads it to S3.
The backup can be used to fully restore the router configuration.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Custom backup filename (without extension). Defaults to '{router-name}-{date}'.",
					},
					"upload_s3": {
						Type:        "boolean",
						Description: "Upload the backup to S3 after creation. Requires AWS config. Default: true if S3 is configured.",
						Default:     true,
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), describes what would be done without creating the backup.",
						Default:     true,
					},
				},
			},
		},
		// ── WiFi Security / Datapath / Interface / Provisioning ─────────────
		{
			Name: "add_wifi_security",
			Description: `Creates a WiFi security profile that can be referenced by CAPsMAN configuration profiles.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Unique profile name (e.g. sec-guest).",
					},
					"passphrase": {
						Type:        "string",
						Description: "WPA2/WPA3 passphrase. Leave empty for open network.",
					},
					"auth_types": {
						Type:        "string",
						Description: "Comma-separated auth types: wpa2-psk, wpa3-psk. Defaults to 'wpa2-psk,wpa3-psk' when passphrase is set.",
					},
					"ft": {
						Type:        "boolean",
						Description: "Enable 802.11r Fast Transition (fast roaming). Default: false.",
						Default:     false,
					},
					"wps": {
						Type:        "boolean",
						Description: "Enable WPS. Default: false.",
						Default:     false,
					},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_wifi_security",
			Description: `Deletes a WiFi security profile. Ensure no configuration profile references it first.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name":   {Type: "string", Description: "Security profile name to delete."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "add_wifi_datapath",
			Description: `Creates a WiFi datapath profile used by CAPsMAN to bridge client traffic.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Unique datapath name (e.g. dp-guest).",
					},
					"bridge": {
						Type:        "string",
						Description: "Bridge interface to forward client traffic to (e.g. bridge1).",
					},
					"vlan_id": {
						Type:        "string",
						Description: "VLAN ID to tag client traffic (1–4094). Optional.",
					},
					"client_isolation": {
						Type:        "boolean",
						Description: "Prevent clients from talking to each other. Default: false.",
						Default:     false,
					},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_wifi_datapath",
			Description: `Deletes a WiFi datapath profile. Ensure no configuration profile references it first.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name":   {Type: "string", Description: "Datapath profile name to delete."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "add_wifi_interface",
			Description: `Creates a virtual WiFi interface (secondary SSID / slave AP) on top of a physical radio.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name", "master_interface"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Name for the new virtual interface (e.g. wifi3-guest).",
					},
					"master_interface": {
						Type:        "string",
						Description: "Physical WiFi interface to create the virtual AP on (e.g. wifi1, wifi2).",
					},
					"configuration": {
						Type:        "string",
						Description: "CAPsMAN configuration profile to assign. Optional.",
					},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_wifi_interface",
			Description: `Deletes a virtual (slave) WiFi interface. Physical interfaces cannot be removed.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name":   {Type: "string", Description: "Virtual WiFi interface name to delete."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "add_wifi_provisioning",
			Description: `Adds a CAPsMAN provisioning rule that maps a radio (by MAC or band) to a configuration profile.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"master_configuration"},
				Properties: map[string]Property{
					"router": routerProp,
					"master_configuration": {
						Type:        "string",
						Description: "CAPsMAN configuration profile to assign (e.g. rede5).",
					},
					"radio_mac": {
						Type:        "string",
						Description: "Radio MAC address to match (e.g. AA:BB:CC:DD:EE:FF). Omit to match all radios.",
					},
					"supported_bands": {
						Type:        "string",
						Description: "Band filter: 2ghz-n, 5ghz-ax, etc. Omit for no band filter.",
					},
					"slave_configuration": {
						Type:        "string",
						Description: "Secondary configuration profile (for dual-band APs). Optional.",
					},
					"action": {
						Type:        "string",
						Description: "Action: create-enabled, create-disabled, create-dynamic-enabled. Default: create-dynamic-enabled.",
						Enum:        []string{"create-enabled", "create-disabled", "create-dynamic-enabled"},
					},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_wifi_provisioning",
			Description: `Removes a CAPsMAN provisioning rule by its ID. Use get_capsman_status to find rule IDs.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]Property{
					"router": routerProp,
					"id":     {Type: "string", Description: "Rule ID from get_capsman_status (e.g. *1, *F3)."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name:        "provision_caps",
			Description: "Forces CAPsMAN to re-provision all managed APs — useful after adding or changing provisioning rules. APs will briefly reconnect.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},

		// ── Bridges / VLANs ─────────────────────────────────────────────────
		{
			Name: "add_bridge",
			Description: `Creates a new bridge interface.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name":   {Type: "string", Description: "Bridge interface name (e.g. bridge2)."},
					"protocol_mode": {
						Type:        "string",
						Description: "STP/RSTP mode: none, stp, rstp, mstp. Default: rstp.",
						Enum:        []string{"none", "stp", "rstp", "mstp"},
					},
					"vlan_filtering": {
						Type:        "boolean",
						Description: "Enable bridge VLAN filtering. Default: false.",
						Default:     false,
					},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_bridge",
			Description: `Deletes a bridge interface and all its ports.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name":   {Type: "string", Description: "Bridge name to delete."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "add_bridge_port",
			Description: `Adds an interface as a port to a bridge.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"bridge", "interface"},
				Properties: map[string]Property{
					"router":    routerProp,
					"bridge":    {Type: "string", Description: "Bridge name (e.g. bridge1)."},
					"interface": {Type: "string", Description: "Interface to add as port (e.g. ether2, vlan10)."},
					"pvid": {
						Type:        "string",
						Description: "Port VLAN ID (1–4094). Only relevant when vlan-filtering is enabled on the bridge.",
					},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_bridge_port",
			Description: `Removes an interface from a bridge.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"interface"},
				Properties: map[string]Property{
					"router":    routerProp,
					"interface": {Type: "string", Description: "Interface to remove from the bridge."},
					"bridge":    {Type: "string", Description: "Bridge name (optional, to disambiguate if interface appears in multiple bridges)."},
					"dry_run":   {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "add_vlan",
			Description: `Creates a VLAN sub-interface on top of a physical or bridge interface.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"interface", "vlan_id"},
				Properties: map[string]Property{
					"router":    routerProp,
					"interface": {Type: "string", Description: "Parent interface (e.g. ether1, bridge1)."},
					"vlan_id":   {Type: "string", Description: "VLAN ID (1–4094)."},
					"name": {
						Type:        "string",
						Description: "Interface name. Defaults to 'vlan<id>' if not provided.",
					},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_vlan",
			Description: `Deletes a VLAN sub-interface.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name":   {Type: "string", Description: "VLAN interface name to delete."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},

		// ── IP Addresses ─────────────────────────────────────────────────────
		{
			Name: "add_ip_address",
			Description: `Assigns an IP address to an interface.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"address", "interface"},
				Properties: map[string]Property{
					"router":    routerProp,
					"address":   {Type: "string", Description: "IP address with prefix length (e.g. 192.168.10.1/24)."},
					"interface": {Type: "string", Description: "Interface to assign the address to (e.g. bridge2, vlan10)."},
					"comment":   {Type: "string", Description: "Optional comment."},
					"dry_run":   {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_ip_address",
			Description: `Removes an IP address from an interface. Use get_ip_addresses to find the address or ID.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router":    routerProp,
					"address":   {Type: "string", Description: "IP address to remove (e.g. 192.168.10.1 or 192.168.10.1/24)."},
					"interface": {Type: "string", Description: "Interface name to disambiguate when same IP appears on multiple interfaces."},
					"id":        {Type: "string", Description: "Address .id from get_ip_addresses (e.g. *3). Takes priority over address+interface."},
					"dry_run":   {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},

		// ── IP Pools ─────────────────────────────────────────────────────────
		{
			Name: "add_ip_pool",
			Description: `Creates an IP address pool used by DHCP servers.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name", "ranges"},
				Properties: map[string]Property{
					"router":  routerProp,
					"name":    {Type: "string", Description: "Pool name (e.g. pool-guest)."},
					"ranges":  {Type: "string", Description: "IP range or comma-separated ranges (e.g. '192.168.10.100-192.168.10.200')."},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_ip_pool",
			Description: `Deletes an IP address pool.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name":   {Type: "string", Description: "Pool name to delete."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},

		// ── DHCP Server ──────────────────────────────────────────────────────
		{
			Name: "add_dhcp_server",
			Description: `Creates a DHCP server on an interface.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name", "interface"},
				Properties: map[string]Property{
					"router":    routerProp,
					"name":      {Type: "string", Description: "DHCP server name (e.g. dhcp-guest)."},
					"interface": {Type: "string", Description: "Interface to serve DHCP on (e.g. bridge2, vlan10)."},
					"address_pool": {
						Type:        "string",
						Description: "IP pool to use (e.g. pool-guest). Created separately with add_ip_pool.",
					},
					"lease_time": {
						Type:        "string",
						Description: "Lease duration (e.g. 1d, 12h, 30m). Default: RouterOS default (10m).",
					},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_dhcp_server",
			Description: `Deletes a DHCP server.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name":   {Type: "string", Description: "DHCP server name to delete."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "add_dhcp_network",
			Description: `Adds a DHCP network entry that defines gateway and DNS options for a subnet.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"address", "gateway"},
				Properties: map[string]Property{
					"router":  routerProp,
					"address": {Type: "string", Description: "Subnet in CIDR notation (e.g. 192.168.10.0/24)."},
					"gateway": {Type: "string", Description: "Default gateway IP (e.g. 192.168.10.1)."},
					"dns_server": {
						Type:        "string",
						Description: "DNS server IP(s) to hand out to clients (e.g. '8.8.8.8,8.8.4.4').",
					},
					"ntp_server": {Type: "string", Description: "NTP server IP(s). Optional."},
					"domain":     {Type: "string", Description: "DNS domain name handed to clients. Optional."},
					"comment":    {Type: "string", Description: "Optional comment."},
					"dry_run":    {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_dhcp_network",
			Description: `Removes a DHCP network entry by its subnet address.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"address"},
				Properties: map[string]Property{
					"router":  routerProp,
					"address": {Type: "string", Description: "Subnet to remove (e.g. 192.168.10.0/24)."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "add_dhcp_lease",
			Description: `Creates a static DHCP lease (MAC → IP reservation).
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"address", "mac_address"},
				Properties: map[string]Property{
					"router":      routerProp,
					"address":     {Type: "string", Description: "Reserved IP address (e.g. 192.168.1.50)."},
					"mac_address": {Type: "string", Description: "Client MAC address (e.g. AA:BB:CC:DD:EE:FF)."},
					"server":      {Type: "string", Description: "DHCP server name to associate this lease with. Optional."},
					"hostname":    {Type: "string", Description: "Client hostname / identifier. Optional."},
					"comment":     {Type: "string", Description: "Optional comment."},
					"dry_run":     {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_dhcp_lease",
			Description: `Removes a static DHCP lease by MAC address or IP address.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router":      routerProp,
					"mac_address": {Type: "string", Description: "Client MAC address (e.g. AA:BB:CC:DD:EE:FF)."},
					"address":     {Type: "string", Description: "Reserved IP address. Alternative to mac_address."},
					"dry_run":     {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},

		// ── DNS Write ────────────────────────────────────────────────────────
		{
			Name: "add_dns_entry",
			Description: `Adds a static DNS entry (hostname → IP) to the router's DNS cache.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name", "address"},
				Properties: map[string]Property{
					"router":  routerProp,
					"name":    {Type: "string", Description: "Hostname to resolve (e.g. nas.home.lan)."},
					"address": {Type: "string", Description: "IP address the hostname resolves to."},
					"ttl":     {Type: "string", Description: "Time-to-live (e.g. '1d', '1h'). Optional."},
					"comment": {Type: "string", Description: "Optional comment."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},
		{
			Name: "remove_dns_entry",
			Description: `Removes a static DNS entry by hostname or IP address.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router":  routerProp,
					"name":    {Type: "string", Description: "Hostname to remove (e.g. nas.home.lan)."},
					"address": {Type: "string", Description: "IP address. Alternative to name."},
					"dry_run": {Type: "boolean", Description: "Preview without making changes.", Default: true},
				},
			},
		},

		// ── WireGuard ────────────────────────────────────────────────────────
		{
			Name:        "get_wireguard_status",
			Description: "Returns WireGuard interface status and all peers: connected/disconnected, last handshake, IP, rx/tx traffic, and enabled state.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"router": routerProp,
				},
			},
		},
		{
			Name: "add_wireguard_peer",
			Description: `Adds a new WireGuard peer (client device) to the VPN.
Generates a Curve25519 key pair and preshared key automatically, assigns the next available IP in the VPN subnet, and returns the complete WireGuard client config ready to import.
IMPORTANT: Always call first with dry_run=true (default). The private key is shown only once upon creation — save it immediately.
Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Unique peer name (e.g. iphone-ernesto, laptop-work).",
					},
					"comment": {
						Type:        "string",
						Description: "Optional human-readable comment displayed in the router UI.",
					},
					"interface": {
						Type:        "string",
						Description: "WireGuard interface to add the peer to. Default: wireguard1.",
						Default:     "wireguard1",
					},
					"full_tunnel": {
						Type:        "boolean",
						Description: "If true (default), all client traffic is routed through the VPN (0.0.0.0/0). If false, only LAN traffic is routed (split tunnel).",
						Default:     true,
					},
					"endpoint": {
						Type:        "string",
						Description: "Router's public hostname or IP (e.g. myhome.dyndns.org). Auto-detected from existing peers if omitted.",
					},
					"dns": {
						Type:        "string",
						Description: "DNS server IP for the client. Auto-detected from existing peers if omitted.",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), previews what would be created without making changes.",
						Default:     true,
					},
				},
			},
		},
		{
			Name: "disable_wireguard_peer",
			Description: `Disables a WireGuard peer, immediately revoking VPN access without deleting the peer config.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Peer name (as shown in get_wireguard_status).",
					},
					"dry_run": {
						Type:    "boolean",
						Description: "If true (default), previews the action without changes.",
						Default: true,
					},
				},
			},
		},
		{
			Name: "enable_wireguard_peer",
			Description: `Re-enables a previously disabled WireGuard peer, restoring VPN access.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Peer name (as shown in get_wireguard_status).",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), previews the action without changes.",
						Default:     true,
					},
				},
			},
		},
		{
			Name: "remove_wireguard_peer",
			Description: `Permanently deletes a WireGuard peer. The client loses VPN access immediately and the config cannot be recovered.
IMPORTANT: Always call first with dry_run=true (default). Only call with dry_run=false after the user explicitly confirms.`,
			InputSchema: InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]Property{
					"router": routerProp,
					"name": {
						Type:        "string",
						Description: "Peer name to permanently delete.",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true (default), previews deletion without removing anything.",
						Default:     true,
					},
				},
			},
		},
	}

	m := make(map[string]Tool, len(tools))
	for _, t := range tools {
		m[t.Name] = t
	}
	return m
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}
