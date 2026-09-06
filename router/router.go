package router

import (
	"Booking-Lapangan/controllers"
	"Booking-Lapangan/handler"
	"Booking-Lapangan/middleware"
	"Booking-Lapangan/repository"

	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	HealthHandler      *handler.HealthHandler
	AuthHandler        *handler.AuthHandler
	UserHandler        *handler.UserHandler
	BlacklistTokenRepo repository.BlacklistTokenRepository
}

func RegisterRoutes(router *gin.Engine, controller *controllers.Controller) {
	// Public routes
	router.POST("/register", controller.Register)
	router.POST("/login", controller.Login)

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", controller.GetProfile)
		protected.PUT("/profile", controller.UpdateProfile)
		protected.POST("/logout", controller.Logout)
	}
}
		