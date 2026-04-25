package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/user/mcp-mikrotik/internal/aws"
	"github.com/user/mcp-mikrotik/internal/mikrotik"
)

const (
	serverName    = "mcp-mikrotik"
	serverVersion = "1.0.0"
	mcpProtoVer   = "2024-11-05"
)

// Server is the MCP server. It routes JSON-RPC requests to tool handlers.
type Server struct {
	cfg     *Config
	logger  *slog.Logger
	routers []*mikrotik.Client
	s3      *aws.S3Client // nil if S3 not configured
	tools   map[string]Tool
}

// NewServer constructs and initialises the server from config.
func NewServer(cfg *Config, logger *slog.Logger) (*Server, error) {
	s := &Server{cfg: cfg, logger: logger}

	for _, rcfg := range cfg.Routers {
		s.routers = append(s.routers, mikrotik.NewClient(rcfg))
	}

	if cfg.AWSAccessKeyID != "" && cfg.AWSS3Bucket != "" {
		s.s3 = aws.NewS3Client(aws.S3Config{
			AccessKeyID:     cfg.AWSAccessKeyID,
			SecretAccessKey: cfg.AWSSecretAccessKey,
			Region:          cfg.AWSRegion,
			Bucket:          cfg.AWSS3Bucket,
			Prefix:          cfg.AWSS3Prefix,
		})
	}

	s.tools = s.buildToolRegistry()
	return s, nil
}

// AuthToken returns the configured Bearer token (may be empty).
func (s *Server) AuthToken() string { return s.cfg.MCPAuthToken }

// Handle processes a single JSON-RPC 2.0 request and returns a Response.
func (s *Server) Handle(ctx context.Context, raw []byte) Response {
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return errResponse(nil, ErrCodeParse, "parse error: "+err.Error())
	}

	s.logger.Debug("rpc request", "method", req.Method, "id", string(req.ID))

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return Response{JSONRPC: "2.0", ID: req.ID}
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return errResponse(req.ID, ErrCodeMethodNotFound, "method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req Request) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: mcpProtoVer,
			ServerInfo:      ServerInfo{Name: serverName, Version: serverVersion},
			Capabilities:    Caps{Tools: &ToolsCap{ListChanged: false}},
		},
	}
}

func (s *Server) handleToolsList(req Request) Response {
	list := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		list = append(list, t)
	}
	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]any{"tools": list},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req Request) Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errResponse(req.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
	}

	handler, ok := s.toolHandlers()[params.Name]
	if !ok {
		return errResponse(req.ID, ErrCodeMethodNotFound, "unknown tool: "+params.Name)
	}

	result, err := handler(ctx, params.Arguments)
	if err != nil {
		s.logger.Error("tool failed", "tool", params.Name, "error", err)
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  ToolResult{IsError: true, Content: []ContentBlock{{Type: "text", Text: err.Error()}}},
		}
	}

	return Response{JSONRPC: "2.0", ID: req.ID, Result: result}
}

type handlerFn func(ctx context.Context, args map[string]any) (ToolResult, error)

func (s *Server) toolHandlers() map[string]handlerFn {
	return map[string]handlerFn{
		// ── Diagnostic (read-only) ─────────────────────────────────────────────
		"list_routers":            s.toolListRouters,
		"get_system_info":         s.toolGetSystemInfo,
		"get_interfaces":          s.toolGetInterfaces,
		"get_ip_addresses":        s.toolGetIPAddresses,
		"get_routing_table":       s.toolGetRoutingTable,
		"get_arp_table":           s.toolGetARPTable,
		"get_dhcp_leases":         s.toolGetDHCPLeases,
		"get_logs":                s.toolGetLogs,
		"ping_from_router":        s.toolPingFromRouter,
		"traceroute_from_router":  s.toolTracerouteFromRouter,
		// ── WiFi / CAPsMAN ────────────────────────────────────────────────────
		"get_wifi_clients":         s.toolGetWiFiClients,
		"get_wifi_interfaces":      s.toolGetWiFiInterfaces,
		"get_wifi_configurations":  s.toolGetWiFiConfigurations,
		"get_capsman_status":       s.toolGetCAPsMANStatus,
		"set_wifi_configuration":   s.toolSetWiFiConfiguration,
		// ── Firewall ──────────────────────────────────────────────────────────
		"get_firewall_rules":   s.toolGetFirewallRules,
		"add_firewall_rule":    s.toolAddFirewallRule,
		"remove_firewall_rule": s.toolRemoveFirewallRule,
		// ── QoS / DNS / Interface ─────────────────────────────────────────────
		"get_queue_stats":   s.toolGetQueueStats,
		"get_dns_entries":   s.toolGetDNSEntries,
		"set_queue_limit":   s.toolSetQueueLimit,
		"restart_interface": s.toolRestartInterface,
		// ── Backup ────────────────────────────────────────────────────────────
		"create_backup": s.toolCreateBackup,
	}
}

// ─── Router lookup ────────────────────────────────────────────────────────────

// router returns the client for the given name, or the first router if name is empty.
func (s *Server) router(name string) (*mikrotik.Client, error) {
	if name == "" {
		if len(s.routers) == 0 {
			return nil, fmt.Errorf("no routers configured")
		}
		return s.routers[0], nil
	}
	for _, r := range s.routers {
		if r.Name() == name {
			return r, nil
		}
	}
	names := make([]string, len(s.routers))
	for i, r := range s.routers {
		names[i] = r.Name()
	}
	return nil, fmt.Errorf("router %q not found; configured: %v", name, names)
}

// routerNames returns the names of all configured routers.
func (s *Server) routerNames() []string {
	names := make([]string, len(s.routers))
	for i, r := range s.routers {
		names[i] = r.Name()
	}
	return names
}

// ─── Result helpers ───────────────────────────────────────────────────────────

func textResult(text string) ToolResult {
	return ToolResult{Content: []ContentBlock{{Type: "text", Text: text}}}
}

func jsonResult(v any) (ToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ToolResult{}, fmt.Errorf("marshal result: %w", err)
	}
	return textResult(string(b)), nil
}

// ─── Argument helpers ─────────────────────────────────────────────────────────

func strArg(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing required argument: %s", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %s must be a string", key)
	}
	return s, nil
}

func strOpt(args map[string]any, key, def string) string {
	v, ok := args[key]
	if !ok {
		return def
	}
	s, _ := v.(string)
	if s == "" {
		return def
	}
	return s
}

func boolOpt(args map[string]any, key string, def bool) bool {
	v, ok := args[key]
	if !ok {
		return def
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true"
	}
	return def
}

func intOpt(args map[string]any, key string, def int) int {
	v, ok := args[key]
	if !ok {
		return def
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	}
	return def
}
