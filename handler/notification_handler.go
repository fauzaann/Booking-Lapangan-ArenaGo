package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/middleware"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

// NotificationHandler menangani request HTTP untuk notifikasi dan pengingat.
type NotificationHandler struct {
	Base
	controller controllers.NotificationController
}

// NewNotificationHandler membuat instance NotificationHandler baru.
func NewNotificationHandler(controller controllers.NotificationController, validate *validator.Validator) *NotificationHandler {
	return &NotificationHandler{
		Base:       NewBase(validate),
		controller: controller,
	}
}

// List menangani GET /api/v1/notifications.
func (h *NotificationHandler) List(c *gin.Context) {
	var query dto.NotificationFilterQuery
	if !h.bindQuery(c, &query) {
		return
	}
	query.PaginationQuery = query.PaginationQuery.Normalize()

	actor := middleware.MustActor(c)
	notifications, total, err := h.controller.ListUserNotifications(
		c.Request.Context(),
		actor.UserID,
		query.Page,
		query.Limit,
		query.UnreadOnly,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	items := make([]dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, dto.NotificationResponse{
			ID:          n.ID.Hex(),
			UserID:      n.UserID,
			BookingID:   n.BookingID,
			BookingCode: n.BookingCode,
			Title:       n.Title,
			Message:     n.Message,
			Type:        n.Type,
			Channels:    n.Channels,
			IsRead:      n.IsRead,
			ReadAt:      n.ReadAt,
			Data:        n.Data,
			CreatedAt:   n.CreatedAt,
		})
	}

	response.Paginated(c, "Notifications retrieved", items, response.NewMeta(query.Page, query.Limit, total))
}

// UnreadCount menangani GET /api/v1/notifications/unread-count.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	actor := middleware.MustActor(c)
	count, err := h.controller.GetUnreadCount(c.Request.Context(), actor.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, "Unread count retrieved", gin.H{
		"unread_count": count,
	})
}

// MarkRead menangani PATCH /api/v1/notifications/:id/read.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	actor := middleware.MustActor(c)

	if err := h.controller.MarkAsRead(c.Request.Context(), id, actor.UserID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, "Notification marked as read", nil)
}

// MarkAllRead menangani PATCH /api/v1/notifications/read-all.
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	actor := middleware.MustActor(c)
	if err := h.controller.MarkAllAsRead(c.Request.Context(), actor.UserID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, "All notifications marked as read", nil)
}

// AdminListReminders menangani GET /api/v1/admin/reminders.
func (h *NotificationHandler) AdminListReminders(c *gin.Context) {
	var query dto.ReminderJobFilterQuery
	if !h.bindQuery(c, &query) {
		return
	}
	query.PaginationQuery = query.PaginationQuery.Normalize()

	jobs, total, err := h.controller.ListReminderJobs(c.Request.Context(), query.Page, query.Limit, query.Status)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginated(c, "Reminder jobs retrieved", jobs, response.NewMeta(query.Page, query.Limit, total))
}
