package apimodels

import (
	"encoding/json"
	"io"

	"github.com/USA-RedDragon/mesh-manager/internal/services/lqm"
)

type SysinfoResponse struct {
	APIVersion              string                   `json:"api_version"`
	SysinfoResponse1Point0  *SysinfoResponse1Point0  `json:"-"`
	SysinfoResponse1Point5  *SysinfoResponse1Point5  `json:"-"`
	SysinfoResponse1Point6  *SysinfoResponse1Point6  `json:"-"`
	SysinfoResponse1Point7  *SysinfoResponse1Point7  `json:"-"`
	SysinfoResponse1Point8  *SysinfoResponse1Point8  `json:"-"`
	SysinfoResponse1Point9  *SysinfoResponse1Point9  `json:"-"`
	SysinfoResponse1Point10 *SysinfoResponse1Point10 `json:"-"`
	SysinfoResponse1Point11 *SysinfoResponse1Point11 `json:"-"`
	SysinfoResponse1Point12 *SysinfoResponse1Point12 `json:"-"`
	SysinfoResponse1Point13 *SysinfoResponse1Point13 `json:"-"`
	SysinfoResponse1Point14 *SysinfoResponse1Point14 `json:"-"`
	SysinfoResponse2Point0  *SysinfoResponse2Point0  `json:"-"`
}

func (s *SysinfoResponse) Decode(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s.APIVersion {
	case "1.0":
		s.SysinfoResponse1Point0 = &SysinfoResponse1Point0{}
		return json.Unmarshal(data, s.SysinfoResponse1Point0)
	case "1.5":
		s.SysinfoResponse1Point5 = &SysinfoResponse1Point5{}
		return json.Unmarshal(data, s.SysinfoResponse1Point5)
	case "1.6":
		s.SysinfoResponse1Point6 = &SysinfoResponse1Point6{}
		return json.Unmarshal(data, s.SysinfoResponse1Point6)
	case "1.7":
		s.SysinfoResponse1Point7 = &SysinfoResponse1Point7{}
		return json.Unmarshal(data, s.SysinfoResponse1Point7)
	case "1.8":
		s.SysinfoResponse1Point8 = &SysinfoResponse1Point8{}
		return json.Unmarshal(data, s.SysinfoResponse1Point8)
	case "1.9":
		s.SysinfoResponse1Point9 = &SysinfoResponse1Point9{}
		return json.Unmarshal(data, s.SysinfoResponse1Point9)
	case "1.10":
		s.SysinfoResponse1Point10 = &SysinfoResponse1Point10{}
		return json.Unmarshal(data, s.SysinfoResponse1Point10)
	case "1.11":
		s.SysinfoResponse1Point11 = &SysinfoResponse1Point11{}
		return json.Unmarshal(data, s.SysinfoResponse1Point11)
	case "1.12":
		s.SysinfoResponse1Point12 = &SysinfoResponse1Point12{}
		return json.Unmarshal(data, s.SysinfoResponse1Point12)
	case "1.13":
		s.SysinfoResponse1Point13 = &SysinfoResponse1Point13{}
		return json.Unmarshal(data, s.SysinfoResponse1Point13)
	case "1.14":
		s.SysinfoResponse1Point14 = &SysinfoResponse1Point14{}
		return json.Unmarshal(data, s.SysinfoResponse1Point14)
	case "2.0":
		s.SysinfoResponse2Point0 = &SysinfoResponse2Point0{}
		return json.Unmarshal(data, s.SysinfoResponse2Point0)
	default:
		return nil
	}
}

func (s *SysinfoResponse) GetObject() any {
	switch s.APIVersion {
	case "1.0":
		return s.SysinfoResponse1Point0
	case "1.5":
		return s.SysinfoResponse1Point5
	case "1.6":
		return s.SysinfoResponse1Point6
	case "1.7":
		return s.SysinfoResponse1Point7
	case "1.8":
		return s.SysinfoResponse1Point8
	case "1.9":
		return s.SysinfoResponse1Point9
	case "1.10":
		return s.SysinfoResponse1Point10
	case "1.11":
		return s.SysinfoResponse1Point11
	case "1.12":
		return s.SysinfoResponse1Point12
	case "1.13":
		return s.SysinfoResponse1Point13
	case "1.14":
		return s.SysinfoResponse1Point14
	case "2.0":
		return s.SysinfoResponse2Point0
	default:
		return nil
	}
}

// GetHosts returns the hosts from the appropriate version of the response
func (s *SysinfoResponse) GetHosts() []Host {
	switch s.APIVersion {
	case "1.0":
		return nil
	case "1.5", "1.6", "1.7", "1.8", "1.9", "1.10", "1.11":
		if s.SysinfoResponse1Point11 != nil {
			return s.SysinfoResponse1Point11.Hosts
		}
		if s.SysinfoResponse1Point10 != nil {
			return s.SysinfoResponse1Point10.Hosts
		}
		if s.SysinfoResponse1Point9 != nil {
			return s.SysinfoResponse1Point9.Hosts
		}
		if s.SysinfoResponse1Point8 != nil {
			return s.SysinfoResponse1Point8.Hosts
		}
		if s.SysinfoResponse1Point7 != nil {
			return s.SysinfoResponse1Point7.Hosts
		}
		if s.SysinfoResponse1Point6 != nil {
			return s.SysinfoResponse1Point6.Hosts
		}
		if s.SysinfoResponse1Point5 != nil {
			return s.SysinfoResponse1Point5.Hosts
		}
	case "1.12", "1.13", "1.14":
		if s.SysinfoResponse1Point14 != nil {
			return s.SysinfoResponse1Point14.Hosts
		}
		if s.SysinfoResponse1Point13 != nil {
			return s.SysinfoResponse1Point13.Hosts
		}
		if s.SysinfoResponse1Point12 != nil {
			return s.SysinfoResponse1Point12.Hosts
		}
	case "2.0":
		if s.SysinfoResponse2Point0 != nil {
			return s.SysinfoResponse2Point0.Hosts
		}
	}
	return nil
}

// GetLatitude returns the latitude from the appropriate version of the response
func (s *SysinfoResponse) GetLatitude() float64 {
	switch s.APIVersion {
	case "1.0":
		if s.SysinfoResponse1Point0 != nil {
			return s.SysinfoResponse1Point0.Latitude
		}
	case "1.5", "1.6", "1.7", "1.8", "1.9", "1.10", "1.11":
		if s.SysinfoResponse1Point11 != nil {
			return s.SysinfoResponse1Point11.Latitude
		}
		if s.SysinfoResponse1Point10 != nil {
			return s.SysinfoResponse1Point10.Latitude
		}
		if s.SysinfoResponse1Point9 != nil {
			return s.SysinfoResponse1Point9.Latitude
		}
		if s.SysinfoResponse1Point8 != nil {
			return s.SysinfoResponse1Point8.Latitude
		}
		if s.SysinfoResponse1Point7 != nil {
			return s.SysinfoResponse1Point7.Latitude
		}
		if s.SysinfoResponse1Point6 != nil {
			return s.SysinfoResponse1Point6.Latitude
		}
		if s.SysinfoResponse1Point5 != nil {
			return s.SysinfoResponse1Point5.Latitude
		}
	case "1.12", "1.13", "1.14":
		if s.SysinfoResponse1Point14 != nil {
			return s.SysinfoResponse1Point14.Latitude
		}
		if s.SysinfoResponse1Point13 != nil {
			return s.SysinfoResponse1Point13.Latitude
		}
		if s.SysinfoResponse1Point12 != nil {
			return s.SysinfoResponse1Point12.Latitude
		}
	case "2.0":
		if s.SysinfoResponse2Point0 != nil {
			return s.SysinfoResponse2Point0.Latitude
		}
	}
	return 0
}

// GetLongitude returns the longitude from the appropriate version of the response
func (s *SysinfoResponse) GetLongitude() float64 {
	switch s.APIVersion {
	case "1.0":
		if s.SysinfoResponse1Point0 != nil {
			return s.SysinfoResponse1Point0.Longitude
		}
	case "1.5", "1.6", "1.7", "1.8", "1.9", "1.10", "1.11":
		if s.SysinfoResponse1Point11 != nil {
			return s.SysinfoResponse1Point11.Longitude
		}
		if s.SysinfoResponse1Point10 != nil {
			return s.SysinfoResponse1Point10.Longitude
		}
		if s.SysinfoResponse1Point9 != nil {
			return s.SysinfoResponse1Point9.Longitude
		}
		if s.SysinfoResponse1Point8 != nil {
			return s.SysinfoResponse1Point8.Longitude
		}
		if s.SysinfoResponse1Point7 != nil {
			return s.SysinfoResponse1Point7.Longitude
		}
		if s.SysinfoResponse1Point6 != nil {
			return s.SysinfoResponse1Point6.Longitude
		}
		if s.SysinfoResponse1Point5 != nil {
			return s.SysinfoResponse1Point5.Longitude
		}
	case "1.12", "1.13", "1.14":
		if s.SysinfoResponse1Point14 != nil {
			return s.SysinfoResponse1Point14.Longitude
		}
		if s.SysinfoResponse1Point13 != nil {
			return s.SysinfoResponse1Point13.Longitude
		}
		if s.SysinfoResponse1Point12 != nil {
			return s.SysinfoResponse1Point12.Longitude
		}
	case "2.0":
		if s.SysinfoResponse2Point0 != nil {
			return s.SysinfoResponse2Point0.Longitude
		}
	}
	return 0
}

// GetMeshSupernode returns whether this is a mesh supernode from the appropriate version of the response
func (s *SysinfoResponse) GetMeshSupernode() bool {
	switch s.APIVersion {
	case "1.0", "1.5", "1.6", "1.7", "1.8", "1.9", "1.10":
		return false
	case "1.11":
		if s.SysinfoResponse1Point11 != nil {
			return s.SysinfoResponse1Point11.NodeDetails.MeshSupernode
		}
	case "1.12", "1.13", "1.14":
		if s.SysinfoResponse1Point14 != nil {
			return s.SysinfoResponse1Point14.NodeDetails.MeshSupernode
		}
		if s.SysinfoResponse1Point13 != nil {
			return s.SysinfoResponse1Point13.NodeDetails.MeshSupernode
		}
		if s.SysinfoResponse1Point12 != nil {
			return s.SysinfoResponse1Point12.NodeDetails.MeshSupernode
		}
	case "2.0":
		if s.SysinfoResponse2Point0 != nil {
			return s.SysinfoResponse2Point0.NodeDetails.MeshSupernode
		}
	}
	return false
}

// GetLinkInfo returns the link info from the appropriate version of the response
func (s *SysinfoResponse) GetLinkInfo() any {
	switch s.APIVersion {
	case "1.0", "1.5", "1.6":
		return nil
	case "1.7", "1.8", "1.9", "1.10", "1.11":
		if s.SysinfoResponse1Point11 != nil {
			return s.SysinfoResponse1Point11.LinkInfo
		}
		if s.SysinfoResponse1Point10 != nil {
			return s.SysinfoResponse1Point10.LinkInfo
		}
		if s.SysinfoResponse1Point9 != nil {
			return s.SysinfoResponse1Point9.LinkInfo
		}
		if s.SysinfoResponse1Point8 != nil {
			return s.SysinfoResponse1Point8.LinkInfo
		}
		if s.SysinfoResponse1Point7 != nil {
			return s.SysinfoResponse1Point7.LinkInfo
		}
	case "1.12", "1.13", "1.14":
		if s.SysinfoResponse1Point14 != nil {
			return s.SysinfoResponse1Point14.LinkInfo
		}
		if s.SysinfoResponse1Point13 != nil {
			return s.SysinfoResponse1Point13.LinkInfo
		}
		if s.SysinfoResponse1Point12 != nil {
			return s.SysinfoResponse1Point12.LinkInfo
		}
	case "2.0":
		if s.SysinfoResponse2Point0 != nil {
			return s.SysinfoResponse2Point0.LinkInfo
		}
	}
	return nil
}

// SetLinkInfo sets the link info for the appropriate version of the response
func (s *SysinfoResponse) SetLinkInfo(in any) {
	switch s.APIVersion {
	case "1.0", "1.5", "1.6":
		return
	case "1.7", "1.8", "1.9", "1.10", "1.11":
		if val, ok := in.(map[string]LinkInfo1Point7); ok {
			if s.SysinfoResponse1Point11 != nil {
				s.SysinfoResponse1Point11.LinkInfo = val
				return
			}
			if s.SysinfoResponse1Point10 != nil {
				s.SysinfoResponse1Point10.LinkInfo = val
				return
			}
			if s.SysinfoResponse1Point9 != nil {
				s.SysinfoResponse1Point9.LinkInfo = val
				return
			}
			if s.SysinfoResponse1Point8 != nil {
				s.SysinfoResponse1Point8.LinkInfo = val
				return
			}
			if s.SysinfoResponse1Point7 != nil {
				s.SysinfoResponse1Point7.LinkInfo = val
				return
			}
		}
	case "1.12", "1.13", "1.14":
		if val, ok := in.(map[string]LinkInfo1Point7); ok {
			if s.SysinfoResponse1Point14 != nil {
				s.SysinfoResponse1Point14.LinkInfo = val
				return
			}
			if s.SysinfoResponse1Point13 != nil {
				s.SysinfoResponse1Point13.LinkInfo = val
				return
			}
			if s.SysinfoResponse1Point12 != nil {
				s.SysinfoResponse1Point12.LinkInfo = val
				return
			}
		}
	case "2.0":
		if val, ok := in.(map[string]LinkInfo2Point0); ok {
			if s.SysinfoResponse2Point0 != nil {
				s.SysinfoResponse2Point0.LinkInfo = val
				return
			}
		}
	}
}

type SysinfoCommon struct {
	Uptime  string     `json:"uptime"`
	Loadavg [3]float64 `json:"loads"`
}

type Sysinfo2Point0 struct {
	SysinfoCommon
	FreeMemory uint64 `json:"freememory,string"`
}

type Sysinfo1Point5 struct {
	SysinfoCommon
}

type Sysinfo1Point11 struct {
	SysinfoCommon
}

type meshRFCommon struct {
	SSID string `json:"ssid,omitempty"`
}

type MeshRF1Point5 struct {
	meshRFCommon
	Channel          int `json:"channel,string,omitempty"`
	ChannelBandwidth int `json:"chanbw,string,omitempty"`
}

type MeshRF1Point6 struct {
	MeshRF1Point5
	Status string `json:"status,omitempty"`
}

type MeshRF1Point7 struct {
	MeshRF1Point5
	Status    string  `json:"status,omitempty"`
	Frequency float64 `json:"freq,string,omitempty"`
}

type MeshRF1Point13 struct {
	MeshRF1Point7
	Azimuth    int     `json:"azimuth,string,omitempty"`
	Elevation  int     `json:"elevation,string,omitempty"`
	Height     float64 `json:"height,string,omitempty"`
	Antenna    any     `json:"antenna,omitempty"`
	AntennaAux any     `json:"antenna_aux,omitempty"`
	Mode       string  `json:"mode,omitempty"`
}

type MeshRF2Point0 struct {
	MeshRF1Point13
	Polarization     string  `json:"polarization,omitempty"`
	Channel          int     `json:"channel,omitempty"`
	ChannelBandwidth int     `json:"chanbw,omitempty"`
	Frequency        float64 `json:"freq,omitempty"`
	Height           float64 `json:"height,omitempty"`
	Azimuth          int     `json:"azimuth,omitempty"`
	Elevation        int     `json:"elevation,omitempty"`
}

type NodeDetailsCommon struct {
	Model                string `json:"model"`
	Description          string `json:"description"`
	BoardID              string `json:"board_id"`
	FirmwareManufacturer string `json:"firmware_mfg"`
	FirmwareVersion      string `json:"firmware_version"`
}

type NodeDetails1Point5 struct {
	NodeDetailsCommon
}

type BoolString bool

func (b *BoolString) UnmarshalJSON(data []byte) error {
	if string(data) == "1" {
		*b = true
	} else {
		*b = false
	}
	return nil
}

type NodeDetails1Point8 struct {
	NodeDetailsCommon
	MeshGateway BoolString `json:"mesh_gateway"`
}

type NodeDetails1Point11 struct {
	NodeDetails1Point8
	MeshSupernode bool `json:"mesh_supernode"`
}

type NodeDetails2Point0 struct {
	NodeDetails1Point11
	MeshGateway bool `json:"mesh_gateway"`
}

type Tunnels1Point5 struct {
	TunnelInstalled   bool `json:"tunnel_installed,string"`
	ActiveTunnelCount int  `json:"active_tunnel_count,string"`
}

type Tunnels1Point10 struct {
	ActiveTunnelCount int `json:"active_tunnel_count"`
}

type Tunnels1Point14 struct {
	Tunnels1Point10
	LegacyTunnelCount    int `json:"legacy_tunnel_count"`
	WireguardTunnelCount int `json:"wireguard_tunnel_count"`
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

type ServiceCommon struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Link     string `json:"link"`
}

type Service1Point5 struct {
	ServiceCommon
}

type Service2Point0 struct {
	ServiceCommon
	IP string `json:"ip"`
}

type LinkType string

const (
	LinkTypeWireguard LinkType = "WIREGUARD"
	LinkTypeDTD       LinkType = "DTD"
	LinkTypeRF        LinkType = "RF"
	LinkTypeTun       LinkType = "TUN"
	LinkTypeSupernode LinkType = "SUPERNODE"
)

type LinkInfoCommon struct {
	Hostname string   `json:"hostname"`
	LinkType LinkType `json:"linkType"`
}

type LinkInfo1Point7 struct {
	LinkInfoCommon
	OLSRInterface       string  `json:"olsrInterface"`
	LinkQuality         float64 `json:"linkQuality"`
	NeighborLinkQuality float64 `json:"neighborLinkQuality"`
	Signal              float64 `json:"signal,omitempty"`
	Noise               float64 `json:"noise,omitempty"`
	TXRate              float64 `json:"tx_rate,omitempty"`
	RXRate              float64 `json:"rx_rate,omitempty"`
}

type LinkInfo2Point0 struct {
	LinkInfoCommon
	Interface string `json:"interface"`
}

type LQM1Point11 struct {
	Enabled bool              `json:"enabled"`
	Config  LQMConfig1Point11 `json:"config"`
	Info    any               `json:"info"`
}

type LQMConfig1Point11 struct {
	MinSNR        int         `json:"min_snr"`
	MarginSNR     int         `json:"margin_snr"`
	MinDistance   int         `json:"min_distance"`
	MaxDistance   int         `json:"max_distance"`
	AutoDistance  int         `json:"auto_distance"`
	MinQuality    int         `json:"min_quality"`
	MarginQuality int         `json:"margin_quality"`
	PingPenalty   int         `json:"ping_penalty"`
	UserBlocks    map[any]any `json:"user_blocks"`
	UserAllows    []string    `json:"user_allowlist"`
}

type SysinfoResponseCommon struct {
	APIVersion string      `json:"api_version"`
	Node       string      `json:"node"`
	Gridsquare string      `json:"grid_square"`
	Interfaces []Interface `json:"interfaces"`
}

type SysinfoResponse1Point0 struct {
	SysinfoResponseCommon
	Model                string  `json:"model"`
	BoardID              string  `json:"board_id"`
	FirmwareManufacturer string  `json:"firmware_mfg"`
	FirmwareVersion      string  `json:"firmware_version"`
	TunnelInstalled      bool    `json:"tunnel_installed,string"`
	SSID                 string  `json:"ssid"`
	Channel              int     `json:"channel,string"`
	ChannelBandwidth     int     `json:"chanbw,string"`
	ActiveTunnelCount    int     `json:"active_tunnel_count,string"`
	Latitude             float64 `json:"lat,string"`
	Longitude            float64 `json:"lon,string"`
}

type SysinfoResponse1Point5 struct {
	SysinfoResponseCommon
	NodeDetails   NodeDetails1Point5 `json:"node_details"`
	MeshRF        MeshRF1Point5      `json:"meshrf"`
	Tunnels       Tunnels1Point5     `json:"tunnels"`
	Latitude      float64            `json:"lat,string"`
	Longitude     float64            `json:"lon,string"`
	Sysinfo       Sysinfo1Point5     `json:"sysinfo"`
	Hosts         []Host             `json:"hosts,omitempty"`
	Services      []Service1Point5   `json:"services,omitempty"`
	ServicesLocal []Service1Point5   `json:"services_local,omitempty"`
}

type SysinfoResponse1Point6 struct {
	SysinfoResponse1Point5
	MeshRF MeshRF1Point6 `json:"meshrf"`
}

type SysinfoResponse1Point7 struct {
	SysinfoResponse1Point6
	MeshRF   MeshRF1Point7              `json:"meshrf"`
	LinkInfo map[string]LinkInfo1Point7 `json:"link_info,omitempty"`
}

type SysinfoResponse1Point8 struct {
	SysinfoResponse1Point7
	NodeDetails NodeDetails1Point8 `json:"node_details"`
}

type SysinfoResponse1Point9 struct {
	SysinfoResponse1Point8
}

type SysinfoResponse1Point10 struct {
	SysinfoResponse1Point9
	Tunnels Tunnels1Point10 `json:"tunnels"`
}

type SysinfoResponse1Point11 struct {
	SysinfoResponse1Point10
	LQM         LQM1Point11         `json:"lqm,omitempty"`
	NodeDetails NodeDetails1Point11 `json:"node_details"`
}

type SysinfoResponse1Point12 struct {
	SysinfoResponse1Point11
	Nodes []Host `json:"nodes,omitempty"`
}

type SysinfoResponse1Point13 struct {
	SysinfoResponse1Point12
	MeshRF   MeshRF1Point13 `json:"meshrf"`
	Topology any            `json:"topology,omitempty"`
}

type SysinfoResponse1Point14 struct {
	SysinfoResponse1Point12
	Tunnels Tunnels1Point14 `json:"tunnels"`
	LQM     lqm.LQM         `json:"lqm,omitempty"`
}

type SysinfoResponse2Point0 struct {
	SysinfoResponse1Point14
	Longitude     float64                    `json:"lon"`
	Latitude      float64                    `json:"lat"`
	Sysinfo       Sysinfo2Point0             `json:"sysinfo"`
	MeshRF        MeshRF2Point0              `json:"meshrf"`
	NodeDetails   NodeDetails2Point0         `json:"node_details"`
	Tunnels       Tunnels1Point10            `json:"tunnels"`
	Services      []Service2Point0           `json:"services,omitempty"`
	ServicesLocal []Service2Point0           `json:"services_local,omitempty"`
	LinkInfo      map[string]LinkInfo2Point0 `json:"link_info,omitempty"`
}
