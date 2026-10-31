package controllers_test

import (
	"context"
	"testing"
	"time"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

type analyticsMockUOW struct {
	payments *analyticsPaymentRepo
}

func (u *analyticsMockUOW) User() repository.UserRepository         { return nil }
func (u *analyticsMockUOW) Field() repository.FieldRepository       { return nil }
func (u *analyticsMockUOW) Schedule() repository.ScheduleRepository { return nil }
func (u *analyticsMockUOW) Booking() repository.BookingRepository   { return nil }
func (u *analyticsMockUOW) Rating() repository.RatingRepository     { return nil }
func (u *analyticsMockUOW) Payment() repository.PaymentRepository   { return u.payments }
func (u *analyticsMockUOW) Checkout() repository.CheckoutRepository { return nil }
func (u *analyticsMockUOW) AuditLog() repository.AuditLogRepository { return nil }
func (u *analyticsMockUOW) Atomic(_ context.Context, fn func(repository.Registry) error) error {
	return fn(u)
}

type analyticsPaymentRepo struct {
	from, to time.Time
	calls    int
	data     *repository.RevenueAnalytics
}

func (*analyticsPaymentRepo) Create(context.Context, *models.Payment) error { return nil }
func (*analyticsPaymentRepo) Update(context.Context, *models.Payment) error { return nil }
func (*analyticsPaymentRepo) FindByID(context.Context, uint) (*models.Payment, error) {
	return nil, nil
}
func (*analyticsPaymentRepo) FindByExternalID(context.Context, string) (*models.Payment, error) {
	return nil, nil
}
func (*analyticsPaymentRepo) FindByBookingID(context.Context, uint) (*models.Payment, error) {
	return nil, nil
}
func (*analyticsPaymentRepo) FindByCheckoutID(context.Context, uint) ([]models.Payment, error) {
	return nil, nil
}
func (*analyticsPaymentRepo) FindAll(context.Context, repository.PaymentFilter) ([]models.Payment, int64, error) {
	return nil, 0, nil
}
func (*analyticsPaymentRepo) SumPaidAmount(context.Context) (float64, error) { return 0, nil }
func (r *analyticsPaymentRepo) RevenueAnalytics(_ context.Context, from, to time.Time) (*repository.RevenueAnalytics, error) {
	r.from, r.to = from, to
	r.calls++
	return r.data, nil
}
func (*analyticsPaymentRepo) ExpireOverdue(context.Context, time.Time) ([]uint, error) {
	return nil, nil
}

func TestAdminRevenueAnalyticsReturnsAnnualTotalsAndAllMonths(t *testing.T) {
	payments := &analyticsPaymentRepo{data: &repository.RevenueAnalytics{
		ByLocation: []repository.RevenueGroup{{Name: "Jakarta", Bookings: 3, Revenue: 400}},
		ByField:    []repository.RevenueGroup{{ID: 7, Name: "Court 7", Location: "Jakarta", Bookings: 3, Revenue: 400}},
		ByMonth: []repository.MonthlyRevenue{
			{Month: 1, Bookings: 1, Revenue: 100},
			{Month: 3, Bookings: 2, Revenue: 300},
		},
		BySlot: []repository.BestSellingSlot{{StartTime: "19:00", Bookings: 4, Revenue: 500}},
	}}
	admin := controllers.NewAdminController(&analyticsMockUOW{payments: payments})

	result, err := admin.RevenueAnalytics(context.Background(), 2025)
	if err != nil {
		t.Fatalf("RevenueAnalytics returned error: %v", err)
	}
	if result.Year != 2025 || result.TotalRevenue != 400 || result.PaidBookings != 3 {
		t.Fatalf("unexpected annual summary: %+v", result)
	}
	if len(result.ByMonth) != 12 || result.ByMonth[0].Revenue != 100 || result.ByMonth[1].Month != 2 || result.ByMonth[1].Revenue != 0 {
		t.Fatalf("expected all months, including zero-value months: %+v", result.ByMonth)
	}
	if len(result.ByLocation) != 1 || result.ByLocation[0].Name != "Jakarta" {
		t.Fatalf("unexpected location report: %+v", result.ByLocation)
	}
	if len(result.ByField) != 1 || result.ByField[0].ID != 7 {
		t.Fatalf("unexpected field report: %+v", result.ByField)
	}
	if len(result.BestSellingSlots) != 1 || result.BestSellingSlots[0].StartTime != "19:00" {
		t.Fatalf("unexpected slot report: %+v", result.BestSellingSlots)
	}
	if payments.calls != 1 || payments.from.Year() != 2025 || payments.from.Month() != time.January || payments.from.Day() != 1 ||
		payments.to.Year() != 2026 || payments.to.Month() != time.January || payments.to.Day() != 1 {
		t.Fatalf("unexpected analytics date range: from %v to %v, calls %d", payments.from, payments.to, payments.calls)
	}
}

func TestAdminRevenueAnalyticsRejectsInvalidYear(t *testing.T) {
	payments := &analyticsPaymentRepo{data: &repository.RevenueAnalytics{}}
	admin := controllers.NewAdminController(&analyticsMockUOW{payments: payments})

	for _, year := range []int{-1, 9999} {
		if _, err := admin.RevenueAnalytics(context.Background(), year); err == nil {
			t.Errorf("RevenueAnalytics(%d) expected error", year)
		}
	}
	if payments.calls != 0 {
		t.Fatalf("invalid years should not query analytics, got %d calls", payments.calls)
	}
}
