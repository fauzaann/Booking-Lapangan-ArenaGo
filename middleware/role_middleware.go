package middleware

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/response"
)

// RequireRole membatasi endpoint hanya untuk role tertentu.
// Dipasang setelah Authenticate.
func RequireRole(roles ...models.Role) gin.HandlerFunc {
	allowed := make(map[models.Role]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}

	return func(c *gin.Context) {
		actor, ok := ActorFrom(c)
		if !ok {
			response.AbortWithError(c, apperror.Unauthorized("authentication is required"))
			return
		}
		if !allowed[actor.Role] {
			response.AbortWithError(c, apperror.Forbidden("you do not have access to this resource"))
			return
		}
		c.Next()
	}
}

// RequireAdmin adalah pintasan RequireRole(models.RoleAdmin).
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}
