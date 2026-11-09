package repository

import (
	"context"
	"testing"
	"time"

	"Booking-Lapangan/internal/booking"
	"Booking-Lapangan/models"
)

type bookingReadStub struct {
	BookingRepository
	detail *models.Booking
	list   []models.Booking
	filter BookingFilter
}

func (s *bookingReadStub) FindDetailByID(context.Context, uint) (*models.Booking, error) {
	return s.detail, nil
}

func (s *bookingReadStub) FindAll(_ context.Context, filter BookingFilter) ([]models.Booking, int64, error) {
	s.filter = filter
	return s.list, int64(len(s.list)), nil
}

func TestBookingReadRepositoryMapsModelsAndFilters(t *testing.T) {
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	token := "token"
	model := models.Booking{
		ID: 11, BookingCode: "BK-11", UserID: 2, FieldID: 3, BookingDate: now,
		StartTime: "10:00", EndTime: "12:00", Duration: 2, TotalPrice: 190000,
		Status: models.BookingStatusConfirmed, CheckInToken: &token, CreatedAt: now,
		Field: &models.Field{Name: "Court A"},
		Items: []models.BookingItem{{StartTime: "10:00", EndTime: "11:00", Price: 95000}},
		Payment: &models.Payment{
			ID: 4, BookingID: 11, ExternalID: "external-11",
			Status: models.PaymentStatusPaid, Amount: 190000, CreatedAt: now,
		},
	}
	stub := &bookingReadStub{detail: &model, list: []models.Booking{model}}
	repository := NewBookingReadRepository(stub)
	userID, fieldID := uint(2), uint(3)

	records, total, err := repository.List(context.Background(), booking.ReadFilter{
		Page: 2, Limit: 5, UserID: &userID, FieldID: &fieldID, Status: "CONFIRMED",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("List returned %d records and total %d, want 1 and 1", len(records), total)
	}
	if stub.filter.Page != 2 || stub.filter.Limit != 5 ||
		stub.filter.UserID != &userID || stub.filter.FieldID != &fieldID ||
		stub.filter.Status != "CONFIRMED" {
		t.Fatalf("filter was not forwarded: %+v", stub.filter)
	}

	detail, err := repository.FindDetail(context.Background(), model.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.FieldName != "Court A" ||
		detail.Payment == nil || detail.Payment.Status != string(models.PaymentStatusPaid) ||
		len(detail.Items) != 1 || detail.Items[0].Price != 95000 ||
		detail.CheckInToken == nil || *detail.CheckInToken != token {
		t.Fatalf("detail was not mapped correctly: %+v", detail)
	}
}
