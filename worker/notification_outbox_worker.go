package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	notificationv1 "Booking-Lapangan/contracts/notification/v1"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

type NotificationOutboxHandler interface {
	HandleOutboxEvent(context.Context, notificationv1.Envelope) error
}

type NotificationOutboxWorker struct {
	repository repository.NotificationOutboxRepository
	handler    NotificationOutboxHandler
	interval   time.Duration
}

func NewNotificationOutboxWorker(
	outbox repository.NotificationOutboxRepository,
	handler NotificationOutboxHandler,
	interval time.Duration,
) *NotificationOutboxWorker {
	if interval <= 0 {
		interval = time.Second
	}
	return &NotificationOutboxWorker{repository: outbox, handler: handler, interval: interval}
}

func (w *NotificationOutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("worker: PostgreSQL notification outbox started (every %s)", w.interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("worker: PostgreSQL notification outbox stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *NotificationOutboxWorker) runOnce(ctx context.Context) {
	now := time.Now()
	events, err := w.repository.ClaimPending(ctx, now, 50)
	if err != nil {
		log.Printf("worker: failed to claim PostgreSQL notification outbox events: %v", err)
		return
	}

	for _, event := range events {
		message, err := notificationEventContract(event)
		if err != nil {
			if retryErr := w.repository.Retry(ctx, event, err, time.Now()); retryErr != nil {
				log.Printf("worker: failed to retry invalid notification outbox event %d after error %v: %v", event.ID, err, retryErr)
				continue
			}
			log.Printf("worker: invalid notification outbox event %d was scheduled for retry: %v", event.ID, err)
			continue
		}
		if err := w.handler.HandleOutboxEvent(ctx, message); err != nil {
			if retryErr := w.repository.Retry(ctx, event, err, time.Now()); retryErr != nil {
				log.Printf("worker: failed to retry notification outbox event %d after processing error %v: %v", event.ID, err, retryErr)
				continue
			}
			log.Printf("worker: notification outbox event %d (%s) failed and was scheduled for retry: %v", event.ID, event.EventType, err)
			continue
		}
		if err := w.repository.MarkProcessed(ctx, event.ID, time.Now()); err != nil {
			log.Printf("worker: notification outbox event %d handled but could not be marked processed: %v", event.ID, err)
		}
	}
}

func notificationEventContract(event models.NotificationEvent) (notificationv1.Envelope, error) {
	if event.ID == 0 {
		return notificationv1.Envelope{}, fmt.Errorf("notification outbox event id is required")
	}
	if event.CreatedAt.IsZero() {
		return notificationv1.Envelope{}, fmt.Errorf("notification outbox event %d has no creation time", event.ID)
	}

	id := fmt.Sprintf("notification-outbox-%d", event.ID)
	switch event.EventType {
	case models.NotificationEventBookingConfirmed:
		return notificationv1.NewBookingEvent(id, notificationv1.EventBookingConfirmed, event.CreatedAt, fmt.Sprintf("%d", event.BookingID))
	case models.NotificationEventBookingCancelled:
		return notificationv1.NewBookingEvent(id, notificationv1.EventBookingCancelled, event.CreatedAt, fmt.Sprintf("%d", event.BookingID))
	case models.NotificationEventBookingRescheduled:
		return notificationv1.NewBookingEvent(id, notificationv1.EventBookingRescheduled, event.CreatedAt, fmt.Sprintf("%d", event.BookingID))
	case models.NotificationEventPaymentInvoiceReady:
		if event.ExpiresAt == nil {
			return notificationv1.Envelope{}, fmt.Errorf("payment invoice ready outbox event %d has no expiry time", event.ID)
		}
		return notificationv1.NewPaymentInvoiceReadyEvent(id, event.CreatedAt, notificationv1.PaymentInvoiceReadyData{
			BookingID:  fmt.Sprintf("%d", event.BookingID),
			PaymentURL: event.PaymentURL,
			ExpiresAt:  *event.ExpiresAt,
		})
	default:
		return notificationv1.Envelope{}, fmt.Errorf("unsupported notification outbox event type %q", event.EventType)
	}
}
