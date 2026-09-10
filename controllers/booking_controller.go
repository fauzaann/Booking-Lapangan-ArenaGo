package controllers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
	"Booking-Lapangan/utils"
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
	UpdateStatus(ctx context.Context, id uint, status models.BookingStatus) (*dto.BookingResponse, error)
}

type bookingController struct {
	uow      repository.UnitOfWork
	invoices utils.InvoiceService
	policy   BookingPolicy
}

// NewBookingController membuat BookingController.
func NewBookingController(uow repository.UnitOfWork, invoices utils.InvoiceService, policy BookingPolicy) BookingController {
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

// Create menjalankan alur: validasi -> cek jadwal -> cek availability (dalam
// transaksi + advisory lock) -> hitung harga -> simpan booking & payment ->
// buat invoice Xendit -> simpan payment_url.
//
// Pemanggilan Xendit sengaja dilakukan DI LUAR transaksi database agar koneksi
// database tidak tertahan selama request HTTP eksternal berlangsung. Konsistensi
// tetap terjaga: jika invoice gagal dibuat, booking dibatalkan otomatis.
func (c *bookingController) Create(ctx context.Context, actor Actor, req dto.CreateBookingRequest) (*dto.BookingCreatedResponse, error) {
	bookingDate, err := utils.ParseDate(req.BookingDate)
	if err != nil {
		return nil, utils.BadRequest(err.Error())
	}
	if bookingDate.Before(utils.NormalizeDate(timeNow())) {
		return nil, utils.Unprocessable("booking_date must not be in the past")
	}

	duration, err := utils.DurationHours(req.StartTime, req.EndTime)
	if err != nil {
		return nil, utils.Unprocessable(err.Error())
	}
	if duration < c.policy.MinHours {
		return nil, utils.Unprocessable(fmt.Sprintf("minimum booking duration is %d hour(s)", c.policy.MinHours))
	}
	if duration > c.policy.MaxHours {
		return nil, utils.Unprocessable(fmt.Sprintf("maximum booking duration is %d hour(s)", c.policy.MaxHours))
	}

	field, err := c.uow.Field().FindByID(ctx, req.FieldID)
	if err != nil {
		return nil, err
	}
	if !field.IsValid() {
		return nil, utils.Unprocessable("field is not available for booking")
	}

	day := models.DayFromWeekday(bookingDate.Weekday())
	schedule, err := c.uow.Schedule().FindByFieldAndDay(ctx, field.ID, day)
	if err != nil {
		if utils.IsNotFound(err) {
			return nil, utils.Unprocessable("field has no operating hours on " + string(day))
		}
		return nil, err
	}
	if schedule.IsActive {
		return nil, utils.Unprocessable("field is closed on " + string(day))
	}
	if !utils.Within(req.StartTime, req.EndTime, schedule.OpenTime, schedule.CloseTime) {
		return nil, utils.Unprocessable(fmt.Sprintf(
			"booking time must be within operating hours %s-%s", schedule.OpenTime, schedule.CloseTime))
	}

	startAt, err := utils.CombineDateTime(bookingDate, req.StartTime)
	if err != nil {
		return nil, utils.BadRequest(err.Error())
	}
	if !startAt.After(timeNow()) {
		return nil, utils.Unprocessable("start_time has already passed")
	}

	// Total harga SELALU dihitung backend, tidak pernah diterima dari client.
	totalPrice := field.Price * float64(duration)

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
			return utils.Conflict("selected time slot is not available")
		}

		sequence, seqErr := r.Booking().NextSequence(ctx, timeNow())
		if seqErr != nil {
			return seqErr
		}

		booking = &models.Booking{
			BookingCode: BuildBookingCode(timeNow(), sequence),
			UserID:      actor.ID,
			FieldID:     field.ID,
			BookingDate: bookingDate,
			StartTime:   req.StartTime,
			EndTime:     req.EndTime,
			Duration:    duration,
			TotalPrice:  totalPrice,
			Status:      models.BookingStatusPending,
			Notes:       strings.TrimSpace(req.Notes),
			Items:       buildBookingItems(req.StartTime, req.EndTime, field.Price),
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

	invoice, err := c.invoices.CreateInvoice(ctx, utils.CreateInvoiceRequest{
		ExternalID:      booking.BookingCode,
		Amount:          totalPrice,
		Description:     fmt.Sprintf("Booking %s - %s (%s %s-%s)", booking.BookingCode, field.Name, req.BookingDate, req.StartTime, req.EndTime),
		InvoiceDuration: c.policy.InvoiceDuration,
		PayerEmail:      actor.Email,
		Customer:        &utils.Customer{Email: actor.Email},
		Items: []utils.Item{{
			Name:     fmt.Sprintf("%s (%d hour)", field.Name, duration),
			Quantity: duration,
			Price:    field.Price,
			Category: string(field.Type),
		}},
		SuccessRedirectURL: c.policy.SuccessRedirectURL,
		FailureRedirectURL: c.policy.FailureRedirectURL,
	})
	if err != nil {
		c.rollbackBooking(ctx, booking, payment)
		return nil, utils.BadGateway("failed to create payment invoice", err)
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
		BookingDate:   booking.BookingDate.Format(utils.DateLayout),
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
		userID := actor.ID
		filter.UserID = &userID
	}

	if query.From != "" {
		from, err := utils.ParseDate(query.From)
		if err != nil {
			return nil, 0, utils.BadRequest("from must use YYYY-MM-DD format")
		}
		filter.DateFrom = &from
	}
	if query.To != "" {
		to, err := utils.ParseDate(query.To)
		if err != nil {
			return nil, 0, utils.BadRequest("to must use YYYY-MM-DD format")
		}
		filter.DateTo = &to
	}

	bookings, total, err := c.uow.Booking().FindAll(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewBookingResponses(bookings), total, nil
}

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

// Cancel membatalkan booking sesuai aturan: booking yang sudah dibayar hanya
// dapat dibatalkan user minimal N jam sebelum jam mulai; admin bebas.
func (c *bookingController) Cancel(ctx context.Context, actor Actor, id uint) (*dto.BookingResponse, error) {
	booking, err := c.uow.Booking().FindDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ensureBookingOwner(actor, booking); err != nil {
		return nil, err
	}

	switch booking.Status {
	case models.BookingStatusCancelled:
		return nil, utils.Conflict("booking is already cancelled")
	case models.BookingStatusCompleted, models.BookingStatusExpired:
		return nil, utils.Conflict("booking can no longer be cancelled")
	}

	isPaid := booking.Status == models.BookingStatusPaid || booking.Status == models.BookingStatusConfirmed
	if isPaid && !actor.IsAdmin() {
		startAt, convErr := utils.CombineDateTime(booking.BookingDate, booking.StartTime)
		if convErr != nil {
			return nil, utils.Internal("failed to read booking schedule", convErr)
		}
		limit := time.Duration(c.policy.CancelMinHours) * time.Hour
		if startAt.Sub(timeNow()) < limit {
			return nil, utils.Unprocessable(fmt.Sprintf(
				"paid booking can only be cancelled at least %d hour(s) before start time", c.policy.CancelMinHours))
		}
	}

	err = c.uow.Atomic(ctx, func(r repository.Registry) error {
		now := timeNow()
		booking.Status = models.BookingStatusCancelled
		booking.CancelledAt = &now
		if updateErr := r.Booking().Update(ctx, booking); updateErr != nil {
			return updateErr
		}

		payment, findErr := r.Payment().FindByBookingID(ctx, booking.ID)
		if findErr != nil {
			if utils.IsNotFound(findErr) {
				return nil
			}
			return findErr
		}
		if payment.Status == models.PaymentStatusPending {
			payment.Status = models.PaymentStatusFailed
			return r.Payment().Update(ctx, payment)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return c.Detail(ctx, actor, booking.ID)
}

func (c *bookingController) UpdateStatus(ctx context.Context, id uint, status models.BookingStatus) (*dto.BookingResponse, error) {
	if !status.Valid() {
		return nil, utils.Unprocessable("invalid booking status")
	}

	booking, err := c.uow.Booking().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if booking.Status == status {
		return c.Detail(ctx, Actor{Role: string(models.RoleAdmin)}, id)
	}

	booking.Status = status
	if status == models.BookingStatusCancelled && booking.CancelledAt == nil {
		now := timeNow()
		booking.CancelledAt = &now
	}
	if err := c.uow.Booking().Update(ctx, booking); err != nil {
		return nil, err
	}
	return c.Detail(ctx, Actor{Role: string(models.RoleAdmin)}, id)
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

func ensureBookingOwner(actor Actor, booking *models.Booking) error {
	if actor.IsAdmin() || booking.UserID == actor.ID {
		return nil
	}
	// 404 dipilih agar keberadaan booking milik orang lain tidak bocor.
	return utils.NotFound("booking not found")
}

// BuildBookingCode menyusun kode booking unik berformat BK-YYYYMMDD-XXXX.
func BuildBookingCode(now time.Time, sequence int) string {
	return fmt.Sprintf("BK-%s-%04d", now.Format("20060102"), sequence)
}

func buildBookingItems(start, end string, pricePerHour float64) []models.BookingItem {
	slots := utils.HourlySlots(start, end)
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
