package repository

import (
	"context"

	"Booking-Lapangan/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScheduleRepository interface {
	FindByFieldID(ctx context.Context, fieldID uint) ([]models.Schedule, error)
	FindByFieldAndDay(ctx context.Context, fieldID uint, day models.Day) (*models.Schedule, error)
	Upsert(ctx context.Context, schedules []models.Schedule) error
	DeleteByFieldID(ctx context.Context, fieldID uint) error
}

type scheduleRepository struct {
	db *gorm.DB
}

func NewScheduleRepository(db *gorm.DB) ScheduleRepository {
	return &scheduleRepository{db: db}
}

func (r *scheduleRepository) FindByFieldID(ctx context.Context, fieldID uint) ([]models.Schedule, error) {
	var schedules []models.Schedule
	err := r.db.WithContext(ctx).
		Where("field_id = ?", fieldID).
		Order("id ASC").
		Find(&schedules).Error
	if err != nil {
		return nil, translate(err, "schedule not found")
	}
	return schedules, nil
}

func (r *scheduleRepository) FindByFieldAndDay(ctx context.Context, fieldID uint, day models.Day) (*models.Schedule, error) {
	var schedule models.Schedule
	err := r.db.WithContext(ctx).
		Where("field_id = ? AND day = ?", fieldID, day).
		First(&schedule).Error
	if err != nil {
		return nil, translate(err, "schedule not found for the selected day")
	}
	return &schedule, nil
}

// Upsert menyimpan jadwal dan menimpa baris yang sudah ada pada
// kombinasi (field_id, day) yang sama.
func (r *scheduleRepository) Upsert(ctx context.Context, schedules []models.Schedule) error {
	if len(schedules) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "field_id"}, {Name: "day"}},
		DoUpdates: clause.AssignmentColumns([]string{"open_time", "close_time", "is_active", "updated_at"}),
	}).Create(&schedules).Error
	return translate(err, "schedule not found")
}

func (r *scheduleRepository) DeleteByFieldID(ctx context.Context, fieldID uint) error {
	err := r.db.WithContext(ctx).Where("field_id = ?", fieldID).Delete(&models.Schedule{}).Error
	return translate(err, "schedule not found")
}
