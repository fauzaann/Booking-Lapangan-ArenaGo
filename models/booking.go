package models

import "time"

// BookingStatus adalah status siklus hidup booking.
type BookingStatus string

const (
	BookingStatusPending        BookingStatus = "PENDING"
	BookingStatusWaitingPayment BookingStatus = "WAITING_PAYMENT"
	BookingStatusPaid           BookingStatus = "PAID"
	BookingStatusConfirmed      BookingStatus = "CONFIRMED"
	BookingStatusCancelled      BookingStatus = "CANCELLED"
	BookingStatusCompleted      BookingStatus = "COMPLETED"
	BookingStatusExpired        BookingStatus = "EXPIRED"
)

// Valid memeriksa apakah status booking dikenal.
func (s BookingStatus) Valid() bool {
	switch s {
	case BookingStatusPending, BookingStatusWaitingPayment, BookingStatusPaid,
		BookingStatusConfirmed, BookingStatusCancelled, BookingStatusCompleted, BookingStatusExpired:
		return true
	}
	return false
}

// BlocksSlot menandakan status yang masih menahan slot waktu lapangan.
func (s BookingStatus) BlocksSlot() bool {
	switch s {
	case BookingStatusCancelled, BookingStatusExpired:
		return false
	default:
		return true
	}
}

// SlotBlockingStatuses adalah daftar status yang membuat slot tidak tersedia.
func SlotBlockingStatuses() []string {
	all := []BookingStatus{
		BookingStatusPending, BookingStatusWaitingPayment, BookingStatusPaid,
		BookingStatusConfirmed, BookingStatusCompleted,
	}
	result := make([]string, 0, len(all))
	for _, status := range all {
		result = append(result, string(status))
	}
	return result
}

// Booking adalah pemesanan slot lapangan oleh user.
type Booking struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	BookingCode string        `gorm:"type:varchar(30);not null;uniqueIndex" json:"booking_code"`
	UserID      uint          `gorm:"not null;index" json:"user_id"`
	FieldID     uint          `gorm:"not null;index:idx_booking_field_date" json:"field_id"`
	BookingDate time.Time     `gorm:"type:date;not null;index:idx_booking_field_date" json:"booking_date"`
	StartTime   string        `gorm:"type:varchar(5);not null" json:"start_time"`
	EndTime     string        `gorm:"type:varchar(5);not null" json:"end_time"`
	Duration    int           `gorm:"not null" json:"duration"`
	TotalPrice  float64       `gorm:"type:numeric(12,2);not null" json:"total_price"`
	Status      BookingStatus `gorm:"type:varchar(20);not null;index" json:"status"`
	Notes       string        `gorm:"type:text" json:"notes"`
	CancelledAt *time.Time    `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	User    *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Field   *Field        `gorm:"foreignKey:FieldID" json:"field,omitempty"`
	Items   []BookingItem `gorm:"foreignKey:BookingID" json:"items,omitempty"`
	Payment *Payment      `gorm:"foreignKey:BookingID" json:"payment,omitempty"`
}
