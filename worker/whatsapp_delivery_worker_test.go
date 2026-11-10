package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"Booking-Lapangan/config"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

type mockWhatsAppOutboxRepo struct {
	jobs       []models.WhatsAppJob
	claimed    []bson.ObjectID
	statuses   map[string]string
	retryCount int
}

func (r *mockWhatsAppOutboxRepo) Create(context.Context, *models.WhatsAppJob) error { return nil }
func (r *mockWhatsAppOutboxRepo) FindDue(context.Context, time.Time, int64) ([]models.WhatsAppJob, error) {
	return r.jobs, nil
}
func (r *mockWhatsAppOutboxRepo) Claim(_ context.Context, id bson.ObjectID, _ time.Time) (bool, error) {
	r.claimed = append(r.claimed, id)
	return true, nil
}
func (r *mockWhatsAppOutboxRepo) MarkSent(_ context.Context, id bson.ObjectID, _ time.Time) error {
	r.setStatus(id, models.WhatsAppJobStatusSent)
	return nil
}
func (r *mockWhatsAppOutboxRepo) Cancel(_ context.Context, id bson.ObjectID, _ string) error {
	r.setStatus(id, models.WhatsAppJobStatusCancelled)
	return nil
}
func (r *mockWhatsAppOutboxRepo) Retry(_ context.Context, id bson.ObjectID, _ string) error {
	r.retryCount++
	r.setStatus(id, models.WhatsAppJobStatusScheduled)
	return nil
}
func (r *mockWhatsAppOutboxRepo) EnsureIndexes(context.Context) error { return nil }
func (r *mockWhatsAppOutboxRepo) setStatus(id bson.ObjectID, status string) {
	if r.statuses == nil {
		r.statuses = make(map[string]string)
	}
	r.statuses[id.Hex()] = status
}

var _ repository.WhatsAppJobRepository = (*mockWhatsAppOutboxRepo)(nil)

func TestWhatsAppDeliveryWorkerDeliversWithCurrentConsent(t *testing.T) {
	jobID := bson.NewObjectID()
	jobs := &mockWhatsAppOutboxRepo{jobs: []models.WhatsAppJob{{
		ID: jobID, UserID: 9, Template: "booking_confirmed", Parameters: []string{"Rizki"},
		ScheduledFor: time.Now().Add(-time.Minute), Status: models.WhatsAppJobStatusScheduled,
	}}}
	users := &mockReminderUserRepo{user: &models.User{
		ID: 9, Phone: "+6281234567890", WhatsAppOptIn: true,
	}}
	sender := &mockReminderWhatsAppSender{}

	worker := NewWhatsAppDeliveryWorker(jobs, users, sender, configForWhatsAppTest(), time.Second)
	worker.runOnce(context.Background())

	if sender.recipient != users.user.Phone || sender.template != "booking_confirmed" ||
		len(sender.parameters) != 1 || sender.parameters[0] != "Rizki" {
		t.Fatalf("unexpected WhatsApp delivery: %+v", sender)
	}
	if jobs.statuses[jobID.Hex()] != models.WhatsAppJobStatusSent {
		t.Fatalf("job status = %q, want SENT", jobs.statuses[jobID.Hex()])
	}
}

func TestWhatsAppDeliveryWorkerCancelsAfterConsentIsWithdrawn(t *testing.T) {
	jobID := bson.NewObjectID()
	jobs := &mockWhatsAppOutboxRepo{jobs: []models.WhatsAppJob{{ID: jobID, UserID: 9}}}
	users := &mockReminderUserRepo{user: &models.User{ID: 9, WhatsAppOptIn: false}}
	sender := &mockReminderWhatsAppSender{}

	worker := NewWhatsAppDeliveryWorker(jobs, users, sender, configForWhatsAppTest(), time.Second)
	worker.runOnce(context.Background())

	if jobs.statuses[jobID.Hex()] != models.WhatsAppJobStatusCancelled {
		t.Fatalf("job status = %q, want CANCELLED", jobs.statuses[jobID.Hex()])
	}
	if sender.recipient != "" {
		t.Fatalf("sent WhatsApp to user who opted out: %+v", sender)
	}
}

func TestWhatsAppDeliveryWorkerRetriesProviderErrors(t *testing.T) {
	jobID := bson.NewObjectID()
	jobs := &mockWhatsAppOutboxRepo{jobs: []models.WhatsAppJob{{ID: jobID, UserID: 9}}}
	users := &mockReminderUserRepo{user: &models.User{ID: 9, WhatsAppOptIn: true, Phone: "+6281234567890"}}
	sender := &mockReminderWhatsAppSender{err: errors.New("provider unavailable")}

	worker := NewWhatsAppDeliveryWorker(jobs, users, sender, configForWhatsAppTest(), time.Second)
	worker.runOnce(context.Background())

	if jobs.retryCount != 1 || jobs.statuses[jobID.Hex()] != models.WhatsAppJobStatusScheduled {
		t.Fatalf("expected job retry, got retries=%d status=%q", jobs.retryCount, jobs.statuses[jobID.Hex()])
	}
}

func configForWhatsAppTest() config.WhatsAppConfig {
	return config.WhatsAppConfig{Language: "id"}
}
