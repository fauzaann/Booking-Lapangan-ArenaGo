package controllers_test

import (
	"context"
	"testing"
	"time"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/xendit"
	"Booking-Lapangan/repository"
)

type paymentMockUOW struct {
	paymentRepo  repository.PaymentRepository
	bookingRepo  repository.BookingRepository
	checkoutRepo repository.CheckoutRepository
	auditLogRepo repository.AuditLogRepository
}

func (m *paymentMockUOW) User() repository.UserRepository         { return nil }
func (m *paymentMockUOW) Field() repository.FieldRepository       { return nil }
func (m *paymentMockUOW) Schedule() repository.ScheduleRepository { return nil }
func (m *paymentMockUOW) Booking() repository.BookingRepository   { return m.bookingRepo }
func (m *paymentMockUOW) Rating() repository.RatingRepository     { return nil }
func (m *paymentMockUOW) Payment() repository.PaymentRepository   { return m.paymentRepo }
func (m *paymentMockUOW) Checkout() repository.CheckoutRepository {
	if m.checkoutRepo != nil {
		return m.checkoutRepo
	}
	return &testCheckoutRepo{}
}
func (m *paymentMockUOW) AuditLog() repository.AuditLogRepository { return m.auditLogRepo }
func (m *paymentMockUOW) Atomic(ctx context.Context, fn func(r repository.Registry) error) error {
	return fn(m)
}

type testPaymentRepo struct {
	payment     *models.Payment
	payments    []models.Payment
	updatedWith *models.Payment
}

func (r *testPaymentRepo) Create(ctx context.Context, payment *models.Payment) error { return nil }
func (r *testPaymentRepo) Update(ctx context.Context, payment *models.Payment) error {
	r.updatedWith = payment
	for i := range r.payments {
		if r.payments[i].ID == payment.ID {
			r.payments[i] = *payment
			return nil
		}
	}
	r.payments = append(r.payments, *payment)
	return nil
}
func (r *testPaymentRepo) FindByID(ctx context.Context, id uint) (*models.Payment, error) {
	return r.payment, nil
}
func (r *testPaymentRepo) FindByExternalID(ctx context.Context, externalID string) (*models.Payment, error) {
	if r.payment != nil && r.payment.ExternalID == externalID {
		return r.payment, nil
	}
	return nil, apperror.NotFound("payment not found")
}
func (r *testPaymentRepo) FindByBookingID(ctx context.Context, bookingID uint) (*models.Payment, error) {
	if r.payment != nil && r.payment.BookingID == bookingID {
		return r.payment, nil
	}
	return nil, apperror.NotFound("payment not found")
}
func (r *testPaymentRepo) FindByCheckoutID(_ context.Context, checkoutID uint) ([]models.Payment, error) {
	var result []models.Payment
	for _, payment := range r.payments {
		if payment.CheckoutID != nil && *payment.CheckoutID == checkoutID {
			result = append(result, payment)
		}
	}
	return result, nil
}
func (r *testPaymentRepo) FindAll(ctx context.Context, filter repository.PaymentFilter) ([]models.Payment, int64, error) {
	return nil, 0, nil
}
func (r *testPaymentRepo) SumPaidAmount(ctx context.Context) (float64, error) { return 0, nil }
func (r *testPaymentRepo) RevenueAnalytics(ctx context.Context, from, to time.Time) (*repository.RevenueAnalytics, error) {
	return &repository.RevenueAnalytics{}, nil
}
func (r *testPaymentRepo) ExpireOverdue(ctx context.Context, now time.Time) ([]uint, error) {
	return nil, nil
}

type testCheckoutRepo struct {
	checkout *models.Checkout
}

func (r *testCheckoutRepo) Create(_ context.Context, checkout *models.Checkout) error {
	r.checkout = checkout
	return nil
}
func (r *testCheckoutRepo) Update(_ context.Context, checkout *models.Checkout) error {
	r.checkout = checkout
	return nil
}
func (r *testCheckoutRepo) FindByID(_ context.Context, id uint) (*models.Checkout, error) {
	if r.checkout != nil && r.checkout.ID == id {
		return r.checkout, nil
	}
	return nil, apperror.NotFound("checkout not found")
}
func (r *testCheckoutRepo) FindByExternalID(_ context.Context, externalID string) (*models.Checkout, error) {
	if r.checkout != nil && r.checkout.ExternalID == externalID {
		return r.checkout, nil
	}
	return nil, apperror.NotFound("checkout not found")
}

type testBookingRepo struct {
	booking     *models.Booking
	bookings    []models.Booking
	checkoutIDs map[uint]uint
	updatedWith *models.Booking
}

func (r *testBookingRepo) Create(ctx context.Context, booking *models.Booking) error { return nil }
func (r *testBookingRepo) Update(ctx context.Context, booking *models.Booking) error {
	r.updatedWith = booking
	for i := range r.bookings {
		if r.bookings[i].ID == booking.ID {
			r.bookings[i] = *booking
			return nil
		}
	}
	r.bookings = append(r.bookings, *booking)
	return nil
}
func (r *testBookingRepo) UpdateSchedule(ctx context.Context, booking *models.Booking) error {
	r.updatedWith = booking
	return nil
}
func (r *testBookingRepo) FindByID(ctx context.Context, id uint) (*models.Booking, error) {
	if r.booking != nil && r.booking.ID == id {
		return r.booking, nil
	}
	for i := range r.bookings {
		if r.bookings[i].ID == id {
			return &r.bookings[i], nil
		}
	}
	return nil, apperror.NotFound("booking not found")
}
func (r *testBookingRepo) FindDetailByID(ctx context.Context, id uint) (*models.Booking, error) {
	return r.FindByID(ctx, id)
}
func (r *testBookingRepo) FindDetailByIDForUpdate(ctx context.Context, id uint) (*models.Booking, error) {
	return r.FindByID(ctx, id)
}
func (r *testBookingRepo) FindByCode(ctx context.Context, code string) (*models.Booking, error) {
	return r.booking, nil
}
func (r *testBookingRepo) FindByCheckoutID(_ context.Context, checkoutID uint) ([]models.Booking, error) {
	var result []models.Booking
	for _, booking := range r.bookings {
		if r.checkoutIDs[booking.ID] == checkoutID {
			result = append(result, booking)
		}
	}
	return result, nil
}
func (r *testBookingRepo) FindAll(ctx context.Context, filter repository.BookingFilter) ([]models.Booking, int64, error) {
	return nil, 0, nil
}
func (r *testBookingRepo) FindActiveByFieldAndDate(ctx context.Context, fieldID uint, date time.Time) ([]models.Booking, error) {
	return nil, nil
}
func (r *testBookingRepo) HasOverlap(ctx context.Context, fieldID uint, date time.Time, startTime, endTime string) (bool, error) {
	return false, nil
}
func (r *testBookingRepo) HasOverlapExcluding(ctx context.Context, fieldID uint, date time.Time, startTime, endTime string, excludeBookingID uint) (bool, error) {
	return false, nil
}
func (r *testBookingRepo) LockFieldDate(ctx context.Context, fieldID uint, date time.Time) error {
	return nil
}
func (r *testBookingRepo) NextSequence(ctx context.Context, date time.Time) (int, error) {
	return 1, nil
}
func (r *testBookingRepo) Count(ctx context.Context) (int64, error) { return 0, nil }
func (r *testBookingRepo) CountByStatus(ctx context.Context) (map[models.BookingStatus]int64, error) {
	return nil, nil
}
func (r *testBookingRepo) MarkExpired(ctx context.Context, bookingIDs []uint) (int64, error) {
	return 0, nil
}
func (r *testBookingRepo) CompleteFinished(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}

type testAuditRepo struct{}

func (r *testAuditRepo) Create(ctx context.Context, log *models.AuditLog) error { return nil }
func (r *testAuditRepo) FindAll(ctx context.Context, params repository.ListParams) ([]models.AuditLog, int64, error) {
	return nil, 0, nil
}

type testInvoiceService struct {
	invoice *xendit.Invoice
}

func (s *testInvoiceService) CreateInvoice(ctx context.Context, req xendit.CreateInvoiceRequest) (*xendit.Invoice, error) {
	return s.invoice, nil
}

func (s *testInvoiceService) GetInvoice(ctx context.Context, invoiceID string) (*xendit.Invoice, error) {
	return s.invoice, nil
}

func TestPaymentController_HandleWebhook_TestCallbackNotFound(t *testing.T) {
	paymentRepo := &testPaymentRepo{}
	bookingRepo := &testBookingRepo{}
	uow := &paymentMockUOW{paymentRepo: paymentRepo, bookingRepo: bookingRepo, auditLogRepo: &testAuditRepo{}}
	token := "valid-secret-token"

	ctrl := controllers.NewPaymentController(uow, &testInvoiceService{}, token)

	// Simulating Xendit Dashboard test webhook with dummy external_id
	dummyPayload := xendit.WebhookPayload{
		ID:         "inv_test123",
		ExternalID: "test_external_non_existent",
		Status:     "PAID",
		Amount:     100000,
	}

	err := ctrl.HandleWebhook(context.Background(), token, dummyPayload)
	if err != nil {
		t.Fatalf("expected nil error on test webhook with non-existent external_id, got: %v", err)
	}
}

func TestPaymentController_HandleWebhook_InvalidToken(t *testing.T) {
	ctrl := controllers.NewPaymentController(&paymentMockUOW{}, &testInvoiceService{}, "valid-secret-token")

	err := ctrl.HandleWebhook(context.Background(), "invalid-token", xendit.WebhookPayload{ExternalID: "ext_1"})
	if err == nil {
		t.Fatal("expected error on invalid webhook token, got nil")
	}
}

func TestPaymentController_HandleWebhook_IssuesCheckInToken(t *testing.T) {
	booking := &models.Booking{ID: 10, Status: models.BookingStatusWaitingPayment}
	payment := &models.Payment{
		ID: 20, BookingID: booking.ID, ExternalID: "booking-10",
		Amount: 150000, Status: models.PaymentStatusPending,
	}
	paymentRepo := &testPaymentRepo{payment: payment}
	bookingRepo := &testBookingRepo{booking: booking}
	uow := &paymentMockUOW{paymentRepo: paymentRepo, bookingRepo: bookingRepo}
	controller := controllers.NewPaymentController(uow, nil, "valid-token")

	err := controller.HandleWebhook(context.Background(), "valid-token", xendit.WebhookPayload{
		ExternalID: "booking-10", Status: xendit.StatusPaid, Amount: 150000, PaidAmount: 150000,
	})
	if err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}
	if bookingRepo.updatedWith == nil || bookingRepo.updatedWith.Status != models.BookingStatusConfirmed {
		t.Fatal("paid webhook did not confirm the booking")
	}
	if bookingRepo.updatedWith.CheckInToken == nil || len(*bookingRepo.updatedWith.CheckInToken) < 32 {
		t.Fatal("paid webhook did not issue a secure check-in token")
	}
}

func TestPaymentController_SyncPaymentStatus_Success(t *testing.T) {
	b := &models.Booking{
		ID:     10,
		UserID: 1,
		Status: models.BookingStatusWaitingPayment,
	}
	p := &models.Payment{
		ID:        20,
		BookingID: 10,
		InvoiceID: "inv_live_123",
		Amount:    150000,
		Status:    models.PaymentStatusPending,
		Booking:   b,
	}

	paymentRepo := &testPaymentRepo{payment: p}
	bookingRepo := &testBookingRepo{booking: b}
	uow := &paymentMockUOW{paymentRepo: paymentRepo, bookingRepo: bookingRepo, auditLogRepo: &testAuditRepo{}}

	paidAt := time.Now()
	invService := &testInvoiceService{
		invoice: &xendit.Invoice{
			ID:             "inv_live_123",
			Status:         "PAID",
			PaymentMethod:  "QRIS",
			PaymentChannel: "BCA",
			PaidAt:         &paidAt,
		},
	}

	ctrl := controllers.NewPaymentController(uow, invService, "valid-token")

	actor := controllers.Actor{UserID: 1, Role: models.RoleUser}
	res, err := ctrl.SyncPaymentStatus(context.Background(), actor, 10)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	if res.Status != string(models.PaymentStatusPaid) {
		t.Errorf("expected payment status PAID, got %s", res.Status)
	}

	if bookingRepo.updatedWith == nil || bookingRepo.updatedWith.Status != models.BookingStatusConfirmed {
		t.Errorf("expected booking to be updated to CONFIRMED")
	}
	if bookingRepo.updatedWith.CheckInToken == nil || len(*bookingRepo.updatedWith.CheckInToken) < 32 {
		t.Error("expected a secure check-in token to be issued after payment")
	}
}

func TestPaymentController_SyncPaymentStatus_StillPending(t *testing.T) {
	b := &models.Booking{
		ID:     10,
		UserID: 1,
		Status: models.BookingStatusWaitingPayment,
	}
	p := &models.Payment{
		ID:        20,
		BookingID: 10,
		InvoiceID: "inv_live_123",
		Amount:    150000,
		Status:    models.PaymentStatusPending,
		Booking:   b,
	}

	paymentRepo := &testPaymentRepo{payment: p}
	bookingRepo := &testBookingRepo{booking: b}
	uow := &paymentMockUOW{paymentRepo: paymentRepo, bookingRepo: bookingRepo, auditLogRepo: &testAuditRepo{}}

	invService := &testInvoiceService{
		invoice: &xendit.Invoice{
			ID:     "inv_live_123",
			Status: "PENDING",
		},
	}

	ctrl := controllers.NewPaymentController(uow, invService, "valid-token")

	actor := controllers.Actor{UserID: 1, Role: models.RoleUser}
	res, err := ctrl.SyncPaymentStatus(context.Background(), actor, 10)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	if res.Status != string(models.PaymentStatusPending) {
		t.Errorf("expected payment status PENDING, got %s", res.Status)
	}

	if bookingRepo.updatedWith != nil {
		t.Errorf("expected booking to NOT be updated when invoice is still pending")
	}
}

func TestPaymentController_HandleWebhook_ConfirmsWholeCheckout(t *testing.T) {
	checkoutID := uint(7)
	checkoutRepo := &testCheckoutRepo{checkout: &models.Checkout{
		ID: 7, UserID: 1, ExternalID: "checkout-7", Amount: 200000,
		Status: models.PaymentStatusPending,
	}}
	paymentRepo := &testPaymentRepo{payments: []models.Payment{
		{ID: 11, BookingID: 21, CheckoutID: &checkoutID, Amount: 100000, Status: models.PaymentStatusPending},
		{ID: 12, BookingID: 22, CheckoutID: &checkoutID, Amount: 100000, Status: models.PaymentStatusPending},
	}}
	bookingRepo := &testBookingRepo{
		bookings: []models.Booking{
			{ID: 21, Status: models.BookingStatusWaitingPayment},
			{ID: 22, Status: models.BookingStatusWaitingPayment},
		},
		checkoutIDs: map[uint]uint{21: checkoutID, 22: checkoutID},
	}
	uow := &paymentMockUOW{
		paymentRepo: paymentRepo, bookingRepo: bookingRepo, checkoutRepo: checkoutRepo,
	}
	controller := controllers.NewPaymentController(uow, nil, "valid-token")
	payload := xendit.WebhookPayload{
		ExternalID: "checkout-7", Status: xendit.StatusPaid, Amount: 200000, PaidAmount: 200000,
	}

	if err := controller.HandleWebhook(context.Background(), "valid-token", payload); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}
	if checkoutRepo.checkout.Status != models.PaymentStatusPaid {
		t.Fatalf("checkout status = %s, want PAID", checkoutRepo.checkout.Status)
	}
	for _, payment := range paymentRepo.payments {
		if payment.Status != models.PaymentStatusPaid {
			t.Errorf("payment %d status = %s, want PAID", payment.ID, payment.Status)
		}
	}
	for _, booking := range bookingRepo.bookings {
		if booking.Status != models.BookingStatusConfirmed {
			t.Errorf("booking %d status = %s, want CONFIRMED", booking.ID, booking.Status)
		}
		if booking.CheckInToken == nil || len(*booking.CheckInToken) < 32 {
			t.Errorf("booking %d has no secure check-in token", booking.ID)
		}
	}

	if err := controller.HandleWebhook(context.Background(), "valid-token", payload); err != nil {
		t.Fatalf("duplicate HandleWebhook() error = %v", err)
	}
}

func TestPaymentController_HandleWebhook_RejectsCheckoutAmountMismatch(t *testing.T) {
	checkoutID := uint(7)
	checkoutRepo := &testCheckoutRepo{checkout: &models.Checkout{
		ID: 7, UserID: 1, ExternalID: "checkout-7", Amount: 200000,
		Status: models.PaymentStatusPending,
	}}
	paymentRepo := &testPaymentRepo{payments: []models.Payment{
		{ID: 11, BookingID: 21, CheckoutID: &checkoutID, Amount: 100000, Status: models.PaymentStatusPending},
	}}
	bookingRepo := &testBookingRepo{
		bookings:    []models.Booking{{ID: 21, Status: models.BookingStatusWaitingPayment}},
		checkoutIDs: map[uint]uint{21: checkoutID},
	}
	uow := &paymentMockUOW{
		paymentRepo: paymentRepo, bookingRepo: bookingRepo, checkoutRepo: checkoutRepo,
	}
	controller := controllers.NewPaymentController(uow, nil, "valid-token")
	err := controller.HandleWebhook(context.Background(), "valid-token", xendit.WebhookPayload{
		ExternalID: "checkout-7", Status: xendit.StatusPaid, Amount: 150000, PaidAmount: 150000,
	})
	if err == nil {
		t.Fatal("expected amount mismatch to be rejected")
	}
	if checkoutRepo.checkout.Status != models.PaymentStatusPending {
		t.Fatalf("checkout changed despite amount mismatch: %s", checkoutRepo.checkout.Status)
	}
}
