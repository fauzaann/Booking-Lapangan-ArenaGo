package controllers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/timeutil"
	"Booking-Lapangan/pkg/xendit"
	"Booking-Lapangan/repository"
)

// BookingPolicy adalah aturan bisnis booking yang berasal dari konfigurasi.
type BookingPolicy struct {
	MinHours           int
	MaxHours           int
	CancelMinHours     int
	InvoiceDuration    int
	SuccessRedirectURL string
	FailureRedirectURL string
}

// BookingController menangani seluruh alur pemesanan lapangan.
type BookingController interface {
	Create(ctx context.Context, actor Actor, req dto.CreateBookingRequest) (*dto.BookingCreatedResponse, error)
	List(ctx context.Context, actor Actor, query dto.BookingFilterQuery) ([]dto.BookingResponse, int64, error)
	Detail(ctx context.Context, actor Actor, id uint) (*dto.BookingResponse, error)
	Cancel(ctx context.Context, actor Actor, id uint) (*dto.BookingResponse, error)
	CancelWithReason(ctx context.Context, actor Actor, id uint, reason string) (*dto.BookingResponse, error)
	Reschedule(ctx context.Context, actor Actor, id uint, req dto.RescheduleBookingRequest) (*dto.BookingResponse, error)
	UpdateStatus(ctx context.Context, id uint, status models.BookingStatus) (*dto.BookingResponse, error)
	SetNotificationController(nc NotificationController)
}

type bookingController struct {
	uow      repository.UnitOfWork
	invoices xendit.InvoiceService
	policy   BookingPolicy
	notif    NotificationController
}

// NewBookingController membuat BookingController.
func NewBookingController(uow repository.UnitOfWork, invoices xendit.InvoiceService, policy BookingPolicy) BookingController {
	if policy.MinHours < 1 {
		policy.MinHours = 1
	}
	if policy.MaxHours < policy.MinHours {
		policy.MaxHours = 8
	}
	if policy.InvoiceDuration <= 0 {
		policy.InvoiceDuration = 3600
	}
	return &bookingController{uow: uow, invoices: invoices, policy: policy}
}

// SetNotificationController menyambungkan controller notifikasi.
func (c *bookingController) SetNotificationController(nc NotificationController) {
	c.notif = nc
}

// Create menjalankan alur: validasi -> cek jadwal -> cek availability (dalam
// transaksi + advisory lock) -> hitung harga -> simpan booking & payment ->
// buat invoice Xendit -> simpan payment_url.
//
// Pemanggilan Xendit sengaja dilakukan DI LUAR transaksi database agar koneksi
// database tidak tertahan selama request HTTP eksternal berlangsung. Konsistensi
// tetap terjaga: jika invoice gagal dibuat, booking dibatalkan otomatis.
func (c *bookingController) Create(ctx context.Context, actor Actor, req dto.CreateBookingRequest) (*dto.BookingCreatedResponse, error) {
	bookingDate, err := timeutil.ParseDate(req.BookingDate)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if bookingDate.Before(timeutil.NormalizeDate(timeNow())) {
		return nil, apperror.Unprocessable("booking_date must not be in the past")
	}

	duration, err := timeutil.DurationHours(req.StartTime, req.EndTime)
	if err != nil {
		return nil, apperror.Unprocessable(err.Error())
	}
	if duration < c.policy.MinHours {
		return nil, apperror.Unprocessable(fmt.Sprintf("minimum booking duration is %d hour(s)", c.policy.MinHours))
	}
	if duration > c.policy.MaxHours {
		return nil, apperror.Unprocessable(fmt.Sprintf("maximum booking duration is %d hour(s)", c.policy.MaxHours))
	}

	field, err := c.uow.Field().FindByID(ctx, req.FieldID)
	if err != nil {
		return nil, err
	}
	if !field.IsActive() {
		return nil, apperror.Unprocessable("field is not available for booking")
	}

	day := models.DayFromWeekday(bookingDate.Weekday())
	schedule, err := c.uow.Schedule().FindByFieldAndDay(ctx, field.ID, day)
	if err != nil {
		if apperror.IsNotFound(err) {
			return nil, apperror.Unprocessable("field has no operating hours on " + string(day))
		}
		return nil, err
	}
	if schedule.IsClosed {
		return nil, apperror.Unprocessable("field is closed on " + string(day))
	}
	if !timeutil.Within(req.StartTime, req.EndTime, schedule.OpenTime, schedule.CloseTime) {
		return nil, apperror.Unprocessable(fmt.Sprintf(
			"booking time must be within operating hours %s-%s", schedule.OpenTime, schedule.CloseTime))
	}

	startAt, err := timeutil.CombineDateTime(bookingDate, req.StartTime)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if !startAt.After(timeNow()) {
		return nil, apperror.Unprocessable("start_time has already passed")
	}

	// Total harga SELALU dihitung backend, tidak pernah diterima dari client.
	totalPrice := field.PricePerHour * float64(duration)

	var (
		booking *models.Booking
		payment *models.Payment
	)

	err = c.uow.Atomic(ctx, func(r repository.Registry) error {
		// Advisory lock mencegah dua request paralel lolos pemeriksaan
		// availability untuk lapangan + tanggal yang sama.
		if lockErr := r.Booking().LockFieldDate(ctx, field.ID, bookingDate); lockErr != nil {
			return lockErr
		}

		overlap, checkErr := r.Booking().HasOverlap(ctx, field.ID, bookingDate, req.StartTime, req.EndTime)
		if checkErr != nil {
			return checkErr
		}
		if overlap {
			return apperror.Conflict("selected time slot is not available")
		}

		sequence, seqErr := r.Booking().NextSequence(ctx, timeNow())
		if seqErr != nil {
			return seqErr
		}

		booking = &models.Booking{
			BookingCode: BuildBookingCode(timeNow(), sequence),
			UserID:      actor.UserID,
			FieldID:     field.ID,
			BookingDate: bookingDate,
			StartTime:   req.StartTime,
			EndTime:     req.EndTime,
			Duration:    duration,
			TotalPrice:  totalPrice,
			Status:      models.BookingStatusPending,
			Notes:       strings.TrimSpace(req.Notes),
			Items:       buildBookingItems(req.StartTime, req.EndTime, field.PricePerHour),
		}
		if createErr := r.Booking().Create(ctx, booking); createErr != nil {
			return createErr
		}

		payment = &models.Payment{
			BookingID:  booking.ID,
			ExternalID: booking.BookingCode,
			Amount:     totalPrice,
			Status:     models.PaymentStatusPending,
		}
		return r.Payment().Create(ctx, payment)
	})
	if err != nil {
		return nil, err
	}

	successURL := c.policy.SuccessRedirectURL
	if successURL != "" {
		sep := "?"
		if strings.Contains(successURL, "?") {
			sep = "&"
		}
		successURL = fmt.Sprintf("%s%sbooking_id=%d&booking_code=%s", successURL, sep, booking.ID, booking.BookingCode)
	}

	failureURL := c.policy.FailureRedirectURL
	if failureURL != "" {
		sep := "?"
		if strings.Contains(failureURL, "?") {
			sep = "&"
		}
		failureURL = fmt.Sprintf("%s%sbooking_id=%d&booking_code=%s", failureURL, sep, booking.ID, booking.BookingCode)
	}

	invoice, err := c.invoices.CreateInvoice(ctx, xendit.CreateInvoiceRequest{
		ExternalID:      booking.BookingCode,
		Amount:          totalPrice,
		Description:     fmt.Sprintf("Booking %s - %s (%s %s-%s)", booking.BookingCode, field.Name, req.BookingDate, req.StartTime, req.EndTime),
		InvoiceDuration: c.policy.InvoiceDuration,
		PayerEmail:      actor.Email,
		Customer:        &xendit.Customer{Email: actor.Email},
		Items: []xendit.Item{{
			Name:     fmt.Sprintf("%s (%d hour)", field.Name, duration),
			Quantity: duration,
			Price:    field.PricePerHour,
			Category: string(field.Type),
		}},
		SuccessRedirectURL: successURL,
		FailureRedirectURL: failureURL,
	})
	if err != nil {
		c.rollbackBooking(ctx, booking, payment)
		return nil, paymentInvoiceError(err)
	}

	expiredAt := invoice.ExpiryDate
	if expiredAt == nil {
		fallback := timeNow().Add(time.Duration(c.policy.InvoiceDuration) * time.Second)
		expiredAt = &fallback
	}

	err = c.uow.Atomic(ctx, func(r repository.Registry) error {
		payment.InvoiceID = invoice.ID
		payment.InvoiceURL = invoice.InvoiceURL
		payment.ExpiredAt = expiredAt
		if updateErr := r.Payment().Update(ctx, payment); updateErr != nil {
			return updateErr
		}
		booking.Status = models.BookingStatusWaitingPayment
		return r.Booking().Update(ctx, booking)
	})
	if err != nil {
		return nil, err
	}

	return &dto.BookingCreatedResponse{
		BookingID:     booking.ID,
		BookingCode:   booking.BookingCode,
		FieldName:     field.Name,
		BookingDate:   booking.BookingDate.Format(timeutil.DateLayout),
		StartTime:     booking.StartTime,
		EndTime:       booking.EndTime,
		Duration:      booking.Duration,
		TotalPrice:    booking.TotalPrice,
		BookingStatus: string(booking.Status),
		PaymentURL:    payment.InvoiceURL,
		PaymentStatus: string(payment.Status),
		ExpiredAt:     payment.ExpiredAt,
	}, nil
}

// paymentInvoiceError menerjemahkan error Xendit menjadi error API yang aman.
func paymentInvoiceError(err error) *apperror.AppError {
	var apiErr *xendit.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401, 403:
			return apperror.BadGateway("Xendit authentication failed; check XENDIT_SECRET_KEY", err)
		case 400:
			return apperror.BadGateway("Xendit rejected the invoice request; check payment configuration", err)
		}
	}
	return apperror.BadGateway("Xendit payment service is unavailable", err)
}

// List mengambil booking milik user atau seluruh booking untuk admin.
func (c *bookingController) List(ctx context.Context, actor Actor, query dto.BookingFilterQuery) ([]dto.BookingResponse, int64, error) {
	query.PaginationQuery = query.PaginationQuery.Normalize()

	filter := repository.BookingFilter{
		ListParams: repository.ListParams{Page: query.Page, Limit: query.Limit},
		FieldID:    query.FieldID,
		Status:     strings.ToUpper(strings.TrimSpace(query.Status)),
	}

	// User biasa hanya boleh melihat booking miliknya sendiri, apa pun isi query.
	if actor.IsAdmin() {
		filter.UserID = query.UserID
	} else {
		userID := actor.UserID
		filter.UserID = &userID
	}

	if query.From != "" {
		from, err := timeutil.ParseDate(query.From)
		if err != nil {
			return nil, 0, apperror.BadRequest("from must use YYYY-MM-DD format")
		}
		filter.DateFrom = &from
	}
	if query.To != "" {
		to, err := timeutil.ParseDate(query.To)
		if err != nil {
			return nil, 0, apperror.BadRequest("to must use YYYY-MM-DD format")
		}
		filter.DateTo = &to
	}

	bookings, total, err := c.uow.Booking().FindAll(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewBookingResponses(bookings), total, nil
}

// Detail mengambil detail booking setelah pemeriksaan kepemilikan.
func (c *bookingController) Detail(ctx context.Context, actor Actor, id uint) (*dto.BookingResponse, error) {
	booking, err := c.uow.Booking().FindDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ensureBookingOwner(actor, booking); err != nil {
		return nil, err
	}
	response := dto.NewBookingResponse(*booking)
	return &response, nil
}

// Cancel preserves the existing controller contract while delegating to the
// owner-initiated cancellation flow.
func (c *bookingController) Cancel(ctx context.Context, actor Actor, id uint) (*dto.BookingResponse, error) {
	return c.CancelWithReason(ctx, actor, id, "")
}

// CancelWithReason cancels an owned booking and preserves paid payment records
// for manual refund review. All state changes and payment updates are atomic.
func (c *bookingController) CancelWithReason(ctx context.Context, actor Actor, id uint, reason string) (*dto.BookingResponse, error) {
	var booking *models.Booking
	err := c.uow.Atomic(ctx, func(r repository.Registry) error {
		locked, err := r.Booking().FindDetailByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if locked.UserID != actor.UserID {
			return apperror.NotFound("booking not found")
		}
		if err := validateBookingCanChange(locked); err != nil {
			return err
		}
		if err := validateCancellationDeadline(locked, c.policy.CancelMinHours, timeNow()); err != nil {
			return err
		}
		if err := r.Booking().LockFieldDate(ctx, locked.FieldID, locked.BookingDate); err != nil {
			return err
		}

		payment, paymentErr := r.Payment().FindByBookingID(ctx, locked.ID)
		if paymentErr != nil && !apperror.IsNotFound(paymentErr) {
			return paymentErr
		}

		now := timeNow()
		locked.Status = models.BookingStatusCancelled
		locked.CancelledAt = &now
		locked.CancellationReason = strings.TrimSpace(reason)
		if payment != nil && payment.Status == models.PaymentStatusPaid {
			locked.RefundReviewRequired = true
		}
		if err := r.Booking().Update(ctx, locked); err != nil {
			return err
		}
		if payment != nil && payment.Status == models.PaymentStatusPending {
			payment.Status = models.PaymentStatusFailed
			if err := r.Payment().Update(ctx, payment); err != nil {
				return err
			}
		}
		locked.Payment = payment
		booking = locked
		return nil
	})
	if err != nil {
		return nil, err
	}

	c.notifyAfterCommit(ctx, "booking cancellation", booking.ID, func(ctx context.Context) error {
		return c.notif.CancelRemindersForBooking(ctx, booking.ID)
	})
	response := dto.NewBookingResponse(*booking)
	return &response, nil
}

// Reschedule validates and saves a new time under transaction-scoped locks.
// Price-changing requests are rejected because the current payment integration
// cannot adjust an existing invoice or collect/refund the difference.
func (c *bookingController) Reschedule(ctx context.Context, actor Actor, id uint, req dto.RescheduleBookingRequest) (*dto.BookingResponse, error) {
	newDate, err := timeutil.ParseDate(req.BookingDate)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	duration, err := timeutil.DurationHours(req.StartTime, req.EndTime)
	if err != nil {
		return nil, apperror.Unprocessable(err.Error())
	}
	if duration < c.policy.MinHours {
		return nil, apperror.Unprocessable(fmt.Sprintf("minimum booking duration is %d hour(s)", c.policy.MinHours))
	}
	if duration > c.policy.MaxHours {
		return nil, apperror.Unprocessable(fmt.Sprintf("maximum booking duration is %d hour(s)", c.policy.MaxHours))
	}

	var booking *models.Booking
	err = c.uow.Atomic(ctx, func(r repository.Registry) error {
		locked, err := r.Booking().FindDetailByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if locked.UserID != actor.UserID {
			return apperror.NotFound("booking not found")
		}
		if err := validateBookingCanChange(locked); err != nil {
			return err
		}
		now := timeNow()
		if newDate.Before(timeutil.NormalizeDate(now)) {
			return apperror.Unprocessable("booking_date must not be in the past")
		}
		if isPaymentSettled(locked) {
			if err := validateCancellationDeadline(locked, c.policy.CancelMinHours, now); err != nil {
				return apperror.Unprocessable("booking is outside the allowed reschedule window")
			}
		}
		startAt, err := timeutil.CombineDateTime(newDate, req.StartTime)
		if err != nil {
			return apperror.BadRequest(err.Error())
		}
		if !startAt.After(now) {
			return apperror.Unprocessable("start_time has already passed")
		}

		field, err := r.Field().FindByID(ctx, locked.FieldID)
		if err != nil {
			return err
		}
		if !field.IsActive() {
			return apperror.Unprocessable("field is not available for booking")
		}
		day := models.DayFromWeekday(newDate.Weekday())
		schedule, err := r.Schedule().FindByFieldAndDay(ctx, field.ID, day)
		if err != nil {
			if apperror.IsNotFound(err) {
				return apperror.Unprocessable("field has no operating hours on " + string(day))
			}
			return err
		}
		if schedule.IsClosed {
			return apperror.Unprocessable("field is closed on " + string(day))
		}
		if !timeutil.Within(req.StartTime, req.EndTime, schedule.OpenTime, schedule.CloseTime) {
			return apperror.Unprocessable(fmt.Sprintf(
				"booking time must be within operating hours %s-%s", schedule.OpenTime, schedule.CloseTime))
		}

		newTotal := field.PricePerHour * float64(duration)
		if math.Abs(newTotal-locked.TotalPrice) > 0.01 {
			return apperror.Conflict(fmt.Sprintf(
				"reschedule would change the total from %.2f to %.2f; payment adjustment is not supported",
				locked.TotalPrice, newTotal))
		}

		dates := []time.Time{locked.BookingDate, newDate}
		if dates[1].Before(dates[0]) {
			dates[0], dates[1] = dates[1], dates[0]
		}
		if err := r.Booking().LockFieldDate(ctx, field.ID, dates[0]); err != nil {
			return err
		}
		if !dates[1].Equal(dates[0]) {
			if err := r.Booking().LockFieldDate(ctx, field.ID, dates[1]); err != nil {
				return err
			}
		}
		overlap, err := r.Booking().HasOverlapExcluding(ctx, field.ID, newDate, req.StartTime, req.EndTime, locked.ID)
		if err != nil {
			return err
		}
		if overlap {
			return apperror.Conflict("selected time slot is not available")
		}

		locked.RescheduleHistory = append(locked.RescheduleHistory, models.BookingScheduleChange{
			PreviousDate:      locked.BookingDate.Format(timeutil.DateLayout),
			PreviousStartTime: locked.StartTime,
			PreviousEndTime:   locked.EndTime,
			NewDate:           newDate.Format(timeutil.DateLayout),
			NewStartTime:      req.StartTime,
			NewEndTime:        req.EndTime,
			ChangedAt:         now,
		})
		locked.BookingDate = newDate
		locked.StartTime = req.StartTime
		locked.EndTime = req.EndTime
		locked.Duration = duration
		locked.Items = buildBookingItems(req.StartTime, req.EndTime, field.PricePerHour)
		locked.Field = field
		if err := r.Booking().UpdateSchedule(ctx, locked); err != nil {
			return err
		}
		booking = locked
		return nil
	})
	if err != nil {
		return nil, err
	}

	c.notifyAfterCommit(ctx, "booking reschedule", booking.ID, func(ctx context.Context) error {
		return c.notif.RescheduleBooking(ctx, booking.ID)
	})
	response := dto.NewBookingResponse(*booking)
	return &response, nil
}

func validateBookingCanChange(booking *models.Booking) error {
	switch booking.Status {
	case models.BookingStatusCancelled:
		return apperror.Conflict("booking is already cancelled")
	case models.BookingStatusCompleted, models.BookingStatusExpired:
		return apperror.Conflict("booking can no longer be changed")
	}
	startAt, err := timeutil.CombineDateTime(booking.BookingDate, booking.StartTime)
	if err != nil {
		return apperror.Internal("failed to read booking schedule", err)
	}
	if !startAt.After(timeNow()) {
		return apperror.Unprocessable("booking has already started")
	}
	return nil
}

func isPaymentSettled(booking *models.Booking) bool {
	return booking.Status == models.BookingStatusPaid ||
		booking.Status == models.BookingStatusConfirmed ||
		(booking.Payment != nil && booking.Payment.Status == models.PaymentStatusPaid)
}

func validateCancellationDeadline(booking *models.Booking, minHours int, now time.Time) error {
	if !isPaymentSettled(booking) {
		return nil
	}
	startAt, err := timeutil.CombineDateTime(booking.BookingDate, booking.StartTime)
	if err != nil {
		return apperror.Internal("failed to read booking schedule", err)
	}
	if startAt.Sub(now) < time.Duration(minHours)*time.Hour {
		return apperror.Unprocessable(fmt.Sprintf(
			"paid booking can only be cancelled or rescheduled at least %d hour(s) before start time", minHours))
	}
	return nil
}

func (c *bookingController) notifyAfterCommit(
	ctx context.Context,
	action string,
	bookingID uint,
	notify func(context.Context) error,
) {
	if c.notif == nil {
		return
	}
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := notify(notifyCtx); err != nil {
		log.Printf("booking: %s committed but notification update failed for booking %d: %v", action, bookingID, err)
	}
}

// UpdateStatus mengubah status booking dari panel admin.
func (c *bookingController) UpdateStatus(ctx context.Context, id uint, status models.BookingStatus) (*dto.BookingResponse, error) {
	if !status.Valid() {
		return nil, apperror.Unprocessable("invalid booking status")
	}

	booking, err := c.uow.Booking().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if booking.Status == status {
		return c.Detail(ctx, Actor{Role: models.RoleAdmin}, id)
	}

	booking.Status = status
	if status == models.BookingStatusCancelled && booking.CancelledAt == nil {
		now := timeNow()
		booking.CancelledAt = &now
	}
	if err := c.uow.Booking().Update(ctx, booking); err != nil {
		return nil, err
	}

	if c.notif != nil {
		if status == models.BookingStatusConfirmed {
			go func() {
				_ = c.notif.ScheduleRemindersForBooking(context.Background(), id)
			}()
		} else if status == models.BookingStatusCancelled {
			go func() {
				_ = c.notif.CancelRemindersForBooking(context.Background(), id)
			}()
		}
	}

	return c.Detail(ctx, Actor{Role: models.RoleAdmin}, id)
}

// rollbackBooking membatalkan booking ketika pembuatan invoice gagal, sehingga
// slot tidak tertahan oleh booking yang tidak akan pernah dibayar.
func (c *bookingController) rollbackBooking(ctx context.Context, booking *models.Booking, payment *models.Payment) {
	// Context baru tanpa cancel: pembersihan tetap harus jalan walau request batal.
	cleanupCtx := context.WithoutCancel(ctx)

	err := c.uow.Atomic(cleanupCtx, func(r repository.Registry) error {
		booking.Status = models.BookingStatusCancelled
		now := timeNow()
		booking.CancelledAt = &now
		if updateErr := r.Booking().Update(cleanupCtx, booking); updateErr != nil {
			return updateErr
		}
		payment.Status = models.PaymentStatusFailed
		return r.Payment().Update(cleanupCtx, payment)
	})
	if err != nil {
		log.Printf("booking: failed to rollback booking %s: %v", booking.BookingCode, err)
	}
}

// ensureBookingOwner memastikan actor boleh mengakses booking tertentu.
func ensureBookingOwner(actor Actor, booking *models.Booking) error {
	if actor.IsAdmin() || booking.UserID == actor.UserID {
		return nil
	}
	// 404 dipilih agar keberadaan booking milik orang lain tidak bocor.
	return apperror.NotFound("booking not found")
}

// BuildBookingCode menyusun kode booking unik berformat BK-YYYYMMDD-XXXX.
func BuildBookingCode(now time.Time, sequence int) string {
	return fmt.Sprintf("BK-%s-%04d", now.Format("20060102"), sequence)
}

// buildBookingItems membuat item harga untuk setiap slot satu jam.
func buildBookingItems(start, end string, pricePerHour float64) []models.BookingItem {
	slots := timeutil.HourlySlots(start, end)
	items := make([]models.BookingItem, 0, len(slots))
	for _, slot := range slots {
		items = append(items, models.BookingItem{
			StartTime: slot[0],
			EndTime:   slot[1],
			Price:     pricePerHour,
		})
	}
	return items
}
