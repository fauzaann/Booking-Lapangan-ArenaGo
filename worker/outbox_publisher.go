package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	notificationv1 "Booking-Lapangan/contracts/notification/v1"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

type NotificationEventPublisher interface {
	Publish(context.Context, notificationv1.Envelope) error
}

type OutboxPublisherWorker struct {
	repository repository.NotificationOutboxRepository
	publisher  NotificationEventPublisher
	interval   time.Duration
}

func NewOutboxPublisherWorker(
	outbox repository.NotificationOutboxRepository,
	publisher NotificationEventPublisher,
	interval time.Duration,
) *OutboxPublisherWorker {
	if interval <= 0 {
		interval = time.Second
	}
	return &OutboxPublisherWorker{repository: outbox, publisher: publisher, interval: interval}
}

func (w *OutboxPublisherWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("worker: notification outbox publisher started (every %s)", w.interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("worker: notification outbox publisher stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *OutboxPublisherWorker) runOnce(ctx context.Context) {
	events, err := w.repository.ClaimPending(ctx, time.Now(), 50)
	if err != nil {
		log.Printf("worker: failed to claim notification outbox events: %v", err)
		return
	}

	for _, event := range events {
		message, processErr := notificationEventContract(event)
		if processErr == nil {
			processErr = w.publisher.Publish(ctx, message)
		}
		if processErr != nil {
			if retryErr := w.repository.Retry(ctx, event, processErr, time.Now()); retryErr != nil {
				log.Printf("worker: failed to retry outbox event %d after %v: %v", event.ID, processErr, retryErr)
				continue
			}
			log.Printf("worker: outbox event %d was scheduled for retry: %v", event.ID, processErr)
			continue
		}
		if err := w.repository.MarkProcessed(ctx, event.ID, time.Now()); err != nil {
			log.Printf("worker: event %d was published but could not be marked processed: %v", event.ID, err)
		}
	}
}

func notificationEventContract(event models.NotificationEvent) (notificationv1.Envelope, error) {
	if event.EventID == "" {
		return notificationv1.Envelope{}, fmt.Errorf("notification outbox event %d has no event id", event.ID)
	}
	if event.CreatedAt.IsZero() {
		return notificationv1.Envelope{}, fmt.Errorf("notification outbox event %d has no creation time", event.ID)
	}

	var eventType notificationv1.EventType
	switch event.EventType {
	case models.NotificationEventBookingConfirmed:
		eventType = notificationv1.EventBookingConfirmed
	case models.NotificationEventBookingCancelled:
		eventType = notificationv1.EventBookingCancelled
	case models.NotificationEventBookingRescheduled:
		eventType = notificationv1.EventBookingRescheduled
	case models.NotificationEventPaymentInvoiceReady:
		eventType = notificationv1.EventPaymentInvoiceReady
	default:
		return notificationv1.Envelope{}, fmt.Errorf("unsupported notification outbox event type %q", event.EventType)
	}

	message := notificationv1.Envelope{
		ID: event.EventID, Type: eventType, Version: event.ContractVersion,
		OccurredAt: event.CreatedAt.UTC(), Data: json.RawMessage(event.Payload),
	}
	if err := message.Validate(); err != nil {
		return notificationv1.Envelope{}, fmt.Errorf("invalid notification outbox event %d payload: %w", event.ID, err)
	}
	return message, nil
}
