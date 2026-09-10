package models

import "time"

// BookingItem adalah rincian slot per jam dari sebuah booking.
// Rincian ini memudahkan pelaporan dan penyesuaian harga per jam
// (misalnya harga prime time) tanpa mengubah struktur booking.
type BookingItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BookingID uint      `gorm:"not null;index" json:"booking_id"`
	StartTime string    `gorm:"type:varchar(5);not null" json:"start_time"`
	EndTime   string    `gorm:"type:varchar(5);not null" json:"end_time"`
	Price     float64   `gorm:"type:numeric(12,2);not null" json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Booking *Booking `gorm:"foreignKey:BookingID;constraint:OnDelete:CASCADE" json:"-"`
}
