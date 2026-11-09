package repository

import (
	"context"

	"Booking-Lapangan/internal/booking"
	"Booking-Lapangan/models"
)

type bookingReadRepository struct {
	bookings BookingRepository
}

func NewBookingReadRepository(bookings BookingRepository) booking.ReadRepository {
	return &bookingReadRepository{bookings: bookings}
}

func (r *bookingReadRepository) List(ctx context.Context, filter booking.ReadFilter) ([]booking.ReadModel, int64, error) {
	bookings, total, err := r.bookings.FindAll(ctx, BookingFilter{
		ListParams: ListParams{Page: filter.Page, Limit: filter.Limit},
		UserID:     filter.UserID,
		FieldID:    filter.FieldID,
		Status:     filter.Status,
		DateFrom:   filter.DateFrom,
		DateTo:     filter.DateTo,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]booking.ReadModel, 0, len(bookings))
	for i := range bookings {
		result = append(result, bookingReadModel(bookings[i]))
	}
	return result, total, nil
}

func (r *bookingReadRepository) FindDetail(ctx context.Context, id uint) (*booking.ReadModel, error) {
	model, err := r.bookings.FindDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := bookingReadModel(*model)
	return &result, nil
}

func bookingReadModel(model models.Booking) booking.ReadModel {
	result := booking.ReadModel{
		ID:                   model.ID,
		BookingCode:          model.BookingCode,
		UserID:               model.UserID,
		FieldID:              model.FieldID,
		BookingDate:          model.BookingDate,
		StartTime:            model.StartTime,
		EndTime:              model.EndTime,
		Duration:             model.Duration,
		OriginalPrice:        model.OriginalPrice,
		DiscountAmount:       model.DiscountAmount,
		DiscountPercent:      model.DiscountPercent,
		PackageName:          model.PackageName,
		TotalPrice:           model.TotalPrice,
		Status:               string(model.Status),
		Notes:                model.Notes,
		CancellationReason:   model.CancellationReason,
		RefundReviewRequired: model.RefundReviewRequired,
		CheckInToken:         model.CheckInToken,
		CheckedInAt:          model.CheckedInAt,
		CreatedAt:            model.CreatedAt,
	}

	for _, change := range model.RescheduleHistory {
		result.RescheduleHistory = append(result.RescheduleHistory, booking.ScheduleChange{
			PreviousDate:      change.PreviousDate,
			PreviousStartTime: change.PreviousStartTime,
			PreviousEndTime:   change.PreviousEndTime,
			NewDate:           change.NewDate,
			NewStartTime:      change.NewStartTime,
			NewEndTime:        change.NewEndTime,
			ChangedAt:         change.ChangedAt,
		})
	}
	if model.Field != nil {
		result.FieldName = model.Field.Name
	}
	for _, item := range model.Items {
		result.Items = append(result.Items, booking.ReadItem{
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Price:     item.Price,
		})
	}
	if model.Payment != nil {
		result.Payment = &booking.ReadPayment{
			ID:             model.Payment.ID,
			BookingID:      model.Payment.BookingID,
			ExternalID:     model.Payment.ExternalID,
			InvoiceID:      model.Payment.InvoiceID,
			InvoiceURL:     model.Payment.InvoiceURL,
			Amount:         model.Payment.Amount,
			PaymentMethod:  model.Payment.PaymentMethod,
			PaymentChannel: model.Payment.PaymentChannel,
			Status:         string(model.Payment.Status),
			PaidAt:         model.Payment.PaidAt,
			ExpiredAt:      model.Payment.ExpiredAt,
			CreatedAt:      model.Payment.CreatedAt,
		}
	}
	return result
}
