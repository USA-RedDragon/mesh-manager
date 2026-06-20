package cmd

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/USA-RedDragon/configulator"
	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/USA-RedDragon/mesh-manager/internal/db"
	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/utils"
	"github.com/spf13/cobra"
)

// verifyAdminPassword checks pw against the stored hash of the "admin" user
// (falling back to the first user), using mesh-manager's own argon2id verifier.
func verifyAdminPassword(cfg *config.Config, users []models.User, pw string) adminCheck {
	var target *models.User
	for i := range users {
		if users[i].Username == "admin" {
			target = &users[i]
			break
		}
	}
	if target == nil && len(users) > 0 {
		target = &users[0]
	}
	if target == nil {
		return adminMismatch
	}
	ok, err := utils.VerifyPassword(pw, target.Password, cfg.PasswordSalt)
	if err != nil || !ok {
		return adminMismatch
	}
	return adminMatched
}

func newMigrateCommand(version, commit string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Generate a report to migrate this node to unmodified upstream AREDN® firmware",
		Long: "Reads this node's configuration and database and emits a Markdown migration\n" +
			"report mapping every piece of mesh-manager state to its native AREDN® equivalent,\n" +
			"including ready-to-paste WireGuard tunnel details. Run it against the same config\n" +
			"and database the node normally uses.\n\n" +
			"AREDN® is a registered trademark of Amateur Radio Emergency Data Network, Inc.\n" +
			"This tool and the mesh-node image are not affiliated with, endorsed by, or\n" +
			"sanctioned by AREDN, Inc.; the image runs unmodified upstream AREDN® firmware.",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Annotations: map[string]string{
			"version": version,
			"commit":  commit,
		},
		RunE:              runMigrate,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}
	cmd.Flags().StringP("output", "o", "", "Write the report to a file instead of stdout")
	cmd.Flags().String("admin-password", "", "Verify this plaintext against the stored admin hash and confirm the AREDN® setpasswd step")
	return cmd
}

// adminCheck records whether an operator-supplied admin password matched the
// stored hash. Values: 0 = not checked, 1 = matched, 2 = did not match.
type adminCheck int

const (
	adminNotChecked adminCheck = iota
	adminMatched
	adminMismatch
)

func runMigrate(cmd *cobra.Command, _ []string) error {
	err := runRoot(cmd, nil)
	if err != nil {
		slog.Error("Encountered an error.", "error", err.Error())
	}

	ctx := cmd.Context()

	c, err := configulator.FromContext[config.Config](ctx)
	if err != nil {
		return fmt.Errorf("failed to get config from context")
	}

	cfg, err := c.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.MakeDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	tunnels, err := models.ListAllTunnels(database)
	if err != nil {
		return fmt.Errorf("failed to list tunnels: %w", err)
	}

	users, err := models.ListUsers(database)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	check := adminNotChecked
	if pw, _ := cmd.Flags().GetString("admin-password"); pw != "" {
		check = verifyAdminPassword(cfg, users, pw)
	}

	report := buildReport(cfg, tunnels, users, check)

	outPath, _ := cmd.Flags().GetString("output")
	if outPath == "" {
		fmt.Print(report)
		return nil
	}

	//nolint:gosec // operator-supplied output path
	if err := os.WriteFile(outPath, []byte(report), 0644); err != nil {
		return fmt.Errorf("failed to write report to %s: %w", outPath, err)
	}
	slog.Info("Wrote migration report", "path", outPath)
	return nil
}

// decodedTunnel holds the WireGuard parameters extracted from a tunnel row.
type decodedTunnel struct {
	models.Tunnel
	valid        bool
	role         string // roleHost (we listen) or roleClient (we dial out)
	serverPriv   string
	serverPub    string
	clientPriv   string
	clientPub    string
	localIP      string // this node's IP on the tunnel
	remoteIP     string // the peer's IP on the tunnel
	endpointHost string // for client tunnels: remote host
	endpointPort string // for client tunnels: remote port
}

const wgKeyLen = 44
const wgPasswordLen = wgKeyLen * 3

// roleHost: this node hosts/listens for the tunnel. roleClient: this node dials out.
const roleHost = "host"
const roleClient = "client"

// uci cursors verified on mesh-node:4.26.1.0: hsmmmesh lives in /etc/local/uci,
// the staged AREDN® config (aredn/setup/wireguard) in /etc/config.mesh.
const uciMesh = "uci -c /etc/config.mesh"
const uciLocal = "uci -c /etc/local/uci"

// writeUCIAdd emits a runnable block that appends a new section of secType to the
// wireguard config and sets each option, committing at the end.
func writeUCIAdd(b *strings.Builder, secType string, opts [][2]string) {
	b.WriteString("```sh\n")
	fmt.Fprintf(b, "%s add wireguard %s\n", uciMesh, secType)
	for _, kv := range opts {
		fmt.Fprintf(b, "%s set wireguard.@%s[-1].%s='%s'\n", uciMesh, secType, kv[0], kv[1])
	}
	fmt.Fprintf(b, "%s commit wireguard\n", uciMesh)
	b.WriteString("```\n\n")
}

func decodeTunnel(t models.Tunnel) decodedTunnel {
	d := decodedTunnel{Tunnel: t}
	if !t.Wireguard {
		return d
	}
	if len(t.Password) != wgPasswordLen {
		return d
	}
	d.serverPub = t.Password[:wgKeyLen]
	d.clientPriv = t.Password[wgKeyLen : wgKeyLen*2]
	d.clientPub = t.Password[wgKeyLen*2:]

	if t.WireguardServerKey != "" {
		// We host this tunnel (UI "WireGuard Server"). tunnel.IP is our (server) IP,
		// peer IP is tunnel.IP + 1. See internal/wireguard/wireguard.go:176-196.
		d.role = roleHost
		d.serverPriv = t.WireguardServerKey
		d.localIP = t.IP
		d.remoteIP = incrementIP(t.IP)
	} else {
		// We dial out (UI "WireGuard Client"). tunnel.IP is the remote (server) IP,
		// our IP is tunnel.IP + 1. Hostname is "host:port".
		d.role = roleClient
		d.remoteIP = t.IP
		d.localIP = incrementIP(t.IP)
		host, port, ok := splitHostPort(t.Hostname)
		if ok {
			d.endpointHost = host
			d.endpointPort = port
		}
	}
	d.valid = true
	return d
}

// incrementIP returns the IPv4 string with its last octet incremented by one,
// mirroring the client-side address offset in internal/wireguard/wireguard.go.
func incrementIP(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ipStr
	}
	ip = ip.To4()
	if ip == nil {
		return ipStr
	}
	out := make(net.IP, len(ip))
	copy(out, ip)
	out[3]++
	if out[3] == 0 {
		out[2]++
	}
	return out.String()
}

func splitHostPort(hostname string) (host, port string, ok bool) {
	parts := strings.Split(hostname, ":")
	if len(parts) != 2 {
		return hostname, "", false
	}
	return parts[0], parts[1], true
}

//nolint:gocyclo // a long but linear report builder
func buildReport(cfg *config.Config, tunnels []models.Tunnel, users []models.User, check adminCheck) string {
	var b strings.Builder

	b.WriteString("# mesh-manager → AREDN® migration report\n\n")
	b.WriteString("Generated by `mesh-manager migrate`. This maps every piece of state this node\n")
	b.WriteString("owns to its native AREDN® equivalent on the `mesh-node` image (unmodified\n")
	b.WriteString("upstream AREDN® firmware). See `docs/MIGRATION.md` for the full walkthrough.\n\n")
	b.WriteString("> AREDN® is a registered trademark of Amateur Radio Emergency Data Network, Inc.\n")
	b.WriteString("> This project is not affiliated with, endorsed by, or sanctioned by AREDN, Inc.\n\n")

	// --- Node identity ---
	b.WriteString("## 1. Node identity\n\n")
	b.WriteString("Set these on the new node via the AREDN® web UI (**Settings → Node**) unless noted.\n\n")
	b.WriteString("| mesh-manager | value | AREDN® native location |\n")
	b.WriteString("|---|---|---|\n")
	fmt.Fprintf(&b, "| `SERVER_NAME` | `%s` | Node name — uci `hsmmmesh.settings.node`, **Basic Setup → Node Name** |\n", cfg.ServerName)
	fmt.Fprintf(&b, "| `NODE_IP` | `%s` | Mesh IP — uci `setup.globals.wifi_ip`, derived from `hsmmmesh.settings.mac2` (see below) |\n", cfg.NodeIP)
	fmt.Fprintf(&b, "| `LATITUDE` | `%s` | Location → Latitude (uci `aredn.@location[0].lat`) |\n", floatOrBlank(cfg.Latitude))
	fmt.Fprintf(&b, "| `LONGITUDE` | `%s` | Location → Longitude (uci `aredn.@location[0].lon`) |\n", floatOrBlank(cfg.Longitude))
	fmt.Fprintf(&b, "| `GRIDSQUARE` | `%s` | Location → Grid square (uci `aredn.@location[0].gridsquare`) |\n", strOrDash(cfg.Gridsquare))
	b.WriteString("\n")
	writeMeshIP(&b, cfg.NodeIP)
	writeLocation(&b, cfg)

	// --- Routing ---
	b.WriteString("## 2. Routing protocol\n\n")
	switch {
	case cfg.Babel.Enabled:
		fmt.Fprintf(&b, "This node runs **Babel** (`BABEL_ENABLED=true`, router-id `%s`). Current AREDN®\n", cfg.Babel.RouterID)
		b.WriteString("also meshes on **Babel** and manages the daemon itself via uci — there is nothing\n")
		b.WriteString("to copy. The mesh-manager router-id does not carry over; AREDN® derives its own.\n\n")
	case cfg.OLSR:
		b.WriteString("This node runs **OLSR** (`OLSR=true`). OLSR is **legacy** in current AREDN®, which\n")
		b.WriteString("meshes on **Babel** instead. No OLSR config carries over; AREDN® manages Babel\n")
		b.WriteString("natively. Your mesh neighbors must also speak Babel for links to form.\n\n")
	default:
		b.WriteString("No routing protocol is enabled in this node's config. Current AREDN® meshes on\n")
		b.WriteString("**Babel** and manages it natively.\n\n")
	}

	// --- Supernode ---
	b.WriteString("## 3. Supernode role\n\n")
	if cfg.Supernode {
		b.WriteString("This node is a **supernode** (`SUPERNODE=true`). In AREDN® this is a config/CLI\n")
		b.WriteString("action, not a web-UI toggle:\n\n")
		b.WriteString("```sh\n")
		fmt.Fprintf(&b, "%s set aredn.@supernode[0].enable='1'\n", uciMesh)
		fmt.Fprintf(&b, "%s commit aredn\n", uciMesh)
		b.WriteString("```\n\n")
		b.WriteString("Supernode tunnels then use the `172.30.x.x` range and UDP ports from `6526`\n")
		b.WriteString("(vs `172.31.x.x` / `5525` for normal nodes). *(Verified on `mesh-node:4.26.1.0`.)*\n\n")
	} else {
		b.WriteString("This node is a normal (non-supernode) node. Nothing to do.\n\n")
	}

	// --- Tunnels ---
	writeTunnelSection(&b, tunnels)

	// --- Services ---
	b.WriteString("## 5. Advertised services\n\n")
	b.WriteString("mesh-manager advertises only this node's own web console automatically\n")
	b.WriteString("(`http://" + cfg.ServerName + "/|tcp|" + cfg.ServerName + "-console`, seeded in\n")
	b.WriteString("`docker/rootfs/usr/bin/start.sh` / generated routing config) and has no UI to add\n")
	b.WriteString("custom advertised services. AREDN® advertises the node console natively; add any\n")
	b.WriteString("extra services under **Settings → Services** (uci `setup.services.service`).\n\n")

	// --- Users / auth ---
	writeUsersSection(&b, users, check)

	// --- Parity gaps ---
	b.WriteString("## 7. Known parity gaps\n\n")
	b.WriteString("- **vtun tunnels:** gone in current AREDN®. Any legacy (non-WireGuard) tunnel must\n")
	b.WriteString("  be rebuilt from scratch as WireGuard with the partner node — keys do not exist to carry over.\n")
	b.WriteString("- **Host (server) tunnel keys via the UI:** adding a tunnel server in the AREDN® UI\n")
	b.WriteString("  generates **new** keys, so the remote client must be re-paired. To avoid touching\n")
	b.WriteString("  the far end, inject the existing keys via uci (section 4, Option B).\n")
	b.WriteString("- **Multiple admin logins:** AREDN® has a single `root` admin; extra mesh-manager\n")
	b.WriteString("  users collapse onto it (section 6).\n")
	b.WriteString("- **RF / Wi-Fi / firmware-upgrade / reboot UI actions:** irrelevant in an RF-less container.\n\n")

	return b.String()
}

// writeMeshIP documents how to preserve the exact node mesh IP on AREDN®.
// Verified against ghcr.io/usa-reddragon/mesh-node:4.26.1.0: the node's mesh IP
// defaults to 10.<mac2> (configuration.uc getDefaultIP) and is applied as
// setup.globals.wifi_ip (node-setup cfg.wifi_ip).
func writeMeshIP(b *strings.Builder, nodeIP string) {
	b.WriteString("> **Mesh IP — preserving `" + nodeIP + "`.** AREDN® derives the node's mesh IP as\n")
	b.WriteString("> `10.<mac2>` from `hsmmmesh.settings.mac2` and applies it as `setup.globals.wifi_ip`.\n")
	if mac2, ok := strings.CutPrefix(nodeIP, "10."); ok {
		b.WriteString("> To keep the same address, set the identity octets **before first configuration**:\n>\n")
		b.WriteString("> ```sh\n")
		fmt.Fprintf(b, "> %s set hsmmmesh.settings.mac2='%s'   # 10.%s -> %s\n", uciLocal, mac2, mac2, nodeIP)
		fmt.Fprintf(b, "> %s commit hsmmmesh\n", uciLocal)
		fmt.Fprintf(b, "> %s set setup.globals.wifi_ip='%s'\n", uciMesh, nodeIP)
		fmt.Fprintf(b, "> %s commit setup\n", uciMesh)
		b.WriteString("> ```\n")
		b.WriteString("> Then run the node's setup/apply (or reboot) so the change propagates to hosts\n")
		b.WriteString("> and DNS. *(Verified against `mesh-node:4.26.1.0`: `/etc/hosts` resolved to the\n")
		b.WriteString("> preserved IP after `node-setup`.)*\n\n")
	} else {
		b.WriteString("> `NODE_IP` is unexpectedly outside `10.0.0.0/8`; set `setup.globals.wifi_ip` manually.\n\n")
	}
}

// writeLocation emits runnable uci commands for any location values that are set.
func writeLocation(b *strings.Builder, cfg *config.Config) {
	var cmds [][2]string
	if cfg.Latitude != 0 {
		cmds = append(cmds, [2]string{"lat", fmt.Sprintf("%g", cfg.Latitude)})
	}
	if cfg.Longitude != 0 {
		cmds = append(cmds, [2]string{"lon", fmt.Sprintf("%g", cfg.Longitude)})
	}
	if cfg.Gridsquare != "" {
		cmds = append(cmds, [2]string{"gridsquare", cfg.Gridsquare})
	}
	if len(cmds) == 0 {
		return
	}
	b.WriteString("Location (or use **Settings → Location**):\n\n")
	b.WriteString("```sh\n")
	for _, kv := range cmds {
		fmt.Fprintf(b, "%s set aredn.@location[0].%s='%s'\n", uciMesh, kv[0], kv[1])
	}
	fmt.Fprintf(b, "%s commit aredn\n", uciMesh)
	b.WriteString("```\n\n")
}

// writeUsersSection maps mesh-manager admin accounts to AREDN®'s single root login.
func writeUsersSection(b *strings.Builder, users []models.User, check adminCheck) {
	b.WriteString("## 6. Users & authentication\n\n")
	b.WriteString("AREDN® has a **single admin** (the `root` login), whose web + system password is\n")
	b.WriteString("set with `setpasswd` (it stores an `/etc/shadow` MD5 crypt — not mesh-manager's\n")
	b.WriteString("argon2id, so the *hash* can't be copied, but the *password* you already use can).\n\n")

	if len(users) > 0 {
		b.WriteString("mesh-manager accounts found:\n\n")
		for _, u := range users {
			fmt.Fprintf(b, "- `%s`\n", u.Username)
		}
		b.WriteString("\nAll of these collapse onto AREDN®'s one `root` admin. On the new node, run\n")
		b.WriteString("(with the password you log into mesh-manager with):\n\n")
	} else {
		b.WriteString("No accounts found. On the new node set the admin password with:\n\n")
	}
	b.WriteString("```sh\n")
	b.WriteString("setpasswd '<your mesh-manager admin password>'\n")
	b.WriteString("```\n\n")

	switch check {
	case adminMatched:
		b.WriteString("> ✅ The `--admin-password` you supplied **matches** the stored admin hash — use\n")
		b.WriteString("> that exact password in the `setpasswd` command above.\n\n")
	case adminMismatch:
		b.WriteString("> ⚠️ The `--admin-password` you supplied did **not** match the stored admin hash.\n\n")
	case adminNotChecked:
		b.WriteString("> Tip: re-run with `--admin-password '<pw>'` to confirm a password matches the\n")
		b.WriteString("> stored hash before you rely on it. (The password is never written to this report.)\n\n")
	}
}

//nolint:gocyclo // linear per-tunnel emitter
func writeTunnelSection(b *strings.Builder, tunnels []models.Tunnel) {
	var hosts, clients, legacy []decodedTunnel
	for _, t := range tunnels {
		d := decodeTunnel(t)
		switch {
		case !t.Wireguard:
			legacy = append(legacy, d)
		case d.role == roleHost:
			hosts = append(hosts, d)
		default:
			clients = append(clients, d)
		}
	}

	b.WriteString("## 4. Tunnels\n\n")
	fmt.Fprintf(b, "Found **%d** tunnel(s): %d WireGuard client (dial-out), %d WireGuard host (server), %d legacy vtun.\n\n",
		len(tunnels), len(clients), len(hosts), len(legacy))
	b.WriteString("Role mapping (note AREDN®'s uci section type is the *inverse* of its UI label):\n\n")
	b.WriteString("- A mesh-manager **client (dial-out)** tunnel → AREDN® **\"WireGuard Client\"** UI\n")
	b.WriteString("  entry, stored as a uci `config server` section.\n")
	b.WriteString("- A mesh-manager **host (server)** tunnel → AREDN® **\"WireGuard Server\"** UI entry,\n")
	b.WriteString("  stored as a uci `config client` section.\n\n")
	b.WriteString("mesh-manager stores a WireGuard tunnel password as the concatenation\n")
	b.WriteString("`<server_pubkey><client_privkey><client_pubkey>` (3×44 chars) — this is exactly\n")
	b.WriteString("AREDN®'s tunnel exchange \"Wireguard key\", so the values below paste straight in.\n")
	b.WriteString("The `uci` commands are **verified against `mesh-node:4.26.1.0`** (run through\n")
	b.WriteString("`node-setup`, producing byte-identical interfaces). After running any of them,\n")
	b.WriteString("apply with `node-setup -a mesh` (or **Save Changes** in the web UI).\n\n")

	if len(legacy) > 0 {
		b.WriteString("### Legacy vtun tunnels (must be rebuilt)\n\n")
		b.WriteString("These are not WireGuard and cannot be auto-migrated. Recreate each as a WireGuard\n")
		b.WriteString("tunnel in AREDN®, coordinating with the partner node.\n\n")
		b.WriteString("| ID | Hostname | IP | Enabled |\n|---|---|---|---|\n")
		for _, d := range legacy {
			fmt.Fprintf(b, "| %d | `%s` | `%s` | %t |\n", d.ID, d.Hostname, d.IP, d.Enabled)
		}
		b.WriteString("\n")
	}

	if len(clients) > 0 {
		b.WriteString("### WireGuard client (dial-out) tunnels → AREDN® \"WireGuard Client\"\n\n")
		b.WriteString("Lowest risk: the far end is unchanged. In AREDN® open **Tunnels → Add as Client**\n")
		b.WriteString("and paste the three exchange lines below (this is the same blob you originally\n")
		b.WriteString("pasted to create the tunnel here).\n\n")
		for _, d := range clients {
			writeClientTunnel(b, d)
		}
	}

	if len(hosts) > 0 {
		b.WriteString("### WireGuard host (server) tunnels → AREDN® \"WireGuard Server\"\n\n")
		b.WriteString("Two options per tunnel:\n\n")
		b.WriteString("- **Option A (UI, simplest):** add a \"WireGuard Server\" in AREDN®, hand the\n")
		b.WriteString("  freshly-generated details to the remote node. New keys ⇒ the **remote must update**.\n")
		b.WriteString("- **Option B (uci, key-preserving):** write the existing keys into\n")
		b.WriteString("  `/etc/config.mesh/wireguard` so the remote node needs **no change**, then apply.\n\n")
		for _, d := range hosts {
			writeHostTunnel(b, d)
		}
	}
}

func writeClientTunnel(b *strings.Builder, d decodedTunnel) {
	fmt.Fprintf(b, "#### Tunnel #%d — to `%s`%s\n\n", d.ID, d.Hostname, disabledNote(d.Enabled))
	if !d.valid {
		b.WriteString("> Could not decode this tunnel's keys (unexpected password length). Skipping details.\n\n")
		return
	}
	b.WriteString("**Option A (UI):** paste into AREDN® **Add Tunnel → Client**:\n\n")
	b.WriteString("```\n")
	fmt.Fprintf(b, "Remote server name: %s\n", d.endpointHost)
	fmt.Fprintf(b, "Wireguard key: %s\n", d.Password)
	fmt.Fprintf(b, "Network:Port: %s:%s\n", d.remoteIP, d.endpointPort)
	b.WriteString("```\n\n")
	b.WriteString("**Option B (uci, for bulk)** — run on the new node:\n\n")
	// node-setup wgs block reads passwd=serverPub+clientPriv+clientPub and uses
	// client_priv as the interface key, server_pub as the peer, host:port as the endpoint.
	writeUCIAdd(b, "server", [][2]string{
		{"enabled", boolToUCI(d.Enabled)},
		{"host", d.endpointHost},
		{"passwd", d.Password},
		{"netip", d.remoteIP + ":" + d.endpointPort},
	})
	fmt.Fprintf(b, "This node's address on the tunnel is `%s`; the server is `%s`.\n\n", d.localIP, d.remoteIP)
}

func writeHostTunnel(b *strings.Builder, d decodedTunnel) {
	fmt.Fprintf(b, "#### Tunnel #%d — for client `%s`%s\n\n", d.ID, d.Hostname, disabledNote(d.Enabled))
	if !d.valid {
		b.WriteString("> Could not decode this tunnel's keys (unexpected password length). Skipping details.\n\n")
		return
	}
	fmt.Fprintf(b, "- Listen UDP port: `%d`\n", d.WireguardPort)
	fmt.Fprintf(b, "- This node (server) IP: `%s`; client IP: `%s`\n", d.localIP, d.remoteIP)
	fmt.Fprintf(b, "- Server private key: `%s`\n", d.serverPriv)
	fmt.Fprintf(b, "- Client public key: `%s`\n\n", d.clientPub)
	b.WriteString("**Option A (UI):** add a \"WireGuard Server\" for client `" + d.Hostname + "`, then send the\n")
	b.WriteString("AREDN®-generated exchange blob to the remote node (its keys will change).\n\n")
	b.WriteString("**Option B (uci, key-preserving)** — run on the new node:\n\n")
	// AREDN reads m1[1]=server_priv as the listen interface's key and m1[4]=client_pub
	// as the peer (node-setup wgc block). key = serverPriv+serverPub+clientPriv+clientPub.
	// clientip is this (listening) node's own tunnel IP and its listen port: node-setup
	// uses the IP directly as the wgc interface address and the port as listen_port.
	writeUCIAdd(b, "client", [][2]string{
		{"enabled", boolToUCI(d.Enabled)},
		{"name", d.Hostname},
		{"key", d.serverPriv + d.Password},
		{"clientip", fmt.Sprintf("%s:%d", d.localIP, d.WireguardPort)},
	})
}

func disabledNote(enabled bool) string {
	if enabled {
		return ""
	}
	return " *(disabled)*"
}

func boolToUCI(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func floatOrBlank(v float64) string {
	if v == 0 {
		return "—"
	}
	return fmt.Sprintf("%g", v)
}

func strOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
