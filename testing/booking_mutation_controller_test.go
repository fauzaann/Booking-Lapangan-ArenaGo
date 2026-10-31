package controllers_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/timeutil"
	"Booking-Lapangan/repository"
)

type mutationBookingRepo struct {
	bookings  map[uint]*models.Booking
	overlap   bool
	updateErr error
}

func (r *mutationBookingRepo) Create(context.Context, *models.Booking) error { return nil }
func (r *mutationBookingRepo) Update(_ context.Context, booking *models.Booking) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.bookings[booking.ID] = cloneMutationBooking(booking)
	return nil
}
func (r *mutationBookingRepo) UpdateSchedule(_ context.Context, booking *models.Booking) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.bookings[booking.ID] = cloneMutationBooking(booking)
	return nil
}
func (r *mutationBookingRepo) FindByID(_ context.Context, id uint) (*models.Booking, error) {
	return r.find(id)
}
func (r *mutationBookingRepo) FindDetailByID(_ context.Context, id uint) (*models.Booking, error) {
	return r.find(id)
}
func (r *mutationBookingRepo) FindDetailByIDForUpdate(_ context.Context, id uint) (*models.Booking, error) {
	return r.find(id)
}
func (r *mutationBookingRepo) find(id uint) (*models.Booking, error) {
	booking, ok := r.bookings[id]
	if !ok {
		return nil, apperror.NotFound("booking not found")
	}
	return cloneMutationBooking(booking), nil
}
func (r *mutationBookingRepo) FindByCode(_ context.Context, code string) (*models.Booking, error) {
	for _, booking := range r.bookings {
		if booking.BookingCode == code {
			return cloneMutationBooking(booking), nil
		}
	}
	return nil, apperror.NotFound("booking not found")
}
func (r *mutationBookingRepo) FindByCheckoutID(context.Context, uint) ([]models.Booking, error) {
	return nil, nil
}
func (r *mutationBookingRepo) FindAll(context.Context, repository.BookingFilter) ([]models.Booking, int64, error) {
	return nil, 0, nil
}
func (r *mutationBookingRepo) FindActiveByFieldAndDate(context.Context, uint, time.Time) ([]models.Booking, error) {
	return nil, nil
}
func (r *mutationBookingRepo) HasOverlap(_ context.Context, fieldID uint, date time.Time, start, end string) (bool, error) {
	return r.hasOverlap(fieldID, date, start, end, 0), nil
}
func (r *mutationBookingRepo) HasOverlapExcluding(_ context.Context, fieldID uint, date time.Time, start, end string, excluded uint) (bool, error) {
	return r.hasOverlap(fieldID, date, start, end, excluded), nil
}
func (r *mutationBookingRepo) hasOverlap(fieldID uint, date time.Time, start, end string, excluded uint) bool {
	if r.overlap {
		return true
	}
	for id, booking := range r.bookings {
		if id == excluded || booking.FieldID != fieldID ||
			!booking.BookingDate.Equal(date) || !booking.Status.BlocksSlot() {
			continue
		}
		if booking.StartTime < end && booking.EndTime > start {
			return true
		}
	}
	return false
}
func (r *mutationBookingRepo) LockFieldDate(context.Context, uint, time.Time) error {
	return nil
}
func (r *mutationBookingRepo) NextSequence(context.Context, time.Time) (int, error) {
	return 1, nil
}
func (r *mutationBookingRepo) Count(context.Context) (int64, error) { return 0, nil }
func (r *mutationBookingRepo) CountByStatus(context.Context) (map[models.BookingStatus]int64, error) {
	return nil, nil
}
func (r *mutationBookingRepo) MarkExpired(context.Context, []uint) (int64, error) {
	return 0, nil
}
func (r *mutationBookingRepo) CompleteFinished(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type mutationUnitOfWork struct {
	tx          sync.Mutex
	bookingRepo *mutationBookingRepo
	fieldRepo   repository.FieldRepository
	schedule    repository.ScheduleRepository
	payment     repository.PaymentRepository
}

type mutationNotifier struct {
	controllers.NotificationController
	cancelled   int
	rescheduled int
	afterCommit func()
}

func (n *mutationNotifier) CancelRemindersForBooking(context.Context, uint) error {
	n.cancelled++
	if n.afterCommit != nil {
		n.afterCommit()
	}
	return nil
}
func (n *mutationNotifier) RescheduleBooking(context.Context, uint) error {
	n.rescheduled++
	if n.afterCommit != nil {
		n.afterCommit()
	}
	return nil
}

func (u *mutationUnitOfWork) User() repository.UserRepository { return nil }
func (u *mutationUnitOfWork) Field() repository.FieldRepository {
	return u.fieldRepo
}
func (u *mutationUnitOfWork) Schedule() repository.ScheduleRepository {
	return u.schedule
}
func (u *mutationUnitOfWork) Booking() repository.BookingRepository {
	return u.bookingRepo
}
func (u *mutationUnitOfWork) Rating() repository.RatingRepository { return nil }
func (u *mutationUnitOfWork) Payment() repository.PaymentRepository {
	return u.payment
}
func (u *mutationUnitOfWork) Checkout() repository.CheckoutRepository { return nil }
func (u *mutationUnitOfWork) AuditLog() repository.AuditLogRepository { return nil }
func (u *mutationUnitOfWork) Atomic(ctx context.Context, fn func(repository.Registry) error) error {
	u.tx.Lock()
	defer u.tx.Unlock()

	snapshot := make(map[uint]*models.Booking, len(u.bookingRepo.bookings))
	for id, booking := range u.bookingRepo.bookings {
		snapshot[id] = cloneMutationBooking(booking)
	}
	err := fn(u)
	if err != nil {
		u.bookingRepo.bookings = snapshot
	}
	return err
}

func newMutationController(bookings ...*models.Booking) (controllers.BookingController, *mutationUnitOfWork) {
	stored := make(map[uint]*models.Booking, len(bookings))
	for _, booking := range bookings {
		stored[booking.ID] = cloneMutationBooking(booking)
	}
	field := &models.Field{
		ID: 1, Name: "Arena Utama", PricePerHour: 100000,
		Status: models.FieldStatusActive,
	}
	uow := &mutationUnitOfWork{
		bookingRepo: &mutationBookingRepo{bookings: stored},
		fieldRepo:   &mockFieldRepo{field: field},
		schedule: &mockScheduleRepo{schedule: &models.Schedule{
			FieldID: 1, Day: models.DayMonday, OpenTime: "06:00", CloseTime: "23:00",
		}},
		payment: &testPaymentRepo{},
	}
	controller := controllers.NewBookingController(uow, nil, controllers.BookingPolicy{
		MinHours: 1, MaxHours: 8, CancelMinHours: 24,
	})
	return controller, uow
}

func futureBookingDate(days int) time.Time {
	date := time.Now().AddDate(0, 0, days)
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

func mutationBooking(id, userID uint, status models.BookingStatus, date time.Time, start, end string) *models.Booking {
	duration, _ := timeutil.DurationHours(start, end)
	return &models.Booking{
		ID: id, BookingCode: "AG-TEST-" + string(rune('0'+id)),
		UserID: userID, FieldID: 1, BookingDate: date,
		StartTime: start, EndTime: end, Duration: duration,
		TotalPrice: float64(duration) * 100000, Status: status,
		Items: []models.BookingItem{{StartTime: start, EndTime: end, Price: 100000}},
	}
}

func cloneMutationBooking(booking *models.Booking) *models.Booking {
	copy := *booking
	copy.Items = append([]models.BookingItem(nil), booking.Items...)
	copy.RescheduleHistory = append([]models.BookingScheduleChange(nil), booking.RescheduleHistory...)
	if booking.Field != nil {
		field := *booking.Field
		copy.Field = &field
	}
	if booking.Payment != nil {
		payment := *booking.Payment
		copy.Payment = &payment
	}
	return &copy
}

func TestBookingCancelValidAndPaymentHandling(t *testing.T) {
	booking := mutationBooking(1, 7, models.BookingStatusWaitingPayment, futureBookingDate(2), "10:00", "12:00")
	controller, uow := newMutationController(booking)
	notifier := &mutationNotifier{afterCommit: func() {
		if uow.bookingRepo.bookings[booking.ID].Status != models.BookingStatusCancelled {
			t.Error("cancellation notification ran before transaction commit")
		}
	}}
	controller.SetNotificationController(notifier)
	paymentRepo := uow.payment.(*testPaymentRepo)
	paymentRepo.payment = &models.Payment{
		BookingID: booking.ID, Status: models.PaymentStatusPending,
	}

	got, err := controller.CancelWithReason(context.Background(), controllers.Actor{UserID: 7}, booking.ID, "Perubahan rencana")
	if err != nil {
		t.Fatalf("CancelWithReason() error = %v", err)
	}
	if got.Status != string(models.BookingStatusCancelled) || got.CancellationReason != "Perubahan rencana" {
		t.Fatalf("unexpected cancellation result: %+v", got)
	}
	if paymentRepo.updatedWith == nil || paymentRepo.updatedWith.Status != models.PaymentStatusFailed {
		t.Fatalf("pending payment should be marked failed, got %+v", paymentRepo.updatedWith)
	}
	if uow.bookingRepo.bookings[booking.ID].CancelledAt == nil {
		t.Fatal("expected cancellation timestamp to be saved")
	}
	if notifier.cancelled != 1 {
		t.Fatalf("expected one post-commit cancellation notification, got %d", notifier.cancelled)
	}
}

func TestBookingCancelRejectsNonOwner(t *testing.T) {
	booking := mutationBooking(2, 7, models.BookingStatusPending, futureBookingDate(2), "10:00", "12:00")
	controller, uow := newMutationController(booking)

	_, err := controller.Cancel(context.Background(), controllers.Actor{UserID: 8}, booking.ID)
	if err == nil {
		t.Fatal("expected non-owner cancellation to fail")
	}
	if uow.bookingRepo.bookings[booking.ID].Status != models.BookingStatusPending {
		t.Fatal("non-owner request changed booking")
	}
}

func TestBookingCancelPaidDeadlineAndRefundReview(t *testing.T) {
	t.Run("within deadline is rejected", func(t *testing.T) {
		booking := mutationBooking(3, 7, models.BookingStatusPaid, futureBookingDate(1), "10:00", "12:00")
		booking.BookingDate = time.Now().Add(20 * time.Hour)
		booking.BookingDate = time.Date(booking.BookingDate.Year(), booking.BookingDate.Month(), booking.BookingDate.Day(), 0, 0, 0, 0, time.Now().Location())
		booking.StartTime = time.Now().Add(20 * time.Hour).Format("15:04")
		controller, uow := newMutationController(booking)
		_, err := controller.Cancel(context.Background(), controllers.Actor{UserID: 7}, booking.ID)
		if err == nil {
			t.Fatal("expected paid booking inside cancellation deadline to be rejected")
		}
		if uow.bookingRepo.bookings[booking.ID].Status != models.BookingStatusPaid {
			t.Fatal("booking status changed after rejected cancellation")
		}
	})

	t.Run("paid booking is flagged without refund", func(t *testing.T) {
		booking := mutationBooking(4, 7, models.BookingStatusPaid, futureBookingDate(3), "10:00", "12:00")
		controller, uow := newMutationController(booking)
		paymentRepo := uow.payment.(*testPaymentRepo)
		paymentRepo.payment = &models.Payment{BookingID: booking.ID, Status: models.PaymentStatusPaid}

		got, err := controller.Cancel(context.Background(), controllers.Actor{UserID: 7}, booking.ID)
		if err != nil {
			t.Fatalf("Cancel() error = %v", err)
		}
		if !got.RefundReviewRequired {
			t.Fatal("expected paid cancellation to be flagged for manual refund review")
		}
		if paymentRepo.updatedWith != nil || paymentRepo.payment.Status != models.PaymentStatusPaid {
			t.Fatal("paid payment status must remain unchanged")
		}
	})
}

func TestBookingCancelRejectsUnchangeableBookings(t *testing.T) {
	for _, status := range []models.BookingStatus{
		models.BookingStatusCancelled, models.BookingStatusCompleted, models.BookingStatusExpired,
	} {
		t.Run(string(status), func(t *testing.T) {
			booking := mutationBooking(5, 7, status, futureBookingDate(2), "10:00", "12:00")
			controller, _ := newMutationController(booking)
			if _, err := controller.Cancel(context.Background(), controllers.Actor{UserID: 7}, booking.ID); err == nil {
				t.Fatalf("expected %s booking cancellation to fail", status)
			}
		})
	}

	now := time.Now()
	startedDate := now.Add(-2 * time.Hour)
	started := mutationBooking(15, 7, models.BookingStatusConfirmed,
		time.Date(startedDate.Year(), startedDate.Month(), startedDate.Day(), 0, 0, 0, 0, startedDate.Location()),
		startedDate.Format("15:04"), now.Add(time.Hour).Format("15:04"))
	controller, _ := newMutationController(started)
	if _, err := controller.Cancel(context.Background(), controllers.Actor{UserID: 7}, started.ID); err == nil {
		t.Fatal("expected already-started booking cancellation to fail")
	}

	checkedIn := mutationBooking(16, 7, models.BookingStatusConfirmed, futureBookingDate(2), "10:00", "12:00")
	now = time.Now()
	checkedIn.CheckedInAt = &now
	controller, _ = newMutationController(checkedIn)
	if _, err := controller.Cancel(context.Background(), controllers.Actor{UserID: 7}, checkedIn.ID); !apperror.IsConflict(err) {
		t.Fatalf("expected checked-in booking cancellation to fail with conflict, got %v", err)
	}
}

func TestBookingRescheduleAvailableSlot(t *testing.T) {
	date := futureBookingDate(3)
	booking := mutationBooking(6, 7, models.BookingStatusConfirmed, date, "10:00", "12:00")
	booking.Payment = &models.Payment{BookingID: booking.ID, Status: models.PaymentStatusPaid}
	controller, uow := newMutationController(booking)
	notifier := &mutationNotifier{afterCommit: func() {
		if uow.bookingRepo.bookings[booking.ID].StartTime != "14:00" {
			t.Error("reschedule notification ran before transaction commit")
		}
	}}
	controller.SetNotificationController(notifier)

	got, err := controller.Reschedule(context.Background(), controllers.Actor{UserID: 7}, booking.ID, dto.RescheduleBookingRequest{
		BookingDate: date.Format(timeutil.DateLayout), StartTime: "14:00", EndTime: "16:00",
	})
	if err != nil {
		t.Fatalf("Reschedule() error = %v", err)
	}
	if got.StartTime != "14:00" || got.EndTime != "16:00" || got.Status != string(models.BookingStatusConfirmed) {
		t.Fatalf("unexpected rescheduled booking: %+v", got)
	}
	stored := uow.bookingRepo.bookings[booking.ID]
	if len(stored.RescheduleHistory) != 1 || stored.RescheduleHistory[0].PreviousStartTime != "10:00" {
		t.Fatalf("schedule history was not recorded: %+v", stored.RescheduleHistory)
	}
	if stored.Payment == nil || stored.Payment.Status != models.PaymentStatusPaid {
		t.Fatal("rescheduling changed payment status")
	}
	if notifier.rescheduled != 1 {
		t.Fatalf("expected one post-commit reschedule notification, got %d", notifier.rescheduled)
	}
}

func TestBookingRescheduleRejectsOverlapPriceChangeAndFailedUpdate(t *testing.T) {
	date := futureBookingDate(4)
	req := dto.RescheduleBookingRequest{BookingDate: date.Format(timeutil.DateLayout), StartTime: "14:00", EndTime: "16:00"}

	t.Run("occupied slot", func(t *testing.T) {
		booking := mutationBooking(7, 7, models.BookingStatusPending, date, "10:00", "12:00")
		controller, uow := newMutationController(booking)
		uow.bookingRepo.overlap = true
		if _, err := controller.Reschedule(context.Background(), controllers.Actor{UserID: 7}, booking.ID, req); err == nil {
			t.Fatal("expected occupied slot to be rejected")
		}
		if uow.bookingRepo.bookings[booking.ID].StartTime != "10:00" {
			t.Fatal("booking changed after overlap rejection")
		}
	})

	t.Run("price change", func(t *testing.T) {
		booking := mutationBooking(8, 7, models.BookingStatusPending, date, "10:00", "12:00")
		controller, uow := newMutationController(booking)
		reqDifferentDuration := dto.RescheduleBookingRequest{
			BookingDate: date.Format(timeutil.DateLayout), StartTime: "14:00", EndTime: "17:00",
		}
		if _, err := controller.Reschedule(context.Background(), controllers.Actor{UserID: 7}, booking.ID, reqDifferentDuration); err == nil {
			t.Fatal("expected unsupported price change to be rejected")
		}
		if uow.bookingRepo.bookings[booking.ID].StartTime != "10:00" {
			t.Fatal("booking changed after price-change rejection")
		}
	})

	t.Run("persistence failure", func(t *testing.T) {
		booking := mutationBooking(9, 7, models.BookingStatusPending, date, "10:00", "12:00")
		controller, uow := newMutationController(booking)
		uow.bookingRepo.updateErr = errors.New("simulated database failure")
		if _, err := controller.Reschedule(context.Background(), controllers.Actor{UserID: 7}, booking.ID, req); err == nil {
			t.Fatal("expected persistence error")
		}
		stored := uow.bookingRepo.bookings[booking.ID]
		if stored.StartTime != "10:00" || len(stored.RescheduleHistory) != 0 {
			t.Fatalf("failed update was persisted: %+v", stored)
		}
	})
}

func TestBookingRescheduleRejectsWrongOwnerAndFinalStates(t *testing.T) {
	for _, test := range []struct {
		name   string
		userID uint
		status models.BookingStatus
	}{
		{name: "wrong owner", userID: 8, status: models.BookingStatusPending},
		{name: "cancelled", userID: 7, status: models.BookingStatusCancelled},
		{name: "completed", userID: 7, status: models.BookingStatusCompleted},
		{name: "expired", userID: 7, status: models.BookingStatusExpired},
	} {
		t.Run(test.name, func(t *testing.T) {
			date := futureBookingDate(5)
			booking := mutationBooking(10, 7, test.status, date, "10:00", "12:00")
			controller, _ := newMutationController(booking)
			_, err := controller.Reschedule(context.Background(), controllers.Actor{UserID: test.userID}, booking.ID, dto.RescheduleBookingRequest{
				BookingDate: date.Format(timeutil.DateLayout), StartTime: "14:00", EndTime: "16:00",
			})
			if err == nil {
				t.Fatal("expected reschedule to fail")
			}
		})
	}
}

func TestConcurrentReschedulesCannotTakeSameSlot(t *testing.T) {
	date := futureBookingDate(6)
	first := mutationBooking(11, 7, models.BookingStatusPending, date, "10:00", "11:00")
	second := mutationBooking(12, 8, models.BookingStatusPending, date, "12:00", "13:00")
	controller, uow := newMutationController(first, second)
	req := dto.RescheduleBookingRequest{
		BookingDate: date.Format(timeutil.DateLayout), StartTime: "14:00", EndTime: "15:00",
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, booking := range []*models.Booking{first, second} {
		wg.Add(1)
		go func(bookingID uint) {
			defer wg.Done()
			actorID := first.UserID
			if bookingID == second.ID {
				actorID = second.UserID
			}
			_, err := controller.Reschedule(context.Background(), controllers.Actor{UserID: actorID}, bookingID, req)
			errs <- err
		}(booking.ID)
	}
	wg.Wait()
	close(errs)

	succeeded, rejected := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case apperror.IsConflict(err):
			rejected++
		default:
			t.Fatalf("unexpected concurrent reschedule error: %v", err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("expected one success and one conflict, got %d success and %d conflict", succeeded, rejected)
	}
	if uow.bookingRepo.bookings[first.ID].StartTime == uow.bookingRepo.bookings[second.ID].StartTime {
		t.Fatal("both bookings acquired the same slot")
	}
}

func TestBookingCheckInRequiresAdminAndPaidUnusedTicket(t *testing.T) {
	token := "secure-ticket-token"
	booking := mutationBooking(13, 7, models.BookingStatusConfirmed, futureBookingDate(2), "10:00", "12:00")
	booking.CheckInToken = &token
	booking.Payment = &models.Payment{BookingID: booking.ID, Status: models.PaymentStatusPaid}
	controller, uow := newMutationController(booking)

	_, err := controller.CheckIn(context.Background(), controllers.Actor{UserID: 8, Role: models.RoleUser}, dto.CheckInRequest{
		BookingCode: booking.BookingCode, CheckInToken: token,
	})
	if err == nil {
		t.Fatal("expected a non-admin check-in to be rejected")
	}
	if uow.bookingRepo.bookings[booking.ID].CheckedInAt != nil {
		t.Fatal("non-admin check-in changed booking state")
	}

	got, err := controller.CheckIn(context.Background(), controllers.Actor{UserID: 99, Role: models.RoleAdmin}, dto.CheckInRequest{
		BookingCode: booking.BookingCode, CheckInToken: token,
	})
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if got.CheckedInAt == nil || got.CheckInToken != "" {
		t.Fatalf("unexpected check-in response: %+v", got)
	}
	stored := uow.bookingRepo.bookings[booking.ID]
	if stored.CheckedInAt == nil || stored.CheckedInBy == nil || *stored.CheckedInBy != 99 {
		t.Fatalf("check-in details were not saved: %+v", stored)
	}

	_, err = controller.CheckIn(context.Background(), controllers.Actor{UserID: 99, Role: models.RoleAdmin}, dto.CheckInRequest{
		BookingCode: booking.BookingCode, CheckInToken: token,
	})
	if !apperror.IsConflict(err) {
		t.Fatalf("second check-in error = %v, want conflict", err)
	}
}
