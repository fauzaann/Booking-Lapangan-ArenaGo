package dto

import (
	"time"

	"Booking-Lapangan/models"
)

// RegisterRequest adalah payload pendaftaran user baru.
type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=100"`
	Email    string `json:"email" validate:"required,email,max=150"`
	Password string `json:"password" validate:"required,min=6,max=72"`
	Phone    string `json:"phone" validate:"required,min=8,max=20"`
}

// LoginRequest adalah payload login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UserResponse adalah representasi user yang aman dikirim ke client.
type UserResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse adalah hasil register/login.
type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

// NewUserResponse memetakan model user menjadi response.
func NewUserResponse(user models.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt,
	}
}

// NewUserResponses memetakan daftar user menjadi daftar response.
func NewUserResponses(users []models.User) []UserResponse {
	result := make([]UserResponse, 0, len(users))
	for _, user := range users {
		result = append(result, NewUserResponse(user))
	}
	return result
}
