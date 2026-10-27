package models

import "time"

// Checkout groups multiple bookings under one payment-gateway invoice.
type Checkout struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	Code           string        `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	UserID         uint          `gorm:"not null;index" json:"user_id"`
	ExternalID     string        `gorm:"type:varchar(80);not null;uniqueIndex" json:"external_id"`
	Amount         float64       `gorm:"type:numeric(12,2);not null" json:"amount"`
	Status         PaymentStatus `gorm:"type:varchar(30);not null;index" json:"status"`
	InvoiceID      string        `gorm:"type:varchar(80);index" json:"invoice_id"`
	InvoiceURL     string        `gorm:"type:varchar(255)" json:"invoice_url"`
	ExpiredAt      *time.Time    `json:"expired_at,omitempty"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	PaymentMethod  string        `gorm:"type:varchar(50)" json:"payment_method,omitempty"`
	PaymentChannel string        `gorm:"type:varchar(50)" json:"payment_channel,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
