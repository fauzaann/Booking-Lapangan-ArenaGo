package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/timeutil"
	"Booking-Lapangan/pkg/xendit"
	"Booking-Lapangan/repository"
)

const maxCheckoutItems = 10

type CheckoutConflict struct {
	ItemID  string `json:"item_id"`
	Message string `json:"message"`
}

type CheckoutConflictError struct {
	Conflicts []CheckoutConflict
}

func (e *CheckoutConflictError) Error() string {
	return "one or more selected court times are no longer available"
}

type CheckoutController interface {
	Create(ctx context.Context, actor Actor, req dto.CheckoutRequest) (*dto.CheckoutCreatedResponse, error)
	Detail(ctx context.Context, actor Actor, id uint) (*dto.CheckoutResponse, error)
}

type checkoutController struct {
	uow      repository.UnitOfWork
	invoices xendit.InvoiceService
	policy   BookingPolicy
}

func NewCheckoutController(uow repository.UnitOfWork, invoices xendit.InvoiceService, policy BookingPolicy) CheckoutController {
	return &checkoutController{uow: uow, invoices: invoices, policy: policy}
}

type preparedCheckoutItem struct {
	request  dto.CheckoutItemRequest
	date     time.Time
	duration int
	field    *models.Field
	booking  *models.Booking
	payment  *models.Payment
}

type checkoutLock struct {
	fieldID uint
	date    time.Time
}

func (c *checkoutController) Create(ctx context.Context, actor Actor, req dto.CheckoutRequest) (*dto.CheckoutCreatedResponse, error) {
	if len(req.Items) == 0 || len(req.Items) > maxCheckoutItems {
		return nil, apperror.Unprocessable(fmt.Sprintf("cart must contain between 1 and %d bookings", maxCheckoutItems))
	}
	if c.invoices == nil {
		return nil, apperror.BadGateway("Xendit payment service is unavailable", errors.New("invoice service is not configured"))
	}

	items := make([]preparedCheckoutItem, len(req.Items))
	seenItemIDs := make(map[string]struct{}, len(req.Items))
	for i, item := range req.Items {
		item.ItemID = strings.TrimSpace(item.ItemID)
		if item.ItemID == "" {
			return nil, apperror.BadRequest("item_id is required for every cart item")
		}
		if _, exists := seenItemIDs[item.ItemID]; exists {
			return nil, apperror.BadRequest("cart item IDs must be unique")
		}
		seenItemIDs[item.ItemID] = struct{}{}
		date, err := timeutil.ParseDate(item.BookingDate)
		if err != nil {
			return nil, apperror.Unprocessable(fmt.Sprintf("cart item %s: %v", item.ItemID, err))
		}
		duration, err := timeutil.DurationHours(item.StartTime, item.EndTime)
		if err != nil {
			return nil, apperror.Unprocessable(fmt.Sprintf("cart item %s: %v", item.ItemID, err))
		}
		if duration < c.policy.MinHours || duration > c.policy.MaxHours {
			return nil, apperror.Unprocessable(fmt.Sprintf(
				"cart item %s duration must be between %d and %d hour(s)", item.ItemID, c.policy.MinHours, c.policy.MaxHours))
		}
		items[i] = preparedCheckoutItem{request: item, date: date, duration: duration}
	}

	if conflicts := findInternalCheckoutConflicts(items); len(conflicts) > 0 {
		return nil, &CheckoutConflictError{Conflicts: conflicts}
	}

	lockSet := make(map[string]checkoutLock, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%d:%s", item.request.FieldID, item.date.Format(timeutil.DateLayout))
		lockSet[key] = checkoutLock{fieldID: item.request.FieldID, date: item.date}
	}
	locks := make([]checkoutLock, 0, len(lockSet))
	for _, lock := range lockSet {
		locks = append(locks, lock)
	}
	sort.Slice(locks, func(i, j int) bool {
		if locks[i].fieldID == locks[j].fieldID {
			return locks[i].date.Before(locks[j].date)
		}
		return locks[i].fieldID < locks[j].fieldID
	})

	code, err := newCheckoutCode(timeNow())
	if err != nil {
		return nil, apperror.Internal("failed to create checkout code", err)
	}
	checkout := &models.Checkout{
		Code: code, ExternalID: code, UserID: actor.UserID,
		Status: models.PaymentStatusPending,
	}
	var conflicts []CheckoutConflict
	err = c.uow.Atomic(ctx, func(r repository.Registry) error {
		for _, lock := range locks {
			if err := r.Booking().LockFieldDate(ctx, lock.fieldID, lock.date); err != nil {
				return err
			}
		}

		conflicts = conflicts[:0]
		total := 0.0
		for i := range items {
			item := &items[i]
			field, err := r.Field().FindByID(ctx, item.request.FieldID)
			if err != nil {
				return err
			}
			if !field.IsActive() {
				return apperror.Unprocessable(fmt.Sprintf("cart item %s: field is not available for booking", item.request.ItemID))
			}
			day := models.DayFromWeekday(item.date.Weekday())
			schedule, err := r.Schedule().FindByFieldAndDay(ctx, field.ID, day)
			if err != nil {
				if apperror.IsNotFound(err) {
					return apperror.Unprocessable(fmt.Sprintf("cart item %s: field has no operating hours on %s", item.request.ItemID, day))
				}
				return err
			}
			if schedule.IsClosed || !timeutil.Within(item.request.StartTime, item.request.EndTime, schedule.OpenTime, schedule.CloseTime) {
				return apperror.Unprocessable(fmt.Sprintf("cart item %s: booking time is outside field operating hours", item.request.ItemID))
			}
			startAt, err := timeutil.CombineDateTime(item.date, item.request.StartTime)
			if err != nil {
				return apperror.BadRequest(err.Error())
			}
			if !startAt.After(timeNow()) {
				return apperror.Unprocessable(fmt.Sprintf("cart item %s: start_time has already passed", item.request.ItemID))
			}
			overlap, err := r.Booking().HasOverlap(ctx, field.ID, item.date, item.request.StartTime, item.request.EndTime)
			if err != nil {
				return err
			}
			if overlap {
				conflicts = append(conflicts, CheckoutConflict{
					ItemID: item.request.ItemID, Message: "Lapangan ini sudah ter-booking",
				})
				continue
			}
			item.field = field
			total += field.PricePerHour * float64(item.duration)
		}
		if len(conflicts) > 0 {
			return nil
		}

		checkout.Amount = total
		if err := r.Checkout().Create(ctx, checkout); err != nil {
			return err
		}
		for i := range items {
			item := &items[i]
			sequence, err := r.Booking().NextSequence(ctx, timeNow())
			if err != nil {
				return err
			}
			booking := &models.Booking{
				BookingCode: BuildBookingCode(timeNow(), sequence),
				UserID:      actor.UserID, FieldID: item.field.ID, BookingDate: item.date,
				StartTime: item.request.StartTime, EndTime: item.request.EndTime,
				Duration: item.duration, TotalPrice: item.field.PricePerHour * float64(item.duration),
				Status: models.BookingStatusPending,
				Items:  buildBookingItems(item.request.StartTime, item.request.EndTime, item.field.PricePerHour),
			}
			if err := r.Booking().Create(ctx, booking); err != nil {
				return err
			}
			checkoutID := checkout.ID
			payment := &models.Payment{
				BookingID: booking.ID, CheckoutID: &checkoutID,
				ExternalID: fmt.Sprintf("%s-%02d", checkout.Code, i+1),
				Amount:     booking.TotalPrice, Status: models.PaymentStatusPending,
			}
			if err := r.Payment().Create(ctx, payment); err != nil {
				return err
			}
			item.booking, item.payment = booking, payment
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(conflicts) > 0 {
		return nil, &CheckoutConflictError{Conflicts: conflicts}
	}

	invoiceItems := make([]xendit.Item, 0, len(items))
	var description strings.Builder
	for _, item := range items {
		if description.Len() > 0 {
			description.WriteString("; ")
		}
		description.WriteString(fmt.Sprintf("%s %s %s-%s", item.field.Name, item.date.Format(timeutil.DateLayout), item.request.StartTime, item.request.EndTime))
		invoiceItems = append(invoiceItems, xendit.Item{
			Name:     fmt.Sprintf("%s (%d hour)", item.field.Name, item.duration),
			Quantity: item.duration, Price: item.field.PricePerHour, Category: string(item.field.Type),
		})
	}
	successURL := appendCheckoutID(c.policy.SuccessRedirectURL, checkout.ID)
	failureURL := appendCheckoutID(c.policy.FailureRedirectURL, checkout.ID)
	invoice, err := c.invoices.CreateInvoice(ctx, xendit.CreateInvoiceRequest{
		ExternalID: checkout.ExternalID, Amount: checkout.Amount,
		Description: description.String(), InvoiceDuration: c.policy.InvoiceDuration,
		PayerEmail: actor.Email, Customer: &xendit.Customer{Email: actor.Email},
		Items: invoiceItems, SuccessRedirectURL: successURL, FailureRedirectURL: failureURL,
	})
	if err != nil {
		if rollbackErr := c.rollbackCheckout(ctx, checkout.ID); rollbackErr != nil {
			return nil, apperror.Internal("invoice creation failed and checkout cleanup also failed", errors.Join(err, rollbackErr))
		}
		return nil, paymentInvoiceError(err)
	}
	if invoice == nil {
		if rollbackErr := c.rollbackCheckout(ctx, checkout.ID); rollbackErr != nil {
			return nil, apperror.Internal("Xendit returned an empty invoice and checkout cleanup also failed", rollbackErr)
		}
		return nil, apperror.BadGateway("Xendit returned an empty invoice", errors.New("invoice response is nil"))
	}
	expiredAt := invoice.ExpiryDate
	if expiredAt == nil {
		fallback := timeNow().Add(time.Duration(c.policy.InvoiceDuration) * time.Second)
		expiredAt = &fallback
	}

	err = c.uow.Atomic(ctx, func(r repository.Registry) error {
		checkout.InvoiceID = invoice.ID
		checkout.InvoiceURL = invoice.InvoiceURL
		checkout.ExpiredAt = expiredAt
		if err := r.Checkout().Update(ctx, checkout); err != nil {
			return err
		}
		payments, err := r.Payment().FindByCheckoutID(ctx, checkout.ID)
		if err != nil {
			return err
		}
		for i := range payments {
			payments[i].InvoiceID = invoice.ID
			payments[i].InvoiceURL = invoice.InvoiceURL
			payments[i].ExpiredAt = expiredAt
			if err := r.Payment().Update(ctx, &payments[i]); err != nil {
				return err
			}
			booking, err := r.Booking().FindByID(ctx, payments[i].BookingID)
			if err != nil {
				return err
			}
			booking.Status = models.BookingStatusWaitingPayment
			if err := r.Booking().Update(ctx, booking); err != nil {
				return err
			}
			for j := range items {
				if items[j].booking.ID == booking.ID {
					items[j].booking = booking
					items[j].payment = &payments[i]
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	bookings := make([]dto.BookingResponse, 0, len(items))
	for i := range items {
		items[i].booking.Payment = items[i].payment
		bookings = append(bookings, dto.NewBookingResponse(*items[i].booking))
	}
	return &dto.CheckoutCreatedResponse{
		CheckoutID: checkout.ID, CheckoutCode: checkout.Code, TotalPrice: checkout.Amount,
		PaymentURL: invoice.InvoiceURL, PaymentStatus: string(models.PaymentStatusPending),
		ExpiredAt: expiredAt, Bookings: bookings,
	}, nil
}

func (c *checkoutController) Detail(ctx context.Context, actor Actor, id uint) (*dto.CheckoutResponse, error) {
	checkout, err := c.uow.Checkout().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if checkout.UserID != actor.UserID && !actor.IsAdmin() {
		return nil, apperror.NotFound("checkout not found")
	}
	bookings, err := c.uow.Booking().FindByCheckoutID(ctx, checkout.ID)
	if err != nil {
		return nil, err
	}
	payments, err := c.uow.Payment().FindByCheckoutID(ctx, checkout.ID)
	if err != nil {
		return nil, err
	}
	status := string(checkout.Status)
	responses := dto.NewBookingResponses(bookings)
	for i := range responses {
		for j := range payments {
			if payments[j].BookingID == responses[i].ID {
				payment := dto.NewPaymentResponse(payments[j])
				responses[i].Payment = &payment
				break
			}
		}
	}
	return &dto.CheckoutResponse{
		CheckoutID: checkout.ID, CheckoutCode: checkout.Code, TotalPrice: checkout.Amount,
		PaymentURL: checkout.InvoiceURL, PaymentStatus: status, ExpiredAt: checkout.ExpiredAt,
		Bookings: responses,
	}, nil
}

func (c *checkoutController) rollbackCheckout(ctx context.Context, checkoutID uint) error {
	cleanupCtx := context.WithoutCancel(ctx)
	err := c.uow.Atomic(cleanupCtx, func(r repository.Registry) error {
		checkout, err := r.Checkout().FindByID(cleanupCtx, checkoutID)
		if err != nil {
			return err
		}
		checkout.Status = models.PaymentStatusFailed
		if err := r.Checkout().Update(cleanupCtx, checkout); err != nil {
			return err
		}
		bookings, err := r.Booking().FindByCheckoutID(cleanupCtx, checkoutID)
		if err != nil {
			return err
		}
		payments, err := r.Payment().FindByCheckoutID(cleanupCtx, checkoutID)
		if err != nil {
			return err
		}
		for i := range payments {
			payments[i].Status = models.PaymentStatusFailed
			if err := r.Payment().Update(cleanupCtx, &payments[i]); err != nil {
				return err
			}
		}
		for i := range bookings {
			bookings[i].Status = models.BookingStatusCancelled
			if err := r.Booking().Update(cleanupCtx, &bookings[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("checkout: failed to roll back checkout %d: %v", checkoutID, err)
	}
	return err
}

func findInternalCheckoutConflicts(items []preparedCheckoutItem) []CheckoutConflict {
	var conflicts []CheckoutConflict
	for i := range items {
		for j := 0; j < i; j++ {
			left, right := items[i], items[j]
			if left.request.FieldID != right.request.FieldID || !left.date.Equal(right.date) {
				continue
			}
			if timeutil.Overlap(left.request.StartTime, left.request.EndTime, right.request.StartTime, right.request.EndTime) {
				conflicts = append(conflicts, CheckoutConflict{
					ItemID: left.request.ItemID, Message: "Jadwal di keranjang bertabrakan dengan item lain",
				})
				break
			}
		}
	}
	return conflicts
}

func newCheckoutCode(now time.Time) (string, error) {
	random := make([]byte, 5)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return fmt.Sprintf("CO-%s-%s", now.Format("20060102"), strings.ToUpper(hex.EncodeToString(random))), nil
}

func appendCheckoutID(base string, checkoutID uint) string {
	if base == "" {
		return ""
	}
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%scheckout_id=%d", base, separator, checkoutID)
}
