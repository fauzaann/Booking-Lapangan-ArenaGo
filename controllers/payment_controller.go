package controllers

import (
	"context"
	"crypto/subtle"
	"log"
	"math"
	"strings"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/xendit"
	"Booking-Lapangan/repository"
)

// PaymentController menangani webhook dan pembacaan data pembayaran.
type PaymentController interface {
	HandleWebhook(ctx context.Context, callbackToken string, payload xendit.WebhookPayload) error
	DetailByBooking(ctx context.Context, actor Actor, bookingID uint) (*dto.PaymentResponse, error)
	List(ctx context.Context, query dto.PaymentFilterQuery) ([]dto.PaymentResponse, int64, error)
}

type paymentController struct {
	uow          repository.UnitOfWork
	invoices     xendit.InvoiceService
	webhookToken string
}

// NewPaymentController membuat PaymentController.
func NewPaymentController(uow repository.UnitOfWork, invoices xendit.InvoiceService, webhookToken string) PaymentController {
	return &paymentController{uow: uow, invoices: invoices, webhookToken: webhookToken}
}

// HandleWebhook memproses callback invoice dari Xendit.
//
// Tiga hal penting:
//  1. Token callback diverifikasi lebih dulu (constant time compare).
//  2. Status diambil ulang dari Xendit bila memungkinkan (server-side
//     verification), bukan sekadar percaya body request.
//  3. Idempotent: payment yang sudah berstatus final diabaikan sehingga
//     webhook berulang tidak menghasilkan perubahan ganda.
func (c *paymentController) HandleWebhook(ctx context.Context, callbackToken string, payload xendit.WebhookPayload) error {
	if c.webhookToken == "" {
		return apperror.Internal("webhook token is not configured", nil)
	}
	if subtle.ConstantTimeCompare([]byte(callbackToken), []byte(c.webhookToken)) != 1 {
		return apperror.Unauthorized("invalid callback token")
	}
	if strings.TrimSpace(payload.ExternalID) == "" {
		return apperror.BadRequest("external_id is required")
	}

	verified := c.verify(ctx, payload)

	return c.uow.Atomic(ctx, func(r repository.Registry) error {
		payment, err := r.Payment().FindByExternalID(ctx, verified.ExternalID)
		if err != nil {
			return err
		}

		if payment.Status != models.PaymentStatusPending {
			log.Printf("payment: webhook for %s ignored, status already %s", payment.ExternalID, payment.Status)
			return nil
		}

		booking, err := r.Booking().FindByID(ctx, payment.BookingID)
		if err != nil {
			return err
		}

		switch strings.ToUpper(strings.TrimSpace(verified.Status)) {
		case xendit.StatusPaid, xendit.StatusSettled:
			paid := verified.PaidAmount
			if paid == 0 {
				paid = verified.Amount
			}
			// Toleransi 1 rupiah untuk pembulatan float.
			if paid+1 < payment.Amount || math.IsNaN(paid) {
				return apperror.BadRequest("paid amount does not match invoice amount")
			}

			paidAt := verified.PaidAt
			if paidAt == nil {
				now := timeNow()
				paidAt = &now
			}
			payment.Status = models.PaymentStatusPaid
			payment.PaidAt = paidAt
			payment.PaymentMethod = verified.PaymentMethod
			payment.PaymentChannel = verified.PaymentChannel
			booking.Status = models.BookingStatusConfirmed

		case xendit.StatusExpired:
			payment.Status = models.PaymentStatusExpired
			if booking.Status == models.BookingStatusPending || booking.Status == models.BookingStatusWaitingPayment {
				booking.Status = models.BookingStatusExpired
			}

		default:
			// Status lain (misal PENDING) tidak mengubah apa pun.
			return nil
		}

		if err := r.Payment().Update(ctx, payment); err != nil {
			return err
		}
		return r.Booking().Update(ctx, booking)
	})
}

// verify mengambil ulang data invoice dari Xendit. Jika gagal (misal jaringan),
// aplikasi tetap memakai payload webhook yang tokennya sudah tervalidasi.
func (c *paymentController) verify(ctx context.Context, payload xendit.WebhookPayload) xendit.WebhookPayload {
	if c.invoices == nil || strings.TrimSpace(payload.ID) == "" {
		return payload
	}

	invoice, err := c.invoices.GetInvoice(ctx, payload.ID)
	if err != nil || invoice == nil {
		log.Printf("payment: unable to verify invoice %s to xendit: %v", payload.ID, err)
		return payload
	}

	verified := payload
	verified.Status = invoice.Status
	verified.Amount = invoice.Amount
	verified.PaidAmount = invoice.PaidAmount
	if invoice.ExternalID != "" {
		verified.ExternalID = invoice.ExternalID
	}
	if invoice.PaidAt != nil {
		verified.PaidAt = invoice.PaidAt
	}
	if invoice.PaymentMethod != "" {
		verified.PaymentMethod = invoice.PaymentMethod
	}
	if invoice.PaymentChannel != "" {
		verified.PaymentChannel = invoice.PaymentChannel
	}
	return verified
}

func (c *paymentController) DetailByBooking(ctx context.Context, actor Actor, bookingID uint) (*dto.PaymentResponse, error) {
	booking, err := c.uow.Booking().FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if err := ensureBookingOwner(actor, booking); err != nil {
		return nil, err
	}

	payment, err := c.uow.Payment().FindByBookingID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	response := dto.NewPaymentResponse(*payment)
	return &response, nil
}

func (c *paymentController) List(ctx context.Context, query dto.PaymentFilterQuery) ([]dto.PaymentResponse, int64, error) {
	query.PaginationQuery = query.PaginationQuery.Normalize()

	payments, total, err := c.uow.Payment().FindAll(ctx, repository.PaymentFilter{
		ListParams: repository.ListParams{Page: query.Page, Limit: query.Limit},
		Status:     strings.ToUpper(strings.TrimSpace(query.Status)),
	})
	if err != nil {
		return nil, 0, err
	}
	return dto.NewPaymentResponses(payments), total, nil
}
