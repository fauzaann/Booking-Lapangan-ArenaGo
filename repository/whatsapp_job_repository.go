package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"Booking-Lapangan/models"
)

type WhatsAppJobRepository interface {
	Create(ctx context.Context, job *models.WhatsAppJob) error
	FindDue(ctx context.Context, now time.Time, limit int64) ([]models.WhatsAppJob, error)
	Claim(ctx context.Context, id bson.ObjectID, now time.Time) (bool, error)
	MarkSent(ctx context.Context, id bson.ObjectID, sentAt time.Time) error
	Cancel(ctx context.Context, id bson.ObjectID, reason string) error
	Retry(ctx context.Context, id bson.ObjectID, reason string) error
	EnsureIndexes(ctx context.Context) error
}

type whatsAppJobRepository struct {
	coll *mongo.Collection
}

func NewWhatsAppJobRepository(db *mongo.Database) WhatsAppJobRepository {
	return &whatsAppJobRepository{coll: db.Collection("whatsapp_outbox")}
}

func (r *whatsAppJobRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "scheduled_for", Value: 1}},
			Options: options.Index().SetName("idx_whatsapp_outbox_due"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "updated_at", Value: 1}},
			Options: options.Index().SetName("idx_whatsapp_outbox_lease"),
		},
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_whatsapp_event_id_unique").SetPartialFilterExpression(bson.M{"event_id": bson.M{"$type": "string"}}),
		},
	})
	return err
}

func (r *whatsAppJobRepository) Create(ctx context.Context, job *models.WhatsAppJob) error {
	now := time.Now()
	if job.ScheduledFor.IsZero() {
		job.ScheduledFor = now
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now
	if job.Status == "" {
		job.Status = models.WhatsAppJobStatusScheduled
	}
	if job.MaxRetries <= 0 {
		job.MaxRetries = 3
	}
	result, err := r.coll.InsertOne(ctx, job)
	if err != nil {
		if job.EventID != "" && mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	}
	if id, ok := result.InsertedID.(bson.ObjectID); ok {
		job.ID = id
	}
	return nil
}

func (r *whatsAppJobRepository) FindDue(ctx context.Context, now time.Time, limit int64) ([]models.WhatsAppJob, error) {
	if limit <= 0 {
		limit = 50
	}
	staleBefore := now.Add(-5 * time.Minute)
	cursor, err := r.coll.Find(ctx, bson.M{
		"$or": bson.A{
			bson.M{"status": models.WhatsAppJobStatusScheduled, "scheduled_for": bson.M{"$lte": now}},
			bson.M{"status": models.WhatsAppJobStatusProcessing, "updated_at": bson.M{"$lte": staleBefore}},
		},
	}, options.Find().SetSort(bson.D{{Key: "scheduled_for", Value: 1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []models.WhatsAppJob
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *whatsAppJobRepository) Claim(ctx context.Context, id bson.ObjectID, now time.Time) (bool, error) {
	staleBefore := now.Add(-5 * time.Minute)
	result, err := r.coll.UpdateOne(ctx, bson.M{
		"_id": id,
		"$or": bson.A{
			bson.M{"status": models.WhatsAppJobStatusScheduled, "scheduled_for": bson.M{"$lte": now}},
			bson.M{"status": models.WhatsAppJobStatusProcessing, "updated_at": bson.M{"$lte": staleBefore}},
		},
	}, bson.M{"$set": bson.M{
		"status":     models.WhatsAppJobStatusProcessing,
		"updated_at": now,
	}})
	if err != nil {
		return false, err
	}
	return result.MatchedCount == 1, nil
}

func (r *whatsAppJobRepository) MarkSent(ctx context.Context, id bson.ObjectID, sentAt time.Time) error {
	return r.updateProcessing(ctx, id, bson.M{
		"status":        models.WhatsAppJobStatusSent,
		"sent_at":       sentAt,
		"updated_at":    sentAt,
		"error_message": "",
	})
}

func (r *whatsAppJobRepository) Cancel(ctx context.Context, id bson.ObjectID, reason string) error {
	return r.updateProcessing(ctx, id, bson.M{
		"status":        models.WhatsAppJobStatusCancelled,
		"error_message": reason,
		"updated_at":    time.Now(),
	})
}

func (r *whatsAppJobRepository) Retry(ctx context.Context, id bson.ObjectID, reason string) error {
	var job models.WhatsAppJob
	if err := r.coll.FindOne(ctx, bson.M{"_id": id, "status": models.WhatsAppJobStatusProcessing}).Decode(&job); err != nil {
		return err
	}
	nextCount := job.RetryCount + 1
	nextStatus := models.WhatsAppJobStatusScheduled
	if nextCount >= job.MaxRetries {
		nextStatus = models.WhatsAppJobStatusFailed
	}
	now := time.Now()
	return r.updateProcessing(ctx, id, bson.M{
		"status":        nextStatus,
		"retry_count":   nextCount,
		"error_message": reason,
		"scheduled_for": now.Add(time.Duration(nextCount) * 30 * time.Second),
		"updated_at":    now,
	})
}

func (r *whatsAppJobRepository) updateProcessing(ctx context.Context, id bson.ObjectID, set bson.M) error {
	result, err := r.coll.UpdateOne(ctx, bson.M{
		"_id":    id,
		"status": models.WhatsAppJobStatusProcessing,
	}, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("WhatsApp job not found or is no longer processing")
	}
	return nil
}
