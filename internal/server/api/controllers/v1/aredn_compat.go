package v1

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/services/lqm"
	"github.com/USA-RedDragon/mesh-manager/internal/services/meshlink"
	"github.com/USA-RedDragon/mesh-manager/internal/services/olsr"
	"github.com/USA-RedDragon/mesh-manager/internal/utils"
	"github.com/gin-gonic/gin"
)

//nolint:gocyclo
func GETIPerf(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.Header("Cache-Control", "no-store")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Writer.WriteHeaderNow()

	server, serverExists := c.GetQuery("server")
	protocol := c.DefaultQuery("protocol", "tcp")
	kill := c.Query("kill") == "1"

	if !serverExists {
		fmt.Fprint(c.Writer, "<html><head><title>ERROR</title></head><body><pre>Provide a server name to run a test between this client and a server\n[/cgi-bin/iperf?server=&lt;ServerName&gt;&amp;protocol=&lt;udp|tcp&gt;]</pre></body></html>\n")
		return
	}

	if matched, _ := regexp.MatchString(`[^0-9a-zA-Z\.\-]`, server); matched {
		fmt.Fprint(c.Writer, "<html><head><title>ERROR</title></head><body><pre>Illegal server name</pre></body></html>\n")
		return
	}

	if server == "" {
		// Server mode
		if kill {
			err := exec.CommandContext(c.Request.Context(), "killall", "-9", "iperf3").Run()
			if err != nil {
				fmt.Fprint(c.Writer, "<html><head><title>ERROR</title></head><body><pre>iperf server failed to stop</pre></body></html>\n")
				return
			}
		} else {
			// Check if running
			out, _ := exec.CommandContext(c.Request.Context(), "pidof", "iperf3").Output()
			if len(out) > 0 {
				fmt.Fprint(c.Writer, "<html><head><title>BUSY</title></head><body><pre>iperf server busy</pre></body></html>\n")
				return
			}
		}

		// Start server
		cmd := exec.CommandContext(c.Request.Context(), "/usr/bin/iperf3", "-s", "-1", "--idle-timeout", "20", "--forceflush", "-B", "0.0.0.0")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Fprint(c.Writer, "<html><head><title>SERVER ERROR</title></head><body><pre>iperf server failed to start</pre></body></html>\n")
			return
		}
		if err := cmd.Start(); err != nil {
			fmt.Fprint(c.Writer, "<html><head><title>SERVER ERROR</title></head><body><pre>iperf server failed to start</pre></body></html>\n")
			return
		}

		// Read one line to ensure it started
		buf := make([]byte, 1024)
		stdout.Read(buf)

		// Drain stdout and wait for process in background to avoid zombies
		go func() {
			_, err := io.Copy(io.Discard, stdout)
			if err != nil {
				slog.Error("error draining iperf server stdout", "error", err)
			}
			if err := cmd.Wait(); err != nil {
				slog.Error("error waiting for iperf server process", "error", err)
			}
		}()

		fmt.Fprint(c.Writer, "<html><head><title>RUNNING</title></head><body><pre>iperf server running</pre></body></html>\n")
		c.Writer.Flush()
		return
	}
	// Client mode
	if !strings.Contains(server, ".") {
		server += ".local.mesh"
	}

	// Resolve IP
	resolver := &net.Resolver{}
	ips, err := resolver.LookupIPAddr(c.Request.Context(), server)
	if err != nil || len(ips) == 0 {
		fmt.Fprint(c.Writer, "<html><head><title>SERVER ERROR</title></head><body><pre>iperf no such server</pre></body></html>\n")
		return
	}
	ip := ips[0].String()

	// Call remote
	killParam := ""
	if kill {
		killParam = "kill=1&"
	}
	remoteURL := fmt.Sprintf("http://%s/cgi-bin/iperf?%sserver=", net.JoinHostPort(ip, "8080"), killParam)
	u, err := url.Parse(remoteURL)
	if err != nil || u.Scheme != "http" {
		fmt.Fprint(c.Writer, "<html><head><title>CLIENT ERROR</title></head><body><pre>iperf invalid remote URL</pre></body></html>\n")
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		fmt.Fprint(c.Writer, "<html><head><title>CLIENT ERROR</title></head><body><pre>iperf request failed</pre></body></html>\n")
		return
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprint(c.Writer, "<html><head><title>CLIENT ERROR</title></head><body><pre>iperf failed to call remote server</pre></body></html>\n")
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Fprint(c.Writer, "<html><head><title>ERROR</title></head><body><pre>iperf unknown error</pre></body></html>\n")
				return
			}
			fmt.Fprint(c.Writer, "<html><head><title>ERROR</title></head><body><pre>iperf unknown error</pre></body></html>\n")
			return
		}

		switch {
		case strings.Contains(line, "CLIENT DISABLED"):
			fmt.Fprint(c.Writer, "<html><head><title>SERVER DISABLED</title></head><body><pre>iperf server is disabled</pre></body></html>\n")
			return
		case strings.Contains(line, "BUSY"):
			fmt.Fprint(c.Writer, "<html><head><title>SERVER BUSY</title></head><body><pre>iperf server is busy</pre></body></html>\n")
			return
		case strings.Contains(line, "ERROR"):
			fmt.Fprint(c.Writer, "<html><head><title>SERVER ERROR</title></head><body><pre>iperf server error</pre></body></html>\n")
			return
		case strings.Contains(line, "RUNNING"):
			// Start local client
			args := []string{"--forceflush", "-b", "0", "-Z", "-c", ip, "-l", "16K"}
			if protocol == "udp" {
				args = append(args, "-u")
			}
			cmd := exec.CommandContext(c.Request.Context(), "/usr/bin/iperf3", args...)

			// Capture stdout and stderr
			pr, pw, _ := os.Pipe()
			cmd.Stdout = pw
			cmd.Stderr = pw

			if err := cmd.Start(); err != nil {
				fmt.Fprint(c.Writer, "<html><head><title>CLIENT ERROR</title></head><body><pre>iperf client failed</pre></body></html>\n")
				return
			}
			pw.Close() // Close write end in parent

			fmt.Fprint(c.Writer, "<html><head><title>SUCCESS</title></head>")

			di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
			nodeName := "Unknown"
			if ok {
				nodeName = di.Config.ServerName
			}

			fmt.Fprintf(c.Writer, "<body><pre>Client: %s\nServer: %s\n", nodeName, server)
			c.Writer.Flush()

			// Stream output
			scanner := bufio.NewScanner(pr)
			for scanner.Scan() {
				fmt.Fprintln(c.Writer, scanner.Text())
				c.Writer.Flush()
			}

			if err := cmd.Wait(); err != nil {
				slog.Error("error waiting for iperf client process", "error", err)
			}
			fmt.Fprint(c.Writer, "</pre></body></html>\n")
		}
	}
}

func GETMesh(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/nodes")
}

func GetAMesh(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/nodes")
}

func GETMetrics(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	if !di.Config.Metrics.Enabled {
		c.JSON(http.StatusGone, gin.H{"error": "Metrics are not enabled"})
		return
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	hostPort := net.JoinHostPort(di.Config.Metrics.NodeExporterHost, "9100")

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, fmt.Sprintf("http://%s/metrics", hostPort), nil)
	if err != nil {
		slog.Error("GETMetrics: Unable to create request", "hostPort", hostPort, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	nodeResp, err := client.Do(req)
	if err != nil {
		slog.Error("GETMetrics: Unable to get node-exporter metrics", "hostPort", hostPort, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer nodeResp.Body.Close()
	nodeMetrics := ""
	buf := make([]byte, 128)
	n, err := nodeResp.Body.Read(buf)
	for err == nil || n > 0 {
		nodeMetrics += string(buf[:n])
		n, err = nodeResp.Body.Read(buf)
	}

	hostPort = net.JoinHostPort("localhost", fmt.Sprintf("%d", di.Config.Metrics.Port))

	req, err = http.NewRequestWithContext(c.Request.Context(), http.MethodGet, fmt.Sprintf("http://%s/metrics", hostPort), nil)
	if err != nil {
		slog.Error("GETMetrics: Unable to create request", "hostPort", hostPort, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	metricsResp, err := client.Do(req)
	if err != nil {
		slog.Error("GETMetrics: Unable to get metrics", "hostPort", hostPort, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer metricsResp.Body.Close()
	metrics := ""
	buf = make([]byte, 128)
	n, err = metricsResp.Body.Read(buf)
	for err == nil || n > 0 {
		metrics += string(buf[:n])
		n, err = metricsResp.Body.Read(buf)
	}

	// Combine the two responses and send them back
	c.String(http.StatusOK, fmt.Sprintf("%s\n%s", nodeMetrics, metrics))
}

func GETSysinfo(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	activeTunnels, err := models.CountAllActiveTunnels(di.DB)
	if err != nil {
		slog.Error("GETSysinfo: Unable to get active tunnels", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var info syscall.Sysinfo_t
	err = syscall.Sysinfo(&info)
	if err != nil {
		slog.Error("GETSysinfo: Unable to get system info", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	hostsStr, exists := c.GetQuery("hosts")
	if !exists {
		hostsStr = "0"
	}
	doHosts := hostsStr == "1"

	nodesStr, exists := c.GetQuery("nodes")
	if !exists {
		nodesStr = "0"
	}
	doNodes := nodesStr == "1"

	servicesStr, exists := c.GetQuery("services")
	if !exists {
		servicesStr = "0"
	}
	doServices := servicesStr == "1"

	servicesStr, exists = c.GetQuery("services_local")
	if !exists {
		servicesStr = "0"
	}
	doLocalServices := servicesStr == "1"

	linkInfoStr, exists := c.GetQuery("link_info")
	if !exists {
		linkInfoStr = "0"
	}
	doLinkInfo := linkInfoStr == "1"

	lqmStr, exists := c.GetQuery("lqm")
	if !exists {
		lqmStr = "0"
	}
	doLQM := lqmStr == "1"

	sysinfo := apimodels.SysinfoResponse{
		Longitude: di.Config.Longitude,
		Latitude:  di.Config.Latitude,
		Sysinfo: apimodels.Sysinfo{
			Uptime: utils.SecondsToClock(info.Uptime),
			Loadavg: [3]float64{
				float64(info.Loads[0]) / float64(1<<16),
				float64(info.Loads[1]) / float64(1<<16),
				float64(info.Loads[2]) / float64(1<<16),
			},
			FreeMemory: uint64(info.Freeram) * uint64(info.Unit),
		},
		APIVersion: "2.0",
		MeshRF: apimodels.MeshRF{
			Status: "off",
		},
		Gridsquare: di.Config.Gridsquare,
		Node:       di.Config.ServerName,
		NodeDetails: apimodels.NodeDetails{
			MeshSupernode:        di.Config.Supernode,
			Description:          "Cloud Tunnel",
			Model:                "Virtual",
			MeshGateway:          true,
			BoardID:              "0x0000",
			FirmwareManufacturer: "github.com/USA-RedDragon/mesh-manager",
			FirmwareVersion:      di.Version,
		},
		Tunnels: apimodels.Tunnels{
			ActiveTunnelCount: activeTunnels,
		},
		LQM: lqm.LQM{
			Enabled: di.Config.LQM.Enabled,
		},
		Interfaces: getInterfaces(),
	}

	if doHosts {
		sysinfo.Hosts = getHosts(di.OLSRHostsParser, di.MeshLinkParser, modeHosts)
	}

	if doServices {
		sysinfo.Services = getServices(c.Request.Context(), di.OLSRServicesParser, di.MeshLinkParser, modeServices)
	}

	if doLocalServices {
		sysinfo.ServicesLocal = getServices(c.Request.Context(), di.OLSRServicesParser, di.MeshLinkParser, modeLocalServices)
	}

	if doNodes {
		sysinfo.Nodes = getHosts(di.OLSRHostsParser, di.MeshLinkParser, modeNodes)
	}

	if doLinkInfo {
		sysinfo.LinkInfo = getLinkInfo(c.Request.Context())
	}

	if doLQM && di.Config.LQM.Enabled {
		lqmInfo := getLQMInfo()
		if lqmInfo != nil {
			sysinfo.LQM.Info = *lqmInfo
		}
	}

	c.JSON(http.StatusOK, sysinfo)
}

func getInterfaces() []apimodels.Interface {
	ret := []apimodels.Interface{}

	ifaces, err := net.Interfaces()
	if err != nil {
		slog.Error("GETSysinfo: Unable to get interfaces", "error", err)
		return nil
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			slog.Error("GETSysinfo: Unable to get addresses for interface", "interface", iface.Name, "error", err)
			continue
		}
		if iface.Name == "lo" || iface.Name == "wg0" {
			continue
		}
		bestAddr := ""
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				slog.Error("GETSysinfo: Unable to parse address", "address", addr.String(), "error", err)
				continue
			}
			// Prefer IPv4
			if ip.To4() != nil {
				bestAddr = ip.String()
				break
			}
		}
		if bestAddr == "" {
			continue
		}
		i := apimodels.Interface{
			Name: iface.Name,
			IP:   bestAddr,
		}
		if len(iface.HardwareAddr) > 0 {
			i.MAC = iface.HardwareAddr.String()
		}
		ret = append(ret, i)
	}
	return ret
}

var (
	regexMid = regexp.MustCompile(`^mid\d+\..*`)
	regexDtd = regexp.MustCompile(`^dtdlink\..*`)
)

type hostMode int

const (
	modeHosts hostMode = iota
	modeNodes
)

func getHosts(olsrParser *olsr.HostsParser, meshlinkParser *meshlink.Parser, mode hostMode) []apimodels.Host {
	olsrHosts := olsrParser.GetHosts()
	meshlinkHosts := meshlinkParser.GetHosts()

	result := make(map[string]apimodels.Host)

	process := func(hostname string, ip net.IP) {
		if regexMid.MatchString(hostname) {
			return
		}
		if regexDtd.MatchString(hostname) {
			return
		}

		var key string
		switch mode {
		case modeHosts:
			key = hostname
		case modeNodes:
			key = ip.String()
		}

		existing, exists := result[key]
		if !exists {
			result[key] = apimodels.Host{Name: hostname, IP: ip.String()}
			return
		}

		// Node mode: prefer simpler (shorter) hostname
		if mode == modeNodes && len(hostname) < len(existing.Name) {
			result[key] = apimodels.Host{Name: hostname, IP: ip.String()}
		}
	}

	for _, h := range olsrHosts {
		process(h.Hostname, h.IP)
	}
	for _, h := range meshlinkHosts {
		process(h.Hostname, h.IP)
	}

	ret := make([]apimodels.Host, 0, len(result))
	for _, e := range result {
		ret = append(ret, e)
	}

	return ret
}

func getLinkInfo(ctx context.Context) map[string]apimodels.LinkInfo {
	ret := make(map[string]apimodels.LinkInfo)
	switch trackers := getLQMInfo().Trackers.(type) {
	case map[string]interface{}:
		for _, tracker := range trackers {
			tracker, ok := tracker.(lqm.Tracker)
			if !ok {
				continue
			}
			ip := tracker.IP
			if ip == "" && tracker.CanonicalIP != "" {
				ip = tracker.CanonicalIP
			}
			ret[ip] = apimodels.LinkInfo{
				LinkType:  apimodels.LinkType(strings.ToUpper(string(tracker.Type))),
				Hostname:  tracker.Hostname,
				Interface: tracker.Device,
			}
		}
	default:
		slog.Error("GETSysinfo: Unable to parse LQM trackers", "type", fmt.Sprintf("%T", trackers))
		return nil
	}
	return ret
}

type serviceMode int

const (
	modeServices serviceMode = iota
	modeLocalServices
)

func getServices(ctx context.Context, parser *olsr.ServicesParser, meshlinkParser *meshlink.Parser, mode serviceMode) []apimodels.Service {
	services := []apimodels.Service{}
	serviceRegex := regexp.MustCompile(`^([^|]*)\|([^|]*)\|(.*)$`)
	zeroRegex := regexp.MustCompile(`:0/`)

	switch mode {
	case modeLocalServices:
		// LOCALLY HOSTED SERVICES ONLY
		file, err := os.Open("/etc/meshlink/services")
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			matches := serviceRegex.FindStringSubmatch(line)
			if matches != nil && len(matches) == 4 {
				link := matches[1]
				if zeroRegex.MatchString(link) {
					link = ""
				}
				services = append(services, apimodels.Service{
					Name:     matches[3],
					Protocol: matches[2],
					Link:     link,
				})
			}
		}

	case modeServices:
		// ALL SERVICES
		entries, err := os.ReadDir("/var/run/meshlink/services")
		if err != nil {
			return nil
		}

		for _, entry := range entries {
			if entry.Name() == "." || entry.Name() == ".." {
				continue
			}

			filePath := "/var/run/meshlink/services/" + entry.Name()
			file, err := os.Open(filePath)
			if err != nil {
				continue
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				matches := serviceRegex.FindStringSubmatch(line)
				if matches != nil && len(matches) == 4 {
					link := matches[1]
					if zeroRegex.MatchString(link) {
						link = ""
					}
					services = append(services, apimodels.Service{
						Name:     matches[3],
						IP:       entry.Name(),
						Link:     link,
						Protocol: matches[2],
					})
				}
			}
			file.Close()
		}
	}

	return services
}

func getLQMInfo() *lqm.LQMInfo {
	// Read /tmp/lqm.info
	file, err := os.Open("/tmp/lqm.info")
	if err != nil {
		return nil
	}
	defer file.Close()

	var lqmData lqm.LQMInfo

	if err := json.NewDecoder(file).Decode(&lqmData); err != nil {
		return nil
	}

	return &lqmData
}
