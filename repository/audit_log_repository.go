package repository

import (
	"context"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
)

type AuditLogRepository interface {
	Create(ctx context.Context, logEntry *models.AuditLog) error
	FindAll(ctx context.Context, params ListParams) ([]models.AuditLog, int64, error)
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

// FindAll mengambil audit log terbaru dengan pagination.
func (r *auditLogRepository) FindAll(ctx context.Context, params ListParams) ([]models.AuditLog, int64, error) {
	params = params.Normalize()
	query := r.db.WithContext(ctx).Model(&models.AuditLog{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translate(err, "audit logs could not be retrieved")
	}

	var logs []models.AuditLog
	if err := query.Order("created_at DESC, id DESC").Limit(params.Limit).Offset(params.Offset()).Find(&logs).Error; err != nil {
		return nil, 0, translate(err, "audit logs could not be retrieved")
	}
	return logs, total, nil
}
