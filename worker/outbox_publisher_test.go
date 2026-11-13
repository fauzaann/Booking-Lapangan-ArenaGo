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

type testNotificationEventPublisher struct {
	err    error
	events []notificationv1.Envelope
}

func (p *testNotificationEventPublisher) Publish(_ context.Context, event notificationv1.Envelope) error {
	p.events = append(p.events, event)
	return p.err
}

func TestOutboxPublisherMarksConfirmedEventProcessed(t *testing.T) {
	occurredAt := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	envelope, err := notificationv1.NewBookingEvent(
		"event-8", notificationv1.EventBookingConfirmed, occurredAt, testBookingSnapshot(),
	)
	if err != nil {
		t.Fatalf("create contract event: %v", err)
	}
	repo := &testNotificationOutboxRepository{
		events: []models.NotificationEvent{{
			ID: 8, EventID: envelope.ID, EventType: models.NotificationEventBookingConfirmed,
			ContractVersion: envelope.Version, Payload: string(envelope.Data), CreatedAt: occurredAt,
		}},
	}
	publisher := &testNotificationEventPublisher{}
	worker := NewOutboxPublisherWorker(repo, publisher, time.Second)

	worker.runOnce(context.Background())

	if len(publisher.events) != 1 || publisher.events[0].ID != "event-8" {
		t.Fatalf("published events = %+v, want event-8", publisher.events)
	}
	if len(repo.processed) != 1 || repo.processed[0] != 8 {
		t.Fatalf("processed events = %v, want [8]", repo.processed)
	}
	if len(repo.retried) != 0 {
		t.Fatalf("unexpected retries: %v", repo.retried)
	}
}

func TestOutboxPublisherRetriesUnconfirmedPublish(t *testing.T) {
	occurredAt := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	envelope, err := notificationv1.NewBookingEvent(
		"event-9", notificationv1.EventBookingCancelled, occurredAt, testBookingSnapshot(),
	)
	if err != nil {
		t.Fatalf("create contract event: %v", err)
	}
	repo := &testNotificationOutboxRepository{
		events: []models.NotificationEvent{{
			ID: 9, EventID: envelope.ID, EventType: models.NotificationEventBookingCancelled,
			ContractVersion: envelope.Version, Payload: string(envelope.Data), CreatedAt: occurredAt,
		}},
	}
	publishErr := errors.New("RabbitMQ unavailable")
	publisher := &testNotificationEventPublisher{err: publishErr}
	worker := NewOutboxPublisherWorker(repo, publisher, time.Second)

	worker.runOnce(context.Background())

	if len(repo.retried) != 1 || repo.retried[0] != 9 {
		t.Fatalf("retried events = %v, want [9]", repo.retried)
	}
	if !errors.Is(repo.retryCause, publishErr) {
		t.Fatalf("retry cause = %v, want %v", repo.retryCause, publishErr)
	}
	if len(repo.processed) != 0 {
		t.Fatalf("unconfirmed event marked processed: %v", repo.processed)
	}
}

func TestNotificationEventContractRejectsInvalidPersistedPayload(t *testing.T) {
	_, err := notificationEventContract(models.NotificationEvent{
		ID: 10, EventID: "event-10", EventType: models.NotificationEventBookingConfirmed,
		ContractVersion: notificationv1.ContractVersion, Payload: `{"booking":{"booking_id":""}}`,
		CreatedAt: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected invalid persisted event payload to fail")
	}
}

func testBookingSnapshot() notificationv1.BookingSnapshot {
	return notificationv1.BookingSnapshot{
		BookingID: "42", BookingCode: "BK-20261112-0001", UserID: "7",
		UserName: "Rizki", UserEmail: "rizki@example.com", UserPhone: "+628123456789",
		UserWhatsAppOptIn: true, FieldID: "3", FieldName: "Court A",
		BookingDate: "2026-11-15", StartTime: "10:00", EndTime: "11:00",
		Duration: 1, TotalPrice: 200000, Status: "CONFIRMED",
	}
}
