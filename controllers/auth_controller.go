package controllers

import (
	"context"
	"strings"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/jwt"
	"Booking-Lapangan/pkg/password"
	"Booking-Lapangan/repository"
)

// AuthController menangani registrasi, login, dan profil user.
type AuthController interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	Profile(ctx context.Context, userID uint) (*dto.UserResponse, error)
}

type authController struct {
	users  repository.UserRepository
	tokens jwt.Manager
}

// NewAuthController membuat AuthController.
func NewAuthController(users repository.UserRepository, tokens jwt.Manager) AuthController {
	return &authController{users: users, tokens: tokens}
}

func (c *authController) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	email := normalizeEmail(req.Email)

	exists, err := c.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.Conflict("email is already registered")
	}

	hashed, err := password.Hash(req.Password)
	if err != nil {
		return nil, apperror.Internal("failed to secure password", err)
	}

	user := &models.User{
		Name:     strings.TrimSpace(req.Name),
		Email:    email,
		Password: hashed,
		Phone:    strings.TrimSpace(req.Phone),
		Role:     models.RoleUser,
	}

	if err := c.users.Create(ctx, user); err != nil {
		if apperror.IsConflict(err) {
			return nil, apperror.Conflict("email is already registered")
		}
		return nil, err
	}

	return c.issueToken(user)
}

func (c *authController) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := c.users.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if apperror.IsNotFound(err) {
			// Pesan sengaja sama dengan password salah agar email tidak dapat dienumerasi.
			return nil, apperror.Unauthorized("invalid email or password")
		}
		return nil, err
	}

	if !password.Verify(user.Password, req.Password) {
		return nil, apperror.Unauthorized("invalid email or password")
	}

	return c.issueToken(user)
}

func (c *authController) Profile(ctx context.Context, userID uint) (*dto.UserResponse, error) {
	user, err := c.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	response := dto.NewUserResponse(*user)
	return &response, nil
}

func (c *authController) issueToken(user *models.User) (*dto.AuthResponse, error) {
	token, expiresAt, err := c.tokens.Generate(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, apperror.Internal("failed to generate token", err)
	}
	return &dto.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      dto.NewUserResponse(*user),
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
