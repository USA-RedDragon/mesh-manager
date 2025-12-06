#/bin/bash

set -euxo pipefail

# Trap signals and exit
trap "exit 0" SIGHUP SIGINT SIGTERM

ip link add dev br-wan type bridge
ip link set dev eth0 master br-wan
ip link set br-wan up

ip link add dev br-dtdlink type bridge
ip link set dev eth1 master br-dtdlink
ip link set br-dtdlink up

GW=$(ip route list 0/0 | awk '{ print $3 }')
IP6_GW=$(ip -6 route list ::/0 | awk '{ print $3 }')

IPV4_ADDRS=$(ip address show eth0 | grep 'inet ' | awk '{ print $2 }')
for IPV4_ADDR in $IPV4_ADDRS; do
    ip address del dev eth0 $IPV4_ADDR
    ip address add dev br-wan $IPV4_ADDR
done
IPV6_ADDRS=$(ip address show eth0 | grep 'inet6 ' | awk '{ print $2 }')
for IPV6_ADDR in $IPV6_ADDRS; do
    # Make sure we don't add the link-local address
    if [[ $IPV6_ADDR == fe80:* ]]; then
        continue
    fi
    ip address del dev eth0 $IPV6_ADDR
    ip address add dev br-wan $IPV6_ADDR
done

mkdir -p /etc/iproute2/
echo "20 babel_in" >> /etc/iproute2/rt_tables
echo "21 babel_super" >> /etc/iproute2/rt_tables
echo "22 wan_in" >> /etc/iproute2/rt_tables
echo "27 local_wan" >> /etc/iproute2/rt_tables
echo "28 wan_out" >> /etc/iproute2/rt_tables
echo "29 local_lan" >> /etc/iproute2/rt_tables
echo "99 blackhole" >> /etc/iproute2/rt_tables

# Table 28 (Default Route)
ip route add default via $GW dev br-wan table 28
ip -6 route del default via $IP6_GW dev eth0
ip -6 route add default via $IP6_GW dev br-wan table 28

# Table 27 (Local WAN Subnet)
WAN_NET=$(ip route show dev br-wan | grep "proto kernel" | awk '{print $1}')
if [ -n "$WAN_NET" ]; then
    ip route add $WAN_NET dev br-wan table 27
fi

# Table 21 (Supernode blackhole)
SUPERNODE=${SUPERNODE:-}
if [ -n "$SUPERNODE" ]; then
    ip route add blackhole 10.0.0.0/8 table 21
fi

# Table 99 (Blackhole)
ip route add blackhole 0.0.0.0/0 table 99

ip address add dev br-dtdlink $NODE_IP/8

ip rule add pref 10 iif br-dtdlink lookup 29
ip rule add pref 20 iif br-dtdlink lookup 20
ip rule add pref 30 iif br-dtdlink lookup 21
# IF MESH TO INTERNET IS ENABLED
ip rule add pref 50 iif br-dtdlink lookup 28
ip rule add pref 60 iif br-dtdlink lookup 22
ip rule add pref 70 iif br-dtdlink lookup 99

ip rule add pref 110 lookup 29
ip rule add pref 120 lookup 20
ip rule add pref 130 lookup 21
ip rule add pref 140 lookup 27
ip rule add pref 150 lookup 28
ip rule add pref 160 lookup 22

mkdir -p /etc/meshlink
echo "${NODE_IP} ${SERVER_NAME}" >> /etc/meshlink/hosts
if [ -n "$SUPERNODE" ]; then
    echo "${NODE_IP} supernode.${SERVER_NAME}.local.mesh" >> /etc/meshlink/hosts
fi
echo "${NODE_IP} dtdlink.${SERVER_NAME}.local.mesh" >> /etc/meshlink/hosts
echo "http://${SERVER_NAME}/|tcp|${SERVER_NAME}-console" >> /etc/meshlink/services

# Firewall

iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X

iptables -P INPUT ACCEPT   # Start permissive to avoid lockout during setup
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

iptables -N ZONE_WAN_IN
iptables -N ZONE_DTD_IN
iptables -N ZONE_VPN_IN

iptables -N ZONE_WAN_FWD
iptables -N ZONE_DTD_FWD
iptables -N ZONE_VPN_FWD

iptables -A INPUT -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT -i br-wan -j ZONE_WAN_IN
iptables -A INPUT -i br-dtdlink -j ZONE_DTD_IN
iptables -A INPUT -i wgs+ -j ZONE_VPN_IN  # WireGuard Servers
iptables -A INPUT -i wgc+ -j ZONE_VPN_IN  # WireGuard Clients

iptables -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables -A FORWARD -i br-wan -j ZONE_WAN_FWD
iptables -A FORWARD -i br-dtdlink -j ZONE_DTD_FWD
iptables -A FORWARD -i wgs+ -j ZONE_VPN_FWD
iptables -A FORWARD -i wgc+ -j ZONE_VPN_FWD

# WAN Zone
# Allow Ping
iptables -A ZONE_WAN_IN -p icmp --icmp-type echo-request -j ACCEPT
iptables -A ZONE_WAN_IN -p tcp -m multiport --dports 80,8080 -j ACCEPT
# We need to allow WIREGUARD_STARTING_PORT to WIREGUARD_STARTING_PORT+100 (udo)
iptables -A ZONE_WAN_IN -p udp -m multiport --dports ${WIREGUARD_STARTING_PORT}:$((${WIREGUARD_STARTING_PORT}+100)) -j ACCEPT
# Drop everything else
iptables -A ZONE_WAN_IN -j REJECT --reject-with icmp-host-prohibited

# NAT: Masquerade outbound WAN traffic (Internet Access)
iptables -t nat -A POSTROUTING -o br-wan -j MASQUERADE

# DTD Zone
# Allow Ping
iptables -A ZONE_DTD_IN -p icmp --icmp-type echo-request -j ACCEPT
# Allow Management: HTTP (80/8080)
iptables -A ZONE_DTD_IN -p tcp -m multiport --dports 80,8080 -j ACCEPT
# Allow Routing/Mesh: OLSR (698), Babel (6696)
iptables -A ZONE_DTD_IN -p udp -m multiport --dports 698,6696 -j ACCEPT
# Allow DNS (53) if Supernode (13-supernode-rules)
if [ -n "$SUPERNODE" ]; then
    iptables -A ZONE_DTD_IN -p udp --dport 53 -j ACCEPT
    iptables -A ZONE_DTD_IN -p tcp --dport 53 -j ACCEPT
fi
# Drop everything else
iptables -A ZONE_DTD_IN -j REJECT --reject-with icmp-host-prohibited

# VPN Zone (WireGuard)
iptables -A ZONE_VPN_IN -p icmp --icmp-type echo-request -j ACCEPT
iptables -A ZONE_VPN_IN -p tcp -m multiport --dports 80,8080 -j ACCEPT
iptables -A ZONE_VPN_IN -p udp -m multiport --dports 698,6696 -j ACCEPT
if [ -n "$SUPERNODE" ]; then
    iptables -A ZONE_VPN_IN -p udp --dport 53 -j ACCEPT
    iptables -A ZONE_VPN_IN -p tcp --dport 53 -j ACCEPT
fi
iptables -A ZONE_VPN_IN -j REJECT --reject-with icmp-host-prohibited

iptables -t nat -A POSTROUTING -o wgs+ -j SNAT --to-source $NODE_IP
iptables -t nat -A POSTROUTING -o wgc+ -j SNAT --to-source $NODE_IP

# Allow DtD <-> VPN
iptables -A ZONE_DTD_FWD -o wgs+ -j ACCEPT
iptables -A ZONE_DTD_FWD -o wgc+ -j ACCEPT
iptables -A ZONE_VPN_FWD -o br-dtdlink -j ACCEPT

# Allow VPN <-> VPN
iptables -A ZONE_VPN_FWD -o wgs+ -j ACCEPT
iptables -A ZONE_VPN_FWD -o wgc+ -j ACCEPT

iptables -P INPUT DROP

sleep 3

mesh-manager generate

# We need the syslog started early
rsyslogd -n &

cat <<EOF > /tmp/resolv.conf.auto
nameserver 127.0.0.11
options ndots:0
EOF

# Use the dnsmasq that's about to run
echo -e 'search local.mesh\nnameserver 127.0.0.1' > /etc/resolv.conf

exec s6-svscan /etc/s6
