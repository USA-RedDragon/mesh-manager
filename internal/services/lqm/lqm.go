package lqm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/vishvananda/netlink"
	"golang.org/x/sync/semaphore"
)

const (
	refreshTimeoutBase   = 12 * 60 * time.Second
	refreshTimeoutRange  = 5 * 60 * time.Second
	refreshRetryTimeout  = 5 * 60 * time.Second
	lastSeenTimeout      = 24 * time.Hour
	txQualityRunAvg      = 0.4
	pingTimeout          = 1.0 * time.Second
	pingTimeRunAvg       = 0.4
	dtdDistance          = 50 // meters
	connectTimeout       = 5 * time.Second
	defaultMaxDistance   = 80550 // meters
	pingPenalty          = 5
	lastUpMargin         = 60 * time.Second
	babelSocketPath      = "/var/run/babel.sock"
	lqmInfoPath          = "/tmp/lqm.info"
)

type Tracker struct {
	LastSeen           time.Time    `json:"lastseen"`
	LastUp             time.Time    `json:"lastup"`
	Type               string       `json:"type"`
	Device             string       `json:"device"`
	MAC                string       `json:"mac"`
	IPv6LL             string       `json:"ipv6ll"`
	Refresh            time.Time    `json:"refresh"`
	LQ                 int          `json:"lq"`
	RxCost             int          `json:"rxcost"`
	TxCost             int          `json:"txcost"`
	RTT                int          `json:"rtt"`
	TxPackets          uint64       `json:"tx_packets"`
	TxFail             uint64       `json:"tx_fail"`
	LastTxPackets      *uint64      `json:"-"`
	LastTxFail         *uint64      `json:"-"`
	AvgTxPackets       float64      `json:"avg_tx_packets"`
	AvgTxFail          float64      `json:"avg_tx_fail"`
	TxQuality          float64      `json:"tx_quality"`
	PingQuality        float64      `json:"ping_quality"`
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
	RevPingSuccessTime float64      `json:"rev_ping_success_time"`
	RevPingQuality     float64      `json:"rev_ping_quality"`
	RevQuality         int          `json:"rev_quality"`
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

type Service struct {
	config          *config.Config
	trackers        map[string]*Tracker
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	lastTick        time.Time
	totalRouteCount int
	pingSem         *semaphore.Weighted
	httpSem         *semaphore.Weighted
	httpClient      *http.Client
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
	if !s.IsEnabled() {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wg.Add(1)
	go s.run()

	return nil
}

func (s *Service) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	return nil
}

func (s *Service) Reload() error {
	return nil
}

func (s *Service) IsRunning() bool {
	return s.ctx != nil && s.ctx.Err() == nil
}

func (s *Service) IsEnabled() bool {
	return s.config.LQM.Enabled
}

func (s *Service) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial run
	s.tick()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Service) tick() {
	now := time.Now()
	s.updateNeighbors()
	s.updateRoutes()
	s.updateStats()
	s.updateRunningAverages()
	s.remoteRefresh()
	s.updateTrackingState()
	s.pruneTrackers(now)
	s.writeState()
	s.lastTick = now
}

func (s *Service) pruneTrackers(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for mac, t := range s.trackers {
		if now.Sub(t.LastSeen) > lastSeenTimeout {
			delete(s.trackers, mac)
		}
	}
}

func (s *Service) updateNeighbors() {
	conn, err := net.Dial("unix", babelSocketPath)
	if err != nil {
		// fmt.Printf("Failed to connect to babel socket: %v\n", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte("dump-neighbors\n"))
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(conn)
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
				continue
			}

			if !exists {
				tracker = &Tracker{
					LastSeen: now,
					LastUp:   now,
					Type:     devType,
					Device:   iface,
					MAC:      mac,
					IPv6LL:   ipv6ll,
				}
				s.trackers[mac] = tracker
			}

			tracker.LastSeen = now
			tracker.LQ = reachToLQ(reach)
			tracker.RxCost = rxcost
			tracker.TxCost = txcost

			// Populate BabelConfig with defaults
			if tracker.BabelConfig == nil {
				// Default for DtD/Wired
				rxcost := 96
				helloInterval := 4000 // Default 4s

				if devType == "Wireguard" {
					rxcost = 206
					helloInterval = 10000 // 10s
				}

				tracker.BabelConfig = &BabelConfig{
					HelloInterval:  helloInterval,
					UpdateInterval: helloInterval * 4,
					RxCost:         rxcost,
				}
			}

			rttMatches := rttRegex.FindStringSubmatch(line)
			if rttMatches != nil {
				if rtt, err := strconv.Atoi(rttMatches[1]); err == nil {
					tracker.RTT = rtt
				}
			}
		}
	}
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
		if devType == "Wireguard" || devType == "DtD" {
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
			tracker.AvgTxPackets = 0
			val := tracker.TxPackets
			tracker.LastTxPackets = &val
		} else {
			diff := float64(0)
			if tracker.TxPackets > *tracker.LastTxPackets {
				diff = float64(tracker.TxPackets - *tracker.LastTxPackets)
			}
			tracker.AvgTxPackets = tracker.AvgTxPackets*txQualityRunAvg + diff*(1-txQualityRunAvg)
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

		if tracker.AvgTxPackets > 0 {
			bad := tracker.AvgTxFail
			tracker.TxQuality = 100 * (1 - math.Min(1, bad/tracker.AvgTxPackets))
		}
	}
}

func (s *Service) remoteRefresh() {
	s.mu.Lock()
	trackersToRefresh := make([]*Tracker, 0)
	now := time.Now()
	for _, t := range s.trackers {
		if t.Refresh.IsZero() || now.After(t.Refresh) {
			// Mark as refreshing immediately to prevent re-scheduling in the next tick
			// if the actual refresh takes longer than the tick interval.
			// We set it to a retry timeout for now; success will overwrite it with the proper interval.
			t.Refresh = now.Add(refreshRetryTimeout)
			trackersToRefresh = append(trackersToRefresh, t)
		}
	}
	s.mu.Unlock()

	for _, t := range trackersToRefresh {
		tracker := t
		// We don't wait for these to finish, but we limit concurrency
		go func() {
			if err := s.httpSem.Acquire(s.ctx, 1); err != nil {
				return
			}
			defer s.httpSem.Release(1)
			s.refreshTracker(tracker)
		}()
	}
}

func (s *Service) refreshTracker(t *Tracker) error {
	if t.IPv6LL == "" {
		return nil
	}

	url := fmt.Sprintf("http://[%s%%%s]:8080/cgi-bin/sysinfo.json?lqm=1", t.IPv6LL, t.Device)

	resp, err := s.httpClient.Get(url)
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

	var info struct {
		Node string `json:"node"`
		Lat  string `json:"lat"`
		Lon  string `json:"lon"`
		NodeDetails struct {
			Model           string `json:"model"`
			FirmwareVersion string `json:"firmware_version"`
		} `json:"node_details"`
		Interfaces []struct {
			Mac string `json:"mac"`
			Ip  string `json:"ip"`
		} `json:"interfaces"`
		Lqm struct {
			Info struct {
				Trackers map[string]struct {
					Hostname        string  `json:"hostname"`
					PingSuccessTime float64 `json:"ping_success_time"`
					PingQuality     float64 `json:"ping_quality"`
					Quality         int     `json:"quality"`
				} `json:"trackers"`
			} `json:"info"`
		} `json:"lqm"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update refresh time on success
	jitter := time.Duration(rand.Int63n(int64(refreshTimeoutRange)))
	t.Refresh = time.Now().Add(refreshTimeoutBase + jitter)

	t.Hostname = canonicalHostname(info.Node)

	if t.Type == "Wireguard" {
		// Skip complex WG IP logic for now
	} else {
		for _, iface := range info.Interfaces {
			if strings.EqualFold(iface.Mac, t.MAC) {
				t.IP = iface.Ip
				break
			}
		}
	}

	if lat, err := strconv.ParseFloat(info.Lat, 64); err == nil {
		t.Lat = lat
	}
	if lon, err := strconv.ParseFloat(info.Lon, 64); err == nil {
		t.Lon = lon
	}

	if s.config.Latitude != "" && s.config.Longitude != "" {
		lat1, err1 := strconv.ParseFloat(s.config.Latitude, 64)
		lon1, err2 := strconv.ParseFloat(s.config.Longitude, 64)
		if err1 == nil && err2 == nil && t.Lat != 0 && t.Lon != 0 {
			t.Distance = calcDistance(lat1, lon1, t.Lat, t.Lon)
			if t.Type == "DtD" && t.Distance < dtdDistance {
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

func (s *Service) updateTrackingState() {
	s.mu.Lock()
	trackers := make([]*Tracker, 0, len(s.trackers))
	for _, t := range s.trackers {
		trackers = append(trackers, t)
	}
	s.mu.Unlock()

	var wg sync.WaitGroup
	for _, t := range trackers {
		// Acquire semaphore BEFORE spawning goroutine to control memory usage
		if err := s.pingSem.Acquire(s.ctx, 1); err != nil {
			break
		}
		wg.Add(1)
		go func(tracker *Tracker) {
			defer s.pingSem.Release(1)
			defer wg.Done()
			s.pingTracker(tracker)
			s.calculateQuality(tracker)
		}(t)
	}
	wg.Wait()
}

func (s *Service) pingTracker(t *Tracker) {
	if t.IPv6LL == "" {
		return
	}

	timeoutSec := strconv.Itoa(int(pingTimeout.Seconds()))
	// Use CommandContext to respect cancellation
	cmd := exec.CommandContext(s.ctx, "ping6", "-c", "1", "-W", timeoutSec, "-I", t.Device, t.IPv6LL)
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
		t.PingQuality += 1
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

	t.PingQuality = math.Max(0, math.Min(100, t.PingQuality))

	if success {
		if !s.lastTick.IsZero() && t.LastSeen.Add(lastUpMargin).Before(s.lastTick) {
			t.LastUp = time.Now()
		}
		t.LastSeen = time.Now()
	}
}

func (s *Service) calculateQuality(t *Tracker) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t.TxQuality > 0 {
		if t.PingQuality > 0 {
			t.Quality = int(math.Round((t.TxQuality + t.PingQuality) / 2))
		} else {
			t.Quality = int(math.Round(t.TxQuality))
		}
	} else if t.PingQuality > 0 {
		t.Quality = int(math.Round(t.PingQuality))
	} else {
		t.Quality = 0
	}
}

func (s *Service) writeState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := struct {
		Now             int64               `json:"now"`
		Trackers        map[string]*Tracker `json:"trackers"`
		Distance        int                 `json:"distance"`
		HiddenNodes     []string            `json:"hidden_nodes"`
		TotalRouteCount int                 `json:"total_route_count"`
	}{
		Now:             time.Now().Unix(),
		Trackers:        s.trackers,
		Distance:        defaultMaxDistance,
		HiddenNodes:     []string{},
		TotalRouteCount: s.totalRouteCount,
	}

	file, err := os.Create(lqmInfoPath)
	if err != nil {
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.Encode(state)
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
		mac := make([]byte, 6)
		mac[0] = ip[8] ^ 0x02
		mac[1] = ip[9]
		mac[2] = ip[10]
		mac[3] = ip[13]
		mac[4] = ip[14]
		mac[5] = ip[15]
		return net.HardwareAddr(mac).String()
	}

	return ipv6ll
}

func deviceToType(device string) string {
	if device == "br-dtdlink" {
		return "DtD"
	}
	if strings.HasPrefix(device, "wg") {
		return "Wireguard"
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

func calcDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const r2 = 12742000               // diameter earth (meters)
	const p = math.Pi / 180

	v := 0.5 - math.Cos((lat2-lat1)*p)/2 + math.Cos(lat1*p)*math.Cos(lat2*p)*(1-math.Cos((lon2-lon1)*p))/2
	return float64(r2) * math.Atan2(math.Sqrt(v), math.Sqrt(1-v))
}

func (s *Service) updateRoutes() {
	conn, err := net.Dial("unix", babelSocketPath)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte("dump-routes\n"))
	if err != nil {
		return
	}

	s.mu.Lock()
	// Reset counts and build lookup map
	ipToTracker := make(map[string]*Tracker)
	for _, t := range s.trackers {
		t.BabelRouteCount = 0
		t.BabelMetric = math.MaxInt32
		t.Routable = false
		if t.IPv6LL != "" {
			ipToTracker[t.IPv6LL] = t
		}
	}
	s.mu.Unlock()

	scanner := bufio.NewScanner(conn)
	// Example: add route prefix ... metric 128 ... via fe80::... if ...
	routeRegex := regexp.MustCompile(`^add route .* metric ([0-9]+) .* via ([^ \t]+)`)

	totalRoutes := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "ok" {
			break
		}

		matches := routeRegex.FindStringSubmatch(line)
		if matches != nil {
			totalRoutes++
			metric, _ := strconv.Atoi(matches[1])
			via := matches[2]

			s.mu.Lock()
			if t, ok := ipToTracker[via]; ok {
				t.Routable = true
				t.BabelRouteCount++
				if metric < t.BabelMetric {
					t.BabelMetric = metric
				}
			}
			s.mu.Unlock()
		}
	}

	s.mu.Lock()
	s.totalRouteCount = totalRoutes
	for _, t := range s.trackers {
		if t.BabelMetric == math.MaxInt32 {
			t.BabelMetric = 0
		}
	}
	s.mu.Unlock()
}
