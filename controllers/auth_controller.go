package controllers

import (
	"context"

	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

// AuthController struct mewakili pengontrol untuk otentikasi pengguna
type AuthController struct {
	userRepo repository.UserRepository
}

func NewAuthController(userRepo repository.UserRepository) *AuthController {
	return &AuthController{userRepo: userRepo}
}

func (c *AuthController) Register(ctx context.Context, username, email, password string) (*models.User, error) {
	// Periksa apakah pengguna dengan email yang sama sudah ada
	exists, err := c.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &models.AppError{Message: "Email already exists", Code: 400}
	}

	// Buat pengguna baru
	user := &models.User{
		Username: username,
		Email:    email,
		Password: password,
		Role:     models.RoleUser,
	}

	if !user.IsValid() {
		return nil, &models.AppError{Message: "Invalid user data", Code: 400}
	}

	if err = c.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (c *AuthController) Login(ctx context.Context, email, password string) (*models.User, error) {
	user, err := c.userRepo.GetUserByUsernameEmail(ctx, email)
	if err != nil {
		return nil, &models.AppError{Message: "Invalid email or password", Code: 401}
	}

	if user.Password != password {
		return nil, &models.AppError{Message: "Invalid email or password", Code: 401}
	}

	return user, nil
}

func (c *AuthController) Profile(ctx context.Context, userID uint) (*models.User, error) {
	user, err := c.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, &models.AppError{Message: "User not found", Code: 404}
	}
	return user, nil
}

func (c *AuthController) GetProfile(ctx context.Context, userID uint) (*models.User, error) {
	return c.Profile(ctx, userID)
}

func (c *AuthController) Logout(ctx context.Context) error {
	return nil
}
