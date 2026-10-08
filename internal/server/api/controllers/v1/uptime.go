package v1

import (
	"log/slog"
	"net/http"
	"syscall"

	"github.com/USA-RedDragon/mesh-manager/internal/utils"
	"github.com/gin-gonic/gin"
)

func GETUptime(c *gin.Context) {
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err != nil {
		slog.Error("GETUptime: Unable to get system info", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "Unable to get system info"})
		return
	}
	uptime := utils.SecondsToClock(info.Uptime)
	if uptime == "" {
		slog.Error("GETUptime: Unable to convert uptime to string")
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "Unable to convert uptime to string"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"uptime": uptime})
}
