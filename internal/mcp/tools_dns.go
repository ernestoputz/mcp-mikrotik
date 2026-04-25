package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Server) toolAddDNSEntry(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	hostname, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	address, err := strArg(args, "address")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	body := map[string]string{"name": hostname, "address": address}
	if v := strOpt(args, "ttl", ""); v != "" {
		body["ttl"] = v
	}
	if v := strOpt(args, "comment", ""); v != "" {
		body["comment"] = v
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create static DNS entry on %s:\n\n", r.Name())
		for k, v := range body {
			fmt.Fprintf(&sb, "  %-15s  %s\n", k, v)
		}
		sb.WriteString("\nTo apply: call add_dns_entry again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	resp, err := r.Put("/ip/dns/static", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create DNS entry: %w", err)
	}
	var created map[string]string
	_ = json.Unmarshal(resp, &created)

	return textResult(fmt.Sprintf("✓ DNS entry %s → %s created (ID: %s) on %s",
		hostname, address, created[".id"], r.Name())), nil
}

func (s *Server) toolRemoveDNSEntry(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	hostname := strOpt(args, "name", "")
	address := strOpt(args, "address", "")
	if hostname == "" && address == "" {
		return ToolResult{}, fmt.Errorf("provide either 'name' or 'address' to identify the DNS entry")
	}

	raw, err := r.Get("/ip/dns/static")
	if err != nil {
		return ToolResult{}, err
	}
	var entries []map[string]string
	if err := json.Unmarshal(raw, &entries); err != nil {
		return ToolResult{}, fmt.Errorf("parse DNS entries: %w", err)
	}
	var target map[string]string
	for _, e := range entries {
		if (hostname != "" && e["name"] == hostname) || (address != "" && e["address"] == address) {
			target = e
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("DNS entry not found")
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE DNS entry: %s → %s (ID: %s) on %s\n",
			target["name"], target["address"], target[".id"], r.Name())
		sb.WriteString("\nTo apply: call remove_dns_entry again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/ip/dns/static/" + target[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("delete DNS entry: %w", err)
	}
	return textResult(fmt.Sprintf("✓ DNS entry %s → %s deleted on %s",
		target["name"], target["address"], r.Name())), nil
}
