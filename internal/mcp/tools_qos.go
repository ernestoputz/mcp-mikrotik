package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *Server) toolGetQueueStats(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Queue Statistics — %s\n%s\n", r.Name(), strings.Repeat("═", 40))

	// Queue Tree
	rawTree, err := r.Get("/queue/tree")
	if err == nil {
		var tree []map[string]string
		if json.Unmarshal(rawTree, &tree) == nil && len(tree) > 0 {
			sb.WriteString("\n── Queue Tree ──\n")
			for _, q := range tree {
				maxLimit := formatBandwidth(q["max-limit"])
				rate := formatBandwidth(q["rate"])
				fmt.Fprintf(&sb, "  %-22s  parent: %-12s  limit: %-12s  current: %-12s  bytes: %s  dropped: %s\n",
					q["name"], q["parent"], maxLimit, rate,
					formatBytes(q["bytes"]), q["dropped"])
			}
		}
	}

	// Queue Simple
	rawSimple, err := r.Get("/queue/simple")
	if err == nil {
		var simple []map[string]string
		if json.Unmarshal(rawSimple, &simple) == nil && len(simple) > 0 {
			sb.WriteString("\n── Simple Queues ──\n")
			for _, q := range simple {
				fmt.Fprintf(&sb, "  %-22s  target: %-18s  max-limit: %-12s  burst: %s\n",
					q["name"], q["target"], q["max-limit"], q["burst-limit"])
			}
		}
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolGetDNSEntries(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	raw, err := r.Get("/ip/dns/static")
	if err != nil {
		return ToolResult{}, err
	}

	var entries []map[string]string
	if err := json.Unmarshal(raw, &entries); err != nil {
		return ToolResult{}, fmt.Errorf("parse dns entries: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Static DNS Entries — %s\n%s\n\n", r.Name(), strings.Repeat("═", 40))

	if len(entries) == 0 {
		sb.WriteString("No static DNS entries configured.\n")
		return textResult(sb.String()), nil
	}

	for _, e := range entries {
		disabled := ""
		if e["disabled"] == "true" {
			disabled = " [DISABLED]"
		}
		entryType := e["type"]
		if entryType == "" {
			entryType = "A"
		}
		fmt.Fprintf(&sb, "  %-30s  %-6s  → %-20s  ttl: %-8s%s\n",
			e["name"], entryType, e["address"]+e["cname"]+e["text"], e["ttl"], disabled)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolSetQueueLimit(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	queueName, err := strArg(args, "queue_name")
	if err != nil {
		return ToolResult{}, err
	}
	maxLimit, err := strArg(args, "max_limit")
	if err != nil {
		return ToolResult{}, err
	}

	dryRun := boolOpt(args, "dry_run", true)

	// Find the queue (try tree first, then simple)
	queueID, queueType, currentLimit, findErr := findQueue(r, queueName)
	if findErr != nil {
		return ToolResult{}, findErr
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would adjust queue limit on %s:\n\n", r.Name())
		fmt.Fprintf(&sb, "  Queue:    %s (%s, ID: %s)\n", queueName, queueType, queueID)
		fmt.Fprintf(&sb, "  Current:  %s\n", formatBandwidth(currentLimit))
		fmt.Fprintf(&sb, "  New:      %s\n", maxLimit)
		fmt.Fprintf(&sb, "\nTo apply: call set_queue_limit again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	path := "/queue/" + queueType + "/" + queueID
	_, err = r.Patch(path, map[string]string{"max-limit": maxLimit})
	if err != nil {
		return ToolResult{}, fmt.Errorf("update queue %s: %w", queueName, err)
	}

	return textResult(fmt.Sprintf("✓ Queue %q limit updated: %s → %s on %s",
		queueName, formatBandwidth(currentLimit), maxLimit, r.Name())), nil
}

func (s *Server) toolRestartInterface(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	ifaceName, err := strArg(args, "interface")
	if err != nil {
		return ToolResult{}, err
	}

	dryRun := boolOpt(args, "dry_run", true)

	// Fetch interface to confirm it exists and get its ID
	raw, err := r.Get("/interface")
	if err != nil {
		return ToolResult{}, err
	}

	var ifaces []map[string]string
	if err := json.Unmarshal(raw, &ifaces); err != nil {
		return ToolResult{}, fmt.Errorf("parse interfaces: %w", err)
	}

	var target map[string]string
	for _, iface := range ifaces {
		if iface["name"] == ifaceName {
			target = iface
			break
		}
	}
	if target == nil {
		return ToolResult{}, fmt.Errorf("interface %q not found on %s", ifaceName, r.Name())
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would restart interface on %s:\n\n", r.Name())
		fmt.Fprintf(&sb, "  Interface: %s (type: %s, ID: %s)\n", ifaceName, target["type"], target[".id"])
		fmt.Fprintf(&sb, "  Status:    %s\n", map[bool]string{true: "running", false: "down"}[target["running"] == "true"])
		fmt.Fprintf(&sb, "\nSequence: disable → 2s pause → enable\n")
		fmt.Fprintf(&sb, "Warning: traffic on this interface will be briefly interrupted.\n")
		fmt.Fprintf(&sb, "\nTo apply: call restart_interface again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	id := target[".id"]
	path := "/interface/" + id

	if _, err := r.Patch(path, map[string]string{"disabled": "true"}); err != nil {
		return ToolResult{}, fmt.Errorf("disable interface %s: %w", ifaceName, err)
	}

	time.Sleep(2 * time.Second)

	if _, err := r.Patch(path, map[string]string{"disabled": "false"}); err != nil {
		return ToolResult{}, fmt.Errorf("re-enable interface %s (it is currently disabled!): %w", ifaceName, err)
	}

	return textResult(fmt.Sprintf("✓ Interface %s restarted on %s — disable→2s→enable complete", ifaceName, r.Name())), nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func findQueue(r interface {
	Get(string) (json.RawMessage, error)
}, name string) (id, qtype, maxLimit string, err error) {
	for _, qt := range []string{"tree", "simple"} {
		raw, e := r.Get("/queue/" + qt)
		if e != nil {
			continue
		}
		var queues []map[string]string
		if json.Unmarshal(raw, &queues) != nil {
			continue
		}
		for _, q := range queues {
			if q["name"] == name {
				return q[".id"], qt, q["max-limit"], nil
			}
		}
	}
	return "", "", "", fmt.Errorf("queue %q not found in tree or simple queues", name)
}

func formatBandwidth(bps string) string {
	var v int64
	fmt.Sscanf(bps, "%d", &v)
	switch {
	case v == 0:
		return "unlimited"
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.0f G", float64(v)/1e9)
	case v >= 1_000_000:
		return fmt.Sprintf("%.0f M", float64(v)/1e6)
	case v >= 1_000:
		return fmt.Sprintf("%.0f K", float64(v)/1e3)
	default:
		return fmt.Sprintf("%d bps", v)
	}
}

func formatBytes(s string) string {
	var v int64
	fmt.Sscanf(s, "%d", &v)
	switch {
	case v >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(v)/(1<<30))
	case v >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(v)/(1<<20))
	case v >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(v)/(1<<10))
	default:
		return fmt.Sprintf("%d B", v)
	}
}
