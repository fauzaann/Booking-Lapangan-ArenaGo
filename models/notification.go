package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Status antrean pengingat di MongoDB.
const (
	ReminderStatusScheduled  = "SCHEDULED"
	ReminderStatusProcessing = "PROCESSING"
	ReminderStatusSent       = "SENT"
	ReminderStatusFailed     = "FAILED"
	ReminderStatusCancelled  = "CANCELLED"
)

// Tipe pengingat jadwal bermain.
const (
	ReminderTypeH24 = "H_24"
	ReminderTypeH2  = "H_2"
)

// Tipe notifikasi inbox.
const (
	NotificationTypeReminderMatch    = "REMINDER_MATCH"
	NotificationTypeBookingConfirmed = "BOOKING_CONFIRMED"
	NotificationTypeBookingCancelled = "BOOKING_CANCELLED"
)

// Channel notifikasi.
const (
	NotificationChannelInApp = "IN_APP"
	NotificationChannelEmail = "EMAIL"
	NotificationChannelWA    = "WHATSAPP"
)

// ReminderJob merepresentasikan satu tugas pengingat jadwal bermain di MongoDB.
type ReminderJob struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	BookingID    uint          `bson:"booking_id" json:"booking_id"`
	BookingCode  string        `bson:"booking_code" json:"booking_code"`
	UserID       uint          `bson:"user_id" json:"user_id"`
	UserName     string        `bson:"user_name" json:"user_name"`
	UserEmail    string        `bson:"user_email" json:"user_email"`
	UserPhone    string        `bson:"user_phone" json:"user_phone"`
	FieldID      uint          `bson:"field_id" json:"field_id"`
	FieldName    string        `bson:"field_name" json:"field_name"`
	BookingDate  time.Time     `bson:"booking_date" json:"booking_date"`
	StartTime    string        `bson:"start_time" json:"start_time"`
	EndTime      string        `bson:"end_time" json:"end_time"`
	PlayTime     time.Time     `bson:"play_time" json:"play_time"`
	ReminderType string        `bson:"reminder_type" json:"reminder_type"` // H_24, H_2
	ScheduledFor time.Time     `bson:"scheduled_for" json:"scheduled_for"`
	Status       string        `bson:"status" json:"status"`               // SCHEDULED, PROCESSING, SENT, FAILED, CANCELLED
	RetryCount   int           `bson:"retry_count" json:"retry_count"`
	MaxRetries   int           `bson:"max_retries" json:"max_retries"`
	SentAt       *time.Time    `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
	ErrorMessage string        `bson:"error_message,omitempty" json:"error_message,omitempty"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// Notification merepresentasikan item notifikasi in-app pada kotak masuk pengguna.
type Notification struct {
	ID          bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID      uint           `bson:"user_id" json:"user_id"`
	BookingID   uint           `bson:"booking_id" json:"booking_id"`
	BookingCode string         `bson:"booking_code" json:"booking_code"`
	Title       string         `bson:"title" json:"title"`
	Message     string         `bson:"message" json:"message"`
	Type        string         `bson:"type" json:"type"`
	Channels    []string       `bson:"channels" json:"channels"`
	IsRead      bool           `bson:"is_read" json:"is_read"`
	ReadAt      *time.Time     `bson:"read_at,omitempty" json:"read_at,omitempty"`
	Data        map[string]any `bson:"data,omitempty" json:"data,omitempty"`
	CreatedAt   time.Time      `bson:"created_at" json:"created_at"`
	ExpiresAt   time.Time      `bson:"expires_at" json:"expires_at"` // Digunakan oleh TTL Index MongoDB
}
