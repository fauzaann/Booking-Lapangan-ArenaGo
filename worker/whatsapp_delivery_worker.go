package worker

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"Booking-Lapangan/config"
	"Booking-Lapangan/pkg/whatsapp"
	"Booking-Lapangan/repository"
)

type WhatsAppDeliveryWorker struct {
	jobs     repository.WhatsAppJobRepository
	users    repository.UserRepository
	sender   whatsapp.Sender
	language string
	interval time.Duration
}

func NewWhatsAppDeliveryWorker(
	jobs repository.WhatsAppJobRepository,
	users repository.UserRepository,
	sender whatsapp.Sender,
	cfg config.WhatsAppConfig,
	interval time.Duration,
) *WhatsAppDeliveryWorker {
	if interval <= 0 {
		interval = time.Minute
	}
	return &WhatsAppDeliveryWorker{
		jobs: jobs, users: users, sender: sender, language: cfg.Language, interval: interval,
	}
}

func (w *WhatsAppDeliveryWorker) Start(ctx context.Context) {
	if w.jobs == nil || w.users == nil {
		log.Println("worker: WhatsApp delivery worker not started (repositories not provided)")
		return
	}
	if w.sender == nil {
		log.Println("worker: WhatsApp delivery worker not started (Cloud API credentials are not configured)")
		return
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	log.Printf("worker: WhatsApp delivery worker started (every %s)", w.interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: WhatsApp delivery worker stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *WhatsAppDeliveryWorker) runOnce(ctx context.Context) {
	now := time.Now()
	jobs, err := w.jobs.FindDue(ctx, now, 50)
	if err != nil {
		log.Printf("worker: failed to query WhatsApp outbox: %v", err)
		return
	}

	for _, job := range jobs {
		claimed, err := w.jobs.Claim(ctx, job.ID, now)
		if err != nil {
			log.Printf("worker: failed to claim WhatsApp job %s: %v", job.ID.Hex(), err)
			continue
		}
		if !claimed {
			continue
		}

		user, err := w.users.FindByID(ctx, job.UserID)
		if err != nil {
			w.retry(ctx, job.ID, err)
			continue
		}
		if !user.WhatsAppOptIn {
			if err := w.jobs.Cancel(ctx, job.ID, "user withdrew WhatsApp consent"); err != nil {
				log.Printf("worker: failed to cancel WhatsApp job %s: %v", job.ID.Hex(), err)
			}
			continue
		}

		sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = w.sender.SendTemplate(sendCtx, user.Phone, job.Template, w.language, job.Parameters)
		cancel()
		if err != nil {
			w.retry(ctx, job.ID, err)
			continue
		}
		if err := w.jobs.MarkSent(ctx, job.ID, time.Now()); err != nil {
			log.Printf("worker: WhatsApp message sent but job %s status update failed: %v", job.ID.Hex(), err)
		}
	}
}

func (w *WhatsAppDeliveryWorker) retry(ctx context.Context, id bson.ObjectID, cause error) {
	if err := w.jobs.Retry(ctx, id, cause.Error()); err != nil {
		log.Printf("worker: failed to record retry for WhatsApp job %s after %v: %v", id.Hex(), cause, err)
	}
}
