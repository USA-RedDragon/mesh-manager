package config

import (
	"errors"
	"net"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type PProf struct {
	Enabled bool `name:"enabled" description:"Enable pprof debugging" default:"false"`
}

type Postgres struct {
	Host     string `name:"host" description:"PostgreSQL host"`
	Port     int    `name:"port" description:"PostgreSQL port" default:"5432"`
	User     string `name:"user" description:"PostgreSQL user"`
	Password string `name:"password" description:"PostgreSQL password"`
	Database string `name:"database" description:"PostgreSQL database"`
}

type Babel struct {
	Enabled  bool   `name:"enabled" description:"Enable Babel routing" default:"false"`
	RouterID string `name:"router-id" description:"Babel router ID"`
}

type Metrics struct {
	Enabled          bool   `name:"enabled" description:"Enable Prometheus metrics"`
	NodeExporterHost string `name:"node-exporter-host" description:"Node exporter host for Prometheus metrics" default:"node-exporter"`
	Port             int    `name:"port" description:"Port for Prometheus metrics" default:"9100"`
}

type Wireguard struct {
	StartingAddress string `name:"starting-address" description:"Starting address for Wireguard"`
	StartingPort    uint16 `name:"starting-port" description:"Starting port for Wireguard" default:"5527"`
}

type Config struct {
	LogLevel                 LogLevel  `name:"log-level" description:"Logging level for the application. One of debug, info, warn, or error" default:"info"`
	Port                     int       `name:"port" description:"Port to listen on for HTTP requests" default:"3333"`
	PasswordSalt             string    `name:"password-salt" description:"Salt used for password hashing"`
	PProf                    PProf     `name:"pprof" description:"pprof debugging settings"`
	Postgres                 Postgres  `name:"postgres" description:"PostgreSQL settings"`
	InitialAdminUserPassword string    `name:"initial-admin-user-password" description:"Initial password for the admin user"`
	Babel                    Babel     `name:"babel" description:"Babel routing settings"`
	OLSR                     bool      `name:"olsr" description:"Enable OLSR routing" default:"true"`
	CORSHosts                []string  `name:"cors-hosts" description:"CORS hosts for the API"`
	TrustedProxies           []string  `name:"trusted-proxies" description:"Trusted proxies for the API"`
	HIBPAPIKey               string    `name:"hibp-api-key" description:"Have I Been Pwned API key"`
	ServerName               string    `name:"server-name" description:"Server name"`
	Supernode                bool      `name:"supernode" description:"Enable supernode mode"`
	NodeIP                   string    `name:"node-ip" description:"Node IP address"`
	Latitude                 float64   `name:"latitude" description:"Server latitude"`
	Longitude                float64   `name:"longitude" description:"Server longitude"`
	Gridsquare               string    `name:"gridsquare" description:"Server gridsquare"`
	Metrics                  Metrics   `name:"metrics" description:"Metrics settings"`
	Wireguard                Wireguard `name:"wireguard" description:"Wireguard settings"`
	SessionSecret            string    `name:"session-secret" description:"Session secret"`
	LQM                      LQM       `name:"lqm" description:"Link Quality Monitoring settings"`
	Walker                   bool      `name:"walker" description:"Enable periodic mesh walking to update meshmap" default:"false"`
}

type LQM struct {
	Enabled bool `name:"enabled" description:"Enable Link Quality Monitoring" default:"true"`
}

var (
	ErrInvalidLogLevel                  = errors.New("invalid log level provided")
	ErrBabelRouterIDRequired            = errors.New("babel router ID is required when Babel is enabled")
	ErrNodeIPRequired                   = errors.New("node IP is required")
	ErrNodeIPInvalid                    = errors.New("node IP is invalid")
	ErrNodeIPNot10_8                    = errors.New("node IP is not in the 10.0.0.0/8 range")
	ErrPasswordSaltRequired             = errors.New("password salt is required")
	ErrServerNameRequired               = errors.New("server name is required")
	ErrWireguardStartingAddressRequired = errors.New("wireguard starting address is required")
	ErrWireguardStartingAddressInvalid  = errors.New("wireguard starting address is invalid")
	ErrWireguardStartingPortRequired    = errors.New("wireguard starting port is required")
	ErrWireguardStartingPortInvalid     = errors.New("wireguard starting port is invalid")
	ErrMetricsPortRequired              = errors.New("metrics port is required")
	ErrMetricsPortInvalid               = errors.New("metrics port is invalid")
	ErrMetricsNodeExporterHostRequired  = errors.New("node exporter host is required")
)

func (c Config) Validate() error {
	if c.LogLevel != LogLevelDebug &&
		c.LogLevel != LogLevelInfo &&
		c.LogLevel != LogLevelWarn &&
		c.LogLevel != LogLevelError {
		return ErrInvalidLogLevel
	}

	if c.PasswordSalt == "" {
		return ErrPasswordSaltRequired
	}

	if c.Babel.Enabled && c.Babel.RouterID == "" {
		return ErrBabelRouterIDRequired
	}

	if c.Metrics.Enabled {
		if c.Metrics.Port == 0 {
			return ErrMetricsPortRequired
		}
		if c.Metrics.Port < 1 || c.Metrics.Port > 65535 {
			return ErrMetricsPortInvalid
		}
		if c.Metrics.NodeExporterHost == "" {
			return ErrMetricsNodeExporterHostRequired
		}
	}

	if c.ServerName == "" {
		return ErrServerNameRequired
	}

	if c.NodeIP == "" {
		return ErrNodeIPRequired
	}

	if c.Wireguard.StartingAddress == "" {
		return ErrWireguardStartingAddressRequired
	}

	ip := net.ParseIP(c.Wireguard.StartingAddress)

	if ip == nil {
		return ErrWireguardStartingAddressInvalid
	}

	if ip.To4() == nil {
		return ErrWireguardStartingAddressInvalid
	}

	if c.Wireguard.StartingPort == 0 {
		return ErrWireguardStartingPortRequired
	}

	if c.Wireguard.StartingPort < 1024 {
		return ErrWireguardStartingPortInvalid
	}

	ip = net.ParseIP(c.NodeIP)

	if ip == nil {
		return ErrNodeIPInvalid
	}

	if ip.To4() == nil {
		return ErrNodeIPInvalid
	}

	if ip.To4()[0] != 10 {
		return ErrNodeIPNot10_8
	}

	return nil
}
