package controllers

import (
	"context"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

// AdminController menyediakan data ringkas untuk kebutuhan administrator.
type AdminController interface {
	Dashboard(ctx context.Context) (*dto.DashboardResponse, error)
	ListUsers(ctx context.Context, query dto.PaginationQuery) ([]dto.UserResponse, int64, error)
}

type adminController struct {
	uow repository.UnitOfWork
}

// NewAdminController membuat AdminController.
func NewAdminController(uow repository.UnitOfWork) AdminController {
	return &adminController{uow: uow}
}

// Dashboard merangkum jumlah user, lapangan, booking per status, dan revenue.
// Revenue dihitung dari pembayaran berstatus PAID, bukan dari total booking.
func (c *adminController) Dashboard(ctx context.Context) (*dto.DashboardResponse, error) {
	totalUsers, err := c.uow.User().Count(ctx, &models.User{})
	if err != nil {
		return nil, err
	}

	totalFields, err := c.uow.Field().Count(ctx)
	if err != nil {
		return nil, err
	}

	totalBookings, err := c.uow.Booking().Count(ctx)
	if err != nil {
		return nil, err
	}

	byStatus, err := c.uow.Booking().CountByStatus(ctx)
	if err != nil {
		return nil, err
	}

	revenue, err := c.uow.Payment().SumPaidAmount(ctx)
	if err != nil {
		return nil, err
	}

	pending := byStatus[models.BookingStatusPending] + byStatus[models.BookingStatusWaitingPayment]

	return &dto.DashboardResponse{
		TotalUsers:        totalUsers,
		TotalFields:       totalFields,
		TotalBookings:     totalBookings,
		TotalRevenue:      revenue,
		PendingBookings:   pending,
		ConfirmedBookings: byStatus[models.BookingStatusConfirmed] + byStatus[models.BookingStatusPaid],
		CompletedBookings: byStatus[models.BookingStatusCompleted],
		CancelledBookings: byStatus[models.BookingStatusCancelled],
		ExpiredBookings:   byStatus[models.BookingStatusExpired],
	}, nil
}

func (c *adminController) ListUsers(ctx context.Context, query dto.PaginationQuery) ([]dto.UserResponse, int64, error) {
	query = query.Normalize()

	// 1. Fungsi Count meminta (*models.User) -> Sesuai dengan repository Anda
	total, err := c.uow.User().Count(ctx, &models.User{})
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []dto.UserResponse{}, 0, nil
	}

	// 2. Fungsi FindAll meminta (string) -> Kita kirim string kosong "" untuk mengambil semua user
	// Jika di masa depan dto.PaginationQuery Anda punya properti .Search, Anda bisa mengirim query.Search ke sini.
	users, err := c.uow.User().FindAll(ctx, "")
	if err != nil {
		return nil, 0, err
	}

	return dto.NewUserResponses(users), total, nil
}
