package dto

import (
	"reflect"
	"testing"
	"time"

	"Booking-Lapangan/internal/booking"
	"Booking-Lapangan/models"
)

func TestNewBookingResponseIncludesPlayingPriceBreakdown(t *testing.T) {
	booking := models.Booking{
		Duration:        3,
		OriginalPrice:   300000,
		DiscountAmount:  30000,
		DiscountPercent: 10,
		PackageName:     "Paket 3 Jam+",
		TotalPrice:      270000,
	}

	response := NewBookingResponse(booking)
	if response.OriginalPrice != 300000 ||
		response.DiscountAmount != 30000 ||
		response.DiscountPercent != 10 ||
		response.PackageName != "Paket 3 Jam+" ||
		response.TotalPrice != 270000 {
		t.Fatalf("unexpected price breakdown: %+v", response)
	}
}

func TestNewBookingResponseUsesTotalAsOriginalForLegacyBooking(t *testing.T) {
	response := NewBookingResponse(models.Booking{TotalPrice: 200000})
	if response.OriginalPrice != 200000 {
		t.Fatalf("OriginalPrice = %v, want 200000", response.OriginalPrice)
	}
}

func TestNewBookingReadResponsePreservesPublicResponse(t *testing.T) {
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	token := "check-in-token"
	model := models.Booking{
		ID: 9, BookingCode: "BK-20261109-0009", UserID: 4, FieldID: 2,
		BookingDate: now, StartTime: "10:00", EndTime: "13:00", Duration: 3,
		OriginalPrice: 300000, DiscountAmount: 30000, DiscountPercent: 10,
		PackageName: "Paket 3 Jam+", TotalPrice: 270000,
		Status: models.BookingStatusConfirmed, CreatedAt: now,
		RescheduleHistory: []models.BookingScheduleChange{{
			PreviousDate: "2026-11-08", PreviousStartTime: "09:00", PreviousEndTime: "12:00",
			NewDate: "2026-11-09", NewStartTime: "10:00", NewEndTime: "13:00", ChangedAt: now,
		}},
		CheckInToken: &token,
		CheckedInAt:  &now,
		Field:        &models.Field{Name: "Court A"},
		Items:        []models.BookingItem{{StartTime: "10:00", EndTime: "11:00", Price: 90000}},
		Payment: &models.Payment{
			ID: 3, BookingID: 9, ExternalID: "external-9", InvoiceID: "invoice-9",
			InvoiceURL: "https://example.test/invoice", Amount: 270000,
			PaymentMethod: "BANK_TRANSFER", PaymentChannel: "BCA",
			Status: models.PaymentStatusPaid, PaidAt: &now, CreatedAt: now,
		},
	}

	got := NewBookingReadResponse(toBookingReadModel(model))
	want := NewBookingResponse(model)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("read-model response differs from legacy mapping:\n got: %#v\nwant: %#v", got, want)
	}
}

func toBookingReadModel(model models.Booking) booking.ReadModel {
	readModel := booking.ReadModel{
		ID: model.ID, BookingCode: model.BookingCode, UserID: model.UserID, FieldID: model.FieldID,
		BookingDate: model.BookingDate, StartTime: model.StartTime, EndTime: model.EndTime,
		Duration: model.Duration, OriginalPrice: model.OriginalPrice, DiscountAmount: model.DiscountAmount,
		DiscountPercent: model.DiscountPercent, PackageName: model.PackageName, TotalPrice: model.TotalPrice,
		Status: string(model.Status), Notes: model.Notes, CancellationReason: model.CancellationReason,
		RefundReviewRequired: model.RefundReviewRequired, CheckInToken: model.CheckInToken,
		CheckedInAt: model.CheckedInAt, CreatedAt: model.CreatedAt,
	}
	if model.Field != nil {
		readModel.FieldName = model.Field.Name
	}
	for _, change := range model.RescheduleHistory {
		readModel.RescheduleHistory = append(readModel.RescheduleHistory, booking.ScheduleChange{
			PreviousDate: change.PreviousDate, PreviousStartTime: change.PreviousStartTime,
			PreviousEndTime: change.PreviousEndTime, NewDate: change.NewDate,
			NewStartTime: change.NewStartTime, NewEndTime: change.NewEndTime, ChangedAt: change.ChangedAt,
		})
	}
	for _, item := range model.Items {
		readModel.Items = append(readModel.Items, booking.ReadItem{
			StartTime: item.StartTime, EndTime: item.EndTime, Price: item.Price,
		})
	}
	if model.Payment != nil {
		readModel.Payment = &booking.ReadPayment{
			ID: model.Payment.ID, BookingID: model.Payment.BookingID,
			ExternalID: model.Payment.ExternalID, InvoiceID: model.Payment.InvoiceID,
			InvoiceURL: model.Payment.InvoiceURL, Amount: model.Payment.Amount,
			PaymentMethod: model.Payment.PaymentMethod, PaymentChannel: model.Payment.PaymentChannel,
			Status: string(model.Payment.Status), PaidAt: model.Payment.PaidAt,
			ExpiredAt: model.Payment.ExpiredAt, CreatedAt: model.Payment.CreatedAt,
		}
	}
	return readModel
}
