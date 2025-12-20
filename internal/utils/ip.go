package utils

import (
	"fmt"
	"net"
)

// Generate a link-local address beginning with fe80 and ending with the 4 octets of the IPv4 address
// The generation logic follows the upstream wireguard-tools implementation:
// 1. Create a pseudo-MAC address: 00:00:IPv4
// 2. Convert to EUI-64 IPv6 Link-Local address
func GenerateIPv6LinkLocalAddress(ipv4 net.IP) (string, error) {
	v4Bytes := ipv4.To4()
	if v4Bytes == nil {
		return "", fmt.Errorf("invalid IPv4 address")
	}

	// Construct the IPv6 address bytes
	// Prefix: fe80::/64
	ipv6Bytes := make([]byte, 16)
	ipv6Bytes[0] = 0xfe
	ipv6Bytes[1] = 0x80

	// Interface ID derived from pseudo-MAC 00:00:v4[0]:v4[1]:v4[2]:v4[3]
	// Modified EUI-64: Flip 7th bit of first byte of MAC, insert ff:fe in middle

	// MAC[0] is 0x00. Modified EUI-64 flips the universal/local bit (bit 1, value 0x02).
	ipv6Bytes[8] = 0x00 ^ 0x02
	// MAC[1] is 0x00.
	ipv6Bytes[9] = 0x00
	// MAC[2] is v4Bytes[0]
	ipv6Bytes[10] = v4Bytes[0]
	// Insert 0xff 0xfe
	ipv6Bytes[11] = 0xff
	ipv6Bytes[12] = 0xfe
	// MAC[3] is v4Bytes[1]
	ipv6Bytes[13] = v4Bytes[1]
	// MAC[4] is v4Bytes[2]
	ipv6Bytes[14] = v4Bytes[2]
	// MAC[5] is v4Bytes[3]
	ipv6Bytes[15] = v4Bytes[3]

	ipv6 := net.IP(ipv6Bytes)
	return ipv6.String(), nil
}
