package v1

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/services"
	"github.com/USA-RedDragon/mesh-manager/internal/services/babel"
	"github.com/USA-RedDragon/mesh-manager/internal/services/olsr"
	"github.com/USA-RedDragon/mesh-manager/internal/wireguard"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func GETTunnels(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	typeStr, exists := c.GetQuery("type")
	if !exists {
		typeStr = "wireguard"
	}

	filter, exists := c.GetQuery("filter")
	if !exists {
		filter = ""
	}

	pageStr, exists := c.GetQuery("page")
	if !exists {
		pageStr = "1"
	}
	pageInt, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil || pageInt < 1 {
		slog.Error("GETTunnels: Error parsing page", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page"})
		return
	}
	page := int(pageInt)

	limitStr, exists := c.GetQuery("limit")
	if !exists {
		limitStr = "10"
	}
	limitInt, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limitInt < 1 {
		slog.Error("GETTunnels: Error parsing limit", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}
	limit := int(limitInt)

	var tunnels []models.Tunnel
	var total int64
	switch typeStr {
	case "wireguard":
		dbQuery := di.DB.Model(&models.Tunnel{}).Where("wireguard = ?", true)
		if filter != "" {
			dbQuery = dbQuery.Where("upper(hostname) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToUpper(filter)))
		}

		if err := dbQuery.Count(&total).Error; err != nil {
			slog.Error("GETTunnels: Error getting tunnel count", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel count"})
			return
		}

		offset := (page - 1) * limit
		if err := dbQuery.Order("id asc").Limit(limit).Offset(offset).Find(&tunnels).Error; err != nil {
			slog.Error("GETTunnels: Error getting tunnels", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnels"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid type"})
		return
	}

	adminStr, exists := c.GetQuery("admin")
	if !exists {
		adminStr = "false"
	}
	admin, err := strconv.ParseBool(adminStr)
	if err != nil {
		slog.Error("GETTunnels: Error parsing admin query", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing admin query"})
		return
	}

	if admin {
		// Check for an active session
		session := sessions.Default(c)
		user := session.Get("user_id")
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Change the json response to include the password
		var tunnelsWithPass []apimodels.TunnelWithPass

		for _, tunnel := range tunnels {
			maybePassword := ""
			if !tunnel.Client {
				maybePassword = tunnel.Password
			}
			tunnelsWithPass = append(tunnelsWithPass, apimodels.TunnelWithPass{
				Enabled:        tunnel.Enabled,
				Wireguard:      tunnel.Wireguard,
				WireguardPort:  tunnel.WireguardPort,
				ID:             tunnel.ID,
				Hostname:       tunnel.Hostname,
				IP:             tunnel.IP,
				Password:       maybePassword,
				Client:         tunnel.Client,
				Active:         tunnel.Active,
				ConnectionTime: tunnel.ConnectionTime,
				CreatedAt:      tunnel.CreatedAt,
			})
		}
		c.JSON(http.StatusOK, gin.H{"total": total, "tunnels": tunnelsWithPass})
	} else {
		c.JSON(http.StatusOK, gin.H{"total": total, "tunnels": tunnels})
	}
}

func GETWireguardTunnelsCount(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	count, err := models.CountWireguardTunnels(di.DB)
	if err != nil {
		slog.Error("GETWireguardTunnelsCount: Error getting tunnel count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func GETWireguardTunnelsCountConnected(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	count, err := models.CountWireguardActiveTunnels(di.DB)
	if err != nil {
		slog.Error("GETWireguardTunnelsCountConnected: Error getting tunnel count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func GETWireguardClientTunnelsCountConnected(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	count, err := models.CountWireguardActiveClientTunnels(di.DB)
	if err != nil {
		slog.Error("GETWireguardClientTunnelsCountConnected: Error getting tunnel count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func GETWireguardServerTunnelsCountConnected(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	count, err := models.CountWireguardActiveServerTunnels(di.DB)
	if err != nil {
		slog.Error("GETWireguardServerTunnelsCountConnected: Error getting tunnel count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func GETWireguardClientTunnelsCount(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	count, err := models.CountWireguardClientTunnels(di.DB)
	if err != nil {
		slog.Error("GETWireguardClientTunnelsCount: Error getting tunnel count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func GETWireguardServerTunnelsCount(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	count, err := models.CountWireguardServerTunnels(di.DB)
	if err != nil {
		slog.Error("GETWireguardServerTunnelsCount: Error getting tunnel count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

//nolint:gocyclo
func POSTTunnel(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var tunnel models.Tunnel
	var json apimodels.CreateTunnel
	err := c.ShouldBindJSON(&json)
	if err != nil {
		slog.Error("POSTTunnel: JSON data is invalid", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON data is invalid"})
	} else {
		if (!json.Wireguard || json.Client) && json.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password cannot be empty"})
			return
		}

		if !json.Wireguard {
			c.JSON(http.StatusBadRequest, gin.H{"error": "VTun is disabled"})
			return
		}

		if !json.Client {
			json.Hostname = strings.ToUpper(json.Hostname)
			isValid, errString := json.IsValidHostname()
			if !isValid {
				c.JSON(http.StatusBadRequest, gin.H{"error": errString})
				return
			}

			// Check if the hostname is already taken
			err := di.DB.Find(&tunnel, "hostname = ? AND wireguard = ?", json.Hostname, json.Wireguard).Error
			if err != nil {
				slog.Error("POSTTunnel: Error getting tunnel", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel"})
				return
			} else if tunnel.ID != 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Hostname is already taken"})
				return
			}

			tunnel = models.Tunnel{
				Enabled:   true,
				Hostname:  json.Hostname,
				Password:  json.Password,
				Client:    json.Client,
				Wireguard: json.Wireguard,
			}

			tunnel.IP, err = models.GetNextWireguardIP(di.DB, di.Config)
			if err != nil {
				slog.Error("POSTTunnel: Error getting next IP", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting next IP"})
				return
			}

			tunnel.WireguardPort, err = models.GetNextWireguardPort(di.DB, di.Config)
			if err != nil {
				slog.Error("POSTTunnel: Error getting next port", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting next port"})
				return
			}

			// Generate a server and client key pair
			serverKey, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				slog.Error("POSTTunnel: Error generating server key", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating server key"})
				return
			}
			clientKey, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				slog.Error("POSTTunnel: Error generating client key", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating client key"})
				return
			}

			tunnel.Password = serverKey.PublicKey().String() + clientKey.String() + clientKey.PublicKey().String()
			tunnel.WireguardServerKey = serverKey.String()

			err = di.DB.Create(&tunnel).Error
			if err != nil {
				slog.Error("POSTTunnel: Error creating tunnel", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating tunnel"})
				return
			}

			err = di.WireguardManager.AddPeer(tunnel)
			if err != nil {
				slog.Error("POSTTunnel: Error adding wireguard peer", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding wireguard peer"})
				return
			}
		} else {
			if json.IP == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "IP cannot be empty"})
				return
			}

			if json.Wireguard {
				// json.Hostname must not contain a port
				if strings.Contains(json.Hostname, ":") {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Server address is invalid"})
					return
				}

				// json.IP must contain a port that needs to be appended to the hostname instead
				ipParts := strings.Split(json.IP, ":")
				if len(ipParts) != 2 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Net is invalid"})
					return
				}
				json.Hostname = json.Hostname + ":" + ipParts[1]
				json.IP = ipParts[0]
			}

			// Check to ensure the IP is in the correct range: 172.16.0.0/12
			ip := net.ParseIP(json.IP)
			if ip == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "IP is not a valid IP address"})
				return
			}
			_, cidr, err := net.ParseCIDR("172.16.0.0/12")
			if err != nil {
				slog.Error("POSTTunnel: Error parsing CIDR", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing CIDR"})
				return
			}
			if !cidr.Contains(ip) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "IP is not in the correct range"})
				return
			}

			// Check to ensure the hostname is either a valid IP or a valid address (without protocol) with an optional port

			// split the hostname by :
			// if len(split) == 1, then it's just an IP address
			// if len(split) == 2, then it's an address with an optional port
			// if len(split) > 2, then it's invalid

			split := strings.Split(json.Hostname, ":")
			if len(split) > 2 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Server address is invalid"})
				return
			}

			// Check if the hostname is an IP address
			if net.ParseIP(split[0]) == nil {
				// Check if the hostname is a valid address
				_, err := url.ParseRequestURI("http://" + split[0])
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Server address is invalid"})
					return
				}

				// Check that the hostname is resolvable
				// _, err = net.LookupIP(split[0])
				// if err != nil {
				// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Server address is not resolvable"})
				// 	return
				// }
			}

			// Check if the port is valid
			if len(split) == 2 {
				port, err := strconv.ParseUint(split[1], 10, 16)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Server port is invalid"})
					return
				}
				if port < 1 || port > 65535 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Server port is invalid"})
					return
				}
			}

			// Check if the IP is already taken
			err = di.DB.Find(&tunnel, "ip = ?", json.IP).Error
			if err != nil {
				slog.Error("POSTTunnel: Error getting tunnel", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel"})
				return
			} else if tunnel.ID != 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "IP address is already taken"})
				return
			}

			tunnel = models.Tunnel{
				Hostname:  json.Hostname,
				Password:  json.Password,
				IP:        json.IP,
				Client:    json.Client,
				Wireguard: json.Wireguard,
			}

			if tunnel.Wireguard {
				// The password will be 3 wireguard keys concatenated together
				// <server_pubkey><client_privkey><client_pubkey>
				if len(json.Password) != 132 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Key is invalid"})
					return
				}
				serverPubkey := json.Password[:44]
				_, err := wgtypes.ParseKey(serverPubkey)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Key is invalid"})
					return
				}
				clientPrivkey := json.Password[44:88]
				_, err = wgtypes.ParseKey(clientPrivkey)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Key is invalid"})
					return
				}
				clientPubkey := json.Password[88:]
				_, err = wgtypes.ParseKey(clientPubkey)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Key is invalid"})
					return
				}
			}

			err = di.DB.Create(&tunnel).Error
			if err != nil {
				slog.Error("POSTTunnel: Error creating tunnel", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating tunnel"})
				return
			}

			err = di.WireguardManager.AddPeer(tunnel)
			if err != nil {
				slog.Error("POSTTunnel: Error adding wireguard peer", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding wireguard peer"})
				return
			}
		}

		if di.Config.OLSR {
			err = olsr.GenerateAndSave(di.Config, di.DB)
			if err != nil {
				slog.Error("POSTTunnel: Error generating olsrd config", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating olsrd config"})
				return
			}

			olsrService, ok := di.ServiceRegistry.Get(services.OLSRServiceName)
			if !ok {
				slog.Error("POSTTunnel: Error getting olsrd service")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}

			err = olsrService.Reload()
			if err != nil {
				slog.Error("POSTTunnel: Error reloading olsrd", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reloading olsrd"})
				return
			}
		}

		if di.Config.Babel.Enabled {
			babelServiceIface, ok := di.ServiceRegistry.Get(services.BabelServiceName)
			if !ok {
				slog.Error("POSTTunnel: Error getting Babel service")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}

			babelService, ok := babelServiceIface.(*babel.Service)
			if !ok {
				slog.Error("POSTTunnel: Error asserting Babel service")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}

			err = babelService.AddTunnel(c.Request.Context(), wireguard.GenerateWireguardInterfaceName(tunnel))
			if err != nil {
				slog.Error("POSTTunnel: Error adding Babel tunnel", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding Babel tunnel"})
				return
			}
		}

		dnsmasqService, ok := di.ServiceRegistry.Get(services.DNSMasqServiceName)
		if !ok {
			slog.Error("POSTTunnel: Error getting DNSMasq service")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		err = dnsmasqService.Reload()
		if err != nil {
			slog.Error("POSTTunnel: Error reloading DNS", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reloading DNS"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Tunnel created"})
	}
}

//nolint:gocyclo
func PATCHTunnel(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var json apimodels.EditTunnel
	err := c.ShouldBindJSON(&json)
	if err != nil {
		slog.Error("PATCHTunnel: JSON data is invalid", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON data is invalid"})
	} else {
		exists, err := models.TunnelIDExists(di.DB, json.ID)
		if err != nil {
			slog.Error("Error checking if tunnel exists", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking if tunnel exists"})
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tunnel does not exist"})
			return
		}
		if json.IP == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "IP cannot be empty"})
			return
		}

		// Check to ensure the IP is valid
		if net.ParseIP(json.IP) == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "IP is not a valid IP address"})
			return
		}

		// Check to ensure the IP is in the correct range: 172.16.0.0/12
		ip := net.ParseIP(json.IP)
		_, cidr, err := net.ParseCIDR("172.16.0.0/12")
		if err != nil {
			slog.Error("Error parsing CIDR", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing CIDR"})
			return
		}
		if !cidr.Contains(ip) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "IP is not in the correct range"})
			return
		}

		// Check to ensure the hostname is either a valid IP or a valid address (without protocol) with an optional port

		// split the hostname by :
		// if len(split) == 1, then it's just an IP address
		// if len(split) == 2, then it's an address with an optional port
		// if len(split) > 2, then it's invalid

		json.Hostname = strings.ToUpper(json.Hostname)

		split := strings.Split(json.Hostname, ":")
		if len(split) > 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Server address is invalid"})
			return
		}

		// Check if the hostname is an IP address
		if net.ParseIP(split[0]) == nil {
			// Check if the hostname is a valid address
			_, err := url.ParseRequestURI("http://" + split[0])
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Server address is invalid"})
				return
			}

			// Check that the hostname is resolvable
			// _, err = net.LookupIP(split[0])
			// if err != nil {
			// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Server address is not resolvable"})
			// 	return
			// }
		}

		// Check if the port is valid
		if len(split) == 2 {
			port, err := strconv.ParseUint(split[1], 10, 16)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Server port is invalid"})
				return
			}
			if port < 1 || port > 65535 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Server port is invalid"})
				return
			}
		}

		// Check if the IP is already taken
		var tunnel models.Tunnel
		err = di.DB.Find(&tunnel, "ip = ?", json.IP).Error
		if err != nil {
			slog.Error("Error getting tunnel", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel"})
			return
		} else if tunnel.ID != 0 && tunnel.ID != json.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "IP address is already taken"})
			return
		}

		origTunnel := tunnel

		tunnel.Hostname = json.Hostname
		tunnel.Password = json.Password
		tunnel.IP = json.IP

		if tunnel.Enabled != *json.Enabled {
			tunnel.Enabled = *json.Enabled
			err = di.DB.Model(&tunnel).Updates(models.Tunnel{Enabled: *json.Enabled}).Error
			if err != nil {
				slog.Error("Error updating tunnel", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating tunnel"})
				return
			}
		}

		if tunnel.Wireguard != *json.Wireguard {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Changing tunnel type not allowed"})
			return
		}

		err = di.DB.Save(&tunnel).Error
		if err != nil {
			slog.Error("Error saving tunnel", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving tunnel"})
			return
		}

		err = di.WireguardManager.RemovePeer(origTunnel)
		if err != nil {
			slog.Error("Error removing wireguard peer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding wireguard peer"})
			return
		}

		err = di.WireguardManager.AddPeer(tunnel)
		if err != nil {
			slog.Error("Error adding wireguard peer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding wireguard peer"})
			return
		}

		if di.Config.OLSR {
			err = olsr.GenerateAndSave(di.Config, di.DB)
			if err != nil {
				slog.Error("Error generating olsrd config", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating olsrd config"})
				return
			}

			olsrService, ok := di.ServiceRegistry.Get(services.OLSRServiceName)
			if !ok {
				slog.Error("Error getting OLSR service")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}

			err = olsrService.Reload()
			if err != nil {
				slog.Error("Error reloading olsrd", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reloading olsrd"})
				return
			}
		}

		dnsmasqService, ok := di.ServiceRegistry.Get(services.DNSMasqServiceName)
		if !ok {
			slog.Error("Error getting DNSMasq service")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		err = dnsmasqService.Reload()
		if err != nil {
			slog.Error("Error reloading DNS", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reloading DNS"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Tunnel updated"})
	}
}

func DELETETunnel(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	idUint64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tunnel ID"})
		return
	}

	exists, err := models.TunnelIDExists(di.DB, uint(idUint64))
	if err != nil {
		slog.Error("Error checking if tunnel exists", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking if tunnel exists"})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tunnel does not exist"})
		return
	}

	tunnel, err := models.FindTunnelByID(di.DB, uint(idUint64))
	if err != nil {
		slog.Error("Error getting tunnel", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting tunnel"})
		return
	}

	err = models.DeleteTunnel(di.DB, uint(idUint64))
	if err != nil {
		slog.Error("Error deleting tunnel", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting tunnel"})
		return
	}

	err = di.WireguardManager.RemovePeer(tunnel)
	if err != nil {
		slog.Error("Error removing wireguard peer", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error removing wireguard peer"})
		return
	}

	if di.Config.OLSR {
		err = olsr.GenerateAndSave(di.Config, di.DB)
		if err != nil {
			slog.Error("Error generating olsrd config", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating olsrd config"})
			return
		}

		olsrService, ok := di.ServiceRegistry.Get(services.OLSRServiceName)
		if !ok {
			slog.Error("Error getting OLSR service")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		err = olsrService.Reload()
		if err != nil {
			slog.Error("Error reloading olsrd", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reloading olsrd"})
			return
		}
	}

	if di.Config.Babel.Enabled {
		babelServiceIface, ok := di.ServiceRegistry.Get(services.BabelServiceName)
		if !ok {
			slog.Error("DELETETunnel: Error getting Babel service")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		babelService, ok := babelServiceIface.(*babel.Service)
		if !ok {
			slog.Error("DELETETunnel: Error asserting Babel service")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		err = babelService.RemoveTunnel(c.Request.Context(), wireguard.GenerateWireguardInterfaceName(tunnel))
		if err != nil {
			slog.Error("DELETETunnel: Error removing Babel tunnel", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error removing Babel tunnel"})
			return
		}
	}

	dnsmasqService, ok := di.ServiceRegistry.Get(services.DNSMasqServiceName)
	if !ok {
		slog.Error("Error getting DNSMasq service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	err = dnsmasqService.Reload()
	if err != nil {
		slog.Error("Error reloading DNS", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reloading DNS"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tunnel deleted"})
}
