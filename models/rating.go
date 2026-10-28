package models

import "time"

// Rating is feedback submitted for a completed booking.
type Rating struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BookingID uint      `gorm:"not null;uniqueIndex" json:"booking_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	FieldID   uint      `gorm:"not null;index" json:"field_id"`
	Rating    int       `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Review    string    `gorm:"type:varchar(1000)" json:"review"`
	IsVisible bool      `gorm:"not null;default:true;index" json:"is_visible"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Booking Booking `gorm:"foreignKey:BookingID" json:"-"`
	User    User    `gorm:"foreignKey:UserID" json:"-"`
	Field   Field   `gorm:"foreignKey:FieldID" json:"-"`
}
