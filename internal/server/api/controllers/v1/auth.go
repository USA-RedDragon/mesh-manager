package v1

import (
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

func POSTLogin(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	// If the IP is from the private ip ranges, reject the login. We cannot encrypt traffic over the mesh
	if net.ParseIP(c.ClientIP()).IsPrivate() && !slices.Contains(di.Config.TrustedProxies, c.ClientIP()) {
		slog.Error("POSTLogin: Login from private IP")
		c.JSON(http.StatusUnavailableForLegalReasons, gin.H{"error": "Cannot encrypt traffic over the mesh. Please use the site via the internet."})
		return
	}

	session := sessions.Default(c)

	var json apimodels.AuthLogin
	err := c.ShouldBindJSON(&json)
	if err != nil {
		slog.Error("POSTLogin: JSON data is invalid", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON data is invalid"})
	} else {
		// Check that one of username is not blank
		if json.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be provided"})
			return
		}
		// Check that password isn't a zero string
		if json.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password cannot be blank"})
			return
		}
		var user models.User
		di.DB.Find(&user, "username = ?", json.Username)
		if di.DB.Error != nil {
			slog.Error("POSTLogin: Error finding user", "error", di.DB.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		slog.Debug("POSTLogin: User found", "user", user)

		verified, err := utils.VerifyPassword(json.Password, user.Password, di.Config.PasswordSalt)
		if verified && err == nil {
			session.Set("user_id", user.ID)
			err = session.Save()
			if err != nil {
				slog.Error("POSTLogin: Error saving session", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving session"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Logged in"})
			return
		}
		slog.Error("POSTLogin: Invalid username or password", "username", json.Username)
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
}

func GETLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	err := session.Save()
	if err != nil {
		slog.Error("GETLogout: Error saving session", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func GETAuthCheck(c *gin.Context) {
	session := sessions.Default(c)

	userID := session.Get("user_id")
	if userID == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	uid, ok := userID.(uint)
	if !ok || uid == 0 {
		c.Status(http.StatusUnauthorized)
		return
	}

	diVal, exists := c.Get(middleware.DepInjectionKey)
	if !exists {
		c.Status(http.StatusUnauthorized)
		return
	}
	di, ok := diVal.(*middleware.DepInjection)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}

	_, err := models.FindUserByID(di.DB, uid)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("GETAuthCheck: Error fetching user", "id", uid, "error", err)
		}
		// Fail closed: if we can't verify the user for any reason, deny access
		c.Status(http.StatusUnauthorized)
		return
	}

	c.Status(http.StatusOK)
}
