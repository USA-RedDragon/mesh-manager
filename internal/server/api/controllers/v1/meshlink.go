package v1

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/services"
	"github.com/gin-gonic/gin"
)

func GETMeshLinkRunning(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgTryAgainLater})
		return
	}

	if !di.Config.Babel.Enabled {
		c.JSON(http.StatusOK, gin.H{keyRunning: false})
		return
	}

	meshLinkService, ok := di.ServiceRegistry.Get(services.MeshLinkServiceName)
	if !ok {
		slog.Error("Error getting MeshLink service")
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgTryAgainLater})
		return
	}
	c.JSON(http.StatusOK, gin.H{keyRunning: meshLinkService.IsRunning()})
}
