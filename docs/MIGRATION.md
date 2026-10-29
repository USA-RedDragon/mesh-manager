# Migrate from mesh-manager to AREDN® firmware

mesh-manager is deprecated. This guide tells you how to move a mesh-manager node to official, unmodified AREDN® firmware. You can run the firmware in a virtual machine (VM) or on bare metal. The node keeps its name, mesh IP, tunnels, WireGuard keys and UDP ports. Remote tunnel partners do not need to change anything.

This procedure moved two production nodes (one normal node and one supernode) from mesh-manager to AREDN® 4.26 firmware in KVM VMs, one VM per node, with all tunnels preserved.

> AREDN® is a registered trademark of Amateur Radio Emergency Data Network, Inc. This project is not affiliated with, endorsed by, or sanctioned by AREDN, Inc.

## Summary

1. Back up the mesh-manager database.
2. Run `mesh-manager migrate` to get a report for your node.
3. Install AREDN® firmware on a VM or a device.
4. Run the commands from the report on the new node, then reboot.
5. Move the public address and UDP port forwards to the new node.
6. If you are on a 4.26 release earlier than 4.26.10.0, prepare the WAN port before you upgrade (see [Upgrade to 4.26.10.0](#upgrade-to-426100)).

## What mesh-manager stores

The database has three tables:

| Table | Contents | Migrate? |
|---|---|---|
| `tunnels` | WireGuard (and legacy vtun) tunnels and their keys | Yes. This is the important part. |
| `users` | mesh-manager admin logins (salted argon2id hash) | No. AREDN® has one admin login. |
| `app_settings` | Internal "has seeded" flag | No. |

All other state is node configuration from environment variables, flags or `config.yaml` (see [`internal/config/config.go`](../internal/config/config.go)): node name, mesh IP, supernode flag, location, routing protocol and WireGuard address and port ranges.

A migration must keep three things: the node identity, the tunnels with their keys, and the supernode role.

## Step 1: Back up

Back up the PostgreSQL database and your `config.yaml` or environment. For example:

```sh
docker compose exec -T postgres pg_dump -U mesh_manager mesh_manager | zstd > mesh-manager.sql.zst
```

Do not delete the mesh-manager containers or database until the new node works.

## Step 2: Generate the migration report

No release contains the `migrate` subcommand, so build it from the `main` branch. The binary is static, so you can copy it into the running container and run it there with the node's own configuration and database:

```sh
git clone https://github.com/USA-RedDragon/mesh-manager
cd mesh-manager
CGO_ENABLED=0 go build -o mesh-manager-migrate .
docker compose cp mesh-manager-migrate cloud-tunnel:/tmp/mesh-manager-migrate
docker compose exec cloud-tunnel /tmp/mesh-manager-migrate migrate -o /tmp/migration.md
docker compose cp cloud-tunnel:/tmp/migration.md ./migration.md
```

Use your own service name in place of `cloud-tunnel`.

The report has the node identity, the routing and supernode settings, every tunnel with ready-to-run commands, and the known parity gaps. Keep the report secret. It contains WireGuard private keys.

To check that you remember the admin password, add `--admin-password '<password>'`. The tool checks the password against the stored hash and does not write it to the report.

## Step 3: Install AREDN® firmware

Install the current AREDN® release. For a VM, use the x86-64 image. Give each node its own VM.

A VM with two network interfaces works well:

- `eth0`: device-to-device (DtD) and LAN
- `eth1`: WAN (internet)

Do not rely on the default port layout. Set the WAN port yourself, as shown in [WAN on a second interface](#wan-on-a-second-interface).

## Step 4: Configure the new node

Run the commands from the report on the new node, in this order. Then reboot. At boot, AREDN® applies the staged configuration in `/etc/config.mesh`.

AREDN® uses two uci directories. `hsmmmesh` is in `/etc/local/uci`. The staged node configuration (`setup`, `aredn`, `wireguard`) is in `/etc/config.mesh`. Always give the `-c` option shown below.

### Mesh IP

mesh-manager's `NODE_IP` is always in `10.0.0.0/8`. AREDN® makes the mesh IP `10.<mac2>` from `hsmmmesh.settings.mac2` and applies it as `setup.globals.wifi_ip`. To keep the address, set both before the first configuration. For example, for `NODE_IP=10.54.27.2`:

```sh
uci -c /etc/local/uci set hsmmmesh.settings.mac2='54.27.2'
uci -c /etc/local/uci commit hsmmmesh
uci -c /etc/config.mesh set setup.globals.wifi_ip='10.54.27.2'
uci -c /etc/config.mesh commit setup
```

### Node name and admin password

Run AREDN®'s first-use setup. It sets the node name and the admin password, and marks the node as configured:

```sh
firstuse_setup '<SERVER_NAME>' '<your mesh-manager admin password>'
```

It prints the new mesh IP. Make sure it matches `NODE_IP`.

AREDN® has one admin login (`root`). All mesh-manager users become this one login. AREDN® stores an MD5 crypt hash and mesh-manager stores argon2id, so you cannot copy the hash. Use the password you already know. To change it later, use `setpasswd`.

### Location

| mesh-manager | AREDN® uci (`/etc/config.mesh`) |
|---|---|
| `LATITUDE` | `aredn.@location[0].lat` |
| `LONGITUDE` | `aredn.@location[0].lon` |
| `GRIDSQUARE` | `aredn.@location[0].gridsquare` |

You can also set these in the web UI.

### Routing

mesh-manager can run OLSR or Babel. Current AREDN® uses Babel and manages it. Nothing carries over: AREDN® makes its own Babel configuration and router ID. OLSR is legacy. If your node used only OLSR, your neighbors must also use Babel.

### Supernode role

If mesh-manager had `SUPERNODE=true`, enable the role with uci. There is no web UI switch:

```sh
uci -c /etc/config.mesh set aredn.@supernode[0].enable='1'
uci -c /etc/config.mesh commit aredn
```

Supernode tunnels use `172.30.x.x` and UDP ports from `6526`. Normal tunnels use `172.31.x.x` and ports from `5525`.

### Tunnels

AREDN® supports only WireGuard tunnels. mesh-manager refuses to make new vtun tunnels, but an old database can still have them (`wireguard = false`). These have no WireGuard keys. Rebuild each one as a WireGuard tunnel with the partner node. The report lists them separately.

#### The password is the AREDN® key

mesh-manager stores a WireGuard tunnel's keys in the `Password` field. The field is three 44-character base64 keys in a row (132 characters):

```text
Password = <server_pubkey><client_privkey><client_pubkey>
```

This is the same value that AREDN® uses as the "Wireguard key" in its tunnel exchange. See [`internal/wireguard/wireguard.go`](../internal/wireguard/wireguard.go) and [`internal/server/api/controllers/v1/tunnels.go`](../internal/server/api/controllers/v1/tunnels.go).

AREDN®'s uci section type is the opposite of its UI label:

| mesh-manager tunnel | Meaning | AREDN® UI | AREDN® uci section |
|---|---|---|---|
| `Client = true`, no `WireguardServerKey` | This node dials out to a remote server | WireGuard Client | `config server` (`wgs` interface) |
| `Client = false`, `WireguardServerKey` set | This node listens for a remote client | WireGuard Server | `config client` (`wgc` interface) |

#### Dial-out tunnels

The remote server does not change. The report gives the values for the web UI (Add Tunnel, Client) and the same tunnel as uci commands:

```sh
uci -c /etc/config.mesh add wireguard server
uci -c /etc/config.mesh set wireguard.@server[-1].enabled='1'
uci -c /etc/config.mesh set wireguard.@server[-1].host='<remote host>'
uci -c /etc/config.mesh set wireguard.@server[-1].passwd='<Password>'
uci -c /etc/config.mesh set wireguard.@server[-1].netip='<tunnel IP>:<remote port>'
uci -c /etc/config.mesh commit wireguard
```

The server IP is the tunnel's `IP`. This node's IP is the tunnel's `IP` plus one, the same as in mesh-manager.

#### Listening tunnels

mesh-manager keeps the server private key in a separate column (`WireguardServerKey`). The AREDN® key for a listening tunnel is four keys:

```text
AREDN key = WireguardServerKey + Password
          = <server_priv><server_pub><client_priv><client_pub>
```

You have two options:

- **Option A (web UI):** Add a WireGuard Server in AREDN®. AREDN® makes new keys, so you must send the new details to the remote node. Use this only if you can coordinate with the partner.
- **Option B (uci, keeps keys):** Write the existing keys and port into the configuration. The remote node does not change. The report has the commands for each tunnel:

```sh
uci -c /etc/config.mesh add wireguard client
uci -c /etc/config.mesh set wireguard.@client[-1].enabled='1'
uci -c /etc/config.mesh set wireguard.@client[-1].name='<remote node name>'
uci -c /etc/config.mesh set wireguard.@client[-1].key='<WireguardServerKey + Password>'
uci -c /etc/config.mesh set wireguard.@client[-1].clientip='<tunnel IP>:<WireguardPort>'
uci -c /etc/config.mesh commit wireguard
```

`clientip` is this node's own tunnel IP and its listen port. AREDN® uses the IP as the interface address and the port as the listen port.

#### How the uci commands were checked

The uci commands were run through AREDN® 4.26.1.0 `node-setup`. The WireGuard interfaces in the generated `/etc/config/network` match what mesh-manager configures:

| Field | mesh-manager | AREDN® `node-setup` output |
|---|---|---|
| Dial-out private key | `Password[44:88]` (client private key) | `wgs0.private_key` |
| Dial-out peer key | `Password[:44]` (server public key) | `wireguard_wgs0.public_key` |
| Dial-out local IP | tunnel `IP` plus one | `wgs0.addresses` |
| Listening private key | `WireguardServerKey` | `wgc0.private_key` |
| Listening peer key | `Password[88:]` (client public key) | `wireguard_wgc0.public_key` |
| Listening IP and port | tunnel `IP` and `WireguardPort` | `wgc0.addresses` and `wgc0.listen_port` |

The same commands then moved the production nodes with all tunnels, keys and ports kept.

#### New tunnels after the migration

AREDN® 4.26.10.0 offers only unused ports when you add a new tunnel, so new tunnels do not collide with migrated ones. On earlier releases, check that the port of a new tunnel is not in use.

### Advertised services

mesh-manager advertises only the node's own web console (see [`docker/rootfs/usr/bin/start.sh`](../docker/rootfs/usr/bin/start.sh)). It has no UI to add services. AREDN® advertises its console itself. Add other services in the AREDN® web UI.

## Step 5: Cut over

1. Stop the mesh-manager containers.
2. Point the node's public hostname and the UDP port forwards for the listening tunnels (the `WireguardPort` values in the report) to the WAN address of the new node.
3. Connect the DtD network to the new node, if you use one.
4. Make sure that each tunnel connects. Make sure that the node shows its neighbors and the mesh routes.

## WAN on a second interface

### AREDN® releases before 4.26.10.0

Set the WAN interface and, if you need it, a static WAN address:

```sh
uci -c /etc/config.mesh set setup.globals.wan_intf='eth1'
uci -c /etc/config.mesh set setup.globals.wan_proto='static'
uci -c /etc/config.mesh set setup.globals.wan_ip='<address>'
uci -c /etc/config.mesh set setup.globals.wan_mask='<netmask>'
uci -c /etc/config.mesh set setup.globals.wan_gw='<gateway>'
uci -c /etc/config.mesh commit setup
```

Leave out the `wan_proto`, `wan_ip`, `wan_mask` and `wan_gw` lines to use DHCP on the WAN.

### AREDN® 4.26.10.0 and later

In 4.26.10.0, `wan_intf` is a list of ports, not a single value. The static address options are the same. We did not test a fresh 4.26.10.0 install. From the 4.26.10.0 source, this sets the WAN port:

```sh
uci -c /etc/config.mesh delete setup.globals.wan_intf
uci -c /etc/config.mesh add_list setup.globals.wan_intf='eth1'
uci -c /etc/config.mesh commit setup
```

## Upgrade to 4.26.10.0

Do this before you upgrade a node with two or more Ethernet ports from an earlier 4.26 release. If you do not, the node loses its WAN after the upgrade.

On first boot, 4.26.10.0 runs `/etc/uci-defaults/97_simplify_ports`. On a device with two or more Ethernet ports, it ignores the old `wan_intf` value. It reads the WAN ports only from `/etc/aredn_include/wan.network.user`. If that file does not exist, the node gets the board's default WAN port, which is not always `eth1`.

1. Take a backup of the node.
2. Copy the node's current WAN block from `/etc/config/network` to `/etc/aredn_include/wan.network.user`. The block is in the `#### WAN configuration` section. It has the `bridge-vlan` with `eth1:u`, the `br-wan` device and the `wan` interface.
3. Run `node-setup`. It sets the reboot-required flag.
4. Reboot.
5. Upgrade to 4.26.10.0.

After the upgrade, `setup.globals.wan_intf` is `['eth1']` and the migration deletes `wan.network.user`.

## Metrics

mesh-manager exported Prometheus metrics through node-exporter (`METRICS_ENABLED`). AREDN® serves Prometheus metrics at `/cgi-bin/metrics` on the node. It is always on and has no switch. Change your Prometheus scrape target to `http://<node>/cgi-bin/metrics`.

## Mesh map

mesh-manager had a built-in mesh walker and mesh map. The `meshmap` AREDN® package from [mesh-walker](https://github.com/USA-RedDragon/mesh-walker) replaces them. Install the package on the node.

## Parity gaps

- **Legacy vtun tunnels:** You cannot migrate them. Rebuild them as WireGuard tunnels with the partner.
- **Listening tunnels made in the web UI:** These get new keys, so the partner must update. Use Option B to keep the keys.
- **More than one admin login:** AREDN® has one `root` admin.
