package repository

import (
	"Booking-Lapangan/models"
	"context"

	"gorm.io/gorm"
)

type FieldFilter struct {
	ListParams
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Location string   `json:"location"`
	Search   string   `json:"search"`
	PriceMax *float64 `json:"price_max"`
	PriceMin *float64 `json:"price_min"`
}

type FieldRepository interface {
	Create(ctx context.Context, field *models.Field) error
	Update(ctx context.Context, field *models.Field) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.Field, error)
	FindDetailByID(ctx context.Context, id uint) (*models.Field, error)
	FindAll(ctx context.Context, filter FieldFilter) ([]models.Field, int64, error)
	Count(ctx context.Context) (int64, error)
}

type fieldRepository struct {
	db *gorm.DB
}

// NewFieldRepository membuat implementasi FieldRepository berbasis GORM.
func NewFieldRepository(db *gorm.DB) FieldRepository {
	return &fieldRepository{db: db}
}

func (r *fieldRepository) Create(ctx context.Context, field *models.Field) error {
	return translate(r.db.WithContext(ctx).Create(field).Error, "field not found")
}

func (r *fieldRepository) Update(ctx context.Context, field *models.Field) error {
	return translate(r.db.WithContext(ctx).Save(field).Error, "field not found")
}

func (r *fieldRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Field{}, id)
	if result.Error != nil {
		return translate(result.Error, "field not found")
	}
	if result.RowsAffected == 0 {
		return translate(gorm.ErrRecordNotFound, "field not found")
	}
	return nil
}

func (r *fieldRepository) FindByID(ctx context.Context, id uint) (*models.Field, error) {
	var field models.Field
	if err := r.db.WithContext(ctx).First(&field, id).Error; err != nil {
		return nil, translate(err, "field not found")
	}
	return &field, nil
}

func (r *fieldRepository) FindDetailByID(ctx context.Context, id uint) (*models.Field, error) {
	var field models.Field
	err := r.db.WithContext(ctx).
		Preload("Schedules", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		First(&field, id).Error
	if err != nil {
		return nil, translate(err, "field not found")
	}
	return &field, nil
}

func (r *fieldRepository) FindAll(ctx context.Context, filter FieldFilter) ([]models.Field, int64, error) {
	filter.ListParams = filter.ListParams.Normalize()

	query := r.db.WithContext(ctx).Model(&models.Field{})

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Location != "" {
		query = query.Where("location ILIKE ?", "%"+filter.Location+"%")
	}
	if filter.Search != "" {
		query = query.Where("name ILIKE ?", "%"+filter.Search+"%")
	}
	if filter.PriceMin != nil {
		query = query.Where("price_per_hour >= ?", *filter.PriceMin)
	}
	if filter.PriceMax != nil {
		query = query.Where("price_per_hour <= ?", *filter.PriceMax)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translate(err, "field not found")
	}

	var fields []models.Field
	err := query.Order("id ASC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&fields).Error
	if err != nil {
		return nil, 0, translate(err, "field not found")
	}
	return fields, total, nil
}

func (r *fieldRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&models.Field{}).Count(&total).Error
	return total, translate(err, "field not found")
}
