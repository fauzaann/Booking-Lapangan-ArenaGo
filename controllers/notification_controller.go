package controllers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"Booking-Lapangan/config"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/timeutil"
	"Booking-Lapangan/repository"
)

// NotificationController mengelola penjadwalan reminder dan inbox notifikasi berbasis MongoDB.
type NotificationController interface {
	ScheduleRemindersForBooking(ctx context.Context, bookingID uint) error
	CancelRemindersForBooking(ctx context.Context, bookingID uint) error
	RescheduleBooking(ctx context.Context, bookingID uint) error
	ListUserNotifications(ctx context.Context, userID uint, page, limit int, unreadOnly bool) ([]models.Notification, int64, error)
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)
	MarkAsRead(ctx context.Context, notifIDHex string, userID uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	ListReminderJobs(ctx context.Context, page, limit int, status string) ([]models.ReminderJob, int64, error)
}

type notificationController struct {
	uow          repository.UnitOfWork
	notifRepo    repository.NotificationRepository
	reminderRepo repository.ReminderRepository
	mongoCfg     config.MongoConfig
}

// NewNotificationController membuat controller notifikasi.
func NewNotificationController(
	uow repository.UnitOfWork,
	notifRepo repository.NotificationRepository,
	reminderRepo repository.ReminderRepository,
	mongoCfg config.MongoConfig,
) NotificationController {
	return &notificationController{
		uow:          uow,
		notifRepo:    notifRepo,
		reminderRepo: reminderRepo,
		mongoCfg:     mongoCfg,
	}
}

// ScheduleRemindersForBooking membuat dokumen pengingat H-24 dan H-2 jam di MongoDB saat booking CONFIRMED.
func (c *notificationController) ScheduleRemindersForBooking(ctx context.Context, bookingID uint) error {
	if c.reminderRepo == nil {
		log.Println("notification: mongodb reminder repository is not initialized, skipping reminder scheduling")
		return nil
	}

	booking, err := c.uow.Booking().FindDetailByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to fetch booking details for reminder: %w", err)
	}

	now := timeNow()
	jobs, err := c.buildReminderJobs(booking, now)
	if err != nil {
		return err
	}

	if len(jobs) > 0 {
		if err := c.reminderRepo.CreateJobs(ctx, jobs); err != nil {
			log.Printf("notification: failed to insert reminder jobs: %v", err)
		} else {
			log.Printf("notification: %d reminder job(s) scheduled for booking %s", len(jobs), booking.BookingCode)
		}
	}

	// Buat notifikasi in-app langsung bahwa booking telah dikonfirmasi
	if c.notifRepo != nil {
		dateStr := booking.BookingDate.Format("02 Jan 2006")
		fieldName := ""
		if booking.Field != nil {
			fieldName = booking.Field.Name
		}
		_ = c.notifRepo.Create(ctx, &models.Notification{
			UserID:      booking.UserID,
			BookingID:   booking.ID,
			BookingCode: booking.BookingCode,
			Title:       "Booking Berhasil Dikonfirmasi 🎉",
			Message:     fmt.Sprintf("Pembayaran booking %s berhasil! Jadwal Anda di %s pada %s pukul %s-%s WIB telah terkonfirmasi.", booking.BookingCode, fieldName, dateStr, booking.StartTime, booking.EndTime),
			Type:        models.NotificationTypeBookingConfirmed,
			Channels:    []string{models.NotificationChannelInApp},
			IsRead:      false,
			Data: map[string]any{
				"booking_code": booking.BookingCode,
				"field_name":   fieldName,
				"date":         dateStr,
				"start_time":   booking.StartTime,
				"end_time":     booking.EndTime,
			},
			CreatedAt: now,
			ExpiresAt: now.AddDate(0, 1, 0), // 30 hari TTL
		})
	}

	return nil
}

// CancelRemindersForBooking membatalkan antrean reminder di MongoDB saat booking CANCELLED.
func (c *notificationController) CancelRemindersForBooking(ctx context.Context, bookingID uint) error {
	var result error
	if c.reminderRepo != nil {
		if err := c.reminderRepo.CancelByBookingID(ctx, bookingID); err != nil {
			log.Printf("notification: failed to cancel reminder jobs for booking %d: %v", bookingID, err)
			result = errors.Join(result, err)
		}
	}

	if c.notifRepo != nil {
		booking, err := c.uow.Booking().FindByID(ctx, bookingID)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to load cancelled booking %d for notification: %w", bookingID, err))
		} else if booking == nil {
			result = errors.Join(result, fmt.Errorf("cancelled booking %d was not found for notification", bookingID))
		} else {
			now := timeNow()
			message := fmt.Sprintf("Jadwal booking Anda dengan kode %s telah dibatalkan.", booking.BookingCode)
			if booking.RefundReviewRequired {
				message += " Pembayaran yang sudah diterima tidak dikembalikan otomatis dan perlu ditinjau untuk proses refund."
			}
			if createErr := c.notifRepo.Create(ctx, &models.Notification{
				UserID:      booking.UserID,
				BookingID:   booking.ID,
				BookingCode: booking.BookingCode,
				Title:       "Booking Dibatalkan",
				Message:     message,
				Type:        models.NotificationTypeBookingCancelled,
				Channels:    []string{models.NotificationChannelInApp},
				IsRead:      false,
				CreatedAt:   now,
				ExpiresAt:   now.AddDate(0, 1, 0),
			}); createErr != nil {
				result = errors.Join(result, createErr)
			}
		}
	}

	return result
}

// RescheduleBooking refreshes pending reminders and creates one in-app
// notification after the PostgreSQL booking change has committed.
func (c *notificationController) RescheduleBooking(ctx context.Context, bookingID uint) error {
	booking, err := c.uow.Booking().FindDetailByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to load rescheduled booking %d: %w", bookingID, err)
	}
	if c.reminderRepo != nil {
		var jobs []models.ReminderJob
		if booking.Status == models.BookingStatusPaid || booking.Status == models.BookingStatusConfirmed {
			jobs, err = c.buildReminderJobs(booking, timeNow())
			if err != nil {
				return err
			}
		}
		if err := c.reminderRepo.RescheduleJobs(ctx, bookingID, jobs); err != nil {
			return err
		}
	}
	if c.notifRepo == nil {
		return nil
	}
	now := timeNow()
	dateStr := booking.BookingDate.Format("02 Jan 2006")
	fieldName := ""
	if booking.Field != nil {
		fieldName = booking.Field.Name
	}
	return c.notifRepo.Create(ctx, &models.Notification{
		UserID:      booking.UserID,
		BookingID:   booking.ID,
		BookingCode: booking.BookingCode,
		Title:       "Jadwal Booking Diubah",
		Message:     fmt.Sprintf("Jadwal booking %s telah diubah menjadi %s pukul %s-%s WIB.", booking.BookingCode, dateStr, booking.StartTime, booking.EndTime),
		Type:        models.NotificationTypeBookingRescheduled,
		Channels:    []string{models.NotificationChannelInApp},
		IsRead:      false,
		Data: map[string]any{
			"booking_code": booking.BookingCode,
			"field_name":   fieldName,
			"date":         dateStr,
			"start_time":   booking.StartTime,
			"end_time":     booking.EndTime,
		},
		CreatedAt: now,
		ExpiresAt: now.AddDate(0, 1, 0),
	})
}

func (c *notificationController) buildReminderJobs(booking *models.Booking, now time.Time) ([]models.ReminderJob, error) {
	playTime, err := timeutil.CombineDateTime(booking.BookingDate, booking.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid booking schedule: %w", err)
	}
	userName, userEmail, userPhone := "", "", ""
	if booking.User != nil {
		userName, userEmail, userPhone = booking.User.Name, booking.User.Email, booking.User.Phone
	}
	fieldName := ""
	if booking.Field != nil {
		fieldName = booking.Field.Name
	}

	jobs := make([]models.ReminderJob, 0, 2)
	for _, reminder := range []struct {
		enabled bool
		kind    string
		lead    time.Duration
	}{
		{c.mongoCfg.ReminderH24, models.ReminderTypeH24, 24 * time.Hour},
		{c.mongoCfg.ReminderH2, models.ReminderTypeH2, 2 * time.Hour},
	} {
		scheduledFor := playTime.Add(-reminder.lead)
		if !reminder.enabled || !playTime.After(now) {
			continue
		}
		if reminder.kind == models.ReminderTypeH2 && !scheduledFor.After(now) {
			scheduledFor = now
		}
		if scheduledFor.After(now) || reminder.kind == models.ReminderTypeH2 && scheduledFor.Equal(now) {
			jobs = append(jobs, models.ReminderJob{
				BookingID: booking.ID, BookingCode: booking.BookingCode,
				UserID: booking.UserID, UserName: userName, UserEmail: userEmail, UserPhone: userPhone,
				FieldID: booking.FieldID, FieldName: fieldName, BookingDate: booking.BookingDate,
				StartTime: booking.StartTime, EndTime: booking.EndTime, PlayTime: playTime,
				ReminderType: reminder.kind, ScheduledFor: scheduledFor,
				Status: models.ReminderStatusScheduled, CreatedAt: now,
			})
		}
	}
	return jobs, nil
}

// ListUserNotifications mengambil notifikasi inbox untuk user yang sedang login.
func (c *notificationController) ListUserNotifications(ctx context.Context, userID uint, page, limit int, unreadOnly bool) ([]models.Notification, int64, error) {
	if c.notifRepo == nil {
		return []models.Notification{}, 0, nil
	}
	return c.notifRepo.FindByUserID(ctx, userID, page, limit, unreadOnly)
}

// GetUnreadCount menghitung jumlah notifikasi yang belum dibaca untuk badge lonceng.
func (c *notificationController) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	if c.notifRepo == nil {
		return 0, nil
	}
	return c.notifRepo.CountUnread(ctx, userID)
}

// MarkAsRead menandai satu notifikasi sebagai sudah dibaca.
func (c *notificationController) MarkAsRead(ctx context.Context, notifIDHex string, userID uint) error {
	if c.notifRepo == nil {
		return nil
	}
	oid, err := bson.ObjectIDFromHex(notifIDHex)
	if err != nil {
		return apperror.BadRequest("invalid notification id")
	}
	if err := c.notifRepo.MarkAsRead(ctx, oid, userID); err != nil {
		return apperror.NotFound("notification not found")
	}
	return nil
}

// MarkAllAsRead menandai seluruh notifikasi user sebagai telah dibaca.
func (c *notificationController) MarkAllAsRead(ctx context.Context, userID uint) error {
	if c.notifRepo == nil {
		return nil
	}
	return c.notifRepo.MarkAllAsRead(ctx, userID)
}

// ListReminderJobs mengambil log daftar tugas pengingat jadwal untuk dipantau admin.
func (c *notificationController) ListReminderJobs(ctx context.Context, page, limit int, status string) ([]models.ReminderJob, int64, error) {
	if c.reminderRepo == nil {
		return []models.ReminderJob{}, 0, nil
	}
	return c.reminderRepo.FindAll(ctx, page, limit, status)
}
