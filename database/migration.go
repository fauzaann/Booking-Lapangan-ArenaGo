// Package database berisi migrasi skema dan data awal (seeder).
package database

import (
	"fmt"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
)

// Migrate menjalankan AutoMigrate untuk seluruh model dan membuat index
// tambahan yang tidak dapat dinyatakan lewat tag struct.
func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.User{},
		&models.Field{},
		&models.Schedule{},
		&models.PlayingPackage{},
		&models.Booking{},
		&models.Rating{},
		&models.BookingItem{},
		&models.Payment{},
		&models.Checkout{},
		&models.AuditLog{},
		&models.NotificationEvent{},
	)
	if err != nil {
		return fmt.Errorf("failed to run auto migration: %w", err)
	}

	statements := []string{
		`UPDATE notification_events SET event_id = 'legacy-' || id::text WHERE event_id IS NULL OR event_id = ''`,
		`UPDATE notification_events AS event
			SET payload = CASE
				WHEN event.event_type = 'PAYMENT_INVOICE_READY' THEN jsonb_build_object(
					'booking', jsonb_build_object(
						'booking_id', booking.id::text,
						'booking_code', booking.booking_code,
						'user_id', booking.user_id::text,
						'user_name', COALESCE(user_record.name, ''),
						'user_email', COALESCE(user_record.email, ''),
						'user_phone', COALESCE(user_record.phone, ''),
						'user_whatsapp_opt_in', COALESCE(user_record.whats_app_opt_in, false),
						'field_id', booking.field_id::text,
						'field_name', COALESCE(field_record.name, ''),
						'booking_date', booking.booking_date::text,
						'start_time', booking.start_time,
						'end_time', booking.end_time,
						'duration', booking.duration,
						'total_price', booking.total_price,
						'status', booking.status,
						'refund_review_required', booking.refund_review_required
					),
					'payment_url', event.payment_url,
					'expires_at', event.expires_at
				)
				ELSE jsonb_build_object(
					'booking', jsonb_build_object(
						'booking_id', booking.id::text,
						'booking_code', booking.booking_code,
						'user_id', booking.user_id::text,
						'user_name', COALESCE(user_record.name, ''),
						'user_email', COALESCE(user_record.email, ''),
						'user_phone', COALESCE(user_record.phone, ''),
						'user_whatsapp_opt_in', COALESCE(user_record.whats_app_opt_in, false),
						'field_id', booking.field_id::text,
						'field_name', COALESCE(field_record.name, ''),
						'booking_date', booking.booking_date::text,
						'start_time', booking.start_time,
						'end_time', booking.end_time,
						'duration', booking.duration,
						'total_price', booking.total_price,
						'status', booking.status,
						'refund_review_required', booking.refund_review_required
					)
				)
			END
			FROM bookings AS booking
			LEFT JOIN users AS user_record ON user_record.id = booking.user_id
			LEFT JOIN fields AS field_record ON field_record.id = booking.field_id
			WHERE event.booking_id = booking.id
				AND event.status IN ('PENDING', 'PROCESSING')
				AND (event.payload IS NULL OR event.payload = '{}'::jsonb)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_events_event_id ON notification_events (event_id)`,
		// Mempercepat pengecekan overlap slot (field + tanggal + jam).
		`CREATE INDEX IF NOT EXISTS idx_bookings_slot ON bookings (field_id, booking_date, start_time, end_time)`,
		// Mempercepat pencarian pembayaran yang kedaluwarsa oleh worker.
		`CREATE INDEX IF NOT EXISTS idx_payments_status_expired ON payments (status, expired_at)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
