package dto

import "time"



type RegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Phone    string `json:"phone" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	User  UserResponse `json:"user"`
}

// NewUserResponse memetakan model user menjadi response.
func NewUserResponse(user *UserResponse) UserResponse {
	return UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Phone:    user.Phone,
		Role:     string(user.Role),
		CreatedAt: user.CreatedAt,
	}
}

// NewUserResponses memetakan daftar user menjadi daftar response.
func NewUserResponses(users []UserResponse) []UserResponse {
	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = NewUserResponse(&user)
	}
	return responses
}