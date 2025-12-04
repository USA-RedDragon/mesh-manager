package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
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
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.Data(http.StatusInternalServerError, "text/html", []byte("<html><head><title>ERROR</title></head><body><pre>Try again later</pre></body></html>\n"))
		return
	}

	server := c.Query("server")
	protocol := c.DefaultQuery("protocol", "tcp")
	killParam := c.Query("kill")
	kill := killParam == "1"

	// Validate protocol - must be exactly "tcp" or "udp"
	if protocol != "tcp" && protocol != "udp" {
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>ERROR</title></head><body><pre>Invalid protocol. Must be tcp or udp</pre></body></html>\n"))
		return
	}

	if server == "" {
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>ERROR</title></head><body><pre>Provide a server name to run a test between this client and a server [/cgi-bin/iperf?server=&lt;ServerName&gt;&amp;protocol=&lt;udp|tcp&gt;]</pre></body></html>\n"))
		return
	}

	// Validate server name - only allow alphanumeric, dots, and hyphens
	for _, ch := range server {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '.' || ch == '-') {
			c.Data(http.StatusOK, "text/html", []byte("<html><head><title>ERROR</title></head><body><pre>Illegal server name</pre></body></html>\n"))
			return
		}
	}

	// iperf client mode
	// Add .local.mesh if no dots in the name
	if !strings.Contains(server, ".") {
		server += ".local.mesh"
	}

	// Resolve the hostname to an IP
	ips, err := net.LookupIP(server)
	if err != nil || len(ips) == 0 {
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>SERVER ERROR</title></head><body><pre>iperf no such server</pre></body></html>\n"))
		return
	}

	ip := ips[0].String()

	// Additional validation: ensure IP is valid IPv4 or IPv6
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>SERVER ERROR</title></head><body><pre>Invalid IP address resolved</pre></body></html>\n"))
		return
	}

	// Call the remote server to start iperf server
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	remoteURL := fmt.Sprintf("http://%s:8080/cgi-bin/iperf?", ip)
	if kill {
		remoteURL += "kill=1&"
	}
	remoteURL += "server="

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, remoteURL, nil)
	if err != nil {
		slog.Error("GETIPerf: Unable to create request", "error", err)
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>CLIENT ERROR</title></head><body><pre>iperf failed to call remote server</pre></body></html>\n"))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("GETIPerf: Unable to call remote server", "error", err)
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>CLIENT ERROR</title></head><body><pre>iperf failed to call remote server</pre></body></html>\n"))
		return
	}
	defer resp.Body.Close()

	// Read the response to check status
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("GETIPerf: Unable to read response", "error", err)
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>ERROR</title></head><body><pre>iperf unknown error</pre></body></html>\n"))
		return
	}

	responseStr := string(body)

	if strings.Contains(responseStr, "CLIENT DISABLED") {
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>SERVER DISABLED</title></head><body><pre>iperf server is disabled</pre></body></html>\n"))
		return
	}

	if strings.Contains(responseStr, "BUSY") {
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>SERVER BUSY</title></head><body><pre>iperf server is busy</pre></body></html>\n"))
		return
	}

	if strings.Contains(responseStr, "ERROR") {
		c.Data(http.StatusOK, "text/html", []byte("<html><head><title>SERVER ERROR</title></head><body><pre>iperf server error</pre></body></html>\n"))
		return
	}

	if strings.Contains(responseStr, "RUNNING") {
		// Server is ready, run iperf3 client
		// Build args slice safely - never pass user input directly to shell
		args := []string{
			"--forceflush",
			"--rcv-timeout", "20000",
			"-b", "0",
			"-Z",
			"-c", ip,
			"-l", "16K",
		}

		if protocol == "udp" {
			args = append(args, "-u")
		}

		cmd := exec.CommandContext(c.Request.Context(), "/usr/bin/iperf3", args...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("GETIPerf: iperf3 client failed", "error", err, "output", string(output))
			c.Data(http.StatusOK, "text/html", []byte("<html><head><title>CLIENT ERROR</title></head><body><pre>iperf client failed</pre></body></html>\n"))
			return
		}

		// Format the response
		response := fmt.Sprintf("<html><head><title>SUCCESS</title></head><body><pre>Client: %s\nServer: %s\n%s</pre></body></html>\n",
			di.Config.ServerName, server, string(output))
		c.Data(http.StatusOK, "text/html", []byte(response))
		return
	}

	// Unknown response
	c.Data(http.StatusOK, "text/html", []byte("<html><head><title>ERROR</title></head><body><pre>iperf unknown error</pre></body></html>\n"))
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
			Enabled: false,
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
