package apimodels

import "github.com/USA-RedDragon/mesh-manager/internal/services/lqm"

type Sysinfo struct {
	Uptime     string     `json:"uptime"`
	Loadavg    [3]float64 `json:"loads"`
	FreeMemory uint64     `json:"freememory"`
}

type MeshRF struct {
	Status string `json:"status"`
}

type NodeDetails struct {
	MeshSupernode        bool   `json:"mesh_supernode"`
	Description          string `json:"description"`
	Model                string `json:"model"`
	MeshGateway          bool   `json:"mesh_gateway"`
	BoardID              string `json:"board_id"`
	FirmwareManufacturer string `json:"firmware_mfg"`
	FirmwareVersion      string `json:"firmware_version"`
}

type Tunnels struct {
	ActiveTunnelCount int `json:"active_tunnel_count"`
}

type Interface struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	MAC  string `json:"mac,omitempty"`
}

type Host struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type Service struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Protocol string `json:"protocol"`
	Link     string `json:"link"`
}

type LinkType string

const (
	LinkTypeWireguard LinkType = "WIREGUARD"
	LinkTypeDTD       LinkType = "DTD"
	LinkTypeTun       LinkType = "TUN"
)

type LinkInfo struct {
	Hostname  string   `json:"hostname"`
	LinkType  LinkType `json:"linkType"`
	Interface string   `json:"interface"`
}

type SysinfoResponse struct {
	Longitude     float64             `json:"lon"`
	Latitude      float64             `json:"lat"`
	Sysinfo       Sysinfo             `json:"sysinfo"`
	APIVersion    string              `json:"api_version"`
	MeshRF        MeshRF              `json:"meshrf"`
	Gridsquare    string              `json:"grid_square"`
	Node          string              `json:"node"`
	Nodes         []Host              `json:"nodes,omitempty"`
	NodeDetails   NodeDetails         `json:"node_details"`
	Tunnels       Tunnels             `json:"tunnels"`
	LQM           lqm.LQM             `json:"lqm,omitempty"`
	Interfaces    []Interface         `json:"interfaces"`
	Hosts         []Host              `json:"hosts,omitempty"`
	Services      []Service           `json:"services,omitempty"`
	ServicesLocal []Service           `json:"services_local,omitempty"`
	LinkInfo      map[string]LinkInfo `json:"link_info,omitempty"`
}
