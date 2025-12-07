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
	"github.com/USA-RedDragon/mesh-manager/internal/services/meshlink"
	"github.com/USA-RedDragon/mesh-manager/internal/services/olsr"
	"github.com/USA-RedDragon/mesh-manager/internal/utils"
	"github.com/gin-gonic/gin"
)

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
			exec.Command("killall", "-9", "iperf3").Run()
		} else {
			// Check if running
			out, _ := exec.Command("pidof", "iperf3").Output()
			if len(out) > 0 {
				fmt.Fprint(c.Writer, "<html><head><title>BUSY</title></head><body><pre>iperf server busy</pre></body></html>\n")
				return
			}
		}

		// Start server
		cmd := exec.Command("/usr/bin/iperf3", "-s", "-1", "--idle-timeout", "20", "--forceflush", "-B", "0.0.0.0")
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
			io.Copy(io.Discard, stdout)
			cmd.Wait()
		}()

		fmt.Fprint(c.Writer, "<html><head><title>RUNNING</title></head><body><pre>iperf server running</pre></body></html>\n")
		c.Writer.Flush()
		return
	} else {
		// Client mode
		if !strings.Contains(server, ".") {
			server += ".local.mesh"
		}

		// Resolve IP
		ips, err := net.LookupIP(server)
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
		remoteURL := fmt.Sprintf("http://%s:8080/cgi-bin/iperf?%sserver=", ip, killParam)
		resp, err := http.Get(remoteURL)
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

			if strings.Contains(line, "CLIENT DISABLED") {
				fmt.Fprint(c.Writer, "<html><head><title>SERVER DISABLED</title></head><body><pre>iperf server is disabled</pre></body></html>\n")
				return
			} else if strings.Contains(line, "BUSY") {
				fmt.Fprint(c.Writer, "<html><head><title>SERVER BUSY</title></head><body><pre>iperf server is busy</pre></body></html>\n")
				return
			} else if strings.Contains(line, "ERROR") {
				fmt.Fprint(c.Writer, "<html><head><title>SERVER ERROR</title></head><body><pre>iperf server error</pre></body></html>\n")
				return
			} else if strings.Contains(line, "RUNNING") {
				// Start local client
				args := []string{"--forceflush", "-b", "0", "-Z", "-c", ip, "-l", "16K"}
				if protocol == "udp" {
					args = append(args, "-u")
				}
				cmd := exec.Command("/usr/bin/iperf3", args...)

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

				cmd.Wait()
				fmt.Fprint(c.Writer, "</pre></body></html>\n")
				return
			}
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

	wgTunnels, err := models.CountWireguardTunnels(di.DB)
	if err != nil {
		slog.Error("GETSysinfo: Unable to get wireguard tunnels", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	legacyTunnels, err := models.CountLegacyTunnels(di.DB)
	if err != nil {
		slog.Error("GETSysinfo: Unable to get legacy tunnels", "error", err)
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

	servicesStr, exists := c.GetQuery("services")
	if !exists {
		servicesStr = "0"
	}
	doServices := servicesStr == "1"

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
		},
		APIVersion: "1.14",
		MeshRF: apimodels.MeshRF{
			Status: "off",
		},
		Gridsquare: di.Config.Gridsquare,
		Node:       di.Config.ServerName,
		NodeDetails: apimodels.NodeDetails{
			MeshSupernode:        di.Config.Supernode,
			Description:          "Cloud Tunnel",
			Model:                "Virtual",
			MeshGateway:          "1",
			BoardID:              "0x0000",
			FirmwareManufacturer: "github.com/USA-RedDragon/mesh-manager",
			FirmwareVersion:      di.Version,
		},
		Tunnels: apimodels.Tunnels{
			ActiveTunnelCount: activeTunnels,
			WireguardTunnelCount: wgTunnels,
			LegacyTunnelCount: legacyTunnels,
		},
		LQM: apimodels.LQM{
			Enabled: di.Config.LQM.Enabled,
		},
		Interfaces: getInterfaces(),
	}

	if doHosts {
		sysinfo.Hosts = getHosts(di.OLSRHostsParser, di.MeshLinkParser)
	}

	if doServices {
		sysinfo.Services = getServices(di.OLSRServicesParser)
	}

	if doLinkInfo {
		sysinfo.LinkInfo = getLinkInfo(c.Request.Context())
	}

	if doLQM && di.Config.LQM.Enabled {
		lqmInfo := getLQMInfo()
		if lqmInfo != nil {
			sysinfo.LQM.Info = lqmInfo
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
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				slog.Error("GETSysinfo: Unable to parse address", "address", addr.String(), "error", err)
				continue
			}
			ret = append(ret, apimodels.Interface{
				Name: iface.Name,
				IP:   ip.String(),
				MAC:  iface.HardwareAddr.String(),
			})
		}
	}
	return ret
}

var (
	regexMid = regexp.MustCompile(`^mid\d+\..*`)
	regexDtd = regexp.MustCompile(`^dtdlink\..*`)
)

func getHosts(olsrParser *olsr.HostsParser, meshlinkParser *meshlink.Parser) []apimodels.Host {
	hosts := olsrParser.GetHosts()
	meshlinkHosts := meshlinkParser.GetHosts()
	hostsMap := make(map[string]net.IP)
	ret := []apimodels.Host{}
	for _, host := range hosts {
		if regexMid.Match([]byte(host.Hostname)) {
			continue
		}

		if regexDtd.Match([]byte(host.Hostname)) {
			continue
		}

		hostsMap[host.Hostname] = host.IP
	}
	for _, host := range meshlinkHosts {
		if regexDtd.Match([]byte(host.Hostname)) {
			continue
		}

		hostsMap[host.Hostname] = host.IP
	}

	for hostname, ip := range hostsMap {
		ret = append(ret, apimodels.Host{
			Name: hostname,
			IP:   ip.String(),
		})
	}

	return ret
}

func getLinkInfo(ctx context.Context) map[string]apimodels.LinkInfo {
	ret := make(map[string]apimodels.LinkInfo)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	// http request http://localhost:9090/links
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:9090/links", nil)
	if err != nil {
		slog.Error("GETSysinfo: Unable to create request", "error", err)
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("GETSysinfo: Unable to get links", "error", err)
		return nil
	}
	defer resp.Body.Close()
	// Grab the body as json
	var links apimodels.OlsrdLinks
	err = json.NewDecoder(resp.Body).Decode(&links)
	if err != nil {
		slog.Error("GETSysinfo: Unable to decode links", "error", err)
		return nil
	}

	for _, link := range links.Links {
		hosts, err := net.LookupAddr(link.RemoteIP)
		if err != nil {
			continue
		}

		var hostname string
		if len(hosts) > 0 {
			hostname = hosts[0]
			// Strip off mid\d. from the hostname if it exists
			regex := regexp.MustCompile(`^[mM][iI][dD]\d+\.(.+)`)
			matches := regex.FindStringSubmatch(hostname)
			if len(matches) == 2 {
				hostname = matches[1]
			}
			// Strip off dtdlink. from the hostname if it exists
			regex = regexp.MustCompile(`^[dD][tT][dD][lL][iI][nN][kK]\.(.+)`)
			matches = regex.FindStringSubmatch(hostname)
			if len(matches) == 2 {
				hostname = matches[1]
			}
			// Make sure the hostname doesn't end with a period
			hostname = strings.TrimSuffix(hostname, ".")
			// Make sure the hostname doesn't end with .local.mesh
			hostname = strings.TrimSuffix(hostname, ".local.mesh")
		} else {
			continue
		}

		ips, err := net.LookupIP(hostname)
		if err != nil {
			continue
		}

		if len(ips) == 0 {
			continue
		}

		var linkType string
		switch {
		case strings.HasPrefix(link.OLSRInterface, "tun"):
			linkType = "TUN"
		case strings.HasPrefix(link.OLSRInterface, "eth"):
			linkType = "DTD"
		case strings.HasPrefix(link.OLSRInterface, "wg"):
			linkType = "WIREGUARD"
		case strings.HasPrefix(link.OLSRInterface, "br"):
			linkType = "DTD"
		default:
			linkType = "UNKNOWN"
		}

		ret[ips[0].String()] = apimodels.LinkInfo{
			HelloTime:           link.HelloTime,
			LostLinkTime:        link.LostLinkTime,
			LinkQuality:         link.LinkQuality,
			VTime:               link.VTime,
			LinkCost:            link.LinkCost,
			LinkType:            linkType,
			Hostname:            hostname,
			PreviousLinkStatus:  link.PreviousLinkStatus,
			CurrentLinkStatus:   link.CurrentLinkStatus,
			NeighborLinkQuality: link.NeighborLinkQuality,
			SymmetryTime:        link.SymmetryTime,
			SeqnoValid:          link.SeqnoValid,
			Pending:             link.Pending,
			LossHelloInterval:   link.LossHelloInterval,
			LossMultiplier:      link.LossMultiplier,
			Hysteresis:          link.Hysteresis,
			Seqno:               link.Seqno,
			LossTime:            link.LossTime,
			ValidityTime:        link.ValidityTime,
			OLSRInterface:       link.OLSRInterface,
			LastHelloTime:       link.LastHelloTime,
			AsymmetryTime:       link.AsymmetryTime,
		}
	}

	return ret
}

func getServices(parser *olsr.ServicesParser) []apimodels.Service {
	svcs := parser.GetServices()
	ret := []apimodels.Service{}
	for _, svc := range svcs {
		// we need to take the hostname from the URL and resolve it to an IP
		url, err := url.Parse(svc.URL)
		if err != nil {
			slog.Error("GETSysinfo: Unable to parse URL", "url", svc.URL, "error", err)
			continue
		}
		ips, err := net.LookupIP(url.Hostname())
		if err != nil {
			continue
		}
		link := svc.URL
		// If the link ends with :0/, then it is a non-http link, so set link to ""
		if strings.HasSuffix(svc.URL, ":0/") {
			link = ""
		}
		ret = append(ret, apimodels.Service{
			Name:     svc.Name,
			IP:       ips[0].String(),
			Protocol: svc.Protocol,
			Link:     link,
		})
	}

	return ret
}

func getLQMInfo() *apimodels.LQMInfo {
	// Read /tmp/lqm.info
	file, err := os.Open("/tmp/lqm.info")
	if err != nil {
		return nil
	}
	defer file.Close()

	var lqmData struct {
		Trackers map[string]map[string]interface{} `json:"trackers"`
	}

	if err := json.NewDecoder(file).Decode(&lqmData); err != nil {
		return nil
	}

	// Convert to apimodels format
	trackers := make(map[string]apimodels.LQMTracker)
	for mac, data := range lqmData.Trackers {
		tracker := apimodels.LQMTracker{}
		
		if hostname, ok := data["hostname"].(string); ok {
			tracker.Hostname = hostname
		}
		if pingSuccessTime, ok := data["ping_success_time"].(float64); ok {
			tracker.PingSuccessTime = pingSuccessTime
		}
		if pingQuality, ok := data["ping_quality"].(float64); ok {
			tracker.PingQuality = pingQuality
		}
		if quality, ok := data["quality"].(float64); ok {
			tracker.Quality = int(quality)
		}
		
		trackers[mac] = tracker
	}

	return &apimodels.LQMInfo{
		Trackers: trackers,
	}
}
