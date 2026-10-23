package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"Booking-Lapangan/models"
)

// ReminderRepository adalah kontrak akses data job pengingat jadwal di MongoDB.
type ReminderRepository interface {
	CreateJobs(ctx context.Context, jobs []models.ReminderJob) error
	RescheduleJobs(ctx context.Context, bookingID uint, jobs []models.ReminderJob) error
	FindDueJobs(ctx context.Context, now time.Time, limit int64) ([]models.ReminderJob, error)
	UpdateStatus(ctx context.Context, id bson.ObjectID, status string, sentAt *time.Time, errMsg string) error
	IncrementRetry(ctx context.Context, id bson.ObjectID, errMsg string) error
	CancelByBookingID(ctx context.Context, bookingID uint) error
	FindAll(ctx context.Context, page, limit int, status string) ([]models.ReminderJob, int64, error)
	EnsureIndexes(ctx context.Context) error
}

type reminderRepository struct {
	coll *mongo.Collection
}

// NewReminderRepository membuat repository pengingat berbasis MongoDB.
func NewReminderRepository(db *mongo.Database) ReminderRepository {
	return &reminderRepository{
		coll: db.Collection("reminder_jobs"),
	}
}

// EnsureIndexes membuat indeks yang dibutuhkan untuk performa query dan integritas data.
func (r *reminderRepository) EnsureIndexes(ctx context.Context) error {
	// 1. Unique index: booking_id + reminder_type (Mencegah reminder ganda)
	uniqueModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "booking_id", Value: 1},
			{Key: "reminder_type", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("idx_booking_reminder_type_unique"),
	}

	// 2. Query index: status + scheduled_for (Optimasi polling worker)
	statusModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "status", Value: 1},
			{Key: "scheduled_for", Value: 1},
		},
		Options: options.Index().SetName("idx_status_scheduled_for"),
	}

	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{uniqueModel, statusModel})
	return err
}

// CreateJobs menyimpan daftar tugas pengingat ke MongoDB.
func (r *reminderRepository) CreateJobs(ctx context.Context, jobs []models.ReminderJob) error {
	if len(jobs) == 0 {
		return nil
	}

	for _, job := range jobs {
		now := time.Now()
		if job.CreatedAt.IsZero() {
			job.CreatedAt = now
		}
		job.UpdatedAt = now
		if job.Status == "" {
			job.Status = models.ReminderStatusScheduled
		}
		if job.MaxRetries == 0 {
			job.MaxRetries = 3
		}

		// InsertOne per job agar jika salah satu duplikat, yang lain tetap tersimpan
		_, err := r.coll.InsertOne(ctx, job)
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to insert reminder job: %w", err)
		}
	}

	return nil
}

// RescheduleJobs refreshes pending reminder documents and keeps already-sent
// reminders immutable.
func (r *reminderRepository) RescheduleJobs(ctx context.Context, bookingID uint, jobs []models.ReminderJob) error {
	types := make([]string, 0, len(jobs))
	now := time.Now()
	for _, job := range jobs {
		if job.CreatedAt.IsZero() {
			job.CreatedAt = now
		}
		types = append(types, job.ReminderType)
		set := bson.M{
			"booking_id":    bookingID,
			"reminder_type": job.ReminderType,
			"booking_code":  job.BookingCode,
			"user_id":       job.UserID,
			"user_name":     job.UserName,
			"user_email":    job.UserEmail,
			"user_phone":    job.UserPhone,
			"field_id":      job.FieldID,
			"field_name":    job.FieldName,
			"booking_date":  job.BookingDate,
			"start_time":    job.StartTime,
			"end_time":      job.EndTime,
			"play_time":     job.PlayTime,
			"scheduled_for": job.ScheduledFor,
			"status":        models.ReminderStatusScheduled,
			"retry_count":   0,
			"max_retries":   3,
			"error_message": "",
			"updated_at":    now,
		}
		filter := bson.M{
			"booking_id":    bookingID,
			"reminder_type": job.ReminderType,
			"status":        bson.M{"$ne": models.ReminderStatusProcessing},
		}
		update := bson.M{
			"$set":         set,
			"$unset":       bson.M{"sent_at": ""},
			"$setOnInsert": bson.M{"created_at": job.CreatedAt},
		}
		_, err := r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("failed to update reminder job for booking %d: %w", bookingID, err)
		}
	}

	filter := bson.M{
		"booking_id": bookingID,
		"status":     models.ReminderStatusScheduled,
	}
	if len(types) > 0 {
		filter["reminder_type"] = bson.M{"$nin": types}
	}
	_, err := r.coll.UpdateMany(ctx, filter, bson.M{"$set": bson.M{
		"status":     models.ReminderStatusCancelled,
		"updated_at": now,
	}})
	if err != nil {
		return fmt.Errorf("failed to cancel obsolete reminder jobs for booking %d: %w", bookingID, err)
	}
	return nil
}

// FindDueJobs mengambil daftar tugas pengingat yang sudah jatuh tempo dan siap dikirim.
func (r *reminderRepository) FindDueJobs(ctx context.Context, now time.Time, limit int64) ([]models.ReminderJob, error) {
	if limit <= 0 {
		limit = 50
	}

	filter := bson.M{
		"status":        models.ReminderStatusScheduled,
		"scheduled_for": bson.M{"$lte": now},
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "scheduled_for", Value: 1}}).
		SetLimit(limit)

	cursor, err := r.coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.ReminderJob
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// UpdateStatus memperbarui status job pengingat.
func (r *reminderRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status string, sentAt *time.Time, errMsg string) error {
	update := bson.M{
		"$set": bson.M{
			"status":        status,
			"error_message": errMsg,
			"updated_at":    time.Now(),
		},
	}
	if sentAt != nil {
		update["$set"].(bson.M)["sent_at"] = sentAt
	}

	res, err := r.coll.UpdateByID(ctx, id, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("reminder job not found")
	}
	return nil
}

// IncrementRetry menambah jumlah percobaan dan menandai FAILED jika sudah melebihi batas.
func (r *reminderRepository) IncrementRetry(ctx context.Context, id bson.ObjectID, errMsg string) error {
	var job models.ReminderJob
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&job); err != nil {
		return err
	}

	nextCount := job.RetryCount + 1
	nextStatus := models.ReminderStatusScheduled
	if nextCount >= job.MaxRetries {
		nextStatus = models.ReminderStatusFailed
	}

	update := bson.M{
		"$set": bson.M{
			"retry_count":   nextCount,
			"status":        nextStatus,
			"error_message": errMsg,
			"updated_at":    time.Now(),
		},
	}

	_, err := r.coll.UpdateByID(ctx, id, update)
	return err
}

// CancelByBookingID membatalkan semua pengingat terjadwal untuk booking tertentu.
func (r *reminderRepository) CancelByBookingID(ctx context.Context, bookingID uint) error {
	filter := bson.M{
		"booking_id": bookingID,
		"status":     models.ReminderStatusScheduled,
	}

	update := bson.M{
		"$set": bson.M{
			"status":     models.ReminderStatusCancelled,
			"updated_at": time.Now(),
		},
	}

	_, err := r.coll.UpdateMany(ctx, filter, update)
	return err
}

// FindAll mengambil daftar log pengingat (untuk dashboard admin).
func (r *reminderRepository) FindAll(ctx context.Context, page, limit int, status string) ([]models.ReminderJob, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	findOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := r.coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []models.ReminderJob
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	return results, total, nil
}
