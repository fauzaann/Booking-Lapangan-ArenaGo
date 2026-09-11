package models

import (
	"time"

	"gorm.io/gorm"
)

// FieldStatus adalah status operasional lapangan.
type FieldStatus string

const (
	FieldStatusActive      FieldStatus = "ACTIVE"
	FieldStatusInactive    FieldStatus = "INACTIVE"
	FieldStatusMaintenance FieldStatus = "MAINTENANCE"
)

// Valid memeriksa apakah status lapangan dikenal.
func (s FieldStatus) Valid() bool {
	return s == FieldStatusActive || s == FieldStatusInactive || s == FieldStatusMaintenance
}

// Field adalah lapangan yang dapat dibooking.
type Field struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"type:varchar(120);not null" json:"name"`
	Description  string         `gorm:"type:text" json:"description"`
	Type         FieldType      `gorm:"type:varchar(30);not null;index" json:"type"`
	Location     string         `gorm:"type:varchar(200);not null;index" json:"location"`
	PricePerHour float64        `gorm:"type:numeric(12,2);not null" json:"price_per_hour"`
	Facilities   string         `gorm:"type:text" json:"facilities"`
	Status       FieldStatus    `gorm:"type:varchar(20);not null;default:'ACTIVE';index" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Schedules []Schedule `gorm:"foreignKey:FieldID" json:"schedules,omitempty"`
	Bookings  []Booking  `gorm:"foreignKey:FieldID" json:"-"`
}

// IsActive menandakan lapangan siap menerima booking.
func (f *Field) IsActive() bool {
	return f.Status == FieldStatusActive
}
