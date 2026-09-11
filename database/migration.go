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
		&models.Booking{},
		&models.BookingItem{},
		&models.Payment{},
	)
	if err != nil {
		return fmt.Errorf("failed to run auto migration: %w", err)
	}

	statements := []string{
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
