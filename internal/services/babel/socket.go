package babel

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"
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

// FetchInstalledRouteMetrics returns a map of IPv4 CIDR prefixes (including /32) to Babel metric (ETX scaled by 256) for installed routes.
// Unreachable routes (metric 65535) and non-IPv4 routes are ignored; for destinations with multiple routes the smallest metric is kept.
func FetchInstalledRouteMetrics(ctx context.Context) (map[string]int, error) {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to babel socket: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	scanner := bufio.NewScanner(conn)

	// Consume banner until "ok"
	for scanner.Scan() {
		if scanner.Text() == "ok" {
			break
		}
	}

	if _, err := conn.Write([]byte("dump-installed-routes\n")); err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	routeRegex := regexp.MustCompile(`^add route .+ prefix ([^ /]+)/([0-9]+) .* installed yes .* metric ([0-9]+)`) // keep spacing to mirror babel output
	etxByPrefix := make(map[string]int)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "ok" {
			break
		}

		matches := routeRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		metric, err := strconv.Atoi(matches[3])
		if err != nil {
			continue
		}

		if metric == 65535 {
			continue
		}

		cidr := matches[1] + "/" + matches[2]
		ip, netw, err := net.ParseCIDR(cidr)
		if err != nil || ip == nil || ip.To4() == nil || netw == nil {
			continue
		}

		key := netw.String()
		if existing, ok := etxByPrefix[key]; ok {
			if metric < existing {
				etxByPrefix[key] = metric
			}
			continue
		}

		etxByPrefix[key] = metric
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return etxByPrefix, nil
}
