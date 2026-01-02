package v1

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/services"
	"github.com/USA-RedDragon/mesh-manager/internal/services/babel"
	"github.com/gin-gonic/gin"
)

func GETBabelHosts(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	if err := di.MeshLinkParser.Parse(); err != nil {
		slog.Error("GETBabelHosts: Error parsing meshlink data", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing mesh hosts"})
		return
	}

	pageStr, exists := c.GetQuery("page")
	if !exists {
		pageStr = "1"
	}
	pageInt, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil {
		slog.Error("error parsing page", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page"})
		return
	}
	page := int(pageInt)

	limitStr, exists := c.GetQuery("limit")
	if !exists {
		limitStr = "50"
	}
	limitInt, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		slog.Error("Error parsing limit:", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}
	limit := int(limitInt)

	filter, exists := c.GetQuery("filter")
	if !exists {
		filter = ""
	}

	total := di.MeshLinkParser.GetHostsCount()

	nodes := di.MeshLinkParser.GetHostsPaginated(page, limit, filter)
	c.JSON(http.StatusOK, gin.H{"nodes": nodes, "total": total})
}

func GETBabelHostsCount(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	if err := di.MeshLinkParser.Parse(); err != nil {
		slog.Error("GETBabelHostsCount: Error parsing meshlink data", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing mesh hosts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes":    di.MeshLinkParser.GetNodeHostsCount(),
		"total":    di.MeshLinkParser.GetTotalHostsCount(),
		"services": di.MeshLinkParser.GetServiceCount(),
	})
}

func GETBabelRunning(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	if !di.Config.Babel.Enabled {
		slog.Info("Babel service is not enabled in the configuration")
		c.JSON(http.StatusOK, gin.H{"running": false})
		return
	}

	babelService, ok := di.ServiceRegistry.Get(services.BabelServiceName)
	if !ok {
		slog.Error("Error getting Babel service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"running": babelService.IsRunning()})
}

// GETBabelETX returns a map of destination IPv4 addresses to their Babel ETX metrics.
// The metrics are derived from the installed routes and filtered to exclude unreachable entries (metric 65535).
func GETBabelETX(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	if !di.Config.Babel.Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "Babel is disabled"})
		return
	}

	etxByIP, err := babel.FetchInstalledRouteMetrics(c.Request.Context())
	if err != nil {
		slog.Error("GETBabelETX: Failed to query Babel", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to query Babel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"etx": etxByIP})
}
