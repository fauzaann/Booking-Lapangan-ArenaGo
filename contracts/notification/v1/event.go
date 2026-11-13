package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const ContractVersion = 1

type EventType string

const (
	EventBookingConfirmed    EventType = "booking.confirmed"
	EventBookingCancelled    EventType = "booking.cancelled"
	EventBookingRescheduled  EventType = "booking.rescheduled"
	EventPaymentInvoiceReady EventType = "payment.invoice_ready"
)

// Envelope is the stable JSON message exchanged with the notification service.
type Envelope struct {
	ID         string          `json:"id"`
	Type       EventType       `json:"type"`
	Version    int             `json:"version"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

// BookingReference is the booking snapshot required to process an event
// without querying the booking or payment database.
type BookingReference struct {
	Booking BookingSnapshot `json:"booking"`
}

type BookingSnapshot struct {
	BookingID            string  `json:"booking_id"`
	BookingCode          string  `json:"booking_code"`
	UserID               string  `json:"user_id"`
	UserName             string  `json:"user_name"`
	UserEmail            string  `json:"user_email"`
	UserPhone            string  `json:"user_phone"`
	UserWhatsAppOptIn    bool    `json:"user_whatsapp_opt_in"`
	FieldID              string  `json:"field_id"`
	FieldName            string  `json:"field_name"`
	BookingDate          string  `json:"booking_date"`
	StartTime            string  `json:"start_time"`
	EndTime              string  `json:"end_time"`
	Duration             int     `json:"duration"`
	TotalPrice           float64 `json:"total_price"`
	Status               string  `json:"status"`
	RefundReviewRequired bool    `json:"refund_review_required"`
}

type PaymentInvoiceReadyData struct {
	Booking    BookingSnapshot `json:"booking"`
	PaymentURL string          `json:"payment_url"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

func NewBookingEvent(id string, eventType EventType, occurredAt time.Time, booking BookingSnapshot) (Envelope, error) {
	if eventType != EventBookingConfirmed && eventType != EventBookingCancelled && eventType != EventBookingRescheduled {
		return Envelope{}, fmt.Errorf("unsupported booking event type %q", eventType)
	}
	return newEnvelope(id, eventType, occurredAt, BookingReference{Booking: booking})
}

func NewPaymentInvoiceReadyEvent(id string, occurredAt time.Time, data PaymentInvoiceReadyData) (Envelope, error) {
	return newEnvelope(id, EventPaymentInvoiceReady, occurredAt, data)
}

func (e Envelope) BookingReference() (BookingReference, error) {
	if err := e.Validate(); err != nil {
		return BookingReference{}, err
	}
	var data BookingReference
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return BookingReference{}, fmt.Errorf("decode %s data: %w", e.Type, err)
	}
	return data, nil
}

func (e Envelope) PaymentInvoiceReady() (PaymentInvoiceReadyData, error) {
	if err := e.Validate(); err != nil {
		return PaymentInvoiceReadyData{}, err
	}
	if e.Type != EventPaymentInvoiceReady {
		return PaymentInvoiceReadyData{}, fmt.Errorf("event type %q is not %q", e.Type, EventPaymentInvoiceReady)
	}
	var data PaymentInvoiceReadyData
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return PaymentInvoiceReadyData{}, fmt.Errorf("decode %s data: %w", e.Type, err)
	}
	return data, nil
}

func (e Envelope) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("event id is required")
	}
	if e.Version != ContractVersion {
		return fmt.Errorf("unsupported notification event contract version %d", e.Version)
	}
	if e.OccurredAt.IsZero() {
		return errors.New("event occurred_at is required")
	}
	if len(e.Data) == 0 || !json.Valid(e.Data) {
		return errors.New("event data must contain valid JSON")
	}
	switch e.Type {
	case EventBookingConfirmed, EventBookingCancelled, EventBookingRescheduled:
		var data BookingReference
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return fmt.Errorf("decode %s data: %w", e.Type, err)
		}
		if err := data.Booking.validate(); err != nil {
			return err
		}
	case EventPaymentInvoiceReady:
		var data PaymentInvoiceReadyData
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return fmt.Errorf("decode %s data: %w", e.Type, err)
		}
		if err := data.Booking.validate(); err != nil {
			return err
		}
		if strings.TrimSpace(data.PaymentURL) == "" {
			return errors.New("payment_url is required")
		}
		if data.ExpiresAt.IsZero() {
			return errors.New("expires_at is required")
		}
	default:
		return fmt.Errorf("unsupported notification event type %q", e.Type)
	}
	return nil
}

func (b BookingSnapshot) validate() error {
	for name, value := range map[string]string{
		"booking_id":   b.BookingID,
		"booking_code": b.BookingCode,
		"user_id":      b.UserID,
		"field_id":     b.FieldID,
		"field_name":   b.FieldName,
		"booking_date": b.BookingDate,
		"start_time":   b.StartTime,
		"end_time":     b.EndTime,
		"status":       b.Status,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if _, err := time.Parse("2006-01-02", b.BookingDate); err != nil {
		return fmt.Errorf("booking_date must use YYYY-MM-DD format: %w", err)
	}
	if b.Duration <= 0 {
		return errors.New("duration must be greater than zero")
	}
	if b.TotalPrice < 0 {
		return errors.New("total_price must not be negative")
	}
	return nil
}

func newEnvelope(id string, eventType EventType, occurredAt time.Time, data any) (Envelope, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, fmt.Errorf("encode %s data: %w", eventType, err)
	}
	event := Envelope{
		ID:         id,
		Type:       eventType,
		Version:    ContractVersion,
		OccurredAt: occurredAt.UTC(),
		Data:       raw,
	}
	if err := event.Validate(); err != nil {
		return Envelope{}, err
	}
	return event, nil
}
