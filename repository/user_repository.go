package repository

import (
	"context"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
)

// UserRepository adalah kontrak akses data user.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindAll(ctx context.Context, params ListParams) ([]models.User, int64, error)
	Count(ctx context.Context) (int64, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat implementasi UserRepository berbasis GORM.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	return translate(r.db.WithContext(ctx).Create(user).Error, "user not found")
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, translate(err, "user not found")
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, translate(err, "user not found")
	}
	return &user, nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, translate(err, "user not found")
	}
	return count > 0, nil
}

func (r *userRepository) FindAll(ctx context.Context, params ListParams) ([]models.User, int64, error) {
	params = params.Normalize()

	var (
		users []models.User
		total int64
	)

	query := r.db.WithContext(ctx).Model(&models.User{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translate(err, "user not found")
	}

	err := query.Order("id DESC").Limit(params.Limit).Offset(params.Offset()).Find(&users).Error
	if err != nil {
		return nil, 0, translate(err, "user not found")
	}
	return users, total, nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&total).Error
	return total, translate(err, "user not found")
}
