package handler

import (
	"net/http"
	"gorm.io/gorm"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct{
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) HealthCheck(c *gin.Context) {
	dbStatus := "connected"
	if h.db == nil {
		sqlDB, err := h.db.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "disconnected"
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "disconnected",
				"message": "Database connection is not available",
				"database": dbStatus,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"message": "Service is running",
		"database": dbStatus,
	})	
}
