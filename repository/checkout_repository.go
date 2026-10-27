package repository

import (
	"context"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
)

type CheckoutRepository interface {
	Create(ctx context.Context, checkout *models.Checkout) error
	Update(ctx context.Context, checkout *models.Checkout) error
	FindByID(ctx context.Context, id uint) (*models.Checkout, error)
	FindByExternalID(ctx context.Context, externalID string) (*models.Checkout, error)
}

type checkoutRepository struct {
	db *gorm.DB
}

func NewCheckoutRepository(db *gorm.DB) CheckoutRepository {
	return &checkoutRepository{db: db}
}

func (r *checkoutRepository) Create(ctx context.Context, checkout *models.Checkout) error {
	return translate(r.db.WithContext(ctx).Create(checkout).Error, "checkout not found")
}

func (r *checkoutRepository) Update(ctx context.Context, checkout *models.Checkout) error {
	return translate(r.db.WithContext(ctx).Save(checkout).Error, "checkout not found")
}

func (r *checkoutRepository) FindByID(ctx context.Context, id uint) (*models.Checkout, error) {
	var checkout models.Checkout
	if err := r.db.WithContext(ctx).First(&checkout, id).Error; err != nil {
		return nil, translate(err, "checkout not found")
	}
	return &checkout, nil
}

func (r *checkoutRepository) FindByExternalID(ctx context.Context, externalID string) (*models.Checkout, error) {
	var checkout models.Checkout
	err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&checkout).Error
	if err != nil {
		return nil, translate(err, "checkout not found")
	}
	return &checkout, nil
}
