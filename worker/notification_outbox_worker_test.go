package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	notificationv1 "Booking-Lapangan/contracts/notification/v1"
	"Booking-Lapangan/models"
)

type testNotificationOutboxRepository struct {
	events     []models.NotificationEvent
	processed  []uint
	retried    []uint
	retryCause error
}

func (r *testNotificationOutboxRepository) Create(context.Context, *models.NotificationEvent) error {
	return nil
}
func (r *testNotificationOutboxRepository) ClaimPending(context.Context, time.Time, int) ([]models.NotificationEvent, error) {
	return r.events, nil
}
func (r *testNotificationOutboxRepository) MarkProcessed(_ context.Context, id uint, _ time.Time) error {
	r.processed = append(r.processed, id)
	return nil
}
func (r *testNotificationOutboxRepository) Retry(_ context.Context, event models.NotificationEvent, cause error, _ time.Time) error {
	r.retried = append(r.retried, event.ID)
	r.retryCause = cause
	return nil
}

type testNotificationOutboxHandler struct {
	err    error
	events []notificationv1.Envelope
}

func (h *testNotificationOutboxHandler) HandleOutboxEvent(_ context.Context, event notificationv1.Envelope) error {
	h.events = append(h.events, event)
	return h.err
}

func TestNotificationOutboxWorkerMarksHandledEventProcessed(t *testing.T) {
	repo := &testNotificationOutboxRepository{
		events: []models.NotificationEvent{{
			ID: 8, EventType: models.NotificationEventBookingConfirmed,
			BookingID: 17, CreatedAt: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC),
		}},
	}
	handler := &testNotificationOutboxHandler{}
	worker := NewNotificationOutboxWorker(repo, handler, time.Second)

	worker.runOnce(context.Background())

	if len(handler.events) != 1 || handler.events[0].ID != "notification-outbox-8" {
		t.Fatalf("handler events = %v, want notification-outbox-8", handler.events)
	}
	if handler.events[0].Type != notificationv1.EventBookingConfirmed || handler.events[0].Version != notificationv1.ContractVersion {
		t.Fatalf("handler contract event = %+v, want booking.confirmed v%d", handler.events[0], notificationv1.ContractVersion)
	}
	data, err := handler.events[0].BookingReference()
	if err != nil {
		t.Fatalf("decode booking reference: %v", err)
	}
	if data.BookingID != "17" {
		t.Fatalf("contract booking_id = %q, want %q", data.BookingID, "17")
	}
	if len(repo.processed) != 1 || repo.processed[0] != 8 {
		t.Fatalf("processed events = %v, want [8]", repo.processed)
	}
	if len(repo.retried) != 0 {
		t.Fatalf("unexpected retries: %v", repo.retried)
	}
}

func TestNotificationOutboxWorkerRetriesFailedEvent(t *testing.T) {
	processErr := errors.New("MongoDB unavailable")
	repo := &testNotificationOutboxRepository{
		events: []models.NotificationEvent{{
			ID: 9, EventType: models.NotificationEventBookingCancelled,
			BookingID: 18, CreatedAt: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC),
		}},
	}
	handler := &testNotificationOutboxHandler{err: processErr}
	worker := NewNotificationOutboxWorker(repo, handler, time.Second)

	worker.runOnce(context.Background())

	if len(repo.retried) != 1 || repo.retried[0] != 9 {
		t.Fatalf("retried events = %v, want [9]", repo.retried)
	}
	if !errors.Is(repo.retryCause, processErr) {
		t.Fatalf("retry cause = %v, want %v", repo.retryCause, processErr)
	}
	if len(repo.processed) != 0 {
		t.Fatalf("failed event marked processed: %v", repo.processed)
	}
}

func TestNotificationOutboxWorkerRetriesInvalidEventContract(t *testing.T) {
	repo := &testNotificationOutboxRepository{
		events: []models.NotificationEvent{{
			ID: 10, EventType: "UNKNOWN", BookingID: 18,
			CreatedAt: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC),
		}},
	}
	handler := &testNotificationOutboxHandler{}
	worker := NewNotificationOutboxWorker(repo, handler, time.Second)

	worker.runOnce(context.Background())

	if len(repo.retried) != 1 || repo.retried[0] != 10 {
		t.Fatalf("retried events = %v, want [10]", repo.retried)
	}
	if len(handler.events) != 0 {
		t.Fatalf("invalid contract was delivered to handler: %+v", handler.events)
	}
	if len(repo.processed) != 0 {
		t.Fatalf("invalid event marked processed: %v", repo.processed)
	}
}
