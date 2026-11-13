package worker

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"Booking-Lapangan/config"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

type mockReminderRepo struct {
	dueJobs  []models.ReminderJob
	sentIDs  []string
	statuses map[string]string
}

func (m *mockReminderRepo) CreateJobs(ctx context.Context, jobs []models.ReminderJob) error {
	return nil
}

func (m *mockReminderRepo) RescheduleJobs(ctx context.Context, bookingID uint, jobs []models.ReminderJob) error {
	return nil
}

func (m *mockReminderRepo) FindDueJobs(ctx context.Context, now time.Time, limit int64) ([]models.ReminderJob, error) {
	return m.dueJobs, nil
}

func (m *mockReminderRepo) ClaimDueJob(_ context.Context, id bson.ObjectID, _ time.Time) (bool, error) {
	if m.statuses == nil {
		m.statuses = make(map[string]string)
	}
	m.statuses[id.Hex()] = models.ReminderStatusProcessing
	return true, nil
}

func (m *mockReminderRepo) UpdateStatus(ctx context.Context, id bson.ObjectID, status string, sentAt *time.Time, errMsg string) error {
	if m.statuses == nil {
		m.statuses = make(map[string]string)
	}
	m.statuses[id.Hex()] = status
	if status == models.ReminderStatusSent {
		m.sentIDs = append(m.sentIDs, id.Hex())
	}
	return nil
}

func (m *mockReminderRepo) MarkInAppSent(context.Context, bson.ObjectID) error { return nil }

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

type mockReminderUserRepo struct {
	repository.UserRepository
	user *models.User
	err  error
}

func (m *mockReminderUserRepo) FindByID(context.Context, uint) (*models.User, error) {
	return m.user, m.err
}

type mockReminderWhatsAppSender struct {
	recipient  string
	template   string
	language   string
	parameters []string
	err        error
}

func (m *mockReminderWhatsAppSender) SendTemplate(_ context.Context, recipient, template, language string, parameters []string) error {
	m.recipient = recipient
	m.template = template
	m.language = language
	m.parameters = parameters
	return m.err
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
				PlayTime:     time.Now().Add(time.Hour),
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
	if notif.Title != "Pengingat: Jadwal Bermain Segera ⏳" {
		t.Errorf("expected immediate reminder title, got %q", notif.Title)
	}
	if !strings.Contains(notif.Message, "Waktu bermain sudah dekat") {
		t.Errorf("expected immediate reminder message, got %q", notif.Message)
	}

	// Pastikan status job diupdate menjadi SENT
	if len(mockReminder.sentIDs) != 1 || mockReminder.sentIDs[0] != jobID.Hex() {
		t.Errorf("expected sent job ID %s, got %v", jobID.Hex(), mockReminder.sentIDs)
	}
}

func TestReminderWorkerSendsPaymentReminderAndChecksCurrentConsent(t *testing.T) {
	jobID := bson.NewObjectID()
	mockReminder := &mockReminderRepo{dueJobs: []models.ReminderJob{{
		ID: jobID, BookingID: 50, BookingCode: "AG-PAY-001", UserID: 7,
		UserName: "Rizki", UserPhone: "+6281234567890", UserWhatsAppOptIn: true,
		FieldName: "Lapangan Beta", BookingDate: time.Now().AddDate(0, 0, 1),
		StartTime: "20:00", EndTime: "21:00", ReminderType: models.ReminderTypePaymentDue,
		ScheduledFor: time.Now().Add(-time.Minute), PaymentURL: "https://pay.example/invoice",
		ExpiredAt: timePointer(time.Now().Add(time.Hour)), Amount: 200000,
	}}}
	mockNotif := &mockNotifRepo{}
	sender := &mockReminderWhatsAppSender{}
	userRepo := &mockReminderUserRepo{user: &models.User{
		ID: 7, Name: "Rizki", Phone: "+6281234567890", WhatsAppOptIn: true,
	}}
	w := NewReminderWorker(mockReminder, mockNotif, time.Second, ReminderWorkerOptions{
		WhatsAppConfig: config.WhatsAppConfig{Language: "id", PaymentReminderTemplate: "payment_reminder"},
		WhatsAppSender: sender,
		UserRepository: userRepo,
	})
	w.runOnce(context.Background())

	if len(mockNotif.delivered) != 1 || mockNotif.delivered[0].Type != models.NotificationTypePaymentReminder {
		t.Fatalf("expected one payment reminder in-app notification, got %+v", mockNotif.delivered)
	}
	if mockNotif.delivered[0].Data["payment_url"] != "https://pay.example/invoice" {
		t.Fatalf("payment URL missing from notification data: %+v", mockNotif.delivered[0].Data)
	}
	if sender.template != "payment_reminder" || sender.recipient != "+6281234567890" {
		t.Fatalf("unexpected WhatsApp delivery: %+v", sender)
	}
	if len(sender.parameters) != 5 || sender.parameters[4] != "https://pay.example/invoice" {
		t.Fatalf("unexpected WhatsApp template parameters: %v", sender.parameters)
	}
	if mockReminder.statuses[jobID.Hex()] != models.ReminderStatusSent {
		t.Fatalf("expected reminder status SENT, got %q", mockReminder.statuses[jobID.Hex()])
	}
}

func TestReminderWorkerCancelsExpiredPaymentReminder(t *testing.T) {
	jobID := bson.NewObjectID()
	mockReminder := &mockReminderRepo{dueJobs: []models.ReminderJob{{
		ID: jobID, BookingID: 51, BookingCode: "AG-PAY-002", UserID: 7,
		ReminderType: models.ReminderTypePaymentDue,
		ScheduledFor: time.Now().Add(-time.Minute), ExpiredAt: timePointer(time.Now().Add(-time.Minute)),
	}}}
	mockNotif := &mockNotifRepo{}
	sender := &mockReminderWhatsAppSender{}
	w := NewReminderWorker(mockReminder, mockNotif, time.Second, ReminderWorkerOptions{
		WhatsAppSender: sender,
	})

	w.runOnce(context.Background())

	if len(mockNotif.delivered) != 0 || sender.template != "" {
		t.Fatal("paid booking should not receive a payment reminder")
	}
	if mockReminder.statuses[jobID.Hex()] != models.ReminderStatusCancelled {
		t.Fatalf("expected reminder status CANCELLED, got %q", mockReminder.statuses[jobID.Hex()])
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
