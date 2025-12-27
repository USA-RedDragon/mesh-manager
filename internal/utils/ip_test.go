package utils

import (
	"net"
	"testing"
)

func TestGenerateIPv6LinkLocalAddress(t *testing.T) {
	tests := []struct {
		name     string
		ipv4     string
		expected string
		wantErr  bool
	}{
		{
			name:     "192.168.1.1",
			ipv4:     "192.168.1.1",
			expected: "fe80::200:c0ff:fea8:101",
			wantErr:  false,
		},
		{
			name:     "10.0.0.1",
			ipv4:     "10.0.0.1",
			expected: "fe80::200:aff:fe00:1",
			wantErr:  false,
		},
		{
			name:     "Invalid IP",
			ipv4:     "256.0.0.1",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipv4 := net.ParseIP(tt.ipv4)
			// ParseIP returns nil for invalid IP, but we need to handle the case where we pass something that isn't an IP string if we want to test ParseIP failure, but here we are testing GenerateIPv6LinkLocalAddress which takes net.IP.
			// If net.ParseIP returns nil, GenerateIPv6LinkLocalAddress should handle it (it calls To4() which returns nil on nil receiver? No, To4 on nil IP returns nil).

			// If ipv4 is nil (invalid parse), we pass it.

			got, err := GenerateIPv6LinkLocalAddress(ipv4)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateIPv6LinkLocalAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("GenerateIPv6LinkLocalAddress() = %v, want %v", got, tt.expected)
			}
		})
	}
}
