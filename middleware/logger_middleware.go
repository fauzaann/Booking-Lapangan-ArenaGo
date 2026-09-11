package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/response"
)

// RequestLogger mencatat setiap request beserta latensi dan status code.
// Error internal yang didaftarkan lewat c.Error ikut tercetak di log,
// tetapi tidak pernah dikirim ke client.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		log.Printf("%s %s %d %s %s",
			c.Request.Method,
			path,
			c.Writer.Status(),
			time.Since(start).Round(time.Millisecond),
			c.ClientIP(),
		)

		for _, ginErr := range c.Errors {
			log.Printf("error: %v", ginErr.Err)
		}
	}
}

// Recovery mengubah panic yang tidak tertangani menjadi response JSON 500
// dengan format yang sama seperti error lainnya.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf("panic recovered: %v", recovered)
		response.AbortWithError(c, apperror.Internal("internal server error", nil))
	})
}
