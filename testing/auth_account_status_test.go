package controllers_test

import (
	"context"
	"net/http"
	"testing"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/password"
	"Booking-Lapangan/repository"
)

type loginAccountRepository struct {
	repository.UserRepository
	user *models.User
}

func (r *loginAccountRepository) FindByEmail(context.Context, string) (*models.User, error) {
	copy := *r.user
	return &copy, nil
}

func TestLoginRejectsInactiveAccountWithGenericCredentialError(t *testing.T) {
	hashed, err := password.Hash("correct-password")
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}
	users := &loginAccountRepository{user: &models.User{
		ID: 7, Email: "inactive@example.com", Password: hashed,
		Role: models.RoleUser, IsActive: false,
	}}
	auth := controllers.NewAuthController(users, nil)

	_, err = auth.Login(context.Background(), dto.LoginRequest{
		Email: "inactive@example.com", Password: "correct-password",
	})
	if err == nil || apperror.From(err).Status != http.StatusUnauthorized {
		t.Fatalf("Login() error = %v, want unauthorized", err)
	}
}
