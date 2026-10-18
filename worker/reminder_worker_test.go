package worker

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"Booking-Lapangan/models"
)

type mockReminderRepo struct {
	dueJobs []models.ReminderJob
	sentIDs []string
}

func (m *mockReminderRepo) CreateJobs(ctx context.Context, jobs []models.ReminderJob) error {
	return nil
}

func (m *mockReminderRepo) FindDueJobs(ctx context.Context, now time.Time, limit int64) ([]models.ReminderJob, error) {
	return m.dueJobs, nil
}

func (m *mockReminderRepo) UpdateStatus(ctx context.Context, id bson.ObjectID, status string, sentAt *time.Time, errMsg string) error {
	if status == models.ReminderStatusSent {
		m.sentIDs = append(m.sentIDs, id.Hex())
	}
	return nil
}

func (m *mockReminderRepo) IncrementRetry(ctx context.Context, id bson.ObjectID, errMsg string) error {
	return nil
}

func (m *mockReminderRepo) CancelByBookingID(ctx context.Context, bookingID uint) error {
	return nil
}

func (m *mockReminderRepo) FindAll(ctx context.Context, page, limit int, status string) ([]models.ReminderJob, int64, error) {
	return nil, 0, nil
}

func (m *mockReminderRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

type mockNotifRepo struct {
	delivered []models.Notification
}

func (m *mockNotifRepo) Create(ctx context.Context, notif *models.Notification) error {
	m.delivered = append(m.delivered, *notif)
	return nil
}

func (m *mockNotifRepo) FindByUserID(ctx context.Context, userID uint, page, limit int, unreadOnly bool) ([]models.Notification, int64, error) {
	return nil, 0, nil
}

func (m *mockNotifRepo) CountUnread(ctx context.Context, userID uint) (int64, error) {
	return 0, nil
}

func (m *mockNotifRepo) MarkAsRead(ctx context.Context, id bson.ObjectID, userID uint) error {
	return nil
}

func (m *mockNotifRepo) MarkAllAsRead(ctx context.Context, userID uint) error {
	return nil
}

func (m *mockNotifRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

func TestReminderWorkerRunOnce(t *testing.T) {
	jobID := bson.NewObjectID()
	mockReminder := &mockReminderRepo{
		dueJobs: []models.ReminderJob{
			{
				ID:           jobID,
				BookingID:    50,
				BookingCode:  "AG-WORKER-001",
				UserID:       7,
				UserName:     "Rizki",
				FieldName:    "Lapangan Padel Beta",
				BookingDate:  time.Now().Add(2 * time.Hour),
				StartTime:    "20:00",
				EndTime:      "22:00",
				ReminderType: models.ReminderTypeH2,
				ScheduledFor: time.Now().Add(-1 * time.Minute),
				Status:       models.ReminderStatusScheduled,
			},
		},
	}
	mockNotif := &mockNotifRepo{}

	w := NewReminderWorker(mockReminder, mockNotif, time.Second)
	ctx := context.Background()

	w.runOnce(ctx)

	// Pastikan notifikasi tersimpan ke in-app inbox
	if len(mockNotif.delivered) != 1 {
		t.Fatalf("expected 1 notification delivered, got %d", len(mockNotif.delivered))
	}
	notif := mockNotif.delivered[0]
	if notif.UserID != 7 {
		t.Errorf("expected user ID 7, got %d", notif.UserID)
	}
	if notif.Type != models.NotificationTypeReminderMatch {
		t.Errorf("expected type %s, got %s", models.NotificationTypeReminderMatch, notif.Type)
	}

	// Pastikan status job diupdate menjadi SENT
	if len(mockReminder.sentIDs) != 1 || mockReminder.sentIDs[0] != jobID.Hex() {
		t.Errorf("expected sent job ID %s, got %v", jobID.Hex(), mockReminder.sentIDs)
	}
}
