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

// BookingReference is the minimum booking-owned data currently required by
// the notification service to retrieve its notification details.
type BookingReference struct {
	BookingID string `json:"booking_id"`
}

type PaymentInvoiceReadyData struct {
	BookingID  string    `json:"booking_id"`
	PaymentURL string    `json:"payment_url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func NewBookingEvent(id string, eventType EventType, occurredAt time.Time, bookingID string) (Envelope, error) {
	if eventType != EventBookingConfirmed && eventType != EventBookingCancelled && eventType != EventBookingRescheduled {
		return Envelope{}, fmt.Errorf("unsupported booking event type %q", eventType)
	}
	return newEnvelope(id, eventType, occurredAt, BookingReference{BookingID: bookingID})
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
		if strings.TrimSpace(data.BookingID) == "" {
			return errors.New("booking_id is required")
		}
	case EventPaymentInvoiceReady:
		var data PaymentInvoiceReadyData
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return fmt.Errorf("decode %s data: %w", e.Type, err)
		}
		if strings.TrimSpace(data.BookingID) == "" {
			return errors.New("booking_id is required")
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
