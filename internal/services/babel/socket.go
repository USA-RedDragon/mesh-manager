package babel

import (
	"context"
	"fmt"
	"net"
)

const (
	socketPath = "/var/run/babel.sock"
)

func (s *Service) AddTunnel(ctx context.Context, iface string) error {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer conn.Close()

	tun := []byte(GenerateTunnelLine(iface, s.config.Supernode))
	n, err := conn.Write(tun)
	if err != nil {
		return fmt.Errorf("failed to write to socket: %w", err)
	}
	if n != len(tun) {
		return fmt.Errorf("failed to write all bytes to socket")
	}

	return nil
}

func (s *Service) RemoveTunnel(ctx context.Context, iface string) error {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer conn.Close()

	tun := []byte("flush interface " + iface + "\n")
	n, err := conn.Write(tun)
	if err != nil {
		return fmt.Errorf("failed to write to socket: %w", err)
	}
	if n != len(tun) {
		return fmt.Errorf("failed to write all bytes to socket")
	}

	return nil
}
