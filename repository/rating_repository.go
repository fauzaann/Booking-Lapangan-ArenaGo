package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
)

type RatingSummary struct {
	Average float64
	Total   int64
}

type FieldRatingSummary struct {
	FieldID uint
	RatingSummary
}

type RatingRepository interface {
	Create(ctx context.Context, rating *models.Rating) error
	FindByID(ctx context.Context, id uint) (*models.Rating, error)
	FindByBookingID(ctx context.Context, bookingID uint) (*models.Rating, error)
	ListPublicByFieldID(ctx context.Context, fieldID uint, params ListParams) ([]models.Rating, int64, error)
	SummaryByFieldID(ctx context.Context, fieldID uint) (RatingSummary, error)
	SummariesByFieldIDs(ctx context.Context, fieldIDs []uint) (map[uint]RatingSummary, error)
	ListAdmin(ctx context.Context, visible *bool, params ListParams) ([]models.Rating, int64, error)
	SetVisibility(ctx context.Context, id uint, visible bool) error
}

type ratingRepository struct {
	db *gorm.DB
}

func NewRatingRepository(db *gorm.DB) RatingRepository {
	return &ratingRepository{db: db}
}

func (r *ratingRepository) Create(ctx context.Context, rating *models.Rating) error {
	err := r.db.WithContext(ctx).Create(rating).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperror.Conflict("this booking already has a rating")
	}
	return translate(err, "rating not found")
}

func (r *ratingRepository) FindByID(ctx context.Context, id uint) (*models.Rating, error) {
	var rating models.Rating
	err := r.db.WithContext(ctx).Preload("Field").First(&rating, id).Error
	if err != nil {
		return nil, translate(err, "rating not found")
	}
	return &rating, nil
}

func (r *ratingRepository) FindByBookingID(ctx context.Context, bookingID uint) (*models.Rating, error) {
	var rating models.Rating
	if err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&rating).Error; err != nil {
		return nil, translate(err, "rating not found")
	}
	return &rating, nil
}

func (r *ratingRepository) ListPublicByFieldID(ctx context.Context, fieldID uint, params ListParams) ([]models.Rating, int64, error) {
	params = params.Normalize()
	query := r.db.WithContext(ctx).Model(&models.Rating{}).
		Where("field_id = ? AND is_visible = ?", fieldID, true)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translate(err, "rating not found")
	}

	var ratings []models.Rating
	err := query.Order("created_at DESC, id DESC").
		Limit(params.Limit).Offset(params.Offset()).Find(&ratings).Error
	if err != nil {
		return nil, 0, translate(err, "rating not found")
	}
	return ratings, total, nil
}

func (r *ratingRepository) SummaryByFieldID(ctx context.Context, fieldID uint) (RatingSummary, error) {
	var summary RatingSummary
	err := r.db.WithContext(ctx).Model(&models.Rating{}).
		Select("COALESCE(AVG(rating), 0) AS average, COUNT(*) AS total").
		Where("field_id = ? AND is_visible = ?", fieldID, true).
		Scan(&summary).Error
	if err != nil {
		return RatingSummary{}, translate(err, "rating not found")
	}
	return summary, nil
}

func (r *ratingRepository) SummariesByFieldIDs(ctx context.Context, fieldIDs []uint) (map[uint]RatingSummary, error) {
	summaries := make(map[uint]RatingSummary, len(fieldIDs))
	if len(fieldIDs) == 0 {
		return summaries, nil
	}

	var rows []FieldRatingSummary
	err := r.db.WithContext(ctx).Model(&models.Rating{}).
		Select("field_id, AVG(rating) AS average, COUNT(*) AS total").
		Where("field_id IN ? AND is_visible = ?", fieldIDs, true).
		Group("field_id").
		Scan(&rows).Error
	if err != nil {
		return nil, translate(err, "rating not found")
	}
	for _, row := range rows {
		summaries[row.FieldID] = row.RatingSummary
	}
	return summaries, nil
}

func (r *ratingRepository) ListAdmin(ctx context.Context, visible *bool, params ListParams) ([]models.Rating, int64, error) {
	params = params.Normalize()
	query := r.db.WithContext(ctx).Model(&models.Rating{})
	if visible != nil {
		query = query.Where("is_visible = ?", *visible)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translate(err, "rating not found")
	}

	var ratings []models.Rating
	err := query.Order("created_at DESC, id DESC").
		Limit(params.Limit).Offset(params.Offset()).
		Preload("Field").Find(&ratings).Error
	if err != nil {
		return nil, 0, translate(err, "rating not found")
	}
	return ratings, total, nil
}

func (r *ratingRepository) SetVisibility(ctx context.Context, id uint, visible bool) error {
	result := r.db.WithContext(ctx).Model(&models.Rating{}).
		Where("id = ?", id).Update("is_visible", visible)
	if result.Error != nil {
		return translate(result.Error, "rating not found")
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := r.db.WithContext(ctx).Model(&models.Rating{}).Where("id = ?", id).Count(&count).Error; err != nil {
			return translate(err, "rating not found")
		}
		if count == 0 {
			return apperror.NotFound("rating not found")
		}
	}
	return nil
}
