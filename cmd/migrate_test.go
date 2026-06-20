package cmd

import (
	"strings"
	"testing"

	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/utils"
)

// Three distinct, valid-length (44-char base64) WireGuard keys for fixtures.
const (
	keyA = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA0="
	keyB = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB0="
	keyC = "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC0="
	keyD = "DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD0="
)

func TestDecodeClientTunnel(t *testing.T) {
	// Dial-out tunnel: WireguardServerKey empty, Client true, password = serverPub+clientPriv+clientPub.
	tun := models.Tunnel{
		ID:        7,
		Hostname:  "hub.example.org:5527",
		IP:        "172.31.150.16",
		Client:    true,
		Wireguard: true,
		Enabled:   true,
		Password:  keyA + keyB + keyC,
	}
	d := decodeTunnel(tun)
	if !d.valid {
		t.Fatalf("expected valid decode")
	}
	if d.role != "client" {
		t.Fatalf("expected role client, got %s", d.role)
	}
	if d.serverPub != keyA || d.clientPriv != keyB || d.clientPub != keyC {
		t.Fatalf("key split incorrect: %q %q %q", d.serverPub, d.clientPriv, d.clientPub)
	}
	if d.endpointHost != "hub.example.org" || d.endpointPort != "5527" {
		t.Fatalf("endpoint parse incorrect: %q %q", d.endpointHost, d.endpointPort)
	}
	// Server IP is tunnel.IP; our IP is +1 (matches internal/wireguard offset).
	if d.remoteIP != "172.31.150.16" || d.localIP != "172.31.150.17" {
		t.Fatalf("ip offset incorrect: local=%s remote=%s", d.localIP, d.remoteIP)
	}
}

func TestDecodeHostTunnel(t *testing.T) {
	// Host tunnel: WireguardServerKey set, Client false.
	tun := models.Tunnel{
		ID:                 3,
		Hostname:           "N0CALL-CLIENT",
		IP:                 "172.31.150.20",
		Client:             false,
		Wireguard:          true,
		Enabled:            true,
		WireguardPort:      5528,
		WireguardServerKey: keyD,
		Password:           keyA + keyB + keyC,
	}
	d := decodeTunnel(tun)
	if !d.valid || d.role != "host" {
		t.Fatalf("expected valid host decode, got valid=%t role=%s", d.valid, d.role)
	}
	if d.serverPriv != keyD {
		t.Fatalf("server priv incorrect: %q", d.serverPriv)
	}
	// AREDN host-side key = serverPriv+serverPub+clientPriv+clientPub.
	wantKey := keyD + keyA + keyB + keyC
	gotKey := d.serverPriv + d.Password
	if gotKey != wantKey {
		t.Fatalf("aredn key incorrect:\n got %q\nwant %q", gotKey, wantKey)
	}
	if d.localIP != "172.31.150.20" || d.remoteIP != "172.31.150.21" {
		t.Fatalf("ip offset incorrect: local=%s remote=%s", d.localIP, d.remoteIP)
	}
}

func TestDecodeInvalidPassword(t *testing.T) {
	d := decodeTunnel(models.Tunnel{Wireguard: true, Password: "too-short"})
	if d.valid {
		t.Fatalf("expected invalid decode for short password")
	}
}

func TestBuildReportContainsSections(t *testing.T) {
	cfg := &config.Config{
		ServerName: "KI5VMF-TEST",
		NodeIP:     "10.54.27.2",
		Supernode:  true,
		Babel:      config.Babel{Enabled: true, RouterID: "01:42:c0:a8:fb:05"},
	}
	tunnels := []models.Tunnel{
		{ID: 1, Hostname: "hub.example.org:5527", IP: "172.31.150.16", Client: true, Wireguard: true, Enabled: true, Password: keyA + keyB + keyC},
		{ID: 2, Hostname: "N0CALL-CLIENT", IP: "172.31.150.20", Wireguard: true, Enabled: true, WireguardPort: 5528, WireguardServerKey: keyD, Password: keyA + keyB + keyC},
		{ID: 3, Hostname: "OLD-VTUN", IP: "172.16.0.5", Wireguard: false, Enabled: true, Password: "secret"},
	}
	users := []models.User{{ID: 0, Username: "admin"}, {ID: 1, Username: "kc1abc"}}
	report := buildReport(cfg, tunnels, users, adminNotChecked)

	for _, want := range []string{
		"# mesh-manager → AREDN® migration report",
		"## 1. Node identity",
		"KI5VMF-TEST",
		"supernode",
		"Babel",
		"Legacy vtun tunnels",
		"Wireguard key: " + keyA + keyB + keyC,         // client paste blob
		"uci -c /etc/config.mesh add wireguard server", // dial-out uci add
		"wireguard.@server[-1].passwd='" + keyA + keyB + keyC + "'",     // dial-out key
		"uci -c /etc/config.mesh add wireguard client",                  // host uci add
		"wireguard.@client[-1].key='" + keyD + keyA + keyB + keyC + "'", // host 4-key
		"wireguard.@client[-1].clientip='172.31.150.20:5528'",           // host's own IP + listen port
		"uci -c /etc/local/uci set hsmmmesh.settings.mac2='54.27.2'",    // mesh IP preservation
		"uci -c /etc/config.mesh set aredn.@supernode[0].enable='1'",    // supernode enable
		"setpasswd", // admin migration
		"`admin`",   // user listed
		"`kc1abc`",
		"## 7. Known parity gaps",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing expected content: %q", want)
		}
	}
}

func TestVerifyAdminPassword(t *testing.T) {
	cfg := &config.Config{PasswordSalt: "saltysalt"}
	hash := utils.HashPassword("hunter2", cfg.PasswordSalt)
	users := []models.User{{Username: "admin", Password: hash}}

	if got := verifyAdminPassword(cfg, users, "hunter2"); got != adminMatched {
		t.Errorf("expected adminMatched, got %v", got)
	}
	if got := verifyAdminPassword(cfg, users, "wrong"); got != adminMismatch {
		t.Errorf("expected adminMismatch, got %v", got)
	}
	if got := verifyAdminPassword(cfg, nil, "x"); got != adminMismatch {
		t.Errorf("expected adminMismatch for no users, got %v", got)
	}
}
