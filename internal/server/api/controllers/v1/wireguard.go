package v1

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
	"github.com/gin-gonic/gin"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func GETWireguardGenkey(c *gin.Context) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		slog.Error("Error generating key", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "Error generating key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key.String()})
}

func POSTWireguardPubkey(c *gin.Context) {
	var req apimodels.WireguardPubkeyRequest
	err := c.BindJSON(&req)
	if err != nil {
		slog.Error("Error binding json", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{keyError: "Invalid request"})
		return
	}

	private, err := wgtypes.ParseKey(req.Privkey)
	if err != nil {
		slog.Error("Error parsing key", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{keyError: "Invalid key"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": private.PublicKey().String()})
}
