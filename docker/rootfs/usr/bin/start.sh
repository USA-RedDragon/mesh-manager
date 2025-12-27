#!/bin/bash

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

iptables -I FORWARD 1 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -I INPUT 1 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -t nat -A POSTROUTING -o wg+ ! -d 255.255.255.255 -m addrtype --src-type LOCAL -j SNAT --to-source $NODE_IP

mkdir -p /etc/meshlink
echo "${NODE_IP} ${SERVER_NAME}" >> /etc/meshlink/hosts
if [ -n "$SUPERNODE" ]; then
    echo "${NODE_IP} supernode.${SERVER_NAME}.local.mesh" >> /etc/meshlink/hosts
fi
echo "${NODE_IP} dtdlink.${SERVER_NAME}.local.mesh" >> /etc/meshlink/hosts
echo "http://${SERVER_NAME}/|tcp|${SERVER_NAME}-console" >> /etc/meshlink/services

sleep 3

mesh-manager generate

# We need the syslog started early
rsyslogd -n &

cat <<EOF > /tmp/resolv.conf.auto
nameserver 127.0.0.11
options ndots:0
EOF

WALKER=${WALKER:-}
if [ -n "$WALKER" ]; then
    (crontab -l ; echo "30 * * * * /usr/bin/mesh-manager walk") | crontab -
    MESHMAP_APP_CONFIG=${MESHMAP_APP_CONFIG:-'{}'}
    echo -n "${MESHMAP_APP_CONFIG}" > /meshmap/appConfig.json
fi

# Use the dnsmasq that's about to run
echo -e 'search local.mesh\nnameserver 127.0.0.1' > /etc/resolv.conf

exec s6-svscan /etc/s6
