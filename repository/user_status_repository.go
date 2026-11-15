package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
)

type UserStatusRepository interface {
	IsActive(ctx context.Context, userID uint) (bool, error)
}

type userStatusRepository struct {
	db *gorm.DB
}

func NewUserStatusRepository(db *gorm.DB) UserStatusRepository {
	return &userStatusRepository{db: db}
}

func (r *userStatusRepository) IsActive(ctx context.Context, userID uint) (bool, error) {
	var account struct {
		IsActive bool
	}
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("is_active").
		Where("id = ?", userID).
		Take(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, translate(err, "user status not found")
	}
	return account.IsActive, nil
}
