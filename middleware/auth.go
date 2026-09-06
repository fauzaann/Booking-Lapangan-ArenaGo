package middleware

import (
	"net/http"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/repository"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(blacklistTokens ...repository.BlacklistTokenRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Header Authorization dibutuhkan",
			})
			c.Abort()
			return
		}

		if len(blacklistTokens) > 0 && blacklistTokens[0] != nil {
			isBlacklisted, err := blacklistTokens[0].IsBlacklisted(token)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "Terjadi kesalahan internal",
				})
				c.Abort()
				return
			}
			if isBlacklisted {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"message": "Token telah diblacklist",
				})
				c.Abort()
				return
			}
		}

		c.Set("actor", controllers.Actor{ID: 1, Username: "guest", Role: "USER"})
		c.Next()
	}
}

func ActorFrom(c *gin.Context) (controllers.Actor, bool) {
	value, exists := c.Get("actor")
	if !exists {
		return controllers.Actor{}, false
	}
	actor, ok := value.(controllers.Actor)
	return actor, ok
}

func MustActor(c *gin.Context) controllers.Actor {
	actor, _ := ActorFrom(c)
	return actor
}


