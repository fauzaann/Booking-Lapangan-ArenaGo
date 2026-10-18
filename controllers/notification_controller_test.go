package controllers

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"Booking-Lapangan/config"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

// mockReminderRepo implements repository.ReminderRepository for unit testing.
type mockReminderRepo struct {
	createdJobs []models.ReminderJob
	cancelledIDs []uint
	statusUpdates map[string]string
}

func newMockReminderRepo() *mockReminderRepo {
	return &mockReminderRepo{
		statusUpdates: make(map[string]string),
	}
}

func (m *mockReminderRepo) CreateJobs(ctx context.Context, jobs []models.ReminderJob) error {
	m.createdJobs = append(m.createdJobs, jobs...)
	return nil
}

func (m *mockReminderRepo) FindDueJobs(ctx context.Context, now time.Time, limit int64) ([]models.ReminderJob, error) {
	return nil, nil
}

func (m *mockReminderRepo) UpdateStatus(ctx context.Context, id bson.ObjectID, status string, sentAt *time.Time, errMsg string) error {
	m.statusUpdates[id.Hex()] = status
	return nil
}

func (m *mockReminderRepo) IncrementRetry(ctx context.Context, id bson.ObjectID, errMsg string) error {
	return nil
}

func (m *mockReminderRepo) CancelByBookingID(ctx context.Context, bookingID uint) error {
	m.cancelledIDs = append(m.cancelledIDs, bookingID)
	return nil
}

func (m *mockReminderRepo) FindAll(ctx context.Context, page, limit int, status string) ([]models.ReminderJob, int64, error) {
	return m.createdJobs, int64(len(m.createdJobs)), nil
}

func (m *mockReminderRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

// mockNotifRepo implements repository.NotificationRepository for unit testing.
type mockNotifRepo struct {
	notifications []models.Notification
	unreadCount   int64
	readIDs       []string
}

func newMockNotifRepo() *mockNotifRepo {
	return &mockNotifRepo{}
}

func (m *mockNotifRepo) Create(ctx context.Context, notif *models.Notification) error {
	m.notifications = append(m.notifications, *notif)
	m.unreadCount++
	return nil
}

func (m *mockNotifRepo) FindByUserID(ctx context.Context, userID uint, page, limit int, unreadOnly bool) ([]models.Notification, int64, error) {
	var filtered []models.Notification
	for _, n := range m.notifications {
		if n.UserID == userID {
			if !unreadOnly || !n.IsRead {
				filtered = append(filtered, n)
			}
		}
	}
	return filtered, int64(len(filtered)), nil
}

func (m *mockNotifRepo) CountUnread(ctx context.Context, userID uint) (int64, error) {
	return m.unreadCount, nil
}

func (m *mockNotifRepo) MarkAsRead(ctx context.Context, id bson.ObjectID, userID uint) error {
	m.readIDs = append(m.readIDs, id.Hex())
	if m.unreadCount > 0 {
		m.unreadCount--
	}
	return nil
}

func (m *mockNotifRepo) MarkAllAsRead(ctx context.Context, userID uint) error {
	m.unreadCount = 0
	return nil
}

func (m *mockNotifRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

// mockBookingRepo implements repository.BookingRepository minimal for test
type mockBookingRepo struct {
	booking *models.Booking
}

func (m *mockBookingRepo) Create(ctx context.Context, booking *models.Booking) error { return nil }
func (m *mockBookingRepo) Update(ctx context.Context, booking *models.Booking) error { return nil }
func (m *mockBookingRepo) FindByID(ctx context.Context, id uint) (*models.Booking, error) {
	return m.booking, nil
}
func (m *mockBookingRepo) FindDetailByID(ctx context.Context, id uint) (*models.Booking, error) {
	return m.booking, nil
}
func (m *mockBookingRepo) FindByCode(ctx context.Context, code string) (*models.Booking, error) {
	return m.booking, nil
}
func (m *mockBookingRepo) FindAll(ctx context.Context, filter repository.BookingFilter) ([]models.Booking, int64, error) {
	return nil, 0, nil
}
func (m *mockBookingRepo) FindActiveByFieldAndDate(ctx context.Context, fieldID uint, date time.Time) ([]models.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) HasOverlap(ctx context.Context, fieldID uint, date time.Time, startTime, endTime string) (bool, error) {
	return false, nil
}
func (m *mockBookingRepo) LockFieldDate(ctx context.Context, fieldID uint, date time.Time) error {
	return nil
}
func (m *mockBookingRepo) NextSequence(ctx context.Context, date time.Time) (int, error) {
	return 1, nil
}
func (m *mockBookingRepo) Count(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockBookingRepo) CountByStatus(ctx context.Context) (map[models.BookingStatus]int64, error) {
	return nil, nil
}
func (m *mockBookingRepo) MarkExpired(ctx context.Context, bookingIDs []uint) (int64, error) {
	return 0, nil
}
func (m *mockBookingRepo) CompleteFinished(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}

// mockUOW implements repository.UnitOfWork for tests
type mockUOW struct {
	bookingRepo repository.BookingRepository
}

func (m *mockUOW) User() repository.UserRepository         { return nil }
func (m *mockUOW) Field() repository.FieldRepository       { return nil }
func (m *mockUOW) Schedule() repository.ScheduleRepository { return nil }
func (m *mockUOW) Booking() repository.BookingRepository   { return m.bookingRepo }
func (m *mockUOW) Payment() repository.PaymentRepository   { return nil }
func (m *mockUOW) AuditLog() repository.AuditLogRepository { return nil }
func (m *mockUOW) Atomic(ctx context.Context, fn func(r repository.Registry) error) error {
	return fn(m)
}

func TestScheduleRemindersForBooking(t *testing.T) {
	baseTime := time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)
	timeNow = func() time.Time {
		return baseTime
	}
	defer func() {
		timeNow = time.Now
	}()

	bookingDate := time.Date(2026, 10, 22, 0, 0, 0, 0, time.Local)
	mockBooking := &models.Booking{
		ID:          100,
		BookingCode: "AG-TEST-001",
		UserID:      10,
		FieldID:     1,
		BookingDate: bookingDate,
		StartTime:   "16:00",
		EndTime:     "18:00",
		User: &models.User{
			ID:    10,
			Name:  "Andi",
			Email: "andi@example.com",
			Phone: "081234567890",
		},
		Field: &models.Field{
			ID:   1,
			Name: "Lapangan Padel Alpha",
		},
	}

	uow := &mockUOW{bookingRepo: &mockBookingRepo{booking: mockBooking}}
	mockReminder := newMockReminderRepo()
	mockNotif := newMockNotifRepo()

	cfg := config.MongoConfig{
		ReminderH24: true,
		ReminderH2:  true,
	}

	controller := NewNotificationController(uow, mockNotif, mockReminder, cfg)

	ctx := context.Background()
	err := controller.ScheduleRemindersForBooking(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Memastikan 2 job reminder dijadwalkan (H-24 dan H-2)
	if len(mockReminder.createdJobs) != 2 {
		t.Fatalf("expected 2 reminder jobs, got %d", len(mockReminder.createdJobs))
	}

	job1 := mockReminder.createdJobs[0]
	if job1.ReminderType != models.ReminderTypeH24 {
		t.Errorf("expected job1 type %s, got %s", models.ReminderTypeH24, job1.ReminderType)
	}
	// PlayTime adalah 2026-10-22 16:00, H-24 adalah 2026-10-21 16:00
	expectedH24 := time.Date(2026, 10, 21, 16, 0, 0, 0, time.Local)
	if !job1.ScheduledFor.Equal(expectedH24) {
		t.Errorf("expected H-24 time %v, got %v", expectedH24, job1.ScheduledFor)
	}

	job2 := mockReminder.createdJobs[1]
	if job2.ReminderType != models.ReminderTypeH2 {
		t.Errorf("expected job2 type %s, got %s", models.ReminderTypeH2, job2.ReminderType)
	}
	// PlayTime adalah 2026-10-22 16:00, H-2 adalah 2026-10-22 14:00
	expectedH2 := time.Date(2026, 10, 22, 14, 0, 0, 0, time.Local)
	if !job2.ScheduledFor.Equal(expectedH2) {
		t.Errorf("expected H-2 time %v, got %v", expectedH2, job2.ScheduledFor)
	}

	// Memastikan in-app notifikasi konfirmasi booking terbuat
	if len(mockNotif.notifications) != 1 {
		t.Fatalf("expected 1 in-app notification, got %d", len(mockNotif.notifications))
	}
	if mockNotif.notifications[0].Type != models.NotificationTypeBookingConfirmed {
		t.Errorf("expected notification type %s, got %s", models.NotificationTypeBookingConfirmed, mockNotif.notifications[0].Type)
	}
}

func TestCancelRemindersForBooking(t *testing.T) {
	uow := &mockUOW{bookingRepo: &mockBookingRepo{booking: &models.Booking{ID: 200, BookingCode: "AG-CANCEL-001", UserID: 10}}}
	mockReminder := newMockReminderRepo()
	mockNotif := newMockNotifRepo()

	controller := NewNotificationController(uow, mockNotif, mockReminder, config.MongoConfig{})
	ctx := context.Background()

	if err := controller.CancelRemindersForBooking(ctx, 200); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockReminder.cancelledIDs) != 1 || mockReminder.cancelledIDs[0] != 200 {
		t.Errorf("expected cancelled booking ID 200, got %v", mockReminder.cancelledIDs)
	}

	if len(mockNotif.notifications) != 1 {
		t.Fatalf("expected 1 cancellation notification, got %d", len(mockNotif.notifications))
	}
	if mockNotif.notifications[0].Type != models.NotificationTypeBookingCancelled {
		t.Errorf("expected type %s, got %s", models.NotificationTypeBookingCancelled, mockNotif.notifications[0].Type)
	}
}

func TestNotificationReadOperations(t *testing.T) {
	mockReminder := newMockReminderRepo()
	mockNotif := newMockNotifRepo()

	oid := bson.NewObjectID()
	mockNotif.notifications = append(mockNotif.notifications, models.Notification{
		ID:     oid,
		UserID: 5,
		IsRead: false,
	})
	mockNotif.unreadCount = 1

	controller := NewNotificationController(nil, mockNotif, mockReminder, config.MongoConfig{})
	ctx := context.Background()

	count, err := controller.GetUnreadCount(ctx, 5)
	if err != nil || count != 1 {
		t.Fatalf("expected unread count 1, got %d (err: %v)", count, err)
	}

	if err := controller.MarkAsRead(ctx, oid.Hex(), 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockNotif.readIDs) != 1 || mockNotif.readIDs[0] != oid.Hex() {
		t.Errorf("expected marked read ID %s, got %v", oid.Hex(), mockNotif.readIDs)
	}

	countAfter, _ := controller.GetUnreadCount(ctx, 5)
	if countAfter != 0 {
		t.Errorf("expected unread count 0, got %d", countAfter)
	}
}
