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

ip route add default via $GW dev br-wan table 28
ip -6 route del default via $IP6_GW dev eth0
ip -6 route add default via $IP6_GW dev br-wan table 28

SUPERNODE=${SUPERNODE:-}
if [ -n "$SUPERNODE" ]; then
    ip route add blackhole 10.0.0.0/8 table 21
fi
ip address add dev br-dtdlink $NODE_IP/8

ip rule add pref 20010 iif br-dtdlink lookup 29
ip rule add pref 20020 iif br-dtdlink lookup 20
ip rule add pref 20030 iif br-dtdlink lookup 30
ip rule add pref 20040 iif br-dtdlink lookup 21
ip rule add pref 20050 iif br-dtdlink lookup 22
ip rule add pref 20060 iif br-dtdlink lookup 28
ip rule add pref 20070 iif br-dtdlink lookup 31
ip rule add pref 20099 iif br-dtdlink unreachable

ip rule add pref 30210 lookup 29
ip rule add pref 30220 lookup 20
ip rule add pref 30230 lookup 30
ip rule add pref 30240 lookup 21
ip rule add pref 30260 lookup main
# ip rule add pref 30270 lookup 22
ip rule add pref 30280 lookup 28
ip rule add pref 30290 lookup 31

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

# Use the dnsmasq that's about to run
echo -e 'search local.mesh\nnameserver 127.0.0.1' > /etc/resolv.conf

exec s6-svscan /etc/s6
