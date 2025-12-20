package v1

import (
	//nolint:gosec
	"crypto/sha1"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	gopwned "github.com/mavjs/goPwned"
)

func GETUsers(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	users, err := models.ListUsers(di.PaginatedDB)
	if err != nil {
		slog.Error("GETUsers: Error getting users", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting users"})
		return
	}

	total, err := models.CountUsers(di.DB)
	if err != nil {
		slog.Error("GETUsers: Error getting user count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting user count"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "users": users})
}

func POSTUser(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var json apimodels.UserRegistration
	err := c.ShouldBindJSON(&json)
	if err != nil {
		slog.Error("POSTUser: JSON data is invalid", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON data is invalid"})
	} else {
		isValid, errString := json.IsValidUsername()
		if !isValid {
			c.JSON(http.StatusBadRequest, gin.H{"error": errString})
			return
		}

		// Check that password isn't a zero string
		if json.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password cannot be blank"})
			return
		}

		// Check if the username is already taken
		var user models.User
		err := di.DB.Find(&user, "username = ?", json.Username).Error
		if err != nil {
			slog.Error("POSTUser: Error getting user", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting user"})
			return
		} else if user.ID != 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username is already taken"})
			return
		}

		if di.Config.HIBPAPIKey != "" {
			goPwned := gopwned.NewClient(nil, di.Config.HIBPAPIKey)
			h := sha1.New() //#nosec G401 -- False positive, we are not using this for crypto, just HIBP
			h.Write([]byte(json.Password))
			sha1HashedPW := fmt.Sprintf("%X", h.Sum(nil))
			frange := sha1HashedPW[0:5]
			lrange := sha1HashedPW[5:40]
			karray, err := goPwned.GetPwnedPasswords(frange, false)
			if err != nil {
				// If the error message starts with "Too many requests", then tell the user to retry in one minute
				if strings.HasPrefix(err.Error(), "Too many requests") {
					c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Please try again in one minute"})
					return
				}
				slog.Error("POSTUser: Error getting pwned passwords", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting pwned passwords"})
				return
			}
			strKArray := string(karray)
			respArray := strings.Split(strKArray, "\r\n")

			var result int64
			for _, resp := range respArray {
				strArray := strings.Split(resp, ":")
				test := strArray[0]

				count, err := strconv.ParseInt(strArray[1], 0, 32)
				if err != nil {
					slog.Error("POSTUser: Error parsing pwned password count", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing pwned password count"})
					return
				}
				if test == lrange {
					result = count
				}
			}
			if result > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Password has been reported in a data breach. Please use another one"})
				return
			}
		}

		// argon2 the password
		hashedPassword := utils.HashPassword(json.Password, di.Config.PasswordSalt)

		user = models.User{
			Username: json.Username,
			Password: hashedPassword,
		}
		err = di.DB.Create(&user).Error
		if err != nil {
			slog.Error("POSTUser: Error creating user", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User created"})
	}
}

func GETUser(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	id := c.Param("id")
	// Convert string id into uint
	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
		return
	}
	user, err := models.FindUserByID(di.DB, uint(userID))
	if err != nil {
		slog.Error("Error finding user", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist"})
	}
	c.JSON(http.StatusOK, user)
}

func PATCHUser(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	id := c.Param("id")
	idInt, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
		return
	}
	var json apimodels.UserPatch
	err = c.ShouldBindJSON(&json)
	if err != nil {
		slog.Error("PATCHUser: JSON data is invalid", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON data is invalid"})
	} else {
		user, err := models.FindUserByID(di.DB, uint(idInt))
		if err != nil {
			slog.Error("Error finding user", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist"})
			return
		}

		if json.Username != "" {
			// Check if the username is already taken
			var existingUser models.User
			err := di.DB.Find(&existingUser, "username = ?", json.Username).Error
			if err != nil {
				slog.Error("Error finding user", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding user"})
				return
			} else if existingUser.ID != 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Username is already taken"})
				return
			}
			user.Username = json.Username
		}

		if json.Password != "" {
			user.Password = utils.HashPassword(json.Password, di.Config.PasswordSalt)
		}

		err = di.DB.Save(&user).Error
		if err != nil {
			slog.Error("Error updating user", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User updated"})
	}
}

func DELETEUser(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	idUint64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	exists, err := models.UserIDExists(di.DB, uint(idUint64))
	if err != nil {
		slog.Error("Error checking if user exists", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking if user exists"})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist"})
		return
	}

	err = models.DeleteUser(di.DB, uint(idUint64))
	if err != nil {
		slog.Error("Error deleting user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func GETUserSelf(c *gin.Context) {
	di, ok := c.MustGet(middleware.DepInjectionKey).(*middleware.DepInjection)
	if !ok {
		slog.Error("Unable to get dependencies from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	session := sessions.Default(c)

	userID := session.Get("user_id")
	if userID == nil {
		slog.Debug("GETUserSelf: user_id is nil")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		slog.Error("GETUserSelf: Unable to convert user_id to uint", "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	user, err := models.FindUserByID(di.DB, uid)
	if err != nil {
		slog.Error("GETUserSelf: Error finding user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding user"})
		return
	}
	c.JSON(http.StatusOK, user)
}
