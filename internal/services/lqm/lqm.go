package lqm

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/vishvananda/netlink"
	"golang.org/x/sync/semaphore"
)

const (
	refreshTimeoutBase  = 12 * 60 * time.Second
	refreshTimeoutRange = 5 * 60 * time.Second
	refreshRetryTimeout = 5 * 60 * time.Second
	lastSeenTimeout     = 24 * time.Hour
	txQualityRunAvg     = 0.4
	lqRunAvg            = 0.4
	pingTimeout         = 1.0 * time.Second
	pingTimeRunAvg      = 0.4
	dtdDistance         = 50 // meters
	connectTimeout      = 5 * time.Second
	defaultMaxDistance  = 80550 // meters
	pingPenalty         = 5
	lastUpMargin        = 60 * time.Second
	babelSocketPath     = "/var/run/babel.sock"
	lqmInfoPath         = "/tmp/lqm.info"
)

type SysinfoResponse struct {
	Node        string             `json:"node"`
	Lat         any                `json:"lat"`
	Lon         any                `json:"lon"`
	NodeDetails SysinfoNodeDetails `json:"node_details"`
	Interfaces  []SysinfoInterface `json:"interfaces"`
	Lqm         LQM                `json:"lqm"`
}

type SysinfoInterface struct {
	Mac string `json:"mac"`
	IP  string `json:"ip"`
}

type SysinfoNodeDetails struct {
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmware_version"`
}

type Tracker struct {
	FirstSeen          int          `json:"-"`
	LastSeen           int          `json:"lastseen"`
	LastUp             int          `json:"lastup"`
	Type               DeviceType   `json:"type"`
	Device             string       `json:"device"`
	MAC                string       `json:"mac"`
	IPv6LL             string       `json:"ipv6ll"`
	Refresh            int          `json:"refresh"`
	LQ                 int          `json:"lq"`
	AvgLQ              float64      `json:"avg_lq"`
	RxCost             int          `json:"rxcost"`
	TxCost             int          `json:"txcost"`
	RTT                int          `json:"rtt"`
	TxPackets          uint64       `json:"tx_packets"`
	TxFail             uint64       `json:"tx_fail"`
	TxRetries          uint64       `json:"-"`
	LastTxPackets      *uint64      `json:"last_tx_packets"`
	LastTxFail         *uint64      `json:"-"`
	LastTxRetries      *uint64      `json:"-"`
	AvgTx              float64      `json:"avg_tx_packets"`
	AvgTxFail          float64      `json:"-"`
	AvgTxRetries       float64      `json:"-"`
	TxQuality          float64      `json:"tx_quality"`
	PingQuality        int          `json:"ping_quality"`
	PingSuccessTime    float64      `json:"ping_success_time"`
	Quality            int          `json:"quality"`
	Hostname           string       `json:"hostname"`
	CanonicalIP        string       `json:"canonical_ip"`
	IP                 string       `json:"ip"`
	Lat                float64      `json:"lat"`
	Lon                float64      `json:"lon"`
	Distance           float64      `json:"distance"`
	LocalArea          bool         `json:"localarea"`
	Model              string       `json:"model"`
	FirmwareVersion    string       `json:"firmware_version"`
	RevLastSeen        int          `json:"-"`
	RevPingSuccessTime float64      `json:"rev_ping_success_time"`
	RevPingQuality     int          `json:"rev_ping_quality"`
	RevQuality         int          `json:"rev_quality"`
	NodeRouteCount     int          `json:"-"`
	BabelRouteCount    int          `json:"babel_route_count"`
	BabelMetric        int          `json:"babel_metric"`
	Routable           bool         `json:"routable"`
	UserBlocks         bool         `json:"user_blocks"`
	BabelConfig        *BabelConfig `json:"babel_config,omitempty"`
}

type BabelConfig struct {
	HelloInterval  int `json:"hello_interval"`
	UpdateInterval int `json:"update_interval"`
	RxCost         int `json:"rxcost"`
}

type LQM struct {
	Enabled bool      `json:"enabled"`
	Config  LQMConfig `json:"config"`
	Info    LQMInfo   `json:"info"`
}

type LQMConfig struct {
	UserBlocks string `json:"user_blocks"`
}

type LQMInfo struct {
	Trackers        map[string]*Tracker `json:"trackers"`
	Start           int64               `json:"start"`
	Now             int64               `json:"now"`
	Distance        int64               `json:"distance"`
	TotalRouteCount int64               `json:"total_route_count"`
}

type DeviceType string

const (
	DeviceTypeDtD       DeviceType = "DtD"
	DeviceTypeWireguard DeviceType = "Wireguard"
)

type Service struct {
	config              *config.Config
	trackers            map[string]*Tracker
	mu                  sync.RWMutex
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	lastTick            time.Time
	startTime           time.Time
	totalRouteCount     int
	totalNodeRouteCount int
	pingSem             *semaphore.Weighted
	httpSem             *semaphore.Weighted
	httpClient          *http.Client
	startStopMu         sync.Mutex
	stopping            bool
	running             atomic.Bool
}

func NewService(config *config.Config) *Service {
	return &Service{
		config:   config,
		trackers: make(map[string]*Tracker),
		pingSem:  semaphore.NewWeighted(10), // Limit concurrent pings
		httpSem:  semaphore.NewWeighted(5),  // Limit concurrent HTTP requests
		httpClient: &http.Client{
			Timeout: connectTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	if s.stopping {
		s.startStopMu.Unlock()
		// Block forever to prevent busy loop in registry during shutdown
		select {}
	}

	if !s.IsEnabled() {
		s.startStopMu.Unlock()
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.startTime = time.Now()
	s.wg.Add(1)
	s.running.Store(true)
	s.startStopMu.Unlock()

	// Run synchronously so that the service registry doesn't spin in a tight loop restarting us.
	// The registry expects Start() to block until the service exits.
	s.run(ctx)

	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	s.stopping = true
	if s.cancel != nil {
		s.cancel()
	}
	s.startStopMu.Unlock()

	s.wg.Wait()
	s.running.Store(false)
	return nil
}

func (s *Service) Reload() error {
	return nil
}

func (s *Service) IsRunning() bool {
	return s.running.Load()
}

func (s *Service) IsEnabled() bool {
	return s.config.LQM.Enabled
}

func (s *Service) run(ctx context.Context) {
	defer func() {
		s.running.Store(false)
		s.wg.Done()
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial run
	s.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Service) tick(ctx context.Context) {
	now := time.Now()
	s.updateNeighbors(ctx)
	s.updateRoutes(ctx)
	s.updateStats()
	s.updateRunningAverages()
	s.remoteRefresh(ctx)
	s.updateTrackingState(ctx)
	s.pruneTrackers(now)
	s.writeState()
	s.lastTick = now
}

func (s *Service) pruneTrackers(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for mac, t := range s.trackers {
		lastSeenTime := time.Unix(int64(t.LastSeen), 0)
		if now.Sub(lastSeenTime) > lastSeenTimeout {
			slog.Info("LQM: Pruning tracker", "mac", mac, "last_seen", t.LastSeen, "age", now.Sub(lastSeenTime))
			delete(s.trackers, mac)
		}
	}
}

func (s *Service) updateNeighbors(ctx context.Context) {
	slog.Info("LQM: updateNeighbors started")
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", babelSocketPath)
	if err != nil {
		slog.Warn("LQM: Failed to connect to babel socket", "error", err)
		return
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	// Consume banner
	for scanner.Scan() {
		if scanner.Text() == "ok" {
			break
		}
	}

	_, err = conn.Write([]byte("dump-neighbors\n"))
	if err != nil {
		slog.Error("LQM: Failed to write to babel socket", "error", err)
		return
	}

	neighborRegex := regexp.MustCompile(`^add.*address ([^ \t]+) if ([^ \t]+) reach ([^ \t]+) .* rxcost ([^ \t]+) txcost ([^ \t]+)`)
	rttRegex := regexp.MustCompile(`rtt ([^ \t]+)`)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "ok" {
			break
		}

		matches := neighborRegex.FindStringSubmatch(line)
		if matches != nil {
			ipv6ll := matches[1]
			iface := matches[2]
			reach := matches[3]
			rxcost, _ := strconv.Atoi(matches[4])
			txcost, _ := strconv.Atoi(matches[5])

			mac := ipv6llToMac(ipv6ll)
			tracker, exists := s.trackers[mac]

			devType := deviceToType(iface)
			if devType == "" {
				slog.Warn("LQM: Skipping neighbor on unsupported interface", "iface", iface, "mac", mac)
				continue
			}

			if !exists {
				slog.Info("LQM: New neighbor detected", "mac", mac, "iface", iface, "type", devType)
				tracker = &Tracker{
					FirstSeen: int(now.Unix()),
					LastSeen:  int(now.Unix()),
					LastUp:    int(now.Unix()),
					Type:      devType,
					Device:    iface,
					MAC:       mac,
					IPv6LL:    ipv6ll,
				}
				// Derive Wireguard peer IP immediately
				if devType == DeviceTypeWireguard {
					tracker.IP = deriveWireguardPeerIP(iface)
				}
				s.trackers[mac] = tracker
			} else if tracker.Type == DeviceTypeWireguard {
				// Always re-derive Wireguard peer IPs to ensure they're current
				tracker.IP = deriveWireguardPeerIP(iface)
				slog.Debug("LQM: Updated IP for existing Wireguard tracker", "mac", mac, "device", iface, "ip", tracker.IP)
			}

			tracker.LastSeen = int(now.Unix())
			tracker.LQ = reachToLQ(reach)
			tracker.RxCost = rxcost
			tracker.TxCost = txcost
			tracker.AvgLQ = math.Min(100, 0.9 * tracker.AvgLQ + 0.1 * float64(tracker.LQ))

			// Populate BabelConfig with defaults
			if tracker.BabelConfig == nil {
				// Default for DtD/Wired
				rxcost := 96
				helloInterval := 6
				updateInterval := 120

				if s.config.Supernode {
					updateInterval = 300
				}

				if devType == DeviceTypeWireguard {
					rxcost = 206
					helloInterval = 10
				}

				tracker.BabelConfig = &BabelConfig{
					HelloInterval:  helloInterval,
					UpdateInterval: updateInterval,
					RxCost:         rxcost,
				}
			}

			rttMatches := rttRegex.FindStringSubmatch(line)
			if rttMatches != nil {
				if rtt, err := strconv.Atoi(rttMatches[1]); err == nil {
					tracker.RTT = rtt
				}
			}
		} else {
			slog.Info("LQM: Failed to match neighbor line", "line", line)
		}
	}
	slog.Info("LQM: updateNeighbors finished")
}

func (s *Service) updateStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	links, err := netlink.LinkList()
	if err != nil {
		return
	}

	for _, link := range links {
		attrs := link.Attrs()
		devType := deviceToType(attrs.Name)
		if devType == DeviceTypeWireguard || devType == DeviceTypeDtD {
			for _, tracker := range s.trackers {
				if tracker.Device == attrs.Name {
					stats := attrs.Statistics
					if stats != nil {
						tracker.TxPackets = stats.TxPackets
						tracker.TxFail = stats.TxErrors // lqm.uc uses tx_errors for tx_fail
					}
					break
				}
			}
		}
	}
}

func (s *Service) updateRunningAverages() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tracker := range s.trackers {
		// Tx Packets
		if tracker.LastTxPackets == nil {
			tracker.AvgTx = 0
			val := tracker.TxPackets
			tracker.LastTxPackets = &val
		} else {
			diff := float64(0)
			if tracker.TxPackets > *tracker.LastTxPackets {
				diff = float64(tracker.TxPackets - *tracker.LastTxPackets)
			}
			tracker.AvgTx = tracker.AvgTx*txQualityRunAvg + diff*(1-txQualityRunAvg)
			*tracker.LastTxPackets = tracker.TxPackets
		}

		// Tx Fail
		if tracker.LastTxFail == nil {
			tracker.AvgTxFail = 0
			val := tracker.TxFail
			tracker.LastTxFail = &val
		} else {
			diff := float64(0)
			if tracker.TxFail > *tracker.LastTxFail {
				diff = float64(tracker.TxFail - *tracker.LastTxFail)
			}
			tracker.AvgTxFail = tracker.AvgTxFail*txQualityRunAvg + diff*(1-txQualityRunAvg)
			*tracker.LastTxFail = tracker.TxFail
		}

		// Tx Retries
		if tracker.LastTxRetries == nil {
			tracker.AvgTxRetries = 0
			val := tracker.TxRetries
			tracker.LastTxRetries = &val
		} else {
			diff := float64(0)
			if tracker.TxRetries > *tracker.LastTxRetries {
				diff = float64(tracker.TxRetries - *tracker.LastTxRetries)
			}
			tracker.AvgTxRetries = tracker.AvgTxRetries*txQualityRunAvg + diff*(1-txQualityRunAvg)
			*tracker.LastTxRetries = tracker.TxRetries
		}

		if tracker.AvgTx > 0 {
			bad := math.Max(tracker.AvgTxFail, tracker.AvgTxRetries)
			tracker.TxQuality = 100 * (1 - math.Min(1, bad/tracker.AvgTx))
		}
	}
}

func (s *Service) remoteRefresh(ctx context.Context) {
	s.mu.Lock()
	trackersToRefresh := make([]*Tracker, 0)
	for _, t := range s.trackers {
		if t.Refresh == 0 || time.Now().After(time.Unix(int64(t.Refresh), 0)) {
			// Mark as refreshing immediately to prevent re-scheduling in the next tick
			// if the actual refresh takes longer than the tick interval.
			// We set it to a retry timeout for now; success will overwrite it with the proper interval.
			t.Refresh = int(time.Now().Add(refreshRetryTimeout).Unix())
			trackersToRefresh = append(trackersToRefresh, t)
		}
	}
	s.mu.Unlock()

	for _, t := range trackersToRefresh {
		tracker := t
		// We don't wait for these to finish, but we limit concurrency
		go func() {
			if err := s.httpSem.Acquire(ctx, 1); err != nil {
				return
			}
			defer s.httpSem.Release(1)
			if err := s.refreshTracker(ctx, tracker); err != nil {
				slog.Warn("LQM: Failed to refresh tracker", "mac", tracker.MAC, "error", err)
			}
		}()
	}
}

func (s *Service) refreshTracker(ctx context.Context, t *Tracker) error {
	if t.IPv6LL == "" {
		return nil
	}

	url := fmt.Sprintf("http://[%s%%25%s]:8080/cgi-bin/sysinfo.json?lqm=1", t.IPv6LL, t.Device)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// Refresh time was already set to retry timeout in remoteRefresh,
		// but we reset stats here.
		s.mu.Lock()
		t.RevPingSuccessTime = 0
		t.RevPingQuality = 0
		t.RevQuality = 0
		s.mu.Unlock()
		return err
	}
	defer resp.Body.Close()

	var info SysinfoResponse

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update refresh time on success
	jitter := refreshTimeoutRange / 2
	if m := big.NewInt(int64(refreshTimeoutRange)); m.Sign() > 0 {
		if n, err := rand.Int(rand.Reader, m); err == nil {
			jitter = time.Duration(n.Int64())
		}
	}
	t.Refresh = int(time.Now().Add(refreshTimeoutBase + jitter).Unix())
	t.RevLastSeen = int(time.Now().Unix())

	switch lat := info.Lat.(type) {
	case float64:
		t.Lat = lat
	case string:
		parsedLat, err := strconv.ParseFloat(lat, 64)
		if err == nil {
			t.Lat = parsedLat
		} else {
			t.Lat = 0.0
		}
	default:
		t.Lat = 0.0
	}

	switch lon := info.Lon.(type) {
	case float64:
		t.Lon = lon
	case string:
		parsedLon, err := strconv.ParseFloat(lon, 64)
		if err == nil {
			t.Lon = parsedLon
		} else {
			t.Lon = 0.0
		}
	default:
		t.Lon = 0.0
	}

	t.Hostname = canonicalHostname(info.Node)
	t.CanonicalIP = meshIPForHostname(ctx, t.Hostname)

	if t.Type == DeviceTypeWireguard {
		// Ensure IP is set for Wireguard
		if t.IP == "" {
			t.IP = deriveWireguardPeerIP(t.Device)
		}
	} else {
		for _, iface := range info.Interfaces {
			if strings.EqualFold(iface.Mac, t.MAC) {
				t.IP = iface.IP
				break
			}
		}
	}

	if s.config.Latitude != "" && s.config.Longitude != "" {
		lat1, err1 := strconv.ParseFloat(s.config.Latitude, 64)
		lon1, err2 := strconv.ParseFloat(s.config.Longitude, 64)
		if err1 == nil && err2 == nil && t.Lat != 0 && t.Lon != 0 {
			t.Distance = calcDistance(lat1, lon1, t.Lat, t.Lon)
			if t.Type == DeviceTypeDtD && t.Distance < dtdDistance {
				t.LocalArea = true
			} else {
				t.LocalArea = false
			}
		}
	}

	t.Model = info.NodeDetails.Model
	t.FirmwareVersion = info.NodeDetails.FirmwareVersion

	// Reverse stats
	myHostname := canonicalHostname(s.config.ServerName)
	if info.Lqm.Info.Trackers != nil {
		for _, rtrack := range info.Lqm.Info.Trackers {
			if myHostname == canonicalHostname(rtrack.Hostname) {
				t.RevPingSuccessTime = rtrack.PingSuccessTime
				t.RevPingQuality = rtrack.PingQuality
				t.RevQuality = rtrack.Quality
				break
			}
		}
	}

	return nil
}

func (s *Service) updateTrackingState(ctx context.Context) {
	slog.Info("LQM: updateTrackingState started")
	s.mu.Lock()
	trackers := make([]*Tracker, 0, len(s.trackers))
	for _, t := range s.trackers {
		trackers = append(trackers, t)
	}
	s.mu.Unlock()

	var wg sync.WaitGroup
	for _, t := range trackers {
		// Acquire semaphore BEFORE spawning goroutine to control memory usage
		if err := s.pingSem.Acquire(ctx, 1); err != nil {
			break
		}
		wg.Add(1)
		go func(tracker *Tracker) {
			defer s.pingSem.Release(1)
			defer wg.Done()
			s.pingTracker(ctx, tracker)
			s.calculateQuality(tracker)
		}(t)
	}
	wg.Wait()
	slog.Info("LQM: updateTrackingState finished")
}

func (s *Service) pingTracker(ctx context.Context, t *Tracker) {
	if t.IPv6LL == "" {
		return
	}

	timeoutSec := strconv.Itoa(int(pingTimeout.Seconds()))
	// Use CommandContext to respect cancellation
	cmd := exec.CommandContext(ctx, "ping6", "-c", "1", "-W", timeoutSec, "-I", t.Device, t.IPv6LL)
	output, err := cmd.Output()

	s.mu.Lock()
	defer s.mu.Unlock()

	success := false
	var ptime float64

	if err == nil {
		re := regexp.MustCompile(`time=([^ \t]+) ms`)
		matches := re.FindStringSubmatch(string(output))
		if matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				ptime = val / 1000.0
				success = true
			}
		}
	}

	if t.PingQuality == 0 {
		t.PingQuality = 100
	} else {
		t.PingQuality++
	}

	if !success {
		t.PingQuality -= pingPenalty
	} else {
		if t.PingSuccessTime == 0 {
			t.PingSuccessTime = ptime
		} else {
			t.PingSuccessTime = t.PingSuccessTime*pingTimeRunAvg + ptime*(1-pingTimeRunAvg)
		}
	}

	t.PingQuality = int(math.Max(0, math.Min(100, float64(t.PingQuality))))

	if success {
		lastSeenTime := time.Unix(int64(t.LastSeen), 0)
		if !s.lastTick.IsZero() && lastSeenTime.Add(lastUpMargin).Before(s.lastTick) {
			t.LastUp = int(time.Now().Unix())
		}
		t.LastSeen = int(time.Now().Unix())
	}
}

func (s *Service) calculateQuality(t *Tracker) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case t.TxQuality > 0:
		if t.PingQuality > 0 {
			t.Quality = int(math.Round((t.TxQuality + float64(t.PingQuality)) / 2))
		} else {
			t.Quality = int(math.Round(t.TxQuality))
		}
	case t.PingQuality > 0:
		t.Quality = int(math.Round(float64(t.PingQuality)))
	default:
		t.Quality = 0
	}
}

func (s *Service) writeState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slog.Info("LQM: Writing state", "trackers_count", len(s.trackers))

	state := LQMInfo{
		Now:             time.Now().Unix(),
		Trackers:        s.trackers,
		Distance:        defaultMaxDistance,
		Start:           s.startTime.Unix(),
		TotalRouteCount: int64(s.totalRouteCount),
	}

	file, err := os.Create(lqmInfoPath)
	if err != nil {
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(state); err != nil {
		slog.Warn("LQM: Failed to encode state", "error", err)
	}
}

func ipv6llToMac(ipv6ll string) string {
	ip := net.ParseIP(ipv6ll)
	if ip == nil {
		return ipv6ll
	}
	if len(ip) != 16 {
		return ipv6ll
	}

	if ip[11] == 0xff && ip[12] == 0xfe {
		mac := make([]byte, 0, 6)
		mac = append(mac,
			ip[8]^0x02,
			ip[9],
			ip[10],
			ip[13],
			ip[14],
			ip[15],
		)
		return net.HardwareAddr(mac).String()
	}

	return ipv6ll
}

func deviceToType(device string) DeviceType {
	if device == "br-dtdlink" {
		return DeviceTypeDtD
	}
	if strings.HasPrefix(device, "wg") {
		return DeviceTypeWireguard
	}
	return ""
}

func reachToLQ(reach string) int {
	val, err := strconv.ParseUint(reach, 16, 16)
	if err != nil {
		return 0
	}

	count := 0
	for i := 0; i < 16; i++ {
		if (val & 1) == 1 {
			count++
		}
		val >>= 1
	}
	return int(math.Ceil(100 * float64(count) / 16))
}

func canonicalHostname(hostname string) string {
	h := strings.ToLower(hostname)
	h = strings.TrimSuffix(h, ".local.mesh")
	h = strings.TrimPrefix(h, "dtdlink.")
	return h
}

func meshIPForHostname(ctx context.Context, hostname string) string {
	resolver := &net.Resolver{}
	addrs, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			return addr.IP.String()
		}
	}
	return ""
}

func calcDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const r2 = 12742000 // diameter earth (meters)
	const p = math.Pi / 180

	v := 0.5 - math.Cos((lat2-lat1)*p)/2 + math.Cos(lat1*p)*math.Cos(lat2*p)*(1-math.Cos((lon2-lon1)*p))/2
	return float64(r2) * math.Atan2(math.Sqrt(v), math.Sqrt(1-v))
}

func (s *Service) updateRoutes(ctx context.Context) {
	slog.Info("LQM: updateRoutes started")
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", babelSocketPath)
	if err != nil {
		slog.Error("LQM: Failed to connect to babel socket for routes", "error", err)
		return
	}
	defer conn.Close()

	// Set deadline to prevent hanging
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		slog.Error("LQM: Failed to set deadline on babel socket", "error", err)
		return
	}

	scanner := bufio.NewScanner(conn)

	// Consume banner
	for scanner.Scan() {
		if scanner.Text() == "ok" {
			break
		}
	}

	_, err = conn.Write([]byte("dump-installed-routes\n"))
	if err != nil {
		return
	}

	s.mu.Lock()
	// Reset counts and build lookup map
	ipToTracker := make(map[string]*Tracker)
	for _, t := range s.trackers {
		t.NodeRouteCount = 0
		t.BabelRouteCount = 0
		t.BabelMetric = math.MaxInt32
		t.Routable = false
		// Map by both IPv6LL and IPv4 if available
		if t.IPv6LL != "" {
			ipToTracker[t.IPv6LL] = t
		}
		if t.IP != "" {
			ipToTracker[t.IP] = t
		}
	}
	s.mu.Unlock()

	// Example: add route prefix 10.51.120.3/32 ... installed yes ... nexthop fe80::... metric 257 ...
	// Match AREDN: only count installed IPv4 routes with metric != 65535
	routeRegex := regexp.MustCompile(`^add route .+ prefix ([^ /]+)/([0-9]+) .* installed yes .* metric ([0-9]+) .* nexthop ([^ \t]+)`)

	totalRoutes := 0
	totalNodeRoutes := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "ok" {
			break
		}

		matches := routeRegex.FindStringSubmatch(line)
		if matches != nil {
			prefix := matches[1]
			prefixLen := matches[2]
			metric, _ := strconv.Atoi(matches[3])
			via := matches[4]

			// Skip non-IPv4 routes and invalid metrics (matching AREDN behavior)
			if !strings.Contains(prefix, ".") || metric == 65535 {
				continue
			}

			// Node routes are /32 (host routes)
			isNodeRoute := prefixLen == "32"

			s.mu.Lock()
			if t, ok := ipToTracker[via]; ok {
				t.Routable = true
				t.BabelRouteCount++
				if isNodeRoute {
					t.NodeRouteCount++
					totalNodeRoutes++
				}
				if metric < t.BabelMetric {
					t.BabelMetric = metric
				}
			}
			s.mu.Unlock()
			totalRoutes++
		}
	}
	slog.Info("LQM: updateRoutes finished")

	s.mu.Lock()
	s.totalRouteCount = totalRoutes
	s.totalNodeRouteCount = totalNodeRoutes
	for _, t := range s.trackers {
		if t.BabelMetric == math.MaxInt32 {
			t.BabelMetric = 0
		}
	}
	s.mu.Unlock()
}

// deriveWireguardPeerIP derives the peer's IPv4 address from a Wireguard interface
// For wgs* (server): peer IP = interface IP + 1
// For wgc* (client): peer IP = interface IP - 1
func deriveWireguardPeerIP(device string) string {
	if device == "" {
		return ""
	}

	links, err := netlink.LinkList()
	if err != nil {
		return ""
	}

	for _, link := range links {
		if link.Attrs().Name == device {
			addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
			if err != nil {
				return ""
			}
			if len(addrs) == 0 {
				return ""
			}

			// Get the IPv4 address and ensure it's in 4-byte format
			ip := addrs[0].IP.To4()
			if ip == nil {
				return ""
			}

			// Make a copy of the IP to avoid modifying the original
			peerIP := make(net.IP, len(ip))
			copy(peerIP, ip)

			if strings.HasPrefix(device, "wgs") {
				// Server interface: peer is IP + 1
				peerIP[3]++
				return peerIP.String()
			} else if strings.HasPrefix(device, "wgc") {
				// Client interface: peer is IP - 1
				peerIP[3]--
				return peerIP.String()
			}
			break
		}
	}
	return ""
}
