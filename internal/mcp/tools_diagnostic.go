package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *Server) toolListRouters(_ context.Context, _ map[string]any) (ToolResult, error) {
	var sb strings.Builder
	sb.WriteString("Configured MikroTik Routers\n")
	sb.WriteString(strings.Repeat("═", 40) + "\n\n")

	for _, r := range s.routers {
		err := r.Ping()
		status := "✓ reachable"
		if err != nil {
			status = "✗ unreachable — " + err.Error()
		}
		fmt.Fprintf(&sb, "• %s — %s\n", r.Name(), status)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetSystemInfo(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/system/resource")
	if err != nil {
		return ToolResult{}, err
	}

	var res map[string]string
	if err := json.Unmarshal(raw, &res); err != nil {
		return ToolResult{}, fmt.Errorf("parse system/resource: %w", err)
	}

	freeMemMB := parseIntField(res["free-memory"]) / 1024 / 1024
	totalMemMB := parseIntField(res["total-memory"]) / 1024 / 1024
	freeHDDMB := parseIntField(res["free-hdd-space"]) / 1024 / 1024
	totalHDDMB := parseIntField(res["total-hdd-space"]) / 1024 / 1024

	out := fmt.Sprintf(`System Info — %s
%s
Board:      %s
Version:    RouterOS %s (built %s)
CPU:        %s × %s cores @ %s MHz — load: %s%%
Memory:     %d / %d MB used
Storage:    %d MB free / %d MB total
Uptime:     %s
`,
		r.Name(), strings.Repeat("═", 40),
		res["board-name"],
		res["version"], res["build-time"],
		res["cpu"], res["cpu-count"], res["cpu-frequency"], res["cpu-load"],
		totalMemMB-freeMemMB, totalMemMB,
		freeHDDMB, totalHDDMB,
		res["uptime"],
	)

	return textResult(out), nil
}

func (s *Server) toolGetInterfaces(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/interface")
	if err != nil {
		return ToolResult{}, err
	}

	var ifaces []map[string]string
	if err := json.Unmarshal(raw, &ifaces); err != nil {
		return ToolResult{}, fmt.Errorf("parse interfaces: %w", err)
	}

	typeFilter := strOpt(args, "type", "")
	var sb strings.Builder
	fmt.Fprintf(&sb, "Interfaces — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))

	count := 0
	for _, iface := range ifaces {
		if typeFilter != "" && iface["type"] != typeFilter {
			continue
		}
		running := "down"
		if iface["running"] == "true" {
			running = "up"
		}
		disabled := ""
		if iface["disabled"] == "true" {
			disabled = " [DISABLED]"
		}
		comment := ""
		if iface["comment"] != "" {
			comment = " (" + iface["comment"] + ")"
		}

		rxMB := parseIntField(iface["rx-byte"]) / 1024 / 1024
		txMB := parseIntField(iface["tx-byte"]) / 1024 / 1024

		fmt.Fprintf(&sb, "%-20s  type=%-8s  %s%s%s\n",
			iface["name"], iface["type"], running, disabled, comment)
		fmt.Fprintf(&sb, "  MAC: %-17s  rx: %d MB  tx: %d MB  drops: %s\n",
			iface["mac-address"], rxMB, txMB, iface["tx-queue-drop"])
		count++
	}

	fmt.Fprintf(&sb, "\nTotal: %d interfaces", count)
	return textResult(sb.String()), nil
}

func (s *Server) toolGetIPAddresses(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/ip/address")
	if err != nil {
		return ToolResult{}, err
	}

	var addrs []map[string]string
	if err := json.Unmarshal(raw, &addrs); err != nil {
		return ToolResult{}, fmt.Errorf("parse ip/address: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "IP Addresses — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))

	for _, a := range addrs {
		dyn := ""
		if a["dynamic"] == "true" {
			dyn = " [dynamic]"
		}
		inv := ""
		if a["invalid"] == "true" {
			inv = " [invalid]"
		}
		comment := ""
		if a["comment"] != "" {
			comment = "  # " + a["comment"]
		}
		fmt.Fprintf(&sb, "%-22s  iface=%-15s%s%s%s\n",
			a["address"], a["interface"], dyn, inv, comment)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetRoutingTable(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/ip/route")
	if err != nil {
		return ToolResult{}, err
	}

	var routes []map[string]string
	if err := json.Unmarshal(raw, &routes); err != nil {
		return ToolResult{}, fmt.Errorf("parse routes: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Routing Table — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))
	fmt.Fprintf(&sb, "%-20s  %-20s  %-15s  dist  active\n", "Destination", "Gateway", "Interface")
	fmt.Fprintf(&sb, "%s\n", strings.Repeat("-", 75))

	for _, rt := range routes {
		active := "no"
		if rt["active"] == "true" {
			active = "yes"
		}
		dis := ""
		if rt["disabled"] == "true" {
			active = "disabled"
		}
		_ = dis
		fmt.Fprintf(&sb, "%-20s  %-20s  %-15s  %-4s  %s\n",
			rt["dst-address"], rt["gateway"], rt["gateway-status"],
			rt["distance"], active)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetARPTable(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/ip/arp")
	if err != nil {
		return ToolResult{}, err
	}

	var entries []map[string]string
	if err := json.Unmarshal(raw, &entries); err != nil {
		return ToolResult{}, fmt.Errorf("parse arp: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "ARP Table — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))
	fmt.Fprintf(&sb, "%-16s  %-19s  %-12s  status\n", "IP", "MAC", "Interface")
	fmt.Fprintf(&sb, "%s\n", strings.Repeat("-", 65))

	for _, e := range entries {
		if e["dynamic"] != "true" && e["dhcp"] != "true" && e["complete"] != "true" {
			continue
		}
		fmt.Fprintf(&sb, "%-16s  %-19s  %-12s  %s\n",
			e["address"], e["mac-address"], e["interface"], e["status"])
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetDHCPLeases(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/ip/dhcp-server/lease")
	if err != nil {
		return ToolResult{}, err
	}

	var leases []map[string]string
	if err := json.Unmarshal(raw, &leases); err != nil {
		return ToolResult{}, fmt.Errorf("parse leases: %w", err)
	}

	showAll := boolOpt(args, "all", false)

	var sb strings.Builder
	fmt.Fprintf(&sb, "DHCP Leases — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))

	count := 0
	for _, l := range leases {
		if !showAll && l["status"] != "bound" {
			continue
		}
		hostname := l["host-name"]
		if hostname == "" {
			hostname = "(unknown)"
		}
		fmt.Fprintf(&sb, "%-16s  %-19s  %-20s  expires: %-15s  status: %s\n",
			l["address"], l["mac-address"], hostname, l["expires-after"], l["status"])
		count++
	}

	if count == 0 {
		sb.WriteString("No active leases found.")
	} else {
		fmt.Fprintf(&sb, "\nTotal: %d leases", count)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetLogs(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/log")
	if err != nil {
		return ToolResult{}, err
	}

	var entries []map[string]string
	if err := json.Unmarshal(raw, &entries); err != nil {
		return ToolResult{}, fmt.Errorf("parse logs: %w", err)
	}

	topicsFilter := strOpt(args, "topics", "")
	count := intOpt(args, "count", 50)

	var sb strings.Builder
	fmt.Fprintf(&sb, "System Logs — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))

	shown := 0
	// Logs are returned oldest-first; show newest last up to count
	start := 0
	if len(entries) > count {
		start = len(entries) - count
	}

	for _, e := range entries[start:] {
		topics := e["topics"]
		if topicsFilter != "" && !strings.Contains(topics, topicsFilter) {
			continue
		}
		fmt.Fprintf(&sb, "[%s] %-25s  %s\n", e["time"], topics, e["message"])
		shown++
	}

	if shown == 0 {
		sb.WriteString("No log entries found.")
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolPingFromRouter(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	address, err := strArg(args, "address")
	if err != nil {
		return ToolResult{}, err
	}

	body := map[string]string{
		"address": address,
		"count":   fmt.Sprintf("%d", intOpt(args, "count", 4)),
	}
	if iface := strOpt(args, "interface", ""); iface != "" {
		body["interface"] = iface
	}

	raw, err := r.Post("/ping", body)
	if err != nil {
		return ToolResult{}, fmt.Errorf("ping from %s to %s: %w", r.Name(), address, err)
	}

	var results []map[string]string
	if err := json.Unmarshal(raw, &results); err != nil {
		return textResult(fmt.Sprintf("Ping result from %s:\n%s", r.Name(), string(raw))), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Ping from %s → %s\n%s\n\n", r.Name(), address, strings.Repeat("═", 40))

	// RouterOS REST API returns incremental summaries per packet.
	// Print per-packet rows first, then the final summary only.
	var lastSummary map[string]string
	for _, res := range results {
		if res["sent"] != "" {
			lastSummary = res
		} else {
			status := "reply"
			if res["status"] != "" && res["status"] != "echo-reply" {
				status = res["status"]
			}
			fmt.Fprintf(&sb, "seq=%-3s  host=%-16s  time=%-10s  ttl=%-4s  %s\n",
				res["seq"], res["host"], res["time"], res["ttl"], status)
		}
	}
	if lastSummary != nil {
		fmt.Fprintf(&sb, "\nResult: sent=%s  received=%s  loss=%s  min=%s  avg=%s  max=%s\n",
			lastSummary["sent"], lastSummary["received"], lastSummary["packet-loss"],
			lastSummary["min-rtt"], lastSummary["avg-rtt"], lastSummary["max-rtt"])
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolTracerouteFromRouter(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	address, err := strArg(args, "address")
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Post("/tool/traceroute", map[string]string{
		"address": address,
		"count":   "3",
	})
	if err != nil {
		return ToolResult{}, fmt.Errorf("traceroute from %s to %s: %w", r.Name(), address, err)
	}

	var hops []map[string]string
	if err := json.Unmarshal(raw, &hops); err != nil {
		return textResult(fmt.Sprintf("Traceroute from %s → %s:\n%s", r.Name(), address, string(raw))), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Traceroute from %s → %s\n%s\n\n", r.Name(), address, strings.Repeat("═", 40))

	for _, hop := range hops {
		loss := hop["loss"]
		if loss != "" {
			loss = " loss=" + loss
		}
		fmt.Fprintf(&sb, "%-3s  %-20s  %-20s  avg=%-8s%s\n",
			hop["#"], hop["address"], hop["host"], hop["avg"], loss)
	}

	return textResult(sb.String()), nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func parseIntField(s string) int64 {
	var v int64
	fmt.Sscanf(s, "%d", &v)
	return v
}

// confirmationPreview builds a standard dry-run preview block.
func confirmationPreview(action, detail, toolName string, params map[string]string) string {
	var sb strings.Builder
	sb.WriteString("⚠️  DRY RUN — no changes made\n")
	sb.WriteString(strings.Repeat("═", 40) + "\n\n")
	fmt.Fprintf(&sb, "Action:  %s\n", action)
	fmt.Fprintf(&sb, "Detail:  %s\n", detail)
	if len(params) > 0 {
		sb.WriteString("\nParameters:\n")
		for k, v := range params {
			fmt.Fprintf(&sb, "  %-20s = %s\n", k, v)
		}
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "To apply: call %s again with dry_run=false\n", toolName)
	_ = time.Now() // keep import
	return sb.String()
}
