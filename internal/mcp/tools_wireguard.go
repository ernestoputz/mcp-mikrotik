package mcp

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

// ─── Read ────────────────────────────────────────────────────────────────────

func (s *Server) toolGetWireGuardStatus(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	rawIfaces, err := r.Get("/interface/wireguard")
	if err != nil {
		return ToolResult{}, err
	}
	var ifaces []map[string]string
	if err := json.Unmarshal(rawIfaces, &ifaces); err != nil {
		return ToolResult{}, fmt.Errorf("parse wireguard interfaces: %w", err)
	}

	rawPeers, err := r.Get("/interface/wireguard/peers")
	if err != nil {
		return ToolResult{}, err
	}
	var peers []map[string]string
	if err := json.Unmarshal(rawPeers, &peers); err != nil {
		return ToolResult{}, fmt.Errorf("parse wireguard peers: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "WireGuard Status — %s\n%s\n", r.Name(), strings.Repeat("═", 50))

	for _, iface := range ifaces {
		running := "DOWN"
		if iface["running"] == "true" {
			running = "UP"
		}
		fmt.Fprintf(&sb, "\nInterface: %-15s  status: %-5s  port: %s\n", iface["name"], running, iface["listen-port"])
		fmt.Fprintf(&sb, "  Public Key: %s\n", iface["public-key"])

		// Count peers per interface
		var ifacePeers []map[string]string
		for _, p := range peers {
			if p["interface"] == iface["name"] {
				ifacePeers = append(ifacePeers, p)
			}
		}

		active, disabled, connected := 0, 0, 0
		for _, p := range ifacePeers {
			if p["disabled"] == "true" {
				disabled++
			} else {
				active++
				if p["current-endpoint-address"] != "" {
					connected++
				}
			}
		}
		fmt.Fprintf(&sb, "  Peers: %d active (%d connected), %d disabled\n", active, connected, disabled)

		sb.WriteString("\n  " + strings.Repeat("─", 46) + "\n")
		for _, p := range ifacePeers {
			dis := ""
			if p["disabled"] == "true" {
				dis = " [DISABLED]"
			}
			conn := "not connected"
			if p["current-endpoint-address"] != "" {
				conn = fmt.Sprintf("connected from %s:%s", p["current-endpoint-address"], p["current-endpoint-port"])
			}
			handshake := p["last-handshake"]
			if handshake == "" {
				handshake = "never"
			}
			name := p["name"]
			comment := p["comment"]
			if comment != "" {
				name = fmt.Sprintf("%s (%s)", name, comment)
			}

			rxMB := parseIntField(p["rx"]) / 1024 / 1024
			txMB := parseIntField(p["tx"]) / 1024 / 1024

			fmt.Fprintf(&sb, "  • %-28s  ip: %-16s  %s%s\n", name, p["client-address"], conn, dis)
			fmt.Fprintf(&sb, "    last handshake: %-15s  rx: %d MB  tx: %d MB\n", handshake, rxMB, txMB)
		}
	}

	return textResult(sb.String()), nil
}

// ─── Action: add peer ────────────────────────────────────────────────────────

func (s *Server) toolAddWireGuardPeer(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	peerName, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}

	ifaceName := strOpt(args, "interface", "wireguard1")
	comment := strOpt(args, "comment", "")
	fullTunnel := boolOpt(args, "full_tunnel", true)
	dryRun := boolOpt(args, "dry_run", true)

	// Fetch interface to get router's public key and listen port
	rawIfaces, err := r.Get("/interface/wireguard")
	if err != nil {
		return ToolResult{}, err
	}
	var ifaces []map[string]string
	if err := json.Unmarshal(rawIfaces, &ifaces); err != nil {
		return ToolResult{}, fmt.Errorf("parse wireguard interfaces: %w", err)
	}

	var iface map[string]string
	for _, i := range ifaces {
		if i["name"] == ifaceName {
			iface = i
			break
		}
	}
	if iface == nil {
		return ToolResult{}, fmt.Errorf("wireguard interface %q not found on %s", ifaceName, r.Name())
	}
	routerPublicKey := iface["public-key"]
	listenPort := iface["listen-port"]

	// Fetch existing peers to find defaults and next available IP
	rawPeers, err := r.Get("/interface/wireguard/peers")
	if err != nil {
		return ToolResult{}, err
	}
	var existingPeers []map[string]string
	if err := json.Unmarshal(rawPeers, &existingPeers); err != nil {
		return ToolResult{}, fmt.Errorf("parse peers: %w", err)
	}

	// Read defaults from existing peers (endpoint, DNS)
	var clientEndpoint, clientDNS string
	usedIPs := map[string]bool{"10.2.0.1": true}
	for _, p := range existingPeers {
		if p["interface"] != ifaceName {
			continue
		}
		if p["client-endpoint"] != "" && clientEndpoint == "" {
			clientEndpoint = p["client-endpoint"]
		}
		if p["client-dns"] != "" && clientDNS == "" {
			clientDNS = p["client-dns"]
		}
		// Track used IPs (client-address is "10.2.0.X/32" format)
		ip := strings.TrimSuffix(p["client-address"], "/32")
		if ip != "" {
			usedIPs[ip] = true
		}
	}

	// Allow overrides
	if v := strOpt(args, "endpoint", ""); v != "" {
		clientEndpoint = v
	}
	if v := strOpt(args, "dns", ""); v != "" {
		clientDNS = v
	}
	if clientEndpoint == "" {
		return ToolResult{}, fmt.Errorf("could not determine router endpoint; set it explicitly via the 'endpoint' argument (e.g. 'router.example.com')")
	}

	// Find next available IP in the /24
	assignedIP, err := nextAvailableIP("10.2.0.0/24", usedIPs)
	if err != nil {
		return ToolResult{}, fmt.Errorf("assign IP: %w", err)
	}

	// Generate WireGuard key pair (client)
	clientPrivKey, clientPubKey, err := generateWireGuardKeyPair()
	if err != nil {
		return ToolResult{}, fmt.Errorf("generate keys: %w", err)
	}

	// Generate preshared key
	psk, err := generatePresharedKey()
	if err != nil {
		return ToolResult{}, fmt.Errorf("generate preshared key: %w", err)
	}

	// Build allowed-address and client-allowed-address based on tunnel mode
	allowedAddr := assignedIP + "/32"
	clientAllowedAddr := "10.1.0.0/24,10.2.0.0/24"
	if fullTunnel {
		allowedAddr = assignedIP + "/32,10.1.0.0/24,::/0,0.0.0.0/0"
		clientAllowedAddr = "0.0.0.0/0,::/0"
	}

	dnsLine := ""
	if clientDNS != "" {
		dnsLine = "\nDNS = " + clientDNS
	}

	clientConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/32%s

[Peer]
PublicKey = %s
PresharedKey = %s
AllowedIPs = %s
Endpoint = %s:%s
PersistentKeepalive = 25`,
		clientPrivKey, assignedIP, dnsLine,
		routerPublicKey, psk, clientAllowedAddr,
		clientEndpoint, listenPort,
	)

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 50) + "\n\n")
		fmt.Fprintf(&sb, "Would add WireGuard peer on %s:\n\n", r.Name())
		fmt.Fprintf(&sb, "  Interface:   %s\n", ifaceName)
		fmt.Fprintf(&sb, "  Name:        %s\n", peerName)
		if comment != "" {
			fmt.Fprintf(&sb, "  Comment:     %s\n", comment)
		}
		fmt.Fprintf(&sb, "  Assigned IP: %s/32\n", assignedIP)
		fmt.Fprintf(&sb, "  Tunnel mode: %s\n", map[bool]string{true: "full (0.0.0.0/0)", false: "split (LAN only)"}[fullTunnel])
		fmt.Fprintf(&sb, "  Endpoint:    %s:%s\n", clientEndpoint, listenPort)
		fmt.Fprintf(&sb, "\nTo apply: call add_wireguard_peer again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	peer := map[string]string{
		"interface":           ifaceName,
		"name":                peerName,
		"allowed-address":     allowedAddr,
		"public-key":          clientPubKey,
		"private-key":         clientPrivKey,
		"preshared-key":       psk,
		"client-address":      assignedIP + "/32",
		"client-allowed-address": clientAllowedAddr,
		"client-endpoint":     clientEndpoint,
		"client-listen-port":  listenPort,
	}
	if clientDNS != "" {
		peer["client-dns"] = clientDNS
	}
	if comment != "" {
		peer["comment"] = comment
	}

	result, err := r.Put("/interface/wireguard/peers", peer)
	if err != nil {
		return ToolResult{}, fmt.Errorf("create wireguard peer: %w", err)
	}

	var created map[string]string
	json.Unmarshal(result, &created)

	var sb strings.Builder
	sb.WriteString("✓ WireGuard peer created\n")
	sb.WriteString(strings.Repeat("═", 50) + "\n\n")
	fmt.Fprintf(&sb, "Router:    %s\n", r.Name())
	fmt.Fprintf(&sb, "Peer:      %s (ID: %s)\n", peerName, created[".id"])
	fmt.Fprintf(&sb, "IP:        %s/32\n\n", assignedIP)
	sb.WriteString("⚠️  Save this config now — private key is shown only once:\n\n")
	sb.WriteString("━━━ WireGuard Client Config ━━━\n")
	sb.WriteString(clientConfig)
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString("Import this config into the WireGuard app on the client device.\n")

	return textResult(sb.String()), nil
}

// ─── Action: enable/disable/remove ───────────────────────────────────────────

func (s *Server) toolDisableWireGuardPeer(_ context.Context, args map[string]any) (ToolResult, error) {
	return s.toggleWireGuardPeer(args, true)
}

func (s *Server) toolEnableWireGuardPeer(_ context.Context, args map[string]any) (ToolResult, error) {
	return s.toggleWireGuardPeer(args, false)
}

func (s *Server) toggleWireGuardPeer(args map[string]any, disable bool) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	peerName, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	peer, err := findWireGuardPeer(r, peerName)
	if err != nil {
		return ToolResult{}, err
	}

	action := "enable"
	if disable {
		action = "disable"
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 50) + "\n\n")
		fmt.Fprintf(&sb, "Would %s WireGuard peer on %s:\n\n", action, r.Name())
		fmt.Fprintf(&sb, "  Name:       %s\n", peerName)
		fmt.Fprintf(&sb, "  Client IP:  %s\n", peer["client-address"])
		fmt.Fprintf(&sb, "  Currently:  disabled=%s\n", peer["disabled"])
		fmt.Fprintf(&sb, "\nTo apply: call %s_wireguard_peer again with dry_run=false\n", action)
		return textResult(sb.String()), nil
	}

	disabledVal := "false"
	if disable {
		disabledVal = "true"
	}

	if _, err := r.Patch("/interface/wireguard/peers/"+peer[".id"], map[string]string{"disabled": disabledVal}); err != nil {
		return ToolResult{}, fmt.Errorf("%s peer %s: %w", action, peerName, err)
	}

	return textResult(fmt.Sprintf("✓ Peer %q %sd on %s (IP: %s)", peerName, action, r.Name(), peer["client-address"])), nil
}

func (s *Server) toolRemoveWireGuardPeer(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	peerName, err := strArg(args, "name")
	if err != nil {
		return ToolResult{}, err
	}
	dryRun := boolOpt(args, "dry_run", true)

	peer, err := findWireGuardPeer(r, peerName)
	if err != nil {
		return ToolResult{}, err
	}

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 50) + "\n\n")
		fmt.Fprintf(&sb, "Would PERMANENTLY DELETE WireGuard peer on %s:\n\n", r.Name())
		fmt.Fprintf(&sb, "  Name:       %s\n", peerName)
		fmt.Fprintf(&sb, "  Client IP:  %s\n", peer["client-address"])
		fmt.Fprintf(&sb, "  ID:         %s\n", peer[".id"])
		fmt.Fprintf(&sb, "\n⚠ This cannot be undone. The client will immediately lose VPN access.\n")
		fmt.Fprintf(&sb, "\nTo apply: call remove_wireguard_peer again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	if err := r.Delete("/interface/wireguard/peers/" + peer[".id"]); err != nil {
		return ToolResult{}, fmt.Errorf("remove peer %s: %w", peerName, err)
	}

	return textResult(fmt.Sprintf("✓ Peer %q (IP: %s) permanently deleted from %s", peerName, peer["client-address"], r.Name())), nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func findWireGuardPeer(r interface {
	Get(string) (json.RawMessage, error)
}, name string) (map[string]string, error) {
	raw, err := r.Get("/interface/wireguard/peers")
	if err != nil {
		return nil, err
	}
	var peers []map[string]string
	if err := json.Unmarshal(raw, &peers); err != nil {
		return nil, fmt.Errorf("parse peers: %w", err)
	}
	for _, p := range peers {
		if p["name"] == name || p["comment"] == name {
			return p, nil
		}
	}
	var names []string
	for _, p := range peers {
		n := p["name"]
		if p["comment"] != "" {
			n += " (" + p["comment"] + ")"
		}
		names = append(names, n)
	}
	return nil, fmt.Errorf("peer %q not found; available: %v", name, names)
}

func nextAvailableIP(cidr string, used map[string]bool) (string, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR %s: %w", cidr, err)
	}

	ip := network.IP.To4()
	if ip == nil {
		return "", fmt.Errorf("only IPv4 supported")
	}

	// Scan from .2 (skip .0 network, .1 router)
	for i := 2; i < 254; i++ {
		candidate := fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], byte(i))
		if !used[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no available IPs in %s", cidr)
}

// generateWireGuardKeyPair produces a clamped Curve25519 private key and its public key.
func generateWireGuardKeyPair() (privateB64, publicB64 string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return
	}
	// Apply RFC 7748 / WireGuard clamping
	raw[0] &= 248
	raw[31] = (raw[31] & 127) | 64

	curve := ecdh.X25519()
	priv, err := curve.NewPrivateKey(raw)
	if err != nil {
		return
	}
	privateB64 = base64.StdEncoding.EncodeToString(raw)
	publicB64 = base64.StdEncoding.EncodeToString(priv.PublicKey().Bytes())
	return
}

func generatePresharedKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
