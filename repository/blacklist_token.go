package repository

import (
	"Booking-Lapangan/models"
	"gorm.io/gorm"
)

type BlacklistTokenRepository interface {
	Create(token *models.BlacklistToken) error
	IsBlacklisted(token string) (bool, error)
}

type blacklistTokenRepositoryImpl struct {
	db *gorm.DB
}

func NewBlacklistTokenRepository(db *gorm.DB) BlacklistTokenRepository {
	return &blacklistTokenRepositoryImpl{db: db}
}

func (r *blacklistTokenRepositoryImpl) Create(token *models.BlacklistToken) error {
	return r.db.Create(token).Error
}

func (r *blacklistTokenRepositoryImpl) IsBlacklisted(token string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.BlacklistToken{}).Where("token = ?", token).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}