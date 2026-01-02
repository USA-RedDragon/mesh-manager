package apimodels

import (
	"regexp"
	"time"
)

const minHostnameLength = 3
const maxHostnameLength = 63

type CreateTunnel struct {
	Wireguard bool   `json:"wireguard"`
	Hostname  string `json:"hostname" binding:"required"`
	Password  string `json:"password"`
	IP        string `json:"ip"`
	Client    bool   `json:"client"`
}

func (r *CreateTunnel) IsValidHostname() (bool, string) {
	if len(r.Hostname) < minHostnameLength {
		return false, "Hostname must be at least 3 characters"
	}
	if len(r.Hostname) > maxHostnameLength {
		return false, "Hostname must be less than 64 characters"
	}
	if !regexp.MustCompile(`^[A-Za-z0-9\-]+$`).MatchString(r.Hostname) {
		return false, "Hostname must be alphanumeric or -"
	}
	return true, ""
}

type TunnelWithPass struct {
	ID             uint      `json:"id"`
	Enabled        bool      `json:"enabled"`
	Wireguard      bool      `json:"wireguard"`
	WireguardPort  uint16    `json:"wireguard_port"`
	Client         bool      `json:"client"`
	Hostname       string    `json:"hostname"`
	IP             string    `json:"ip"`
	Password       string    `json:"password"`
	Active         bool      `json:"active"`
	ConnectionTime time.Time `json:"connection_time"`
	CreatedAt      time.Time `json:"created_at"`
}

type TunnelLQMResponse struct {
	ID        uint     `json:"id"`
	IP        string   `json:"ip,omitempty"`
	Hostname  string   `json:"hostname,omitempty"`
	RXQuality *int     `json:"rx_quality"`
	Quality   *int     `json:"quality"`
	RTT       *float64 `json:"rtt"`
	Errors    *int     `json:"errors"`
}

type EditTunnel struct {
	ID        uint   `json:"id" binding:"required"`
	Enabled   *bool  `json:"enabled" binding:"required"`
	Wireguard *bool  `json:"wireguard" binding:"required"`
	Hostname  string `json:"hostname" binding:"required"`
	Password  string `json:"password"`
	IP        string `json:"ip" binding:"required"`
}
