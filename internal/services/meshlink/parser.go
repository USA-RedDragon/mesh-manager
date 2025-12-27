package meshlink

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const hostsDir = "/var/run/meshlink/hosts"
const servicesDir = "/var/run/meshlink/services"

type Parser struct {
	currentHosts []*Host
	nodesCount   int
	totalCount   int
	serviceCount int
	isParsing    atomic.Bool
	needParse    atomic.Bool // set if we get a call to Parse() while already parsing. We'll run Parse() again after the current parse is done to ensure we have the latest data
	lastModTime  atomic.Int64
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) GetHosts() []*Host {
	return p.currentHosts
}

func (p *Parser) GetHostsCount() int {
	return len(p.currentHosts)
}

func (p *Parser) GetServiceCount() int {
	return p.serviceCount
}

func (p *Parser) GetNodeHostsCount() int {
	return p.nodesCount
}

func (p *Parser) GetTotalHostsCount() int {
	return p.totalCount + p.nodesCount
}

func (p *Parser) GetHostsPaginated(page int, limit int, filter string) []*Host {
	ret := []*Host{}
	for _, host := range p.currentHosts {
		filter = strings.ToLower(filter)
		hostNameLower := strings.ToLower(host.Hostname)
		if strings.Contains(hostNameLower, filter) {
			ret = append(ret, host)
		}
	}
	start := (page - 1) * limit
	end := start + limit
	if start > len(ret) {
		return []*Host{}
	}
	if end > len(ret) {
		end = len(ret)
	}
	return ret[start:end]
}

func (p *Parser) Parse() error {
	latestModTime, err := newestModTime(hostsDir, servicesDir)
	if err != nil {
		return err
	}

	latestNanos := latestModTime.UnixNano()
	if latestNanos == 0 {
		latestNanos = time.Now().UnixNano()
	}

	if last := p.lastModTime.Load(); last != 0 && latestNanos <= last {
		return nil
	}

	if p.isParsing.Load() {
		p.needParse.Store(true)
		return nil
	}

	p.isParsing.Store(true)
	defer p.isParsing.Store(false)

	hosts, nodeCount, totalCount, serviceCount, err := parseHosts()
	if err != nil {
		return err
	}

	p.nodesCount = nodeCount
	p.totalCount = totalCount
	p.currentHosts = hosts
	p.serviceCount = serviceCount
	p.lastModTime.Store(latestNanos)

	if p.needParse.Load() {
		go func() {
			p.needParse.Store(false)
			if err := p.Parse(); err != nil {
				slog.Error("Error re-parsing hosts", "error", err)
			}
		}()
	}

	return nil
}

type HostData struct {
	Hostname string         `json:"hostname"`
	IP       net.IP         `json:"ip"`
	Services []*MeshService `json:"services"`
}

type MeshService struct {
	URL        string `json:"url"`
	Protocol   string `json:"protocol"`
	Name       string `json:"name"`
	ShouldLink bool   `json:"should_link"`
	Tag        string `json:"type"`
}

func (s *MeshService) String() string {
	ret := fmt.Sprintf("%s:\n\t", s.Name)
	ret += fmt.Sprintf("%s\t%s", s.Protocol, s.URL)
	return ret
}

type Host struct {
	HostData
	Children []HostData `json:"children"`
}

func (h *Host) addChild(child HostData) {
	h.Children = append(h.Children, child)
}

func (h *Host) String() string {
	ret := fmt.Sprintf("%s: %s\n", h.Hostname, h.IP)
	for _, child := range h.Children {
		ret += fmt.Sprintf("\t%s: %s\n", child.Hostname, child.IP)
	}
	return ret
}

var (
	regexMesh   = regexp.MustCompile(`\s[^\.]+$`)
	taggedRegex = regexp.MustCompile(`^(.*)\s+\[(.*)\]$`)
)

func newestModTime(paths ...string) (time.Time, error) {
	var newest time.Time

	for _, root := range paths {
		err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			if info.ModTime().After(newest) {
				newest = info.ModTime()
			}

			return nil
		})
		if err != nil {
			return time.Time{}, err
		}
	}

	return newest, nil
}

func parseHosts() (ret []*Host, count int, totalCount int, serviceCount int, err error) {
	err = fs.WalkDir(os.DirFS(hostsDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("Error reading hosts directory", "error", err)
			return err
		}
		if d.IsDir() {
			return nil // Skip directories
		}

		file := filepath.Join(hostsDir, path)

		slog.Debug("parseHosts: Processing hosts file", "file", file)

		entries, err := os.ReadFile(file)
		if err != nil {
			slog.Error("parseHosts: Error reading hosts directory entry", "entry", file, "error", err)
		}

		totalCount++

		var host *Host

		for _, line := range strings.Split(string(entries), "\n") {
			// Ignore empty lines
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}

			slog.Debug("parseHosts: Processing hosts file line", "file", file, "line", line)

			fields := strings.Fields(line)
			if len(fields) < 2 {
				slog.Warn("parseHosts: Invalid host entry", "file", file, "line", line)
				continue
			}

			if regexMesh.Match([]byte(line)) && host == nil {
				slog.Debug("parseHosts: Found host entry", "file", file, "line", line)
				host = &Host{
					HostData: HostData{
						Hostname: strings.TrimSpace(fields[1]),
						IP:       net.ParseIP(strings.TrimSpace(fields[0])),
					},
				}
				if host.IP == nil {
					slog.Warn("parseHosts: Invalid IP in hosts file", "file", file, "line", line)
					continue
				}
				// Check if the same base filename exists under the services directory
				servicesFile := filepath.Join(servicesDir, host.IP.To4().String())
				if services, err := os.ReadFile(servicesFile); err == nil {
					slog.Debug("parseHosts: Found services file for host", "file", servicesFile)
					var servicesList []*MeshService
					for _, svcLine := range strings.Split(string(services), "\n") {
						line := strings.TrimSpace(svcLine)

						// Ignore empty lines
						if len(line) == 0 {
							slog.Debug("parseHosts: Skipping empty line in services file", "file", servicesFile)
							continue
						}

						// Lines are of the form:
						// url|protocol|name

						// Split the line by '|'
						split := strings.Split(line, "|")
						if len(split) != 3 {
							slog.Warn("parseHosts: Invalid service line format", "line", line, "file", servicesFile)
							continue
						}

						url, err := url.Parse(split[0])
						if err != nil {
							slog.Warn("parseHosts: Error parsing URL", "url", split[0], "error", err)
							continue
						}

						// Name can have an optional tag suffix like 'Meshchat [chat]'
						name := split[2]
						tag := ""
						if matches := taggedRegex.FindStringSubmatch(name); len(matches) == 3 {
							name = matches[1]
							tag = matches[2]
						}

						service := &MeshService{
							URL:        url.String(),
							Protocol:   split[1],
							Name:       name,
							ShouldLink: url.Port() != "0",
							Tag:        tag,
						}

						serviceCount++

						servicesList = append(servicesList, service)
					}
					host.Services = append(host.Services, servicesList...)
				}
			} else {
				slog.Debug("parseHosts: Found child host entry", "file", file, "line", line)
				if host == nil {
					slog.Warn("parseHosts: Found a host entry without a parent host", "line", line)
					continue
				}
				// This is a child of the last host
				child := HostData{
					Hostname: strings.TrimSuffix(strings.TrimSpace(fields[1]), ".local.mesh"),
					IP:       net.ParseIP(strings.TrimSpace(fields[0])),
				}
				if child.IP == nil {
					slog.Warn("parseHosts: Invalid IP in hosts file", "line", line)
					continue
				}
				if strings.HasPrefix(child.Hostname, "lan.") ||
					strings.HasPrefix(child.Hostname, "dtdlink.") ||
					strings.HasPrefix(child.Hostname, "babel.") ||
					strings.HasPrefix(child.Hostname, "supernode.") {
					continue
				}
				host.addChild(child)
			}
		}

		if host != nil {
			ret = append(ret, host)
			count++
		}
		return nil
	})

	return
}
