package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/mcp-mikrotik/internal/mcp"
	"github.com/user/mcp-mikrotik/internal/transport"
)

func main() {
	// Logs go to stderr — stdout is reserved for the stdio JSON-RPC channel.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slogLevel(),
	}))
	slog.SetDefault(logger)

	cfg, err := mcp.LoadConfig()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	server, err := mcp.NewServer(cfg, logger)
	if err != nil {
		slog.Error("failed to start MCP server", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	mode := os.Getenv("MCP_TRANSPORT")
	if mode == "" {
		mode = "stdio"
	}

	switch mode {
	case "stdio":
		slog.Info("mcp-mikrotik started", "transport", "stdio")
		if err := transport.RunStdio(ctx, server); err != nil {
			slog.Error("stdio error", "error", err)
			os.Exit(1)
		}
	case "http":
		addr := fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort)
		slog.Info("mcp-mikrotik started", "transport", "http+sse", "addr", addr)
		if err := transport.RunHTTP(ctx, server, addr); err != nil {
			slog.Error("http error", "error", err)
			os.Exit(1)
		}
	default:
		slog.Error("unknown MCP_TRANSPORT", "value", mode, "valid", []string{"stdio", "http"})
		os.Exit(1)
	}
}

func slogLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
