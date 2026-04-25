package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Server) toolGetFirewallRules(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	tableFilter := strOpt(args, "table", "")
	chainFilter := strOpt(args, "chain", "")

	tables := []string{"filter", "nat", "mangle"}
	if tableFilter != "" {
		tables = []string{tableFilter}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Firewall Rules — %s\n%s\n", r.Name(), strings.Repeat("═", 40))

	for _, table := range tables {
		raw, err := r.Get("/ip/firewall/" + table)
		if err != nil {
			fmt.Fprintf(&sb, "\n[%s] error: %v\n", table, err)
			continue
		}

		var rules []map[string]string
		if err := json.Unmarshal(raw, &rules); err != nil {
			continue
		}

		fmt.Fprintf(&sb, "\n── Table: %s (%d rules) ──\n", strings.ToUpper(table), len(rules))

		for _, rule := range rules {
			if chainFilter != "" && rule["chain"] != chainFilter {
				continue
			}

			disabled := ""
			if rule["disabled"] == "true" {
				disabled = " [DISABLED]"
			}
			dynamic := ""
			if rule["dynamic"] == "true" {
				dynamic = " [dynamic]"
			}
			comment := ""
			if rule["comment"] != "" {
				comment = "  # " + rule["comment"]
			}

			// Build match summary
			var match []string
			if rule["src-address"] != "" {
				match = append(match, "src="+rule["src-address"])
			}
			if rule["dst-address"] != "" {
				match = append(match, "dst="+rule["dst-address"])
			}
			if rule["src-address-list"] != "" {
				match = append(match, "src-list="+rule["src-address-list"])
			}
			if rule["dst-address-list"] != "" {
				match = append(match, "dst-list="+rule["dst-address-list"])
			}
			if rule["protocol"] != "" {
				match = append(match, "proto="+rule["protocol"])
			}
			if rule["dst-port"] != "" {
				match = append(match, "dport="+rule["dst-port"])
			}
			if rule["src-port"] != "" {
				match = append(match, "sport="+rule["src-port"])
			}
			if rule["in-interface"] != "" {
				match = append(match, "in="+rule["in-interface"])
			}
			if rule["out-interface"] != "" {
				match = append(match, "out="+rule["out-interface"])
			}
			if rule["connection-state"] != "" {
				match = append(match, "state="+rule["connection-state"])
			}
			if rule["to-addresses"] != "" {
				match = append(match, "to="+rule["to-addresses"]+":"+rule["to-ports"])
			}

			matchStr := strings.Join(match, "  ")
			if matchStr == "" {
				matchStr = "(all)"
			}

			fmt.Fprintf(&sb, "  ID:%-8s  chain:%-12s  action:%-18s  %s%s%s%s\n",
				rule[".id"], rule["chain"], rule["action"], matchStr, comment, disabled, dynamic)
		}
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolAddFirewallRule(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	table, err := strArg(args, "table")
	if err != nil {
		return ToolResult{}, err
	}
	chain, err := strArg(args, "chain")
	if err != nil {
		return ToolResult{}, err
	}
	action, err := strArg(args, "action")
	if err != nil {
		return ToolResult{}, err
	}

	dryRun := boolOpt(args, "dry_run", true)

	rule := map[string]string{
		"chain":  chain,
		"action": action,
	}
	optStr := func(k, rk string) {
		if v := strOpt(args, k, ""); v != "" {
			rule[rk] = v
		}
	}
	optStr("src_address", "src-address")
	optStr("dst_address", "dst-address")
	optStr("protocol", "protocol")
	optStr("dst_port", "dst-port")
	optStr("src_port", "src-port")
	optStr("in_interface", "in-interface")
	optStr("out_interface", "out-interface")
	optStr("comment", "comment")

	// Build human-readable summary
	var parts []string
	parts = append(parts, "table="+table, "chain="+chain, "action="+action)
	for k, v := range rule {
		if k == "chain" || k == "action" {
			continue
		}
		parts = append(parts, k+"="+v)
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would add firewall rule on %s:\n\n", r.Name())
		for _, p := range parts {
			fmt.Fprintf(&sb, "  %s\n", p)
		}
		fmt.Fprintf(&sb, "\nTo apply: call add_firewall_rule again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	// Handle position (prepend or insert after)
	path := "/ip/firewall/" + table
	position := strOpt(args, "position", "")
	if position == "top" {
		rule[".before"] = "*0"
	} else if position != "" {
		rule[".before"] = position
	}

	result, err := r.Put(path, rule)
	if err != nil {
		return ToolResult{}, fmt.Errorf("add firewall rule: %w", err)
	}

	var created map[string]string
	json.Unmarshal(result, &created)

	var sb strings.Builder
	sb.WriteString("✓ Firewall rule added\n")
	sb.WriteString(strings.Repeat("═", 40) + "\n\n")
	fmt.Fprintf(&sb, "Router: %s\nTable:  %s\nNew ID: %s\n\nRule:\n", r.Name(), table, created[".id"])
	for _, p := range parts {
		fmt.Fprintf(&sb, "  %s\n", p)
	}

	return textResult(sb.String()), nil
}

func (s *Server) toolRemoveFirewallRule(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	ruleID, err := strArg(args, "rule_id")
	if err != nil {
		return ToolResult{}, err
	}
	table, err := strArg(args, "table")
	if err != nil {
		return ToolResult{}, err
	}

	dryRun := boolOpt(args, "dry_run", true)

	// Fetch the rule so we can show its details in the preview
	raw, err := r.Get("/ip/firewall/" + table + "/" + ruleID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("fetch rule %s from %s table: %w", ruleID, table, err)
	}
	var rule map[string]string
	json.Unmarshal(raw, &rule)

	comment := ""
	if rule["comment"] != "" {
		comment = " (" + rule["comment"] + ")"
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would DELETE firewall rule from %s:\n\n", r.Name())
		fmt.Fprintf(&sb, "  Table:  %s\n", table)
		fmt.Fprintf(&sb, "  ID:     %s%s\n", ruleID, comment)
		fmt.Fprintf(&sb, "  Chain:  %s\n", rule["chain"])
		fmt.Fprintf(&sb, "  Action: %s\n", rule["action"])
		fmt.Fprintf(&sb, "\nThis action is irreversible. To apply: call remove_firewall_rule with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/ip/firewall/" + table + "/" + ruleID); err != nil {
		return ToolResult{}, fmt.Errorf("delete rule %s: %w", ruleID, err)
	}

	return textResult(fmt.Sprintf("✓ Rule %s deleted from table %s on %s%s", ruleID, table, r.Name(), comment)), nil
}
