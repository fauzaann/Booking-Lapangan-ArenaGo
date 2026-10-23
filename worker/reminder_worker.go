package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

// ReminderWorker bertugas memantau antrean pengingat di MongoDB dan
// mengirimkan notifikasi saat waktu pengingat telah tiba.
type ReminderWorker struct {
	reminderRepo repository.ReminderRepository
	notifRepo    repository.NotificationRepository
	interval     time.Duration
}

// NewReminderWorker membuat worker pengingat jadwal baru.
func NewReminderWorker(
	reminderRepo repository.ReminderRepository,
	notifRepo repository.NotificationRepository,
	interval time.Duration,
) *ReminderWorker {
	if interval <= 0 {
		interval = time.Minute
	}
	return &ReminderWorker{
		reminderRepo: reminderRepo,
		notifRepo:    notifRepo,
		interval:     interval,
	}
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
		// Tandai sedang diproses
		_ = w.reminderRepo.UpdateStatus(ctx, job.ID, models.ReminderStatusProcessing, nil, "")

		// Buat konten pesan notifikasi berdasarkan jenis pengingat
		var title, message string
		dateStr := job.BookingDate.Format("02 Jan 2006")

		if job.ReminderType == models.ReminderTypeH24 {
			title = "Pengingat: Jadwal Bermain Besok 🎾"
			message = fmt.Sprintf(
				"Halo %s, jadwal bermain Anda di %s adalah besok (%s) pukul %s-%s WIB. Jangan lupa konfirmasi kehadiran tim Anda!",
				job.UserName, job.FieldName, dateStr, job.StartTime, job.EndTime,
			)
		} else {
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
		}

		// Simpan notifikasi ke kotak masuk in-app
		notif := &models.Notification{
			UserID:      job.UserID,
			BookingID:   job.BookingID,
			BookingCode: job.BookingCode,
			Title:       title,
			Message:     message,
			Type:        models.NotificationTypeReminderMatch,
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
			ExpiresAt: now.AddDate(0, 1, 0), // TTL 30 hari
		}

		if err := w.notifRepo.Create(ctx, notif); err != nil {
			log.Printf("worker: failed to deliver reminder to user %d (booking %s): %v", job.UserID, job.BookingCode, err)
			_ = w.reminderRepo.IncrementRetry(ctx, job.ID, err.Error())
			continue
		}

		// Sukses dikirim
		sentAt := time.Now()
		if err := w.reminderRepo.UpdateStatus(ctx, job.ID, models.ReminderStatusSent, &sentAt, ""); err != nil {
			log.Printf("worker: failed to update reminder job status %s: %v", job.ID.Hex(), err)
		} else {
			log.Printf("worker: reminder %s delivered to user %s for booking %s", job.ReminderType, job.UserName, job.BookingCode)
		}
	}
}
