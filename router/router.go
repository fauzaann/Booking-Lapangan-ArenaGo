// Package router memetakan seluruh endpoint HTTP beserta middleware-nya.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"Booking-Lapangan/config"
	"Booking-Lapangan/handler"
	"Booking-Lapangan/middleware"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/jwt"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/repository"
)

// Dependencies adalah seluruh komponen yang dibutuhkan router.
// Semua di-inject dari main.go (dependency injection).
type Dependencies struct {
	Config       *config.Config
	JWT          jwt.Manager
	Auth         *handler.AuthHandler
	User         *handler.UserHandler
	Field        *handler.FieldHandler
	Schedule     *handler.ScheduleHandler
	Booking      *handler.BookingHandler
	Checkout     *handler.CheckoutHandler
	Payment      *handler.PaymentHandler
	Assistant    *handler.AssistantHandler
	Admin        *handler.AdminHandler
	Notification *handler.NotificationHandler
	AuditLog     repository.AuditLogRepository
}

// New menyusun seluruh route aplikasi.
func New(deps Dependencies) *gin.Engine {
	if deps.Config.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(middleware.Recovery())
	engine.Use(middleware.RequestLogger())
	engine.Use(middleware.AuditLogger(deps.AuditLog))
	engine.Use(middleware.CORS(deps.Config.App.AllowedOrigins))

	engine.NoRoute(func(c *gin.Context) {
		response.Error(c, apperror.NotFound("endpoint not found"))
	})
	engine.NoMethod(func(c *gin.Context) {
		response.Error(c, apperror.New(http.StatusMethodNotAllowed, "method not allowed", nil))
	})

	engine.GET("/health", func(c *gin.Context) {
		response.OK(c, "Service is healthy", gin.H{"status": "ok", "env": deps.Config.App.Env})
	})

	authenticated := middleware.Authenticate(deps.JWT)
	adminOnly := middleware.RequireRole(models.RoleAdmin)

	v1 := engine.Group("/api/v1")

	// ---------- Authentication ----------
	auth := v1.Group("/auth")
	{
		auth.POST("/register", deps.Auth.Register)
		auth.POST("/login", deps.Auth.Login)
	}

	v1.GET("/me", authenticated, deps.User.Profile)

	// ---------- Fields (publik) ----------
	fields := v1.Group("/fields")
	{
		fields.GET("", deps.Field.List)
		fields.GET("/:id", deps.Field.Detail)
		fields.GET("/:id/availability", deps.Field.Availability)
		fields.GET("/:id/schedules", deps.Schedule.List)
	}

	// ---------- Bookings (user login) ----------
	bookings := v1.Group("/bookings", authenticated)
	{
		bookings.POST("", deps.Booking.Create)
		bookings.GET("", deps.Booking.List)
		bookings.GET("/:id", deps.Booking.Detail)
		bookings.GET("/:id/payment", deps.Payment.DetailByBooking)
		bookings.POST("/:id/sync-payment", deps.Payment.SyncPayment)
		bookings.POST("/:id/cancel", deps.Booking.CancelPost)
		bookings.PATCH("/:id/reschedule", deps.Booking.Reschedule)
		bookings.DELETE("/:id", deps.Booking.Cancel)
	}

	checkouts := v1.Group("/checkouts", authenticated)
	{
		checkouts.POST("", deps.Checkout.Create)
		checkouts.GET("/:id", deps.Checkout.Detail)
		checkouts.POST("/:id/sync-payment", deps.Payment.SyncCheckout)
	}

	// ---------- Payments ----------
	// Webhook sengaja tanpa JWT: pemanggilnya adalah Xendit, bukan user.
	// Autentikasinya memakai header x-callback-token.
	payments := v1.Group("/payments")
	{
		payments.POST("/webhook", deps.Payment.Webhook)
	}

	// ---------- Assistant ----------
	v1.POST("/assistant/chat", deps.Assistant.Chat)

	// ---------- Notifications (user login) ----------
	if deps.Notification != nil {
		notifications := v1.Group("/notifications", authenticated)
		{
			notifications.GET("", deps.Notification.List)
			notifications.GET("/unread-count", deps.Notification.UnreadCount)
			notifications.PATCH("/:id/read", deps.Notification.MarkRead)
			notifications.PATCH("/read-all", deps.Notification.MarkAllRead)
		}
	}

	// ---------- Admin ----------
	admin := v1.Group("/admin", authenticated, adminOnly)
	{
		admin.GET("/dashboard", deps.Admin.Dashboard)
		admin.GET("/audit-logs", deps.Admin.AuditLogs)
		admin.GET("/users", deps.User.List)

		admin.GET("/fields", deps.Field.List)
		admin.POST("/fields", deps.Field.Create)
		admin.PUT("/fields/:id", deps.Field.Update)
		admin.DELETE("/fields/:id", deps.Field.Delete)
		admin.GET("/fields/:id/schedules", deps.Schedule.List)
		admin.PUT("/fields/:id/schedules", deps.Schedule.Upsert)

		admin.GET("/bookings", deps.Booking.List)
		admin.GET("/bookings/:id", deps.Booking.Detail)
		admin.PATCH("/bookings/:id/status", deps.Booking.UpdateStatus)

		admin.GET("/payments", deps.Payment.List)

		if deps.Notification != nil {
			admin.GET("/reminders", deps.Notification.AdminListReminders)
		}
	}

	return engine
}
