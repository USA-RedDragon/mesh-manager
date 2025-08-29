package apimodels

type Sysinfo struct {
	Uptime  string     `json:"uptime"`
	Loadavg [3]float64 `json:"loads"`
}

type MeshRF struct {
	Status string `json:"status"`
}

type NodeDetails struct {
	MeshSupernode        bool   `json:"mesh_supernode"`
	Description          string `json:"description"`
	Model                string `json:"model"`
	MeshGateway          string `json:"mesh_gateway"`
	BoardID              string `json:"board_id"`
	FirmwareManufacturer string `json:"firmware_mfg"`
	FirmwareVersion      string `json:"firmware_version"`
}

type Tunnels struct {
	ActiveTunnelCount int `json:"active_tunnel_count"`
	WireguardTunnelCount int `json:"wireguard_tunnel_count"`
	LegacyTunnelCount int `json:"legacy_tunnel_count"`
}

type LQM struct {
	Enabled bool `json:"enabled"`
}

type Interface struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	MAC  string `json:"mac"`
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

type LinkInfo struct {
	HelloTime           uint64  `json:"helloTime"`
	LostLinkTime        uint64  `json:"lostLinkTime"`
	LinkQuality         float32 `json:"linkQuality"`
	VTime               uint64  `json:"vtime"`
	LinkCost            float32 `json:"linkCost"`
	LinkType            string  `json:"linkType"`
	Hostname            string  `json:"hostname"`
	PreviousLinkStatus  string  `json:"previousLinkStatus"`
	CurrentLinkStatus   string  `json:"currentLinkStatus"`
	NeighborLinkQuality float32 `json:"neighborLinkQuality"`
	SymmetryTime        uint64  `json:"symmetryTime"`
	SeqnoValid          bool    `json:"seqnoValid"`
	Pending             bool    `json:"pending"`
	LossHelloInterval   uint64  `json:"lossHelloInterval"`
	LossMultiplier      uint64  `json:"lossMultiplier"`
	Hysteresis          float32 `json:"hysteresis"`
	Seqno               uint64  `json:"seqno"`
	LossTime            uint64  `json:"lossTime"`
	ValidityTime        uint64  `json:"validityTime"`
	OLSRInterface       string  `json:"olsrInterface"`
	LastHelloTime       uint64  `json:"lastHelloTime"`
	AsymmetryTime       uint64  `json:"asymmetryTime"`
}

type SysinfoResponse struct {
	Longitude   string              `json:"lon"`
	Latitude    string              `json:"lat"`
	Sysinfo     Sysinfo             `json:"sysinfo"`
	APIVersion  string              `json:"api_version"`
	MeshRF      MeshRF              `json:"meshrf"`
	Gridsquare  string              `json:"grid_square"`
	Node        string              `json:"node"`
	NodeDetails NodeDetails         `json:"node_details"`
	Tunnels     Tunnels             `json:"tunnels"`
	LQM         LQM                 `json:"lqm"`
	Interfaces  []Interface         `json:"interfaces"`
	Hosts       []Host              `json:"hosts"`
	Services    []Service           `json:"services"`
	LinkInfo    map[string]LinkInfo `json:"link_info"`
}
