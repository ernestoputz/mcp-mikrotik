package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Server) toolGetWiFiClients(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/interface/wifi/registration-table")
	if err != nil {
		return ToolResult{}, err
	}

	var clients []map[string]string
	if err := json.Unmarshal(raw, &clients); err != nil {
		return ToolResult{}, fmt.Errorf("parse wifi clients: %w", err)
	}

	ifaceFilter := strOpt(args, "interface", "")

	var sb strings.Builder
	fmt.Fprintf(&sb, "WiFi Clients — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))

	// Group by interface
	grouped := map[string][]map[string]string{}
	order := []string{}
	for _, c := range clients {
		iface := c["interface"]
		if ifaceFilter != "" && iface != ifaceFilter {
			continue
		}
		if _, ok := grouped[iface]; !ok {
			order = append(order, iface)
		}
		grouped[iface] = append(grouped[iface], c)
	}

	total := 0
	for _, iface := range order {
		clist := grouped[iface]
		fmt.Fprintf(&sb, "Interface: %s (%d client(s))\n", iface, len(clist))
		for _, c := range clist {
			signal := c["signal"]
			band := c["band"]
			rxRate := formatRate(c["rx-rate"])
			txRate := formatRate(c["tx-rate"])
			auth := c["auth-type"]
			ssid := c["ssid"]

			fmt.Fprintf(&sb, "  • %-18s  signal: %-6s dBm  band: %-10s  rx: %-12s  tx: %-12s  auth: %-10s  ssid: %s  uptime: %s\n",
				c["mac-address"], signal, band, rxRate, txRate, auth, ssid, c["uptime"])
		}
		total += len(clist)
		sb.WriteString("\n")
	}

	if total == 0 {
		sb.WriteString("No WiFi clients connected.\n")
	} else {
		fmt.Fprintf(&sb, "Total: %d client(s)\n", total)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetWiFiInterfaces(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/interface/wifi")
	if err != nil {
		return ToolResult{}, err
	}

	var ifaces []map[string]string
	if err := json.Unmarshal(raw, &ifaces); err != nil {
		return ToolResult{}, fmt.Errorf("parse wifi interfaces: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "WiFi Interfaces — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))

	for _, iface := range ifaces {
		running := "DOWN"
		if iface["running"] == "true" {
			running = "UP"
		}
		inactive := ""
		if iface["inactive"] == "true" {
			inactive = " [inactive]"
		}

		about := iface[".about"]
		cap := iface["cap"]

		fmt.Fprintf(&sb, "%-15s  %-4s  band: %-12s  freq: %-20s  width: %-16s%s\n",
			iface["name"], running,
			iface["channel.band"], iface["channel.frequency"], iface["channel.width"],
			inactive)
		fmt.Fprintf(&sb, "  config: %-15s  mac: %-17s  ssid: %s\n",
			iface["configuration"], iface["mac-address"], iface["configuration.ssid"])
		if about != "" {
			fmt.Fprintf(&sb, "  info: %s\n", about)
		}
		if cap != "" {
			fmt.Fprintf(&sb, "  managed-by-cap: %s\n", cap)
		}
		sb.WriteString("\n")
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetWiFiConfigurations(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/interface/wifi/configuration")
	if err != nil {
		return ToolResult{}, err
	}

	var cfgs []map[string]string
	if err := json.Unmarshal(raw, &cfgs); err != nil {
		return ToolResult{}, fmt.Errorf("parse wifi configurations: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "CAPsMAN WiFi Configurations — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))

	for _, c := range cfgs {
		fmt.Fprintf(&sb, "Profile: %s  (ID: %s)\n", c["name"], c[".id"])
		fmt.Fprintf(&sb, "  SSID:        %s\n", c["ssid"])
		fmt.Fprintf(&sb, "  Band:        %s\n", c["channel.band"])
		fmt.Fprintf(&sb, "  Frequency:   %s MHz\n", c["channel.frequency"])
		fmt.Fprintf(&sb, "  Width:       %s\n", c["channel.width"])
		fmt.Fprintf(&sb, "  TX Power:    %s dBm\n", c["tx-power"])
		fmt.Fprintf(&sb, "  Security:    %s\n", c["security.authentication-types"])
		fmt.Fprintf(&sb, "  FT (roam):   %s\n", c["security.ft"])
		fmt.Fprintf(&sb, "  Country:     %s\n", c["country"])
		fmt.Fprintf(&sb, "  Manager:     %s\n", c["manager"])
		fmt.Fprintf(&sb, "  Disabled:    %s\n", c["disabled"])
		sb.WriteString("\n")
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetCAPsMANStatus(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	rawMgr, err := r.Get("/interface/wifi/capsman")
	if err != nil {
		return ToolResult{}, err
	}

	var mgr map[string]string
	if err := json.Unmarshal(rawMgr, &mgr); err != nil {
		return ToolResult{}, fmt.Errorf("parse capsman: %w", err)
	}

	rawProv, err := r.Get("/interface/wifi/provisioning")
	if err != nil {
		return ToolResult{}, err
	}

	var provisioning []map[string]string
	if err := json.Unmarshal(rawProv, &provisioning); err != nil {
		return ToolResult{}, fmt.Errorf("parse provisioning: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "CAPsMAN Status — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))
	fmt.Fprintf(&sb, "Controller enabled:  %s\n", mgr["enabled"])
	fmt.Fprintf(&sb, "Interfaces:          %s\n", mgr["interfaces"])
	fmt.Fprintf(&sb, "Certificate:         %s\n", mgr["generated-certificate"])
	fmt.Fprintf(&sb, "Upgrade policy:      %s\n", mgr["upgrade-policy"])

	sb.WriteString("\nProvisioning Rules:\n")
	for _, p := range provisioning {
		comment := ""
		if p["comment"] != "" {
			comment = " (" + p["comment"] + ")"
		}
		disabled := ""
		if p["disabled"] == "true" {
			disabled = " [disabled]"
		}
		fmt.Fprintf(&sb, "  • ID: %-8s  radio-mac: %-20s  config: %-12s  action: %s%s%s\n",
			p[".id"], p["radio-mac"], p["master-configuration"], p["action"], comment, disabled)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolSetWiFiConfiguration(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	configName, err := strArg(args, "configuration")
	if err != nil {
		return ToolResult{}, err
	}

	dryRun := boolOpt(args, "dry_run", true)

	// Fetch current configurations to find the target
	raw, err := r.Get("/interface/wifi/configuration")
	if err != nil {
		return ToolResult{}, err
	}

	var cfgs []map[string]string
	if err := json.Unmarshal(raw, &cfgs); err != nil {
		return ToolResult{}, fmt.Errorf("parse configurations: %w", err)
	}

	var target map[string]string
	for _, c := range cfgs {
		if c["name"] == configName {
			target = c
			break
		}
	}
	if target == nil {
		names := make([]string, len(cfgs))
		for i, c := range cfgs {
			names[i] = c["name"]
		}
		return ToolResult{}, fmt.Errorf("configuration %q not found; available: %v", configName, names)
	}

	// Build the patch body with only fields that were provided
	patch := map[string]string{}
	changes := map[string]string{}

	if v := strOpt(args, "ssid", ""); v != "" {
		patch["ssid"] = v
		changes["ssid"] = fmt.Sprintf("%q → %q", target["ssid"], v)
	}
	if v := strOpt(args, "channel_frequency", ""); v != "" {
		patch["channel.frequency"] = v
		changes["channel.frequency"] = fmt.Sprintf("%q → %q", target["channel.frequency"], v)
	}
	if v := strOpt(args, "channel_width", ""); v != "" {
		patch["channel.width"] = v
		changes["channel.width"] = fmt.Sprintf("%q → %q", target["channel.width"], v)
	}
	if v := strOpt(args, "tx_power", ""); v != "" {
		patch["tx-power"] = v
		changes["tx-power"] = fmt.Sprintf("%q → %q", target["tx-power"], v)
	}

	if len(patch) == 0 {
		return ToolResult{}, fmt.Errorf("no changes specified; provide at least one of: ssid, channel_frequency, channel_width, tx_power")
	}

	var sb strings.Builder
	if dryRun {
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would modify CAPsMAN configuration: %s (ID: %s)\n", configName, target[".id"])
		fmt.Fprintf(&sb, "Router: %s\n\nChanges:\n", r.Name())
		for field, change := range changes {
			fmt.Fprintf(&sb, "  %-22s  %s\n", field, change)
		}
		fmt.Fprintf(&sb, "\nNote: channel changes cause APs to briefly reconnect.\n")
		fmt.Fprintf(&sb, "\nTo apply: call set_wifi_configuration again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	_, err = r.Patch("/interface/wifi/configuration/"+target[".id"], patch)
	if err != nil {
		return ToolResult{}, fmt.Errorf("update configuration %s: %w", configName, err)
	}

	sb.WriteString("✓ Configuration updated successfully\n")
	sb.WriteString(strings.Repeat("═", 40) + "\n\n")
	fmt.Fprintf(&sb, "Profile: %s (ID: %s) on %s\n\nApplied changes:\n", configName, target[".id"], r.Name())
	for field, change := range changes {
		fmt.Fprintf(&sb, "  %-22s  %s\n", field, change)
	}

	return textResult(sb.String()), nil
}

// formatRate converts a bits-per-second value string to a human-readable rate.
func formatRate(bps string) string {
	var v int64
	fmt.Sscanf(bps, "%d", &v)
	switch {
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.1f Gbps", float64(v)/1e9)
	case v >= 1_000_000:
		return fmt.Sprintf("%.1f Mbps", float64(v)/1e6)
	case v >= 1_000:
		return fmt.Sprintf("%.0f Kbps", float64(v)/1e3)
	default:
		return fmt.Sprintf("%d bps", v)
	}
}
