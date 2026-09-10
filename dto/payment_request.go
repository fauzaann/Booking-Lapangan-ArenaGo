package dto

import (
	"time"

	"Booking-Lapangan/models"
)

// XenditInvoiceCallback adalah body webhook invoice dari Xendit.
// Struktur ini hanya memuat field yang dibutuhkan aplikasi.
type XenditInvoiceCallback struct {
	ID             string     `json:"id"`
	ExternalID     string     `json:"external_id"`
	Status         string     `json:"status"`
	Amount         float64    `json:"amount"`
	PaidAmount     float64    `json:"paid_amount"`
	PaymentMethod  string     `json:"payment_method"`
	PaymentChannel string     `json:"payment_channel"`
	PaidAt         *time.Time `json:"paid_at"`
}

// PaymentResponse adalah representasi pembayaran untuk client.
type PaymentResponse struct {
	ID             uint       `json:"id"`
	BookingID      uint       `json:"booking_id"`
	ExternalID     string     `json:"external_id"`
	InvoiceID      string     `json:"invoice_id,omitempty"`
	InvoiceURL     string     `json:"invoice_url,omitempty"`
	Amount         float64    `json:"amount"`
	PaymentMethod  string     `json:"payment_method,omitempty"`
	PaymentChannel string     `json:"payment_channel,omitempty"`
	Status         string     `json:"status"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	ExpiredAt      *time.Time `json:"expired_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// NewPaymentResponse memetakan model pembayaran menjadi response.
func NewPaymentResponse(payment models.Payment) PaymentResponse {
	return PaymentResponse{
		ID:             payment.ID,
		BookingID:      payment.BookingID,
		ExternalID:     payment.ExternalID,
		InvoiceID:      payment.InvoiceID,
		InvoiceURL:     payment.InvoiceURL,
		Amount:         payment.Amount,
		PaymentMethod:  payment.PaymentMethod,
		PaymentChannel: payment.PaymentChannel,
		Status:         string(payment.Status),
		PaidAt:         payment.PaidAt,
		ExpiredAt:      payment.ExpiredAt,
		CreatedAt:      payment.CreatedAt,
	}
}

// NewPaymentResponses memetakan daftar pembayaran menjadi daftar response.
func NewPaymentResponses(payments []models.Payment) []PaymentResponse {
	result := make([]PaymentResponse, 0, len(payments))
	for _, payment := range payments {
		result = append(result, NewPaymentResponse(payment))
	}
	return result
}

// PaymentFilterQuery adalah query string daftar pembayaran (admin).
type PaymentFilterQuery struct {
	PaginationQuery
	Status string `form:"status"`
}
