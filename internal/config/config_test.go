package config_test

import (
	"errors"
	"testing"

	"github.com/USA-RedDragon/configulator/v2"
	"github.com/USA-RedDragon/mesh-manager/internal/config"
)

func TestLogLevelConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		logLevel config.LogLevel
		valid    bool
	}{
		{"debug level", config.LogLevelDebug, true},
		{"info level", config.LogLevelInfo, true},
		{"warn level", config.LogLevelWarn, true},
		{"error level", config.LogLevelError, true},
		{"invalid level", "invalid", false},
	}

	defConfig, err := configulator.New(config.ConfigSchema()).Default()
	if err != nil {
		t.Fatalf("failed to create default config: %v", err)
	}
	defConfig.PasswordSalt = "test-salt"
	defConfig.ServerName = "test-server"
	defConfig.NodeIP = "10.0.0.0"
	defConfig.Wireguard.StartingAddress = "171.31.0.0"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defConfig
			cfg.LogLevel = tt.logLevel
			err := cfg.Validate()
			if tt.valid {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else if !errors.Is(err, config.ErrInvalidLogLevel) {
				t.Errorf("Validate() error = %v, want %v", err, config.ErrInvalidLogLevel)
			}
		})
	}
}
