package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ─── Bridges ──────────────────────────────────────────────────────────────────

func (s *Server) toolAddBridge(_ context.Context, args map[string]any) (ToolResult, error) {
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
	if v := strOpt(args, "protocol_mode", ""); v != "" {
		body["protocol-mode"] = v
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}
	if _, ok := args["vlan_filtering"]; ok {
		body["vlan-filtering"] = boolToYesNo(boolOpt(args, "vlan_filtering", false))
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create bridge on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-20s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_bridge again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/interface/bridge", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create bridge: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ Bridge %q created (ID: %s) on %s", name, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveBridge(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/interface/bridge")
	if err != nil {
		return ToolResult{}, err
	}
	var bridges []map[string]string
	if err := json.Unmarshal(raw, &bridges); err != nil {
		return ToolResult{}, fmt.Errorf("parse bridges: %w", err)
	}
	var target map[string]string
	for _, b := range bridges {
		if b["name"] == name {
			target = b
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("bridge %q not found", name)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE bridge %q (ID: %s) on %s\n", name, target[".id"], r.Name())
		sb.WriteString("Warning: also removes all bridge ports assigned to this bridge.\n")
		sb.WriteString("\nTo apply: call remove_bridge again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/interface/bridge/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete bridge: %w", err)
	}
	return textResult(fmt.Sprintf("✓ Bridge %q deleted on %s", name, r.Name())), nil
}

// ─── Bridge Ports ─────────────────────────────────────────────────────────────

func (s *Server) toolAddBridgePort(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	bridge, err := strArg(args, "bridge")
	if err != nil {
		return ToolResult{}, err
	}
	iface, err := strArg(args, "interface")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{
		"bridge":    bridge,
		"interface": iface,
	}
	if v := strOpt(args, "pvid", ""); v != "" {
		body["pvid"] = v
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would add %q to bridge %q on %s:\n\n", iface, bridge, r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-15s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_bridge_port again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/interface/bridge/port", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("add bridge port: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ Interface %q added to bridge %q (port ID: %s) on %s",
		iface, bridge, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveBridgePort(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	iface, err := strArg(args, "interface")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)
	bridgeName := strOpt(args, "bridge", "")

	raw, err := r.Get("/interface/bridge/port")
	if err != nil {
		return ToolResult{}, err
	}
	var ports []map[string]string
	if err := json.Unmarshal(raw, &ports); err != nil {
		return ToolResult{}, fmt.Errorf("parse bridge ports: %w", err)
	}

	var target map[string]string
	for _, p := range ports {
		if p["interface"] == iface && (bridgeName == "" || p["bridge"] == bridgeName) {
			target = p
			break
		}
	}
	if target == nil {
		if bridgeName != "" {
			return ToolResult{}, fmt.Errorf("port for %q on bridge %q not found", iface, bridgeName)
		}
		return ToolResult{}, fmt.Errorf("bridge port for interface %q not found", iface)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would remove %q from bridge %q (port ID: %s) on %s\n",
			target["interface"], target["bridge"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_bridge_port again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/interface/bridge/port/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("remove bridge port: %w", err)
	}
	return textResult(fmt.Sprintf("✓ Interface %q removed from bridge %q on %s",
		iface, target["bridge"], r.Name())), nil
}

// ─── VLANs ────────────────────────────────────────────────────────────────────

func (s *Server) toolAddVLAN(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	iface, err := strArg(args, "interface")
	if err != nil {
		return ToolResult{}, err
	}
	vlanID, err := strArg(args, "vlan_id")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	name := strOpt(args, "name", "vlan"+vlanID)
	body := map[string]string{
		"name":      name,
		"interface": iface,
		"vlan-id":   vlanID,
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create VLAN on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-15s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_vlan again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/interface/vlan", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create VLAN: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ VLAN %s (%s on %s) created (ID: %s) on %s",
		vlanID, name, iface, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveVLAN(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	name, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	raw, err := r.Get("/interface/vlan")
	if err != nil {
		return ToolResult{}, err
	}
	var vlans []map[string]string
	if err := json.Unmarshal(raw, &vlans); err != nil {
		return ToolResult{}, fmt.Errorf("parse VLANs: %w", err)
	}
	var target map[string]string
	for _, v := range vlans {
		if v["name"] == name {
			target = v
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("VLAN %q not found", name)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE VLAN %q (vlan-id: %s, parent: %s, ID: %s) on %s\n",
			name, target["vlan-id"], target["interface"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_vlan again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/interface/vlan/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete VLAN: %w", err)
	}
	return textResult(fmt.Sprintf("✓ VLAN %q deleted on %s", name, r.Name())), nil
}
