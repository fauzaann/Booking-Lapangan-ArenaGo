// Package middleware berisi seluruh middleware HTTP: autentikasi, otorisasi,
// logging, CORS, dan recovery.
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/jwt"
	"Booking-Lapangan/pkg/response"
)

// ContextActorKey adalah kunci penyimpanan identitas pemanggil di gin.Context.
const ContextActorKey = "auth_actor"

type AccountStatusChecker interface {
	IsActive(ctx context.Context, userID uint) (bool, error)
}

// Authenticate memverifikasi header Authorization: Bearer <token>.
func Authenticate(manager jwt.Manager, statusCheckers ...AccountStatusChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			response.AbortWithError(c, apperror.Unauthorized("authorization header is required"))
			return
		}

		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.AbortWithError(c, apperror.Unauthorized("authorization header must use Bearer scheme"))
			return
		}

		claims, err := manager.Verify(parts[1])
		if err != nil {
			response.AbortWithError(c, apperror.Unauthorized("invalid or expired token"))
			return
		}

		role := models.Role(strings.ToUpper(claims.Role))
		if !role.Valid() {
			response.AbortWithError(c, apperror.Unauthorized("token contains an unknown role"))
			return
		}
		if len(statusCheckers) > 0 && statusCheckers[0] != nil {
			active, err := statusCheckers[0].IsActive(c.Request.Context(), claims.UserID)
			if err != nil {
				response.AbortWithError(c, apperror.Internal("failed to validate account status", err))
				return
			}
			if !active {
				response.AbortWithError(c, apperror.Unauthorized("invalid or expired token"))
				return
			}
		}

		c.Set(ContextActorKey, controllers.Actor{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   role,
		})
		c.Next()
	}
}

// ActorFrom membaca identitas pemanggil dari context.
func ActorFrom(c *gin.Context) (controllers.Actor, bool) {
	value, exists := c.Get(ContextActorKey)
	if !exists {
		return controllers.Actor{}, false
	}
	actor, ok := value.(controllers.Actor)
	return actor, ok
}

// MustActor mengembalikan identitas pemanggil. Selalu dipakai setelah
// middleware Authenticate sehingga nilainya dijamin ada.
func MustActor(c *gin.Context) controllers.Actor {
	actor, _ := ActorFrom(c)
	return actor
}
