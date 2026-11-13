package worker

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"Booking-Lapangan/config"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/whatsapp"
	"Booking-Lapangan/repository"
)

type ReminderWorkerOptions struct {
	WhatsAppConfig config.WhatsAppConfig
	WhatsAppSender whatsapp.Sender
	UserRepository repository.UserRepository
}

// ReminderWorker bertugas memantau antrean pengingat di MongoDB dan
// mengirimkan notifikasi saat waktu pengingat telah tiba.
type ReminderWorker struct {
	reminderRepo repository.ReminderRepository
	notifRepo    repository.NotificationRepository
	interval     time.Duration
	whatsAppCfg  config.WhatsAppConfig
	whatsApp     whatsapp.Sender
	userRepo     repository.UserRepository
}

// NewReminderWorker membuat worker pengingat jadwal baru.
func NewReminderWorker(
	reminderRepo repository.ReminderRepository,
	notifRepo repository.NotificationRepository,
	interval time.Duration,
	options ...ReminderWorkerOptions,
) *ReminderWorker {
	if interval <= 0 {
		interval = time.Minute
	}
	worker := &ReminderWorker{
		reminderRepo: reminderRepo,
		notifRepo:    notifRepo,
		interval:     interval,
	}
	if len(options) > 0 {
		worker.whatsAppCfg = options[0].WhatsAppConfig
		worker.whatsApp = options[0].WhatsAppSender
		worker.userRepo = options[0].UserRepository
	}
	return worker
}

// Start menjalankan loop pengingat hingga context dibatalkan.
func (w *ReminderWorker) Start(ctx context.Context) {
	if w.reminderRepo == nil || w.notifRepo == nil {
		log.Println("worker: reminder worker not started (mongodb repositories not provided)")
		return
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("worker: reminder worker started (every %s)", w.interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: reminder worker stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

// runOnce memproses tugas pengingat yang jatuh tempo.
func (w *ReminderWorker) runOnce(ctx context.Context) {
	now := time.Now()
	dueJobs, err := w.reminderRepo.FindDueJobs(ctx, now, 50)
	if err != nil {
		log.Printf("worker: failed to query due reminder jobs: %v", err)
		return
	}

	if len(dueJobs) == 0 {
		return
	}

	log.Printf("worker: processing %d due reminder job(s)", len(dueJobs))

	for _, job := range dueJobs {
		claimed, err := w.reminderRepo.ClaimDueJob(ctx, job.ID, now)
		if err != nil {
			log.Printf("worker: failed to claim reminder job %s: %v", job.ID.Hex(), err)
			continue
		}
		if !claimed {
			continue
		}
		if job.ReminderType == models.ReminderTypePaymentDue && job.ExpiredAt != nil && !job.ExpiredAt.After(now) {
			if err := w.reminderRepo.UpdateStatus(ctx, job.ID, models.ReminderStatusCancelled, nil, "invoice expired before payment reminder"); err != nil {
				log.Printf("worker: failed to cancel expired payment reminder %s: %v", job.ID.Hex(), err)
			}
			continue
		}
		var title, message string
		dateStr := job.BookingDate.Format("02 Jan 2006")

		switch job.ReminderType {
		case models.ReminderTypePaymentDue:
			title = "Pengingat Pembayaran Booking"
			message = fmt.Sprintf("Pembayaran booking %s di %s masih menunggu. Silakan selesaikan sebelum invoice kedaluwarsa.", job.BookingCode, job.FieldName)
		case models.ReminderTypeH24:
			title = "Pengingat: Jadwal Bermain Besok 🎾"
			message = fmt.Sprintf(
				"Halo %s, jadwal bermain Anda di %s adalah besok (%s) pukul %s-%s WIB. Jangan lupa konfirmasi kehadiran tim Anda!",
				job.UserName, job.FieldName, dateStr, job.StartTime, job.EndTime,
			)
		case models.ReminderTypeH2:
			if remaining := job.PlayTime.Sub(now); remaining < 2*time.Hour {
				title = "Pengingat: Jadwal Bermain Segera ⏳"
				message = fmt.Sprintf(
					"Halo %s, jadwal bermain Anda di %s dimulai pukul %s WIB. Waktu bermain sudah dekat, segera bersiap!",
					job.UserName, job.FieldName, job.StartTime,
				)
			} else {
				title = "Pengingat: Jadwal Bermain 2 Jam Lagi ⏳"
				message = fmt.Sprintf(
					"Halo %s, jadwal bermain Anda di %s dimulai 2 jam lagi pukul %s WIB. Siapkan raket dan segera meluncur ke lapangan!",
					job.UserName, job.FieldName, job.StartTime,
				)
			}
		default:
			err := fmt.Errorf("unknown reminder type %q", job.ReminderType)
			log.Printf("worker: %v", err)
			if updateErr := w.reminderRepo.UpdateStatus(ctx, job.ID, models.ReminderStatusFailed, nil, err.Error()); updateErr != nil {
				log.Printf("worker: failed to mark unknown reminder job %s as failed: %v", job.ID.Hex(), updateErr)
			}
			continue
		}

		if !job.InAppSent {
			notif := &models.Notification{
				UserID:      job.UserID,
				BookingID:   job.BookingID,
				BookingCode: job.BookingCode,
				Title:       title,
				Message:     message,
				Type:        reminderNotificationType(job.ReminderType),
				Channels:    []string{models.NotificationChannelInApp},
				IsRead:      false,
				Data: map[string]any{
					"booking_code":  job.BookingCode,
					"field_name":    job.FieldName,
					"date":          dateStr,
					"start_time":    job.StartTime,
					"end_time":      job.EndTime,
					"reminder_type": job.ReminderType,
				},
				CreatedAt: now,
				ExpiresAt: now.AddDate(0, 1, 0),
			}
			if job.ReminderType == models.ReminderTypePaymentDue {
				notif.Data["payment_url"] = job.PaymentURL
			}
			if err := w.notifRepo.Create(ctx, notif); err != nil {
				log.Printf("worker: failed to create reminder in-app notification for user %d (booking %s): %v", job.UserID, job.BookingCode, err)
				w.retry(ctx, job, err)
				continue
			}
			if err := w.reminderRepo.MarkInAppSent(ctx, job.ID); err != nil {
				log.Printf("worker: failed to mark in-app reminder %s as sent: %v", job.ID.Hex(), err)
				w.retry(ctx, job, err)
				continue
			}
		}

		userOptedIn := job.UserWhatsAppOptIn
		if w.whatsApp != nil && w.userRepo != nil {
			user, err := w.userRepo.FindByID(ctx, job.UserID)
			if err != nil {
				log.Printf("worker: failed to verify WhatsApp consent for user %d: %v", job.UserID, err)
				w.retry(ctx, job, err)
				continue
			}
			userOptedIn = user.WhatsAppOptIn
			job.UserPhone = user.Phone
		}
		if userOptedIn && w.whatsApp != nil {
			template, parameters := reminderTemplate(w.whatsAppCfg, job, dateStr)
			if err := w.whatsApp.SendTemplate(ctx, job.UserPhone, template, w.whatsAppCfg.Language, parameters); err != nil {
				log.Printf("worker: failed to send WhatsApp reminder to user %d (booking %s): %v", job.UserID, job.BookingCode, err)
				w.retry(ctx, job, err)
				continue
			}
		} else if userOptedIn {
			log.Printf("worker: WhatsApp reminder for user %d (booking %s) skipped because the provider is not configured", job.UserID, job.BookingCode)
			w.retry(ctx, job, fmt.Errorf("WhatsApp Cloud API is not configured"))
			continue
		}

		sentAt := time.Now()
		if err := w.reminderRepo.UpdateStatus(ctx, job.ID, models.ReminderStatusSent, &sentAt, ""); err != nil {
			log.Printf("worker: failed to update reminder job status %s: %v", job.ID.Hex(), err)
		} else {
			log.Printf("worker: reminder %s delivered to user %s for booking %s", job.ReminderType, job.UserName, job.BookingCode)
		}
	}
}

func (w *ReminderWorker) retry(ctx context.Context, job models.ReminderJob, cause error) {
	if err := w.reminderRepo.IncrementRetry(ctx, job.ID, cause.Error()); err != nil {
		log.Printf("worker: failed to retry reminder job %s after %v: %v", job.ID.Hex(), cause, err)
	}
}

func reminderNotificationType(reminderType string) string {
	if reminderType == models.ReminderTypePaymentDue {
		return models.NotificationTypePaymentReminder
	}
	return models.NotificationTypeReminderMatch
}

func reminderTemplate(cfg config.WhatsAppConfig, job models.ReminderJob, date string) (string, []string) {
	if job.ReminderType == models.ReminderTypePaymentDue {
		return cfg.PaymentReminderTemplate, []string{
			job.UserName, job.BookingCode, job.FieldName,
			strconv.FormatFloat(job.Amount, 'f', 0, 64), job.PaymentURL,
		}
	}
	return cfg.MatchReminderTemplate, []string{
		job.UserName, job.FieldName, date, job.StartTime, job.EndTime, job.ReminderType,
	}
}
