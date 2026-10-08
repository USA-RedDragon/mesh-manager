package v1

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/gin-gonic/gin"
)

func GETStats(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgTryAgainLater})
		return
	}
	statsJSON := gin.H{
		"total_rx_mb":            di.NetworkStats.TotalRXMB,
		"total_tx_mb":            di.NetworkStats.TotalTXMB,
		"total_rx_bytes_per_sec": di.NetworkStats.TotalRXBandwidth,
		"total_tx_bytes_per_sec": di.NetworkStats.TotalTXBandwidth,
	}
	c.JSON(http.StatusOK, gin.H{"stats": statsJSON})
}
