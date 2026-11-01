package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
)

type AuditLogFilter struct {
	ListParams
	ActorEmail string
	Path       string
	Method     string
	StatusCode int
	From       *time.Time
	To         *time.Time
}

type AuditLogRepository interface {
	Create(ctx context.Context, logEntry *models.AuditLog) error
	FindAll(ctx context.Context, filter AuditLogFilter) ([]models.AuditLog, int64, error)
}

type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository membuat repository audit log berbasis GORM.
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

// Create menyimpan satu aktivitas request ke database.
func (r *auditLogRepository) Create(ctx context.Context, logEntry *models.AuditLog) error {
	return translate(r.db.WithContext(ctx).Create(logEntry).Error, "audit log could not be created")
}

// FindAll mengambil audit log terbaru dengan filter dan pagination.
func (r *auditLogRepository) FindAll(ctx context.Context, filter AuditLogFilter) ([]models.AuditLog, int64, error) {
	filter.ListParams = filter.ListParams.Normalize()
	query := r.db.WithContext(ctx).Model(&models.AuditLog{})
	if filter.ActorEmail != "" {
		query = query.Where("actor_email ILIKE ?", "%"+filter.ActorEmail+"%")
	}
	if filter.Path != "" {
		query = query.Where("path ILIKE ?", "%"+filter.Path+"%")
	}
	if filter.Method != "" {
		query = query.Where("method = ?", filter.Method)
	}
	if filter.StatusCode > 0 {
		query = query.Where("status_code = ?", filter.StatusCode)
	}
	if filter.From != nil {
		query = query.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("created_at < ?", *filter.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translate(err, "audit logs could not be retrieved")
	}

	var logs []models.AuditLog
	if err := query.Order("created_at DESC, id DESC").Limit(filter.Limit).Offset(filter.Offset()).Find(&logs).Error; err != nil {
		return nil, 0, translate(err, "audit logs could not be retrieved")
	}
	return logs, total, nil
}
