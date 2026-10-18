package dto

import "time"

// NotificationFilterQuery mendefinisikan query parameter untuk mengambil notifikasi pengguna.
type NotificationFilterQuery struct {
	PaginationQuery
	UnreadOnly bool `form:"unread_only"`
}

// NotificationResponse adalah format respons item notifikasi ke client.
type NotificationResponse struct {
	ID          string         `json:"id"`
	UserID      uint           `json:"user_id"`
	BookingID   uint           `json:"booking_id"`
	BookingCode string         `json:"booking_code"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	Type        string         `json:"type"`
	Channels    []string       `json:"channels"`
	IsRead      bool           `json:"is_read"`
	ReadAt      *time.Time     `json:"read_at,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ReminderJobFilterQuery mendefinisikan query parameter untuk filter monitoring pengingat oleh admin.
type ReminderJobFilterQuery struct {
	PaginationQuery
	Status string `form:"status"`
}
