package v1

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/services"
	"github.com/gin-gonic/gin"
)

func POSTNotify(c *gin.Context) {
	if (c.RemoteIP() != "127.0.0.1" && c.RemoteIP() != "::1") || c.GetHeader("X-Forwarded-For") != "" {
		slog.Warn("POSTNotify: Forbidden notify", "ip", c.RemoteIP())
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	dnsmasqService, ok := di.ServiceRegistry.Get(services.DNSMasqServiceName)
	if !ok {
		slog.Error("POSTNotify: Error getting DNSMasq service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	err := dnsmasqService.Reload()
	if err != nil {
		slog.Error("POSTNotify: Error reloading DNSMasq service", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error regenerating DNS"})
		return
	}

	go func() {
		err := di.OLSRHostsParser.Parse()
		if err != nil {
			slog.Error("POSTNotify: Error parsing hosts", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing hosts"})
			return
		}

		err = di.OLSRServicesParser.Parse()
		if err != nil {
			slog.Error("POSTNotify: Error parsing services", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing services"})
			return
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}
