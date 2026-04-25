package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// toolRunCommand executes any RouterOS REST API call directly.
func (s *Server) toolRunCommand(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	path, err := strArg(args, "path")
	if err != nil {
		return ToolResult{}, err
	}

	method := strings.ToUpper(strOpt(args, "method", "GET"))
	bodyStr := strOpt(args, "body", "")

	isWrite := method != "GET" && method != "HEAD"
	dryRun := isWrite && boolOpt(args, "dry_run", true)

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would execute on %s:\n\n", r.Name())
		fmt.Fprintf(&sb, "  Method:  %s\n", method)
		fmt.Fprintf(&sb, "  Path:    /rest%s\n", path)
		if bodyStr != "" {
			fmt.Fprintf(&sb, "  Body:    %s\n", bodyStr)
		}
		sb.WriteString("\nTo apply: call run_command again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	var body any
	if bodyStr != "" {
		if err := json.Unmarshal([]byte(bodyStr), &body); err != nil {
			return ToolResult{}, fmt.Errorf("invalid JSON body: %w", err)
		}
	}

	var raw json.RawMessage
	switch method {
	case "GET":
		raw, err = r.Get(path)
	case "POST":
		raw, err = r.Post(path, body)
	case "PUT":
		raw, err = r.Put(path, body)
	case "PATCH":
		raw, err = r.Patch(path, body)
	case "DELETE":
		err = r.Delete(path)
	default:
		return ToolResult{}, fmt.Errorf("unsupported method: %s (use GET, POST, PUT, PATCH, DELETE)", method)
	}
	if err != nil {
		return ToolResult{}, fmt.Errorf("%s /rest%s: %w", method, path, err)
	}

	if len(raw) == 0 || string(raw) == "null" || string(raw) == "" {
		return textResult(fmt.Sprintf("✓ %s /rest%s — success (no output)", method, path)), nil
	}

	var pretty bytes.Buffer
	if json.Indent(&pretty, raw, "", "  ") == nil {
		return textResult(pretty.String()), nil
	}
	return textResult(string(raw)), nil
}

// toolRunScript creates a temporary RouterOS script, runs it, then deletes it.
// Script output is not returned directly (RouterOS REST API doesn't support this);
// use get_logs with topic "script" to see any output from :log statements.
func (s *Server) toolRunScript(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	source, err := strArg(args, "source")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would execute RouterOS script on %s:\n\n", r.Name())
		sb.WriteString("─── script ───\n")
		sb.WriteString(source)
		sb.WriteString("\n──────────────\n")
		sb.WriteString("\nNote: script output is not returned via REST API.\n")
		sb.WriteString("Use :log info message=... in your script to capture output,\n")
		sb.WriteString("then call get_logs with topics=script to retrieve it.\n")
		sb.WriteString("\nTo apply: call run_script again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	// Use a timestamped name to avoid collisions
	scriptName := fmt.Sprintf("mcp-tmp-%d", time.Now().UnixMilli())

	// Create temp script
	_, err = r.Put("/ip/script", map[string]string{
		"name":   scriptName,
		"source": source,
	})
	if err != nil {
		return ToolResult{}, fmt.Errorf("create script: %w", err)
	}

	// Run it
	_, runErr := r.Post("/ip/script/"+scriptName+"/run", nil)

	// Always clean up
	_ = r.Delete("/ip/script/" + scriptName)

	if runErr != nil {
		return ToolResult{}, fmt.Errorf("script execution failed: %w", runErr)
	}

	var sb strings.Builder
	sb.WriteString("✓ Script executed on ")
	sb.WriteString(r.Name())
	sb.WriteString("\n\n")
	sb.WriteString("Script output is not captured via REST API.\n")
	sb.WriteString("If your script used ':log info message=...' statements,\n")
	sb.WriteString("call get_logs with topics=\"script\" to retrieve the output.\n")

	return textResult(sb.String()), nil
}
