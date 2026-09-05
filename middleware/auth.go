package middleware

import (
	"net/http"
    "Booking-Lapangan/repository"
    "github.com/gin-gonic/gin"
)

func AuthMiddleware(blacklist_tokens repository.BlacklistTokenRepository) gin.HandlerFunc {
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

		isBlacklisted, err := blacklist_tokens.IsBlacklisted(token)
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

		c.Next()
	}		
}


