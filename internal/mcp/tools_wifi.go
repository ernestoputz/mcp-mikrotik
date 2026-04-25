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
	if v := strOpt(args, "passphrase", ""); v != "" {
		patch["security.passphrase"] = v
		changes["security.passphrase"] = "*** → updated (hidden)"
	}
	if v := strOpt(args, "auth_types", ""); v != "" {
		patch["security.authentication-types"] = v
		changes["security.authentication-types"] = fmt.Sprintf("%q → %q", target["security.authentication-types"], v)
	}
	if _, ok := args["ft"]; ok {
		ftStr := boolToYesNo(boolOpt(args, "ft", false))
		patch["security.ft"] = ftStr
		changes["security.ft"] = fmt.Sprintf("%q → %q", target["security.ft"], ftStr)
	}
	if _, ok := args["wps"]; ok {
		wpsStr := boolToYesNo(boolOpt(args, "wps", false))
		patch["security.wps"] = wpsStr
		changes["security.wps"] = fmt.Sprintf("%q → %q", target["security.wps"], wpsStr)
	}

	if len(patch) == 0 {
		return ToolResult{}, fmt.Errorf("no changes specified; provide at least one of: ssid, channel_frequency, channel_width, tx_power, passphrase, auth_types, ft, wps")
	}

	var sb strings.Builder
	if dryRun {
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would modify CAPsMAN configuration: %s (ID: %s)\n", configName, target[".id"])
		fmt.Fprintf(&sb, "Router: %s\n\nChanges:\n", r.Name())
		for field, change := range changes {
			fmt.Fprintf(&sb, "  %-30s  %s\n", field, change)
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
		fmt.Fprintf(&sb, "  %-30s  %s\n", field, change)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolCreateWiFiNetwork(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	ssid, err := strArg(args, "ssid")
	if err != nil {
		return ToolResult{}, err
	}
	band, err := strArg(args, "band")
	if err != nil {
		return ToolResult{}, err
	}

	dryRun := boolOpt(args, "dry_run", true)

	// Verify name doesn't already exist
	rawCfgs, err := r.Get("/interface/wifi/configuration")
	if err != nil {
		return ToolResult{}, err
	}
	var existingCfgs []map[string]string
	if err := json.Unmarshal(rawCfgs, &existingCfgs); err != nil {
		return ToolResult{}, fmt.Errorf("parse configurations: %w", err)
	}
	for _, c := range existingCfgs {
		if c["name"] == name {
			return ToolResult{}, fmt.Errorf("configuration %q already exists; use set_wifi_configuration to modify it", name)
		}
	}

	// Build configuration body
	body := map[string]string{
		"name":         name,
		"ssid":         ssid,
		"channel.band": band,
		"manager":      "capsman",
	}

	if v := strOpt(args, "channel_frequency", ""); v != "" {
		body["channel.frequency"] = v
	}
	if v := strOpt(args, "channel_width", ""); v != "" {
		body["channel.width"] = v
	}
	if v := strOpt(args, "tx_power", ""); v != "" {
		body["tx-power"] = v
	}
	if v := strOpt(args, "country", ""); v != "" {
		body["country"] = v
	}
	if v := strOpt(args, "datapath", ""); v != "" {
		body["datapath"] = v
	}

	passphrase := strOpt(args, "passphrase", "")
	authTypes := strOpt(args, "auth_types", "")
	if passphrase != "" {
		body["security.passphrase"] = passphrase
		if authTypes == "" {
			authTypes = "wpa2-psk,wpa3-psk"
		}
	}
	if authTypes != "" {
		body["security.authentication-types"] = authTypes
	}
	if _, ok := args["ft"]; ok {
		body["security.ft"] = boolToYesNo(boolOpt(args, "ft", false))
	}
	if _, ok := args["wps"]; ok {
		body["security.wps"] = boolToYesNo(boolOpt(args, "wps", false))
	}

	var sb strings.Builder
	if dryRun {
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create CAPsMAN WiFi configuration on %s:\n\n", r.Name())
		for k, v := range body {
			display := v
			if k == "security.passphrase" {
				display = "***"
			}
			fmt.Fprintf(&sb, "  %-35s  %s\n", k, display)
		}
		sb.WriteString("\nNote: After creation, assign this config to APs by adding provisioning\n")
		sb.WriteString("rules in RouterOS (use get_capsman_status to review existing rules).\n")
		sb.WriteString("\nTo apply: call create_wifi_network again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	respRaw, err := r.Put("/interface/wifi/configuration", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create configuration %s: %w", name, err)
	}

	var created map[string]string
	_ = json.Unmarshal(respRaw, &created)

	sb.WriteString("✓ WiFi configuration created successfully\n")
	sb.WriteString(strings.Repeat("═", 40) + "\n\n")
	fmt.Fprintf(&sb, "Profile: %s  SSID: %s  Band: %s\n", name, ssid, band)
	fmt.Fprintf(&sb, "Router: %s\n", r.Name())
	if id := created[".id"]; id != "" {
		fmt.Fprintf(&sb, "ID: %s\n", id)
	}
	sb.WriteString("\nNext steps:\n")
	sb.WriteString("  • Run get_capsman_status to review provisioning rules\n")
	sb.WriteString("  • Add provisioning rules in RouterOS to assign this config to APs\n")

	return textResult(sb.String()), nil
}

func (s *Server) toolDeleteWiFiNetwork(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}

	dryRun := boolOpt(args, "dry_run", true)
	removeProvisioning := boolOpt(args, "remove_provisioning", true)

	// Find configuration
	rawCfgs, err := r.Get("/interface/wifi/configuration")
	if err != nil {
		return ToolResult{}, err
	}
	var cfgs []map[string]string
	if err := json.Unmarshal(rawCfgs, &cfgs); err != nil {
		return ToolResult{}, fmt.Errorf("parse configurations: %w", err)
	}

	var target map[string]string
	for _, c := range cfgs {
		if c["name"] == name {
			target = c
			break
		}
	}
	if target == nil {
		names := make([]string, len(cfgs))
		for i, c := range cfgs {
			names[i] = c["name"]
		}
		return ToolResult{}, fmt.Errorf("configuration %q not found; available: %v", name, names)
	}

	// Collect provisioning rules that reference this config
	var provRules []map[string]string
	if removeProvisioning {
		rawProv, err := r.Get("/interface/wifi/provisioning")
		if err != nil {
			return ToolResult{}, err
		}
		var allProv []map[string]string
		if err := json.Unmarshal(rawProv, &allProv); err != nil {
			return ToolResult{}, fmt.Errorf("parse provisioning: %w", err)
		}
		for _, p := range allProv {
			if p["master-configuration"] == name || p["slave-configuration"] == name {
				provRules = append(provRules, p)
			}
		}
	}

	var sb strings.Builder
	if dryRun {
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE CAPsMAN configuration: %s (ID: %s)\n", name, target[".id"])
		fmt.Fprintf(&sb, "Router: %s\n", r.Name())
		fmt.Fprintf(&sb, "SSID: %s   Band: %s\n", target["ssid"], target["channel.band"])
		if removeProvisioning && len(provRules) > 0 {
			fmt.Fprintf(&sb, "\nWould also remove %d provisioning rule(s):\n", len(provRules))
			for _, p := range provRules {
				comment := ""
				if p["comment"] != "" {
					comment = " (" + p["comment"] + ")"
				}
				fmt.Fprintf(&sb, "  • ID: %-8s  radio-mac: %-20s%s\n", p[".id"], p["radio-mac"], comment)
			}
		} else if removeProvisioning {
			sb.WriteString("\nNo provisioning rules reference this configuration.\n")
		}
		sb.WriteString("\nWarning: This action is irreversible.\n")
		sb.WriteString("\nTo apply: call delete_wifi_network again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	// Remove provisioning rules first to avoid orphan references
	for _, p := range provRules {
		if err := r.Delete("/interface/wifi/provisioning/" + p[".id"]); err != nil {
			return ToolResult{}, fmt.Errorf("remove provisioning rule %s: %w", p[".id"], err)
		}
	}

	if err := r.Delete("/interface/wifi/configuration/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete configuration %s: %w", name, err)
	}

	sb.WriteString("✓ WiFi configuration deleted\n")
	sb.WriteString(strings.Repeat("═", 40) + "\n\n")
	fmt.Fprintf(&sb, "Deleted: %s (ID: %s) on %s\n", name, target[".id"], r.Name())
	if len(provRules) > 0 {
		fmt.Fprintf(&sb, "Also removed %d provisioning rule(s).\n", len(provRules))
	}

	return textResult(sb.String()), nil
}

// ─── WiFi Security Profiles ───────────────────────────────────────────────────

func (s *Server) toolAddWiFiSecurity(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{"name": name}
	if v := strOpt(args, "passphrase", ""); v != "" {
		body["passphrase"] = v
	}
	if v := strOpt(args, "auth_types", ""); v != "" {
		body["authentication-types"] = v
	} else if body["passphrase"] != "" {
		body["authentication-types"] = "wpa2-psk,wpa3-psk"
	}
	if _, ok := args["ft"]; ok {
		body["ft"] = boolToYesNo(boolOpt(args, "ft", false))
	}
	if _, ok := args["wps"]; ok {
		body["wps"] = boolToYesNo(boolOpt(args, "wps", false))
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create WiFi security profile on %s:\n\n", r.Name())
		for k, v := range body {
			if k == "passphrase" {
				v = "***"
			}
			fmt.Fprintf(&sb, "  %-25s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_wifi_security again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/interface/wifi/security", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create WiFi security profile: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ WiFi security profile %q created (ID: %s) on %s",
		name, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveWiFiSecurity(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/interface/wifi/security")
	if err != nil {
		return ToolResult{}, err
	}
	var profiles []map[string]string
	if err := json.Unmarshal(raw, &profiles); err != nil {
		return ToolResult{}, fmt.Errorf("parse security profiles: %w", err)
	}
	var target map[string]string
	for _, p := range profiles {
		if p["name"] == name {
			target = p
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("security profile %q not found", name)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE WiFi security profile %q (ID: %s) on %s\n",
			name, target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_wifi_security again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/interface/wifi/security/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete security profile: %w", err)
	}
	return textResult(fmt.Sprintf("✓ WiFi security profile %q deleted on %s", name, r.Name())), nil
}

// ─── WiFi Datapaths ───────────────────────────────────────────────────────────

func (s *Server) toolAddWiFiDatapath(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{"name": name}
	if v := strOpt(args, "bridge", ""); v != "" {
		body["bridge"] = v
	}
	if v := strOpt(args, "vlan_id", ""); v != "" {
		body["vlan-id"] = v
	}
	if _, ok := args["client_isolation"]; ok {
		body["client-to-client-forwarding"] = boolToYesNo(!boolOpt(args, "client_isolation", false))
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create WiFi datapath on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-35s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_wifi_datapath again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/interface/wifi/datapath", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create WiFi datapath: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ WiFi datapath %q created (ID: %s) on %s",
		name, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveWiFiDatapath(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/interface/wifi/datapath")
	if err != nil {
		return ToolResult{}, err
	}
	var datapaths []map[string]string
	if err := json.Unmarshal(raw, &datapaths); err != nil {
		return ToolResult{}, fmt.Errorf("parse datapaths: %w", err)
	}
	var target map[string]string
	for _, d := range datapaths {
		if d["name"] == name {
			target = d
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("datapath %q not found", name)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE WiFi datapath %q (bridge: %s, ID: %s) on %s\n",
			name, target["bridge"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_wifi_datapath again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/interface/wifi/datapath/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete datapath: %w", err)
	}
	return textResult(fmt.Sprintf("✓ WiFi datapath %q deleted on %s", name, r.Name())), nil
}

// ─── Virtual WiFi Interfaces ──────────────────────────────────────────────────

func (s *Server) toolAddWiFiInterface(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	master, err := strArg(args, "master_interface")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{
		"name":             name,
		"master-interface": master,
	}
	if v := strOpt(args, "configuration", ""); v != "" {
		body["configuration"] = v
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create virtual WiFi interface on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-20s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_wifi_interface again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/interface/wifi", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create WiFi interface: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ Virtual WiFi interface %q created on %s (ID: %s) on %s",
		name, master, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveWiFiInterface(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/interface/wifi")
	if err != nil {
		return ToolResult{}, err
	}
	var ifaces []map[string]string
	if err := json.Unmarshal(raw, &ifaces); err != nil {
		return ToolResult{}, fmt.Errorf("parse WiFi interfaces: %w", err)
	}
	var target map[string]string
	for _, i := range ifaces {
		if i["name"] == name {
			target = i
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("WiFi interface %q not found", name)
	}
	if target["master-interface"] == "" {
		return ToolResult{}, fmt.Errorf("%q is a physical interface — only virtual (slave) interfaces can be removed", name)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE virtual WiFi interface %q (master: %s, ID: %s) on %s\n",
			name, target["master-interface"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_wifi_interface again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/interface/wifi/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete WiFi interface: %w", err)
	}
	return textResult(fmt.Sprintf("✓ Virtual WiFi interface %q deleted on %s", name, r.Name())), nil
}

// ─── CAPsMAN Provisioning ─────────────────────────────────────────────────────

func (s *Server) toolAddWiFiProvisioning(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	masterCfg, err := strArg(args, "master_configuration")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	action := strOpt(args, "action", "create-dynamic-enabled")
	body := map[string]string{
		"master-configuration": masterCfg,
		"action":               action,
	}
	if v := strOpt(args, "radio_mac", ""); v != "" {
		body["radio-mac"] = v
	}
	if v := strOpt(args, "supported_bands", ""); v != "" {
		body["supported-bands"] = v
	}
	if v := strOpt(args, "slave_configuration", ""); v != "" {
		body["slave-configurations"] = v
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create CAPsMAN provisioning rule on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-25s  %s\n", k, v)
		}
		if body["radio-mac"] == "" {
			sb.WriteString("\nNote: no radio-mac set — this rule will match ALL radios.\n")
		}
		sb.WriteString("\nTo apply: call add_wifi_provisioning again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/interface/wifi/provisioning", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create provisioning rule: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ CAPsMAN provisioning rule created (config: %s, ID: %s) on %s",
		masterCfg, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveWiFiProvisioning(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	ruleID, err := strArg(args, "id")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/interface/wifi/provisioning/" + ruleID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("fetch provisioning rule %s: %w", ruleID, err)
	}
	var rule map[string]string
	_ = json.Unmarshal(raw, &rule)

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE provisioning rule %s:\n", ruleID)
		fmt.Fprintf(&sb, "  master-config: %s\n", rule["master-configuration"])
		fmt.Fprintf(&sb, "  radio-mac:     %s\n", rule["radio-mac"])
		fmt.Fprintf(&sb, "  action:        %s\n", rule["action"])
		fmt.Fprintf(&sb, "\nRouter: %s\n", r.Name())
		sb.WriteString("\nTo apply: call remove_wifi_provisioning again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/interface/wifi/provisioning/" + ruleID); err != nil {
		return ToolResult{}, fmt.Errorf("delete provisioning rule: %w", err)
	}
	return textResult(fmt.Sprintf("✓ CAPsMAN provisioning rule %s deleted on %s", ruleID, r.Name())), nil
}

func (s *Server) toolProvisionCAPs(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	_, err = r.Post("/interface/wifi/capsman/provision", map[string]string{})
	if err != nil {
		return ToolResult{}, fmt.Errorf("force CAP provisioning: %w", err)
	}
	return textResult(fmt.Sprintf("✓ CAPsMAN re-provisioning triggered on %s — APs will reconnect and re-apply their configurations.", r.Name())), nil
}

// boolToYesNo converts a boolean to RouterOS "yes"/"no".
func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
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
