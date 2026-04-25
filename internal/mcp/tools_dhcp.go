package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ─── IP Addresses ─────────────────────────────────────────────────────────────

func (s *Server) toolAddIPAddress(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	address, err := strArg(args, "address")
	if err != nil {
		return ToolResult{}, err
	}
	iface, err := strArg(args, "interface")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{"address": address, "interface": iface}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would add IP address on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-15s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_ip_address again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/ip/address", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("add IP address: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ IP address %s added to %s (ID: %s) on %s",
		address, iface, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveIPAddress(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	addrID := strOpt(args, "id", "")
	address := strOpt(args, "address", "")
	iface := strOpt(args, "interface", "")
	if addrID == "" && address == "" {
		return ToolResult{}, fmt.Errorf("provide either 'id' or 'address' to identify the IP address")
	}

	raw, err := r.Get("/ip/address")
	if err != nil {
		return ToolResult{}, err
	}
	var addrs []map[string]string
	if err := json.Unmarshal(raw, &addrs); err != nil {
		return ToolResult{}, fmt.Errorf("parse IP addresses: %w", err)
	}

	var target map[string]string
	for _, a := range addrs {
		if addrID != "" && a[".id"] == addrID {
			target = a
			break
		}
		if address != "" && (a["address"] == address || strings.HasPrefix(a["address"], address+"/")) {
			if iface == "" || a["interface"] == iface {
				target = a
				break
			}
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("IP address not found")
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would remove IP address %s from %s (ID: %s) on %s\n",
			target["address"], target["interface"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_ip_address again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/ip/address/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("remove IP address: %w", err)
	}
	return textResult(fmt.Sprintf("✓ IP address %s removed from %s on %s",
		target["address"], target["interface"], r.Name())), nil
}

// ─── IP Pools ─────────────────────────────────────────────────────────────────

func (s *Server) toolAddIPPool(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	ranges, err := strArg(args, "ranges")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{"name": name, "ranges": ranges}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create IP pool on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-15s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_ip_pool again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/ip/pool", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create IP pool: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ IP pool %q created (ranges: %s, ID: %s) on %s",
		name, ranges, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveIPPool(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/ip/pool")
	if err != nil {
		return ToolResult{}, err
	}
	var pools []map[string]string
	if err := json.Unmarshal(raw, &pools); err != nil {
		return ToolResult{}, fmt.Errorf("parse IP pools: %w", err)
	}
	var target map[string]string
	for _, p := range pools {
		if p["name"] == name {
			target = p
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("IP pool %q not found", name)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE IP pool %q (ranges: %s, ID: %s) on %s\n",
			name, target["ranges"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_ip_pool again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/ip/pool/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete IP pool: %w", err)
	}
	return textResult(fmt.Sprintf("✓ IP pool %q deleted on %s", name, r.Name())), nil
}

// ─── DHCP Server ──────────────────────────────────────────────────────────────

func (s *Server) toolAddDHCPServer(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	iface, err := strArg(args, "interface")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{"name": name, "interface": iface}
	if v := strOpt(args, "address_pool", ""); v != "" {
		body["address-pool"] = v
	}
	if v := strOpt(args, "lease_time", ""); v != "" {
		body["lease-time"] = v
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create DHCP server on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-20s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_dhcp_server again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/ip/dhcp-server", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create DHCP server: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ DHCP server %q created on %s (ID: %s) on %s",
		name, iface, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveDHCPServer(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/ip/dhcp-server")
	if err != nil {
		return ToolResult{}, err
	}
	var servers []map[string]string
	if err := json.Unmarshal(raw, &servers); err != nil {
		return ToolResult{}, fmt.Errorf("parse DHCP servers: %w", err)
	}
	var target map[string]string
	for _, srv := range servers {
		if srv["name"] == name {
			target = srv
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("DHCP server %q not found", name)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE DHCP server %q (interface: %s, ID: %s) on %s\n",
			name, target["interface"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_dhcp_server again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/ip/dhcp-server/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete DHCP server: %w", err)
	}
	return textResult(fmt.Sprintf("✓ DHCP server %q deleted on %s", name, r.Name())), nil
}

// ─── DHCP Networks ────────────────────────────────────────────────────────────

func (s *Server) toolAddDHCPNetwork(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	address, err := strArg(args, "address")
	if err != nil {
		return ToolResult{}, err
	}
	gateway, err := strArg(args, "gateway")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{"address": address, "gateway": gateway}
	if v := strOpt(args, "dns_server", ""); v != "" {
		body["dns-server"] = v
	}
	if v := strOpt(args, "ntp_server", ""); v != "" {
		body["ntp-server"] = v
	}
	if v := strOpt(args, "domain", ""); v != "" {
		body["domain"] = v
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create DHCP network on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-15s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_dhcp_network again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/ip/dhcp-server/network", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create DHCP network: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ DHCP network %s created (gateway: %s, ID: %s) on %s",
		address, gateway, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveDHCPNetwork(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	address, err := strArg(args, "address")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/ip/dhcp-server/network")
	if err != nil {
		return ToolResult{}, err
	}
	var networks []map[string]string
	if err := json.Unmarshal(raw, &networks); err != nil {
		return ToolResult{}, fmt.Errorf("parse DHCP networks: %w", err)
	}
	var target map[string]string
	for _, n := range networks {
		if n["address"] == address {
			target = n
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("DHCP network %q not found", address)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE DHCP network %s (gateway: %s, ID: %s) on %s\n",
			address, target["gateway"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_dhcp_network again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/ip/dhcp-server/network/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete DHCP network: %w", err)
	}
	return textResult(fmt.Sprintf("✓ DHCP network %s deleted on %s", address, r.Name())), nil
}

// ─── DHCP Leases ──────────────────────────────────────────────────────────────

func (s *Server) toolAddDHCPLease(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	address, err := strArg(args, "address")
	if err != nil {
		return ToolResult{}, err
	}
	mac, err := strArg(args, "mac_address")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{"address": address, "mac-address": mac}
	if v := strOpt(args, "server", ""); v != "" {
		body["server"] = v
	}
	if v := strOpt(args, "hostname", ""); v != "" {
		body["client-id"] = v
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create static DHCP lease on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-20s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_dhcp_lease again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/ip/dhcp-server/lease", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create DHCP lease: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ Static DHCP lease created: %s → %s (ID: %s) on %s",
		mac, address, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveDHCPLease(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	mac := strOpt(args, "mac_address", "")
	address := strOpt(args, "address", "")
	if mac == "" && address == "" {
		return ToolResult{}, fmt.Errorf("provide either 'mac_address' or 'address' to identify the lease")
	}

	raw, err := r.Get("/ip/dhcp-server/lease")
	if err != nil {
		return ToolResult{}, err
	}
	var leases []map[string]string
	if err := json.Unmarshal(raw, &leases); err != nil {
		return ToolResult{}, fmt.Errorf("parse DHCP leases: %w", err)
	}
	var target map[string]string
	for _, l := range leases {
		if (mac != "" && l["mac-address"] == mac) || (address != "" && l["address"] == address) {
			target = l
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("DHCP lease not found")
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE DHCP lease: %s → %s (ID: %s) on %s\n",
			target["mac-address"], target["address"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_dhcp_lease again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/ip/dhcp-server/lease/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete DHCP lease: %w", err)
	}
	return textResult(fmt.Sprintf("✓ DHCP lease %s → %s deleted on %s",
		target["mac-address"], target["address"], r.Name())), nil
}
