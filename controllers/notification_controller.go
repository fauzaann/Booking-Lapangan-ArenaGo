package controllers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"Booking-Lapangan/config"
	notificationv1 "Booking-Lapangan/contracts/notification/v1"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/timeutil"
	"Booking-Lapangan/repository"
)

type VenueAnnouncementResult struct {
	Recipients         int  `json:"recipients"`
	InAppEnabled       bool `json:"in_app_enabled"`
	WhatsAppRecipients int  `json:"whatsapp_recipients"`
	WhatsAppSent       int  `json:"whatsapp_sent"`
	WhatsAppQueued     int  `json:"whatsapp_queued"`
	WhatsAppFailed     int  `json:"whatsapp_failed"`
	InAppFailed        int  `json:"in_app_failed"`
	WhatsAppEnabled    bool `json:"whatsapp_enabled"`
}

type NotificationOptions struct {
	WhatsAppConfig WhatsAppNotificationConfig
	WhatsAppJobs   repository.WhatsAppJobRepository
}

type WhatsAppNotificationConfig struct {
	PaymentReminderDelay       time.Duration
	BookingConfirmedTemplate   string
	BookingCancelledTemplate   string
	BookingRescheduledTemplate string
	VenueAnnouncementTemplate  string
}

// NotificationController mengelola penjadwalan reminder dan inbox notifikasi berbasis MongoDB.
type NotificationController interface {
	ScheduleRemindersForBooking(ctx context.Context, bookingID uint) error
	SchedulePaymentReminder(ctx context.Context, bookingID uint, paymentURL string, expiresAt time.Time) error
	CancelRemindersForBooking(ctx context.Context, bookingID uint) error
	RescheduleBooking(ctx context.Context, bookingID uint) error
	PublishVenueAnnouncement(ctx context.Context, title, message string) (*VenueAnnouncementResult, error)
	ListUserNotifications(ctx context.Context, userID uint, page, limit int, unreadOnly bool) ([]models.Notification, int64, error)
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)
	MarkAsRead(ctx context.Context, notifIDHex string, userID uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	ListReminderJobs(ctx context.Context, page, limit int, status string) ([]models.ReminderJob, int64, error)
	HandleOutboxEvent(ctx context.Context, event notificationv1.Envelope) error
}

type notificationController struct {
	uow          repository.UnitOfWork
	notifRepo    repository.NotificationRepository
	reminderRepo repository.ReminderRepository
	whatsAppJobs repository.WhatsAppJobRepository
	mongoCfg     config.MongoConfig
	whatsAppCfg  WhatsAppNotificationConfig
}

// NewNotificationController membuat controller notifikasi.
func NewNotificationController(
	uow repository.UnitOfWork,
	notifRepo repository.NotificationRepository,
	reminderRepo repository.ReminderRepository,
	mongoCfg config.MongoConfig,
	options ...NotificationOptions,
) NotificationController {
	controller := &notificationController{
		uow:          uow,
		notifRepo:    notifRepo,
		reminderRepo: reminderRepo,
		mongoCfg:     mongoCfg,
	}
	if len(options) > 0 {
		controller.whatsAppCfg = options[0].WhatsAppConfig
		controller.whatsAppJobs = options[0].WhatsAppJobs
	}
	return controller
}

func (c *notificationController) HandleOutboxEvent(ctx context.Context, event notificationv1.Envelope) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid notification event %q: %w", event.ID, err)
	}
	switch event.Type {
	case notificationv1.EventBookingConfirmed:
		data, err := event.BookingReference()
		if err != nil {
			return err
		}
		booking, err := bookingFromSnapshot(data.Booking)
		if err != nil {
			return err
		}
		return c.scheduleRemindersForBooking(ctx, booking, event.ID)
	case notificationv1.EventBookingCancelled:
		data, err := event.BookingReference()
		if err != nil {
			return err
		}
		booking, err := bookingFromSnapshot(data.Booking)
		if err != nil {
			return err
		}
		return c.cancelRemindersForBooking(ctx, booking, event.ID)
	case notificationv1.EventBookingRescheduled:
		data, err := event.BookingReference()
		if err != nil {
			return err
		}
		booking, err := bookingFromSnapshot(data.Booking)
		if err != nil {
			return err
		}
		return c.rescheduleBooking(ctx, booking, event.ID)
	case notificationv1.EventPaymentInvoiceReady:
		data, err := event.PaymentInvoiceReady()
		if err != nil {
			return err
		}
		booking, err := bookingFromSnapshot(data.Booking)
		if err != nil {
			return err
		}
		return c.schedulePaymentReminder(ctx, booking, data.PaymentURL, data.ExpiresAt)
	default:
		return fmt.Errorf("unsupported notification event type %q", event.Type)
	}
}

func bookingFromSnapshot(snapshot notificationv1.BookingSnapshot) (*models.Booking, error) {
	bookingID, err := parseNotificationID(snapshot.BookingID, "booking_id")
	if err != nil {
		return nil, err
	}
	userID, err := parseNotificationID(snapshot.UserID, "user_id")
	if err != nil {
		return nil, err
	}
	fieldID, err := parseNotificationID(snapshot.FieldID, "field_id")
	if err != nil {
		return nil, err
	}
	bookingDate, err := time.Parse(timeutil.DateLayout, snapshot.BookingDate)
	if err != nil {
		return nil, fmt.Errorf("invalid notification event booking_date: %w", err)
	}
	return &models.Booking{
		ID: bookingID, BookingCode: snapshot.BookingCode,
		UserID: userID, FieldID: fieldID, BookingDate: bookingDate,
		StartTime: snapshot.StartTime, EndTime: snapshot.EndTime,
		Duration: snapshot.Duration, TotalPrice: snapshot.TotalPrice,
		Status:               models.BookingStatus(snapshot.Status),
		RefundReviewRequired: snapshot.RefundReviewRequired,
		User: &models.User{
			ID: userID, Name: snapshot.UserName, Email: snapshot.UserEmail,
			Phone: snapshot.UserPhone, WhatsAppOptIn: snapshot.UserWhatsAppOptIn,
		},
		Field: &models.Field{ID: fieldID, Name: snapshot.FieldName},
	}, nil
}

func parseNotificationID(value, field string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, strconv.IntSize)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid notification event %s %q", field, value)
	}
	return uint(id), nil
}

// ScheduleRemindersForBooking membuat dokumen pengingat H-24 dan H-2 jam di MongoDB saat booking CONFIRMED.
func (c *notificationController) ScheduleRemindersForBooking(ctx context.Context, bookingID uint) error {
	booking, err := c.uow.Booking().FindDetailByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to fetch booking details for reminder: %w", err)
	}
	return c.scheduleRemindersForBooking(ctx, booking, "")
}

func (c *notificationController) scheduleRemindersForBooking(ctx context.Context, booking *models.Booking, eventID string) error {
	bookingID := booking.ID
	now := timeNow()
	var result error
	if c.reminderRepo != nil {
		if err := c.reminderRepo.CancelByBookingID(ctx, bookingID); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to cancel stale reminder jobs for booking %d: %w", bookingID, err))
		} else {
			jobs, err := c.buildReminderJobs(booking, now)
			if err != nil {
				result = errors.Join(result, err)
			} else if len(jobs) > 0 {
				if err := c.reminderRepo.CreateJobs(ctx, jobs); err != nil {
					result = errors.Join(result, fmt.Errorf("failed to insert reminder jobs: %w", err))
				} else {
					log.Printf("notification: %d reminder job(s) scheduled for booking %s", len(jobs), booking.BookingCode)
				}
			}
		}
	}

	dateStr := booking.BookingDate.Format("02 Jan 2006")
	field := fieldName(booking)
	userName := ""
	if booking.User != nil {
		userName = booking.User.Name
	}
	channels := []string{models.NotificationChannelInApp}
	whatsAppQueued, whatsAppErr := c.enqueueWhatsApp(ctx, booking.User, c.whatsAppCfg.BookingConfirmedTemplate, []string{
		userName, booking.BookingCode, field, dateStr, booking.StartTime, booking.EndTime,
	}, eventID)
	if whatsAppQueued {
		channels = append(channels, models.NotificationChannelWA)
	}
	if c.notifRepo != nil {
		createErr := c.notifRepo.Create(ctx, &models.Notification{
			EventID:     eventID,
			UserID:      booking.UserID,
			BookingID:   booking.ID,
			BookingCode: booking.BookingCode,
			Title:       "Booking Berhasil Dikonfirmasi 🎉",
			Message:     fmt.Sprintf("Pembayaran booking %s berhasil! Jadwal Anda di %s pada %s pukul %s-%s WIB telah terkonfirmasi.", booking.BookingCode, field, dateStr, booking.StartTime, booking.EndTime),
			Type:        models.NotificationTypeBookingConfirmed,
			Channels:    channels,
			IsRead:      false,
			Data: map[string]any{
				"booking_code": booking.BookingCode,
				"field_name":   field,
				"date":         dateStr,
				"start_time":   booking.StartTime,
				"end_time":     booking.EndTime,
			},
			CreatedAt: now,
			ExpiresAt: now.AddDate(0, 1, 0), // 30 hari TTL
		})
		result = errors.Join(result, createErr)
	}
	if whatsAppErr != nil {
		result = errors.Join(result, whatsAppErr)
	}
	return result
}

// SchedulePaymentReminder membuat satu pengingat tertunda untuk invoice yang belum dibayar.
func (c *notificationController) SchedulePaymentReminder(ctx context.Context, bookingID uint, paymentURL string, expiresAt time.Time) error {
	if c.reminderRepo == nil {
		return nil
	}
	booking, err := c.uow.Booking().FindDetailByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to fetch booking details for payment reminder: %w", err)
	}
	return c.schedulePaymentReminder(ctx, booking, paymentURL, expiresAt)
}

func (c *notificationController) schedulePaymentReminder(ctx context.Context, booking *models.Booking, paymentURL string, expiresAt time.Time) error {
	if c.reminderRepo == nil {
		return nil
	}
	if booking.User == nil {
		return fmt.Errorf("booking %d does not include user details for payment reminder", booking.ID)
	}
	delay := c.whatsAppCfg.PaymentReminderDelay
	if delay <= 0 {
		delay = 10 * time.Minute
	}
	now := timeNow()
	scheduledFor := now.Add(delay)
	if !expiresAt.After(scheduledFor) {
		return nil
	}
	fieldName := ""
	if booking.Field != nil {
		fieldName = booking.Field.Name
	}
	job := models.ReminderJob{
		BookingID: booking.ID, BookingCode: booking.BookingCode,
		UserID: booking.UserID, UserName: booking.User.Name, UserEmail: booking.User.Email,
		UserPhone: booking.User.Phone, UserWhatsAppOptIn: booking.User.WhatsAppOptIn,
		FieldID: booking.FieldID, FieldName: fieldName, BookingDate: booking.BookingDate,
		StartTime: booking.StartTime, EndTime: booking.EndTime,
		ReminderType: models.ReminderTypePaymentDue, ScheduledFor: scheduledFor,
		PaymentURL: paymentURL, ExpiredAt: &expiresAt, Amount: booking.TotalPrice,
		Status: models.ReminderStatusScheduled, CreatedAt: now,
	}
	if err := c.reminderRepo.CreateJobs(ctx, []models.ReminderJob{job}); err != nil {
		return fmt.Errorf("failed to schedule payment reminder for booking %s: %w", booking.BookingCode, err)
	}
	return nil
}

// CancelRemindersForBooking membatalkan antrean reminder di MongoDB saat booking CANCELLED.
func (c *notificationController) CancelRemindersForBooking(ctx context.Context, bookingID uint) error {
	booking, err := c.uow.Booking().FindDetailByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to load cancelled booking %d for notification: %w", bookingID, err)
	}
	return c.cancelRemindersForBooking(ctx, booking, "")
}

func (c *notificationController) cancelRemindersForBooking(ctx context.Context, booking *models.Booking, eventID string) error {
	if booking == nil {
		return fmt.Errorf("cancelled booking was not found for notification")
	}
	var result error
	if c.reminderRepo != nil {
		if err := c.reminderRepo.CancelByBookingID(ctx, booking.ID); err != nil {
			log.Printf("notification: failed to cancel reminder jobs for booking %d: %v", booking.ID, err)
			result = errors.Join(result, err)
		}
	}
	message := fmt.Sprintf("Jadwal booking Anda dengan kode %s telah dibatalkan.", booking.BookingCode)
	if booking.RefundReviewRequired {
		message += " Pembayaran yang sudah diterima tidak dikembalikan otomatis dan perlu ditinjau untuk proses refund."
	}
	channels := []string{models.NotificationChannelInApp}
	var whatsAppErr error
	if booking.User != nil {
		whatsAppQueued, sendErr := c.enqueueWhatsApp(ctx, booking.User, c.whatsAppCfg.BookingCancelledTemplate, []string{
			booking.User.Name, booking.BookingCode, fieldName(booking),
		}, eventID)
		whatsAppErr = sendErr
		if whatsAppQueued {
			channels = append(channels, models.NotificationChannelWA)
		}
	}
	if c.notifRepo != nil {
		now := timeNow()
		if createErr := c.notifRepo.Create(ctx, &models.Notification{
			EventID:     eventID,
			UserID:      booking.UserID,
			BookingID:   booking.ID,
			BookingCode: booking.BookingCode,
			Title:       "Booking Dibatalkan",
			Message:     message,
			Type:        models.NotificationTypeBookingCancelled,
			Channels:    channels,
			IsRead:      false,
			CreatedAt:   now, ExpiresAt: now.AddDate(0, 1, 0),
		}); createErr != nil {
			result = errors.Join(result, createErr)
		}
	}
	return errors.Join(result, whatsAppErr)
}

// RescheduleBooking refreshes pending reminders and creates one in-app
// notification after the PostgreSQL booking change has committed.
func (c *notificationController) RescheduleBooking(ctx context.Context, bookingID uint) error {
	booking, err := c.uow.Booking().FindDetailByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to load rescheduled booking %d: %w", bookingID, err)
	}
	return c.rescheduleBooking(ctx, booking, "")
}

func (c *notificationController) rescheduleBooking(ctx context.Context, booking *models.Booking, eventID string) error {
	bookingID := booking.ID
	if c.reminderRepo != nil {
		var jobs []models.ReminderJob
		if booking.Status == models.BookingStatusPaid || booking.Status == models.BookingStatusConfirmed {
			var err error
			jobs, err = c.buildReminderJobs(booking, timeNow())
			if err != nil {
				return err
			}
		}
		if err := c.reminderRepo.RescheduleJobs(ctx, bookingID, jobs); err != nil {
			return err
		}
	}
	now := timeNow()
	dateStr := booking.BookingDate.Format("02 Jan 2006")
	channels := []string{models.NotificationChannelInApp}
	var whatsAppErr error
	if booking.User != nil {
		whatsAppQueued, sendErr := c.enqueueWhatsApp(ctx, booking.User, c.whatsAppCfg.BookingRescheduledTemplate, []string{
			booking.User.Name, booking.BookingCode, fieldName(booking), dateStr, booking.StartTime, booking.EndTime,
		}, eventID)
		whatsAppErr = sendErr
		if whatsAppQueued {
			channels = append(channels, models.NotificationChannelWA)
		}
	}
	var createErr error
	if c.notifRepo != nil {
		createErr = c.notifRepo.Create(ctx, &models.Notification{
			EventID:     eventID,
			UserID:      booking.UserID,
			BookingID:   booking.ID,
			BookingCode: booking.BookingCode,
			Title:       "Jadwal Booking Diubah",
			Message:     fmt.Sprintf("Jadwal booking %s telah diubah menjadi %s pukul %s-%s WIB.", booking.BookingCode, dateStr, booking.StartTime, booking.EndTime),
			Type:        models.NotificationTypeBookingRescheduled,
			Channels:    channels,
			IsRead:      false,
			Data: map[string]any{
				"booking_code": booking.BookingCode,
				"field_name":   fieldName(booking),
				"date":         dateStr,
				"start_time":   booking.StartTime,
				"end_time":     booking.EndTime,
			},
			CreatedAt: now,
			ExpiresAt: now.AddDate(0, 1, 0),
		})
	}
	return errors.Join(createErr, whatsAppErr)
}

// PublishVenueAnnouncement mengirimkan pengumuman ke seluruh pengguna terdaftar.
func (c *notificationController) PublishVenueAnnouncement(ctx context.Context, title, message string) (*VenueAnnouncementResult, error) {
	result := &VenueAnnouncementResult{
		InAppEnabled:    c.notifRepo != nil,
		WhatsAppEnabled: c.whatsAppJobs != nil,
	}
	page := 1
	const pageSize = 100
	for {
		users, total, err := c.uow.User().FindAll(ctx, repository.ListParams{Page: page, Limit: pageSize})
		if err != nil {
			return result, fmt.Errorf("failed to list users for venue announcement: %w", err)
		}
		if len(users) == 0 {
			break
		}
		for i := range users {
			user := &users[i]
			result.Recipients++
			now := timeNow()
			channels := []string{models.NotificationChannelInApp}
			if user.WhatsAppOptIn {
				queued, queueErr := c.enqueueWhatsApp(ctx, user, c.whatsAppCfg.VenueAnnouncementTemplate, []string{title, message}, "")
				if queueErr != nil {
					result.WhatsAppFailed++
					log.Printf("notification: failed to queue venue announcement for user %d: %v", user.ID, queueErr)
				} else if queued {
					result.WhatsAppQueued++
					channels = append(channels, models.NotificationChannelWA)
				}
				result.WhatsAppRecipients++
			}
			if c.notifRepo != nil {
				if err := c.notifRepo.Create(ctx, &models.Notification{
					UserID: user.ID, Title: title, Message: message,
					Type:     models.NotificationTypeVenueAnnouncement,
					Channels: channels,
					IsRead:   false, CreatedAt: now, ExpiresAt: now.AddDate(0, 1, 0),
				}); err != nil {
					result.InAppFailed++
					log.Printf("notification: failed to create venue announcement for user %d: %v", user.ID, err)
				}
			}
		}
		if int64(page*pageSize) >= total || len(users) < pageSize {
			break
		}
		page++
	}
	return result, nil
}

func (c *notificationController) enqueueWhatsApp(ctx context.Context, user *models.User, template string, parameters []string, eventID string) (bool, error) {
	if user == nil || !user.WhatsAppOptIn {
		return false, nil
	}
	if c.whatsAppJobs == nil {
		return false, fmt.Errorf("WhatsApp delivery outbox is not available")
	}
	job := &models.WhatsAppJob{
		EventID:      eventID,
		UserID:       user.ID,
		Template:     template,
		Parameters:   append([]string(nil), parameters...),
		ScheduledFor: timeNow(),
		Status:       models.WhatsAppJobStatusScheduled,
	}
	if err := c.whatsAppJobs.Create(ctx, job); err != nil {
		return false, err
	}
	return true, nil
}

func fieldName(booking *models.Booking) string {
	if booking != nil && booking.Field != nil {
		return booking.Field.Name
	}
	return ""
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
				UserWhatsAppOptIn: booking.User != nil && booking.User.WhatsAppOptIn,
				FieldID:           booking.FieldID, FieldName: fieldName, BookingDate: booking.BookingDate,
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
