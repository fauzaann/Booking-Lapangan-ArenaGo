package v1

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBookingEventUsesVersionedTransportPayload(t *testing.T) {
	occurredAt := time.Date(2026, 11, 12, 12, 30, 0, 0, time.FixedZone("WIB", 7*60*60))
	event, err := NewBookingEvent("evt-123", EventBookingConfirmed, occurredAt, testSnapshot())
	if err != nil {
		t.Fatalf("create booking event: %v", err)
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal contract event: %v", err)
	}
	jsonText := string(raw)
	for _, field := range []string{
		`"id":"evt-123"`,
		`"type":"booking.confirmed"`,
		`"version":1`,
		`"booking":{"booking_id":"42"`,
	} {
		if !strings.Contains(jsonText, field) {
			t.Errorf("event JSON missing %s: %s", field, jsonText)
		}
	}

	decoded, err := event.BookingReference()
	if err != nil {
		t.Fatalf("decode booking reference: %v", err)
	}
	if decoded.Booking.BookingID != "42" {
		t.Fatalf("booking_id = %q, want %q", decoded.Booking.BookingID, "42")
	}
	if !event.OccurredAt.Equal(occurredAt.UTC()) {
		t.Fatalf("occurred_at = %s, want %s", event.OccurredAt, occurredAt.UTC())
	}
}

func TestPaymentInvoiceReadyEventRequiresCompleteData(t *testing.T) {
	expiresAt := time.Date(2026, 11, 12, 14, 0, 0, 0, time.UTC)
	event, err := NewPaymentInvoiceReadyEvent(
		"evt-payment",
		time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC),
		PaymentInvoiceReadyData{Booking: testSnapshot(), PaymentURL: "https://pay.example/invoice", ExpiresAt: expiresAt},
	)
	if err != nil {
		t.Fatalf("create payment invoice event: %v", err)
	}
	data, err := event.PaymentInvoiceReady()
	if err != nil {
		t.Fatalf("decode payment invoice event: %v", err)
	}
	if data.Booking.BookingID != "42" || data.PaymentURL != "https://pay.example/invoice" || !data.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("payment invoice payload = %+v", data)
	}
}

func TestValidateRejectsUnsupportedVersionAndMissingPayload(t *testing.T) {
	event, err := NewBookingEvent("evt-123", EventBookingCancelled, time.Now(), testSnapshot())
	if err != nil {
		t.Fatalf("create booking event: %v", err)
	}

	event.Version++
	if err := event.Validate(); err == nil {
		t.Fatal("expected unsupported contract version to fail")
	}

	event.Version = ContractVersion
	event.Data = json.RawMessage(`{"booking":{"booking_id":""}}`)
	if err := event.Validate(); err == nil {
		t.Fatal("expected missing booking_id to fail")
	}
}

func testSnapshot() BookingSnapshot {
	return BookingSnapshot{
		BookingID: "42", BookingCode: "BK-20261112-0001", UserID: "7",
		UserName: "Rizki", UserEmail: "rizki@example.com", UserPhone: "+628123456789",
		UserWhatsAppOptIn: true, FieldID: "3", FieldName: "Court A",
		BookingDate: "2026-11-15", StartTime: "10:00", EndTime: "11:00",
		Duration: 1, TotalPrice: 200000, Status: "CONFIRMED",
	}
}
