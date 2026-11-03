package models

import "time"

// PlayingPackage defines a duration-based discount tier for bookings.
type PlayingPackage struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	MinimumDuration int       `gorm:"not null" json:"minimum_duration"`
	DiscountPercent float64   `gorm:"type:numeric(5,2);not null" json:"discount_percent"`
	IsActive        bool      `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
