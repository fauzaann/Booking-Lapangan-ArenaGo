package models

import "time"

// PaymentStatus adalah status pembayaran pada payment gateway.
type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusPaid    PaymentStatus = "PAID"
	PaymentStatusExpired PaymentStatus = "EXPIRED"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

// Valid memeriksa apakah status pembayaran dikenal.
func (s PaymentStatus) Valid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusPaid, PaymentStatusExpired, PaymentStatusFailed:
		return true
	}
	return false
}

// Payment adalah catatan pembayaran sebuah booking.
// ExternalID unik dan menjadi kunci idempotensi saat memproses webhook.
type Payment struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	BookingID      uint          `gorm:"not null;uniqueIndex" json:"booking_id"`
	ExternalID     string        `gorm:"type:varchar(80);not null;uniqueIndex" json:"external_id"`
	InvoiceID      string        `gorm:"type:varchar(80);index" json:"invoice_id"`
	InvoiceURL     string        `gorm:"type:varchar(255)" json:"invoice_url"`
	Amount         float64       `gorm:"type:numeric(12,2);not null" json:"amount"`
	PaymentMethod  string        `gorm:"type:varchar(50)" json:"payment_method"`
	PaymentChannel string        `gorm:"type:varchar(50)" json:"payment_channel"`
	Status         PaymentStatus `gorm:"type:varchar(20);not null;index" json:"status"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	ExpiredAt      *time.Time    `json:"expired_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`

	Booking *Booking `gorm:"foreignKey:BookingID" json:"-"`
}
