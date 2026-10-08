package middleware

import (
	"log/slog"
	"math"
	"net/http"

	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	keyError      = "error"
	msgAuthFailed = "Authentication failed"
)

func RequireLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		defer func() {
			if recover() != nil {
				slog.Error("RequireLogin: Recovered from panic", "error", recover())
				// Delete the session cookie
				c.SetCookie("sessions", "", -1, "/", "", false, true)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{keyError: msgAuthFailed})
			}
		}()
		userID := session.Get("user_id")

		if userID == nil {
			slog.Debug("RequireLogin: user_id is nil")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{keyError: msgAuthFailed})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			slog.Error("RequireLogin: Unable to convert user_id to uint", "user_id", userID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{keyError: msgAuthFailed})
			return
		}
		if uid > math.MaxInt32 {
			slog.Error("RequireLogin: user_id is out of range", "user_id", uid)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{keyError: msgAuthFailed})
			return
		}
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			span.SetAttributes(
				attribute.String("http.auth", "RequireLogin"),
				attribute.Int("user.id", int(uid)),
			)
		}

		valid := true
		// Open up the DB and check if the user exists
		di, ok := c.MustGet(DepInjectionKey).(*DepInjection)
		if !ok {
			slog.Error("Unable to get dependencies from context")
			c.JSON(http.StatusInternalServerError, gin.H{keyError: "Try again later"})
			return
		}
		var user models.User
		di.DB.Find(&user, "id = ?", uid)
		if user.CreatedAt.IsZero() {
			valid = false
		}

		if !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{keyError: msgAuthFailed})
		}
	}
}
