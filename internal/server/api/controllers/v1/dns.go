package v1

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/services"
	"github.com/gin-gonic/gin"
)

func GETDNSRunning(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgTryAgainLater})
		return
	}

	dnsmasqService, ok := di.ServiceRegistry.Get(services.DNSMasqServiceName)
	if !ok {
		slog.Error("Error getting DNSMasq service")
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgTryAgainLater})
		return
	}
	c.JSON(http.StatusOK, gin.H{keyRunning: dnsmasqService.IsRunning()})
}
