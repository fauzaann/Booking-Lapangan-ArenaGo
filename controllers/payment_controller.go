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
	SyncPaymentStatus(ctx context.Context, actor Actor, bookingID uint) (*dto.PaymentResponse, error)
	SyncCheckoutStatus(ctx context.Context, actor Actor, checkoutID uint) (*dto.CheckoutResponse, error)
	SetNotificationController(nc NotificationController)
}

type paymentController struct {
	uow          repository.UnitOfWork
	invoices     xendit.InvoiceService
	webhookToken string
	notif        NotificationController
}

// NewPaymentController membuat PaymentController.
func NewPaymentController(uow repository.UnitOfWork, invoices xendit.InvoiceService, webhookToken string) PaymentController {
	return &paymentController{uow: uow, invoices: invoices, webhookToken: webhookToken}
}

// SetNotificationController menyambungkan controller notifikasi untuk memicu reminder saat pembayaran sukses.
func (c *paymentController) SetNotificationController(nc NotificationController) {
	c.notif = nc
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

	var confirmedBookingIDs []uint

	err := c.uow.Atomic(ctx, func(r repository.Registry) error {
		checkout, checkoutErr := r.Checkout().FindByExternalID(ctx, verified.ExternalID)
		if checkoutErr == nil {
			if checkout.Status != models.PaymentStatusPending {
				log.Printf("payment: checkout webhook for %s ignored, status already %s", checkout.ExternalID, checkout.Status)
				return nil
			}
			return applyCheckoutWebhook(ctx, r, checkout, verified, &confirmedBookingIDs)
		}
		if !apperror.IsNotFound(checkoutErr) {
			return checkoutErr
		}

		payment, err := r.Payment().FindByExternalID(ctx, verified.ExternalID)
		if err != nil {
			if apperror.IsNotFound(err) {
				// Webhook simulasi/test dari dashboard Xendit (misal external_id dummy) diakui sukses 200 OK
				log.Printf("payment: webhook for external_id '%s' not found in database, acknowledging test callback", verified.ExternalID)
				return nil
			}
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
			confirmedBookingIDs = append(confirmedBookingIDs, booking.ID)

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
	if err != nil {
		return err
	}

	c.scheduleConfirmedBookings(confirmedBookingIDs)
	return nil
}

func applyCheckoutWebhook(
	ctx context.Context,
	r repository.Registry,
	checkout *models.Checkout,
	verified xendit.WebhookPayload,
	confirmedBookingIDs *[]uint,
) error {
	status := strings.ToUpper(strings.TrimSpace(verified.Status))
	switch status {
	case xendit.StatusPaid, xendit.StatusSettled:
		paid := verified.PaidAmount
		if paid == 0 {
			paid = verified.Amount
		}
		if paid+1 < checkout.Amount || math.IsNaN(paid) {
			return apperror.BadRequest("paid amount does not match invoice amount")
		}
	case xendit.StatusExpired:
	default:
		return nil
	}

	payments, err := r.Payment().FindByCheckoutID(ctx, checkout.ID)
	if err != nil {
		return err
	}
	bookings, err := r.Booking().FindByCheckoutID(ctx, checkout.ID)
	if err != nil {
		return err
	}
	now := timeNow()
	switch status {
	case xendit.StatusPaid, xendit.StatusSettled:
		paidAt := verified.PaidAt
		if paidAt == nil {
			paidAt = &now
		}
		checkout.Status = models.PaymentStatusPaid
		checkout.PaidAt = paidAt
		checkout.PaymentMethod = verified.PaymentMethod
		checkout.PaymentChannel = verified.PaymentChannel
		for i := range payments {
			payments[i].Status = models.PaymentStatusPaid
			payments[i].PaidAt = paidAt
			payments[i].PaymentMethod = verified.PaymentMethod
			payments[i].PaymentChannel = verified.PaymentChannel
			if err := r.Payment().Update(ctx, &payments[i]); err != nil {
				return err
			}
		}
		for i := range bookings {
			bookings[i].Status = models.BookingStatusConfirmed
			if err := r.Booking().Update(ctx, &bookings[i]); err != nil {
				return err
			}
			*confirmedBookingIDs = append(*confirmedBookingIDs, bookings[i].ID)
		}
	case xendit.StatusExpired:
		checkout.Status = models.PaymentStatusExpired
		for i := range payments {
			payments[i].Status = models.PaymentStatusExpired
			if err := r.Payment().Update(ctx, &payments[i]); err != nil {
				return err
			}
		}
		for i := range bookings {
			if bookings[i].Status == models.BookingStatusPending || bookings[i].Status == models.BookingStatusWaitingPayment {
				bookings[i].Status = models.BookingStatusExpired
				if err := r.Booking().Update(ctx, &bookings[i]); err != nil {
					return err
				}
			}
		}
	}
	return r.Checkout().Update(ctx, checkout)
}

func (c *paymentController) scheduleConfirmedBookings(bookingIDs []uint) {
	if c.notif == nil {
		return
	}
	for _, bookingID := range bookingIDs {
		id := bookingID
		go func() {
			if err := c.notif.ScheduleRemindersForBooking(context.Background(), id); err != nil {
				log.Printf("payment: failed to schedule reminder for booking %d: %v", id, err)
			}
		}()
	}
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

// DetailByBooking mengambil pembayaran setelah memastikan actor berhak melihat booking.
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
	if payment.CheckoutID != nil {
		checkout, err := c.uow.Checkout().FindByID(ctx, *payment.CheckoutID)
		if err != nil {
			return nil, err
		}
		return c.syncCheckoutPayment(ctx, actor, checkout, payment)
	}
	response := dto.NewPaymentResponse(*payment)
	return &response, nil
}

func (c *paymentController) SyncCheckoutStatus(ctx context.Context, actor Actor, checkoutID uint) (*dto.CheckoutResponse, error) {
	checkout, err := c.uow.Checkout().FindByID(ctx, checkoutID)
	if err != nil {
		return nil, err
	}
	if checkout.UserID != actor.UserID && !actor.IsAdmin() {
		return nil, apperror.NotFound("checkout not found")
	}
	payments, err := c.uow.Payment().FindByCheckoutID(ctx, checkout.ID)
	if err != nil {
		return nil, err
	}
	if len(payments) > 0 {
		_, err := c.syncCheckoutPayment(ctx, actor, checkout, &payments[0])
		if err != nil {
			return nil, err
		}
	}
	checkoutController := &checkoutController{uow: c.uow, invoices: c.invoices}
	return checkoutController.Detail(ctx, actor, checkout.ID)
}

func (c *paymentController) syncCheckoutPayment(
	ctx context.Context,
	actor Actor,
	checkout *models.Checkout,
	payment *models.Payment,
) (*dto.PaymentResponse, error) {
	if checkout.UserID != actor.UserID && !actor.IsAdmin() {
		return nil, apperror.NotFound("checkout not found")
	}
	if checkout.Status != models.PaymentStatusPending || checkout.InvoiceID == "" || c.invoices == nil {
		response := dto.NewPaymentResponse(*payment)
		return &response, nil
	}
	invoice, err := c.invoices.GetInvoice(ctx, checkout.InvoiceID)
	if err != nil {
		log.Printf("payment: failed to fetch checkout invoice from xendit: %v", err)
		response := dto.NewPaymentResponse(*payment)
		return &response, nil
	}
	var confirmedIDs []uint
	err = c.uow.Atomic(ctx, func(r repository.Registry) error {
		current, err := r.Checkout().FindByID(ctx, checkout.ID)
		if err != nil {
			return err
		}
		if current.Status != models.PaymentStatusPending {
			return nil
		}
		payload := xendit.WebhookPayload{
			ExternalID: current.ExternalID, Status: invoice.Status, Amount: invoice.Amount,
			PaidAmount: invoice.PaidAmount, PaidAt: invoice.PaidAt,
			PaymentMethod: invoice.PaymentMethod, PaymentChannel: invoice.PaymentChannel,
		}
		return applyCheckoutWebhook(ctx, r, current, payload, &confirmedIDs)
	})
	if err != nil {
		return nil, err
	}
	c.scheduleConfirmedBookings(confirmedIDs)
	updated, err := c.uow.Payment().FindByBookingID(ctx, payment.BookingID)
	if err != nil {
		return nil, err
	}
	response := dto.NewPaymentResponse(*updated)
	return &response, nil
}

// List mengambil daftar pembayaran untuk kebutuhan admin.
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

// SyncPaymentStatus mengambil status invoice terbaru langsung dari Xendit dan memperbarui status payment & booking.
func (c *paymentController) SyncPaymentStatus(ctx context.Context, actor Actor, bookingID uint) (*dto.PaymentResponse, error) {
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
	if payment.CheckoutID != nil {
		checkout, err := c.uow.Checkout().FindByID(ctx, *payment.CheckoutID)
		if err != nil {
			return nil, err
		}
		return c.syncCheckoutPayment(ctx, actor, checkout, payment)
	}

	// Jika payment sudah berstatus final (PAID), kembalikan langsung
	if payment.Status == models.PaymentStatusPaid {
		res := dto.NewPaymentResponse(*payment)
		return &res, nil
	}

	// Jika belum ada invoice_id atau client Xendit tidak aktif, kembalikan data lokal
	if payment.InvoiceID == "" || c.invoices == nil {
		res := dto.NewPaymentResponse(*payment)
		return &res, nil
	}

	// Ambil status invoice aktual dari Xendit API (on-demand reconciliation)
	invoice, err := c.invoices.GetInvoice(ctx, payment.InvoiceID)
	if err != nil {
		log.Printf("payment: failed to fetch invoice from xendit for sync: %v", err)
		res := dto.NewPaymentResponse(*payment)
		return &res, nil
	}

	var confirmedBookingID uint
	switch strings.ToUpper(strings.TrimSpace(invoice.Status)) {
	case xendit.StatusPaid, xendit.StatusSettled:
		err = c.uow.Atomic(ctx, func(r repository.Registry) error {
			now := timeNow()
			paidAt := invoice.PaidAt
			if paidAt == nil {
				paidAt = &now
			}
			payment.Status = models.PaymentStatusPaid
			payment.PaidAt = paidAt
			if invoice.PaymentMethod != "" {
				payment.PaymentMethod = invoice.PaymentMethod
			}
			if invoice.PaymentChannel != "" {
				payment.PaymentChannel = invoice.PaymentChannel
			}
			if err := r.Payment().Update(ctx, payment); err != nil {
				return err
			}

			booking.Status = models.BookingStatusConfirmed
			confirmedBookingID = booking.ID
			return r.Booking().Update(ctx, booking)
		})
		if err != nil {
			return nil, err
		}

		if confirmedBookingID > 0 && c.notif != nil {
			go func() {
				_ = c.notif.ScheduleRemindersForBooking(context.Background(), confirmedBookingID)
			}()
		}

	case xendit.StatusExpired:
		_ = c.uow.Atomic(ctx, func(r repository.Registry) error {
			payment.Status = models.PaymentStatusExpired
			if err := r.Payment().Update(ctx, payment); err != nil {
				return err
			}
			if booking.Status == models.BookingStatusPending || booking.Status == models.BookingStatusWaitingPayment {
				booking.Status = models.BookingStatusExpired
				return r.Booking().Update(ctx, booking)
			}
			return nil
		})
	}

	res := dto.NewPaymentResponse(*payment)
	return &res, nil
}
