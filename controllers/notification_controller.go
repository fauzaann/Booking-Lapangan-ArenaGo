package controllers

import (
	"context"
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
	ListUserNotifications(ctx context.Context, userID uint, page, limit int, unreadOnly bool) ([]models.Notification, int64, error)
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)
	MarkAsRead(ctx context.Context, notifIDHex string, userID uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	ListReminderJobs(ctx context.Context, page, limit int, status string) ([]models.ReminderJob, int64, error)
}

type notificationController struct {
	uow            repository.UnitOfWork
	notifRepo      repository.NotificationRepository
	reminderRepo   repository.ReminderRepository
	mongoCfg       config.MongoConfig
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

	playTime, err := timeutil.CombineDateTime(booking.BookingDate, booking.StartTime)
	if err != nil {
		return fmt.Errorf("invalid booking schedule: %w", err)
	}

	now := timeNow()
	var jobs []models.ReminderJob

	userName := ""
	userEmail := ""
	userPhone := ""
	if booking.User != nil {
		userName = booking.User.Name
		userEmail = booking.User.Email
		userPhone = booking.User.Phone
	}

	fieldName := ""
	if booking.Field != nil {
		fieldName = booking.Field.Name
	}

	// 1. Pengingat H-24 Jam
	if c.mongoCfg.ReminderH24 {
		h24Scheduled := playTime.Add(-24 * time.Hour)
		if h24Scheduled.After(now) {
			jobs = append(jobs, models.ReminderJob{
				BookingID:    booking.ID,
				BookingCode:  booking.BookingCode,
				UserID:       booking.UserID,
				UserName:     userName,
				UserEmail:    userEmail,
				UserPhone:    userPhone,
				FieldID:      booking.FieldID,
				FieldName:    fieldName,
				BookingDate:  booking.BookingDate,
				StartTime:    booking.StartTime,
				EndTime:      booking.EndTime,
				PlayTime:     playTime,
				ReminderType: models.ReminderTypeH24,
				ScheduledFor: h24Scheduled,
				Status:       models.ReminderStatusScheduled,
				CreatedAt:    now,
			})
		}
	}

	// 2. Pengingat H-2 Jam
	if c.mongoCfg.ReminderH2 {
		h2Scheduled := playTime.Add(-2 * time.Hour)
		if h2Scheduled.After(now) {
			jobs = append(jobs, models.ReminderJob{
				BookingID:    booking.ID,
				BookingCode:  booking.BookingCode,
				UserID:       booking.UserID,
				UserName:     userName,
				UserEmail:    userEmail,
				UserPhone:    userPhone,
				FieldID:      booking.FieldID,
				FieldName:    fieldName,
				BookingDate:  booking.BookingDate,
				StartTime:    booking.StartTime,
				EndTime:      booking.EndTime,
				PlayTime:     playTime,
				ReminderType: models.ReminderTypeH2,
				ScheduledFor: h2Scheduled,
				Status:       models.ReminderStatusScheduled,
				CreatedAt:    now,
			})
		}
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
	if c.reminderRepo != nil {
		if err := c.reminderRepo.CancelByBookingID(ctx, bookingID); err != nil {
			log.Printf("notification: failed to cancel reminder jobs for booking %d: %v", bookingID, err)
		}
	}

	if c.notifRepo != nil {
		booking, err := c.uow.Booking().FindByID(ctx, bookingID)
		if err == nil && booking != nil {
			now := timeNow()
			_ = c.notifRepo.Create(ctx, &models.Notification{
				UserID:      booking.UserID,
				BookingID:   booking.ID,
				BookingCode: booking.BookingCode,
				Title:       "Booking Dibatalkan",
				Message:     fmt.Sprintf("Jadwal booking Anda dengan kode %s telah dibatalkan.", booking.BookingCode),
				Type:        models.NotificationTypeBookingCancelled,
				Channels:    []string{models.NotificationChannelInApp},
				IsRead:      false,
				CreatedAt:   now,
				ExpiresAt:   now.AddDate(0, 1, 0),
			})
		}
	}

	return nil
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
