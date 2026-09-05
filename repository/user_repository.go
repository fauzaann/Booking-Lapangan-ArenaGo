package repository

import (
	"context"

	"Booking-Lapangan/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	GetUserByUsernameEmail(ctx context.Context, email string) (*models.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id uint) error
	FindAll(ctx context.Context, username string) ([]models.User, error)
	Count(ctx context.Context, user *models.User) (int64, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// CreateUser membuat pengguna baru dalam database
func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetUserByID mengambil pengguna berdasarkan ID dari database
func (r *userRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, translate(err, "User not found")
	}
	return &user, nil
}

// GetUserByUsernameEmail mengambil pengguna berdasarkan username atau email dari database
func (r *userRepository) GetUserByUsernameEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("username = ? OR email = ?", email, email).First(&user).Error; err != nil {
		return nil, translate(err, "User not found")
	}
	return &user, nil
}

// ExistsByEmail memeriksa apakah pengguna dengan email tertentu ada dalam database
func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, translate(err, "User not found")
	}
	return true, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) DeleteUser(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

func (r *userRepository) FindAll(ctx context.Context, username string) ([]models.User, error) {
	var users []models.User
	query := r.db.WithContext(ctx).Model(&models.User{})
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, translate(err, "Users not found")
	}

	return users, nil
}

func (r *userRepository) Count(ctx context.Context, user *models.User) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where(user).Count(&total).Error; err != nil {
		return 0, translate(err, "users not found")
	}
	return total, nil
}
