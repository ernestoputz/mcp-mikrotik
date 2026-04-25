package mcp

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/user/mcp-mikrotik/internal/mikrotik"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Routers []mikrotik.RouterConfig

	// AWS S3 — optional, required only for backup uploads
	AWSAccessKeyID     string // AWS_ACCESS_KEY_ID
	AWSSecretAccessKey string // AWS_SECRET_ACCESS_KEY
	AWSRegion          string // AWS_REGION
	AWSS3Bucket        string // AWS_S3_BUCKET
	AWSS3Prefix        string // AWS_S3_PREFIX (default: "mikrotik-backups/")

	// HTTP transport — used when MCP_TRANSPORT=http
	HTTPHost     string // HTTP_HOST (default: 127.0.0.1)
	HTTPPort     string // HTTP_PORT (default: 8080)
	MCPAuthToken string // MCP_AUTH_TOKEN — Bearer token for HTTP clients
}

// LoadConfig reads all configuration from environment variables.
// Routers are discovered via MIKROTIK_ROUTER_1_HOST … MIKROTIK_ROUTER_N_HOST.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		HTTPHost:     envOr("HTTP_HOST", "127.0.0.1"),
		HTTPPort:     envOr("HTTP_PORT", "8080"),
		MCPAuthToken: os.Getenv("MCP_AUTH_TOKEN"),

		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSRegion:          os.Getenv("AWS_REGION"),
		AWSS3Bucket:        os.Getenv("AWS_S3_BUCKET"),
		AWSS3Prefix:        envOr("AWS_S3_PREFIX", "mikrotik-backups/"),
	}

	for i := 1; ; i++ {
		pfx := fmt.Sprintf("MIKROTIK_ROUTER_%d_", i)
		host := os.Getenv(pfx + "HOST")
		if host == "" {
			break
		}
		skipVerify, _ := strconv.ParseBool(os.Getenv(pfx + "TLS_SKIP_VERIFY"))
		cfg.Routers = append(cfg.Routers, mikrotik.RouterConfig{
			Name:          envOr(pfx+"NAME", fmt.Sprintf("router-%d", i)),
			Host:          host,
			Port:          os.Getenv(pfx + "PORT"),
			Scheme:        envOr(pfx+"SCHEME", "https"),
			User:          os.Getenv(pfx + "USER"),
			Pass:          os.Getenv(pfx + "PASS"),
			TLSSkipVerify: skipVerify,
		})
	}

	var errs []string
	if len(cfg.Routers) == 0 {
		errs = append(errs, "no routers configured — set MIKROTIK_ROUTER_1_HOST, MIKROTIK_ROUTER_1_USER, MIKROTIK_ROUTER_1_PASS")
	}
	for i, r := range cfg.Routers {
		if r.User == "" {
			errs = append(errs, fmt.Sprintf("MIKROTIK_ROUTER_%d_USER is required", i+1))
		}
		if r.Pass == "" {
			errs = append(errs, fmt.Sprintf("MIKROTIK_ROUTER_%d_PASS is required", i+1))
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("config:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
