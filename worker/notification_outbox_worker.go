package worker

import (
	"context"
	"log"
	"time"

	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

type NotificationOutboxHandler interface {
	HandleOutboxEvent(context.Context, models.NotificationEvent) error
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
		if err := w.handler.HandleOutboxEvent(ctx, event); err != nil {
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
