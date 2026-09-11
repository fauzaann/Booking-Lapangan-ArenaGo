package dto

import (
	"time"

	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/timeutil"
)

// CreateBookingRequest adalah payload pembuatan booking.
// total_price sengaja TIDAK diterima dari client; harga dihitung backend.
type CreateBookingRequest struct {
	FieldID     uint   `json:"field_id" validate:"required,gt=0"`
	BookingDate string `json:"booking_date" validate:"required,dateonly,notpast"`
	StartTime   string `json:"start_time" validate:"required,clock"`
	EndTime     string `json:"end_time" validate:"required,clock"`
	Notes       string `json:"notes" validate:"omitempty,max=255"`
}

// BookingFilterQuery adalah query string daftar booking.
type BookingFilterQuery struct {
	PaginationQuery
	Status  string `form:"status"`
	FieldID *uint  `form:"field_id"`
	UserID  *uint  `form:"user_id"`
	From    string `form:"from"`
	To      string `form:"to"`
}

// UpdateBookingStatusRequest adalah payload perubahan status oleh admin.
type UpdateBookingStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=PENDING WAITING_PAYMENT PAID CONFIRMED CANCELLED COMPLETED EXPIRED"`
}

// BookingItemResponse adalah rincian slot per jam.
type BookingItemResponse struct {
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Price     float64 `json:"price"`
}

// BookingResponse adalah representasi booking untuk client.
type BookingResponse struct {
	ID          uint                  `json:"id"`
	BookingCode string                `json:"booking_code"`
	UserID      uint                  `json:"user_id"`
	FieldID     uint                  `json:"field_id"`
	FieldName   string                `json:"field_name,omitempty"`
	BookingDate string                `json:"booking_date"`
	StartTime   string                `json:"start_time"`
	EndTime     string                `json:"end_time"`
	Duration    int                   `json:"duration"`
	TotalPrice  float64               `json:"total_price"`
	Status      string                `json:"status"`
	Notes       string                `json:"notes,omitempty"`
	Items       []BookingItemResponse `json:"items,omitempty"`
	Payment     *PaymentResponse      `json:"payment,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
}

// BookingCreatedResponse adalah response ringkas setelah booking dibuat.
type BookingCreatedResponse struct {
	BookingID     uint       `json:"booking_id"`
	BookingCode   string     `json:"booking_code"`
	FieldName     string     `json:"field_name"`
	BookingDate   string     `json:"booking_date"`
	StartTime     string     `json:"start_time"`
	EndTime       string     `json:"end_time"`
	Duration      int        `json:"duration"`
	TotalPrice    float64    `json:"total_price"`
	BookingStatus string     `json:"booking_status"`
	PaymentURL    string     `json:"payment_url"`
	PaymentStatus string     `json:"payment_status"`
	ExpiredAt     *time.Time `json:"expired_at,omitempty"`
}

// NewBookingResponse memetakan model booking menjadi response.
func NewBookingResponse(booking models.Booking) BookingResponse {
	response := BookingResponse{
		ID:          booking.ID,
		BookingCode: booking.BookingCode,
		UserID:      booking.UserID,
		FieldID:     booking.FieldID,
		BookingDate: booking.BookingDate.Format(timeutil.DateLayout),
		StartTime:   booking.StartTime,
		EndTime:     booking.EndTime,
		Duration:    booking.Duration,
		TotalPrice:  booking.TotalPrice,
		Status:      string(booking.Status),
		Notes:       booking.Notes,
		CreatedAt:   booking.CreatedAt,
	}

	if booking.Field != nil {
		response.FieldName = booking.Field.Name
	}
	if len(booking.Items) > 0 {
		items := make([]BookingItemResponse, 0, len(booking.Items))
		for _, item := range booking.Items {
			items = append(items, BookingItemResponse{
				StartTime: item.StartTime,
				EndTime:   item.EndTime,
				Price:     item.Price,
			})
		}
		response.Items = items
	}
	if booking.Payment != nil {
		payment := NewPaymentResponse(*booking.Payment)
		response.Payment = &payment
	}
	return response
}

// NewBookingResponses memetakan daftar booking menjadi daftar response.
func NewBookingResponses(bookings []models.Booking) []BookingResponse {
	result := make([]BookingResponse, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, NewBookingResponse(booking))
	}
	return result
}
