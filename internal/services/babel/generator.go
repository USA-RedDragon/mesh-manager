package babel

import (
	"fmt"
	"os"

	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/wireguard"
	"gorm.io/gorm"
)

// This file will generate the babel.conf file

func GenerateAndSave(config *config.Config, db *gorm.DB) error {
	conf := Generate(config, db)
	if conf == "" {
		return fmt.Errorf("failed to generate babel.conf")
	}

	//nolint:gosec
	return os.WriteFile("/tmp/babel-generated.conf", []byte(conf), 0644)
}

func Generate(config *config.Config, db *gorm.DB) string {
	// Yay this config format is much easier to generate.
	var ret string
	ret += "router-id " + config.Babel.RouterID + "\n"
	ret += "interface br-dtdlink type wired\n"
	ret += "interface br-dtdlink rxcost 96\n"
	ret += "interface br-dtdlink enable-timestamps false\n"
	ret += "interface br-dtdlink split-horizon true\n"

	tunnelInterfaces := make([]string, 0)
	tunnels, err := models.ListWireguardTunnels(db)
	if err != nil {
		panic(err)
	}

	if len(tunnels) > 0 {
		for _, tunnel := range tunnels {
			if !tunnel.Enabled {
				continue
			}
			tunnelInterfaces = append(tunnelInterfaces, wireguard.GenerateWireguardInterfaceName(tunnel))
		}
	}

	for _, iface := range tunnelInterfaces {
		ret += GenerateTunnelLine(iface, config.Supernode)
	}

	if config.Supernode {
		ret += "smoothing-half-life 0\n"
		ret += "protocol-buffer-size 32768\n"
		ret += "import-table 21\n"
		ret += "redistribute anyproto ip 10.0.0.0/8 allow\n"
		ret += "out if br-dtdlink ip 10.0.0.0/8 eq 8 allow\n"
		ret += "out ip 10.0.0.0/8 eq 8 deny\n"
		ret += "out if br-dtdlink ip " + config.NodeIP + " allow\n"
		ret += "out if br-dtdlink deny\n"

		ret += "redistribute anyproto ip 172.30.0.0/16 eq 32 deny\n"
	} else {
		ret += "smoothing-half-life 10\n"
		ret += "protocol-buffer-size 65536\n"
		ret += "redistribute anyproto ip 10.0.0.0/8 ge 24 allow\n"
		ret += "redistribute anyproto ip 44.0.0.0/8 ge 24 allow\n"
		ret += "redistribute anyproto ip 172.31.0.0/16 eq 32 deny\n"
	}
	ret += "redistribute anyproto if br0 deny\n"
	ret += "redistribute deny\n"

	if config.Supernode {
		ret += "install ip 10.0.0.0/8 eq 8 deny\n"
	} else {
		ret += "install ip 10.0.0.0/8 eq 8 allow table 21\n"
		ret += "install ip 44.0.0.0/8 le 23 allow table 21\n"
	}

	ret += "install ip 0.0.0.0/0 eq 0 allow table 22\n"
	ret += "install ip 10.0.0.0/8 ge 24 allow table 20\n"
	ret += "install ip 44.0.0.0/8 ge 24 allow table 20\n"
	ret += "install ip 0.0.0.0/0 ge 0 deny\n"

	return ret
}

func GenerateTunnelLine(iface string, supernode bool) string {
	ret := fmt.Sprintf("interface %s type tunnel\n", iface)
	ret += fmt.Sprintf("interface %s rxcost 206\n", iface)
	ret += fmt.Sprintf("interface %s hello-interval 10\n", iface)
	ret += fmt.Sprintf("interface %s split-horizon true\n", iface)
	updateInterval := 120
	if supernode {
		updateInterval = 300
	}
	ret += fmt.Sprintf("interface %s update-interval %d\n", iface, updateInterval)
	ret += fmt.Sprintf("interface %s rtt-min 10\n", iface)
	ret += fmt.Sprintf("interface %s rtt-max 400\n", iface)
	ret += fmt.Sprintf("interface %s max-rtt-penalty 400\n", iface)
	ret += fmt.Sprintf("redistribute anyproto if %s deny\n", iface)
	return ret
}
