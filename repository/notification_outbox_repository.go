package repository

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"Booking-Lapangan/models"
)

type NotificationOutboxRepository interface {
	Create(ctx context.Context, event *models.NotificationEvent) error
	ClaimPending(ctx context.Context, now time.Time, limit int) ([]models.NotificationEvent, error)
	MarkProcessed(ctx context.Context, id uint, now time.Time) error
	Retry(ctx context.Context, event models.NotificationEvent, cause error, now time.Time) error
}

type notificationOutboxRepository struct {
	db *gorm.DB
}

func NewNotificationOutboxRepository(db *gorm.DB) NotificationOutboxRepository {
	return &notificationOutboxRepository{db: db}
}

func (r *notificationOutboxRepository) Create(ctx context.Context, event *models.NotificationEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.NewString()
	}
	if event.ContractVersion <= 0 {
		return errors.New("notification event contract version is required")
	}
	if len(event.Payload) == 0 || !json.Valid([]byte(event.Payload)) {
		return errors.New("notification event payload must be valid JSON")
	}
	if event.Status == "" {
		event.Status = models.NotificationEventPending
	}
	if event.MaxAttempts <= 0 {
		event.MaxAttempts = 5
	}
	if event.AvailableAt.IsZero() {
		event.AvailableAt = time.Now()
	}
	return translate(r.db.WithContext(ctx).Create(event).Error, "failed to create notification outbox event")
}

func (r *notificationOutboxRepository) ClaimPending(ctx context.Context, now time.Time, limit int) ([]models.NotificationEvent, error) {
	if limit <= 0 {
		return nil, nil
	}

	var events []models.NotificationEvent
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where(
				"(status = ? AND available_at <= ?) OR (status = ? AND locked_at <= ?)",
				models.NotificationEventPending, now,
				models.NotificationEventProcessing, now.Add(-2*time.Minute),
			).
			Order("available_at ASC, id ASC").
			Limit(limit).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Find(&events).Error; err != nil {
			return err
		}

		for i := range events {
			events[i].Status = models.NotificationEventProcessing
			events[i].Attempts++
			events[i].LockedAt = &now
			if err := tx.Model(&models.NotificationEvent{}).
				Where("id = ?", events[i].ID).
				Updates(map[string]any{
					"status":     events[i].Status,
					"attempts":   events[i].Attempts,
					"locked_at":  now,
					"last_error": "",
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, translate(err, "failed to claim notification outbox events")
	}
	return events, nil
}

func (r *notificationOutboxRepository) MarkProcessed(ctx context.Context, id uint, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.NotificationEvent{}).Where("id = ?", id).Updates(map[string]any{
		"status":       models.NotificationEventProcessed,
		"processed_at": now,
		"locked_at":    nil,
		"last_error":   "",
		"payload":      "{}",
		"payment_url":  "",
		"expires_at":   nil,
	})
	if result.Error != nil {
		return translate(result.Error, "failed to mark notification event processed")
	}
	if result.RowsAffected == 0 {
		return errors.New("notification outbox event not found")
	}
	return nil
}

func (r *notificationOutboxRepository) Retry(ctx context.Context, event models.NotificationEvent, cause error, now time.Time) error {
	status := models.NotificationEventPending
	availableAt := now.Add(time.Duration(math.Min(math.Pow(2, float64(event.Attempts)), 30)) * time.Minute)
	if event.Attempts >= event.MaxAttempts {
		status = models.NotificationEventFailed
	}
	lastError := ""
	if cause != nil {
		lastError = strings.TrimSpace(cause.Error())
	}
	result := r.db.WithContext(ctx).Model(&models.NotificationEvent{}).Where("id = ?", event.ID).Updates(map[string]any{
		"status":       status,
		"available_at": availableAt,
		"locked_at":    nil,
		"last_error":   lastError,
	})
	if result.Error != nil {
		return translate(result.Error, "failed to retry notification event")
	}
	if result.RowsAffected == 0 {
		return errors.New("notification outbox event not found")
	}
	return nil
}
