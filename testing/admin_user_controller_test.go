package controllers_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/repository"
)

type adminUserRepository struct {
	repository.UserRepository
	users map[uint]*models.User
}

func (r *adminUserRepository) FindByID(_ context.Context, id uint) (*models.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, apperror.NotFound("user not found")
	}
	copy := *user
	return &copy, nil
}

func (r *adminUserRepository) FindAll(_ context.Context, params repository.ListParams) ([]models.User, int64, error) {
	params = params.Normalize()
	users := make([]models.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, *user)
	}
	return users, int64(len(users)), nil
}

func (r *adminUserRepository) Update(_ context.Context, user *models.User) error {
	copy := *user
	r.users[user.ID] = &copy
	return nil
}

type adminUserUnitOfWork struct {
	repository.UnitOfWork
	users repository.UserRepository
}

func (u *adminUserUnitOfWork) User() repository.UserRepository {
	return u.users
}

func (u *adminUserUnitOfWork) Atomic(ctx context.Context, fn func(repository.Registry) error) error {
	return fn(u)
}

func TestAdminListUsersReturnsOnlyApprovedAccountFields(t *testing.T) {
	users := &adminUserRepository{users: map[uint]*models.User{
		1: {
			ID: 1, Name: "Ayu", Email: "ayu@example.com", Phone: "+628123456789",
			Password: "must-not-be-returned", WhatsAppOptIn: true, IsActive: false,
			Role: models.RoleUser, CreatedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		},
	}}
	admin := controllers.NewAdminController(&adminUserUnitOfWork{users: users})

	result, total, err := admin.ListUsers(context.Background(), dto.PaginationQuery{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if total != 1 || len(result) != 1 || result[0].Name != "Ayu" || result[0].IsActive {
		t.Fatalf("unexpected admin user response: total=%d users=%+v", total, result)
	}
	encoded, err := json.Marshal(result[0])
	if err != nil {
		t.Fatalf("marshal admin user response: %v", err)
	}
	for _, privateField := range []string{"phone", "password", "whatsapp_opt_in"} {
		if strings.Contains(string(encoded), `"`+privateField+`"`) {
			t.Errorf("admin user response unexpectedly exposes %s: %s", privateField, encoded)
		}
	}
}

func TestAdminCanChangeUserStatusButCannotDeactivateAdminsOrSelf(t *testing.T) {
	users := &adminUserRepository{users: map[uint]*models.User{
		1: {ID: 1, Name: "Admin", Email: "admin@example.com", Role: models.RoleAdmin, IsActive: true},
		2: {ID: 2, Name: "User", Email: "user@example.com", Role: models.RoleUser, IsActive: true},
		3: {ID: 3, Name: "Second Admin", Email: "admin2@example.com", Role: models.RoleAdmin, IsActive: true},
	}}
	admin := controllers.NewAdminController(&adminUserUnitOfWork{users: users})

	updated, err := admin.UpdateUserStatus(context.Background(), 1, 2, false)
	if err != nil {
		t.Fatalf("deactivate user: %v", err)
	}
	if updated.IsActive || users.users[2].IsActive {
		t.Fatal("user account was not deactivated")
	}

	if _, err := admin.UpdateUserStatus(context.Background(), 1, 3, false); !apperror.IsConflict(err) {
		t.Errorf("deactivate another admin error = %v, want conflict", err)
	}
	if _, err := admin.UpdateUserStatus(context.Background(), 1, 1, false); !apperror.IsConflict(err) {
		t.Errorf("deactivate own account error = %v, want conflict", err)
	}
}
