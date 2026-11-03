package controllers_test

import (
	"context"
	"testing"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

type notificationPreferenceUserRepo struct {
	repository.UserRepository
	user    *models.User
	updated bool
}

func (r *notificationPreferenceUserRepo) FindByID(context.Context, uint) (*models.User, error) {
	copy := *r.user
	return &copy, nil
}

func (r *notificationPreferenceUserRepo) Update(_ context.Context, user *models.User) error {
	r.user = user
	r.updated = true
	return nil
}

func TestUpdateWhatsAppPreferenceValidatesAndPersistsConsent(t *testing.T) {
	users := &notificationPreferenceUserRepo{user: &models.User{ID: 7, Phone: "0812-0000-0000"}}
	auth := controllers.NewAuthController(users, nil)
	phone := "0812-3456-7890"

	response, err := auth.UpdateWhatsAppPreference(context.Background(), 7, true, &phone)
	if err != nil {
		t.Fatalf("UpdateWhatsAppPreference() error = %v", err)
	}
	if !response.WhatsAppOptIn || response.Phone != "+6281234567890" || !users.updated {
		t.Fatalf("WhatsApp preference was not saved correctly: response=%+v user=%+v", response, users.user)
	}

	response, err = auth.UpdateWhatsAppPreference(context.Background(), 7, false, nil)
	if err != nil {
		t.Fatalf("disabling WhatsApp preference returned error: %v", err)
	}
	if response.WhatsAppOptIn {
		t.Fatal("WhatsApp consent remained enabled after opt-out")
	}
}

func TestUpdateWhatsAppPreferenceRejectsInvalidPhone(t *testing.T) {
	users := &notificationPreferenceUserRepo{user: &models.User{ID: 7, Phone: "not-a-phone"}}
	auth := controllers.NewAuthController(users, nil)

	if _, err := auth.UpdateWhatsAppPreference(context.Background(), 7, true, nil); err == nil {
		t.Fatal("expected invalid phone error")
	}
	if users.updated {
		t.Fatal("invalid phone should not update user preferences")
	}
}
