package controllers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	notificationv1 "Booking-Lapangan/contracts/notification/v1"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/timeutil"
)

func newNotificationOutboxEvent(
	eventType string,
	booking *models.Booking,
	paymentURL string,
	expiresAt *time.Time,
) (*models.NotificationEvent, error) {
	if booking == nil || booking.User == nil || booking.Field == nil {
		return nil, fmt.Errorf("booking, user, and field snapshots are required for notification event")
	}

	snapshot := notificationv1.BookingSnapshot{
		BookingID:            fmt.Sprintf("%d", booking.ID),
		BookingCode:          booking.BookingCode,
		UserID:               fmt.Sprintf("%d", booking.UserID),
		UserName:             booking.User.Name,
		UserEmail:            booking.User.Email,
		UserPhone:            booking.User.Phone,
		UserWhatsAppOptIn:    booking.User.WhatsAppOptIn,
		FieldID:              fmt.Sprintf("%d", booking.FieldID),
		FieldName:            booking.Field.Name,
		BookingDate:          booking.BookingDate.Format(timeutil.DateLayout),
		StartTime:            booking.StartTime,
		EndTime:              booking.EndTime,
		Duration:             booking.Duration,
		TotalPrice:           booking.TotalPrice,
		Status:               string(booking.Status),
		RefundReviewRequired: booking.RefundReviewRequired,
	}
	eventID := uuid.NewString()
	occurredAt := time.Now().UTC()

	var (
		event notificationv1.Envelope
		err   error
	)
	switch eventType {
	case models.NotificationEventBookingConfirmed:
		event, err = notificationv1.NewBookingEvent(eventID, notificationv1.EventBookingConfirmed, occurredAt, snapshot)
	case models.NotificationEventBookingCancelled:
		event, err = notificationv1.NewBookingEvent(eventID, notificationv1.EventBookingCancelled, occurredAt, snapshot)
	case models.NotificationEventBookingRescheduled:
		event, err = notificationv1.NewBookingEvent(eventID, notificationv1.EventBookingRescheduled, occurredAt, snapshot)
	case models.NotificationEventPaymentInvoiceReady:
		if expiresAt == nil {
			return nil, fmt.Errorf("payment invoice ready event requires expiry time")
		}
		event, err = notificationv1.NewPaymentInvoiceReadyEvent(eventID, occurredAt, notificationv1.PaymentInvoiceReadyData{
			Booking: snapshot, PaymentURL: paymentURL, ExpiresAt: *expiresAt,
		})
	default:
		return nil, fmt.Errorf("unsupported notification outbox event type %q", eventType)
	}
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(event.Data)
	if err != nil {
		return nil, fmt.Errorf("encode notification outbox payload: %w", err)
	}
	return &models.NotificationEvent{
		EventID:         eventID,
		EventType:       eventType,
		ContractVersion: notificationv1.ContractVersion,
		BookingID:       booking.ID,
		PaymentURL:      paymentURL,
		ExpiresAt:       expiresAt,
		Payload:         string(payload),
		CreatedAt:       occurredAt,
	}, nil
}
