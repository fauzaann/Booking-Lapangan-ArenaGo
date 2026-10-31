package controllers

import (
	"context"
	"time"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/repository"
)

// AdminController menyediakan data ringkas untuk kebutuhan administrator.
type AdminController interface {
	Dashboard(ctx context.Context) (*dto.DashboardResponse, error)
	RevenueAnalytics(ctx context.Context, year int) (*dto.RevenueAnalyticsResponse, error)
	ListUsers(ctx context.Context, query dto.PaginationQuery) ([]dto.UserResponse, int64, error)
	ListAuditLogs(ctx context.Context, query dto.PaginationQuery) ([]dto.AuditLogResponse, int64, error)
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
	totalUsers, err := c.uow.User().Count(ctx)
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

func (c *adminController) RevenueAnalytics(ctx context.Context, year int) (*dto.RevenueAnalyticsResponse, error) {
	now := time.Now()
	if year == 0 {
		year = now.Year()
	}
	if year < 1 || year > 9998 {
		return nil, apperror.BadRequest("year must be between 1 and 9998")
	}

	from := time.Date(year, time.January, 1, 0, 0, 0, 0, now.Location())
	to := from.AddDate(1, 0, 0)
	analytics, err := c.uow.Payment().RevenueAnalytics(ctx, from, to)
	if err != nil {
		return nil, err
	}

	response := &dto.RevenueAnalyticsResponse{
		Year:             year,
		ByLocation:       make([]dto.RevenueGroupResponse, 0, len(analytics.ByLocation)),
		ByField:          make([]dto.RevenueGroupResponse, 0, len(analytics.ByField)),
		ByMonth:          make([]dto.MonthlyRevenue, 12),
		BestSellingSlots: make([]dto.BestSellingSlot, 0, len(analytics.BySlot)),
	}
	for month := range response.ByMonth {
		response.ByMonth[month].Month = month + 1
	}
	for _, row := range analytics.ByLocation {
		response.ByLocation = append(response.ByLocation, dto.RevenueGroupResponse{
			Name: row.Name, Bookings: row.Bookings, Revenue: row.Revenue,
		})
	}
	for _, row := range analytics.ByField {
		response.ByField = append(response.ByField, dto.RevenueGroupResponse{
			ID: row.ID, Name: row.Name, Location: row.Location, Bookings: row.Bookings, Revenue: row.Revenue,
		})
	}
	for _, row := range analytics.ByMonth {
		if row.Month >= 1 && row.Month <= len(response.ByMonth) {
			response.ByMonth[row.Month-1] = dto.MonthlyRevenue{
				Month: row.Month, Bookings: row.Bookings, Revenue: row.Revenue,
			}
		}
		response.TotalRevenue += row.Revenue
		response.PaidBookings += row.Bookings
	}
	for _, row := range analytics.BySlot {
		response.BestSellingSlots = append(response.BestSellingSlots, dto.BestSellingSlot{
			StartTime: row.StartTime, Bookings: row.Bookings, Revenue: row.Revenue,
		})
	}
	return response, nil
}

// ListUsers mengambil daftar user untuk panel admin.
func (c *adminController) ListUsers(ctx context.Context, query dto.PaginationQuery) ([]dto.UserResponse, int64, error) {
	query = query.Normalize()

	users, total, err := c.uow.User().FindAll(ctx, repository.ListParams{Page: query.Page, Limit: query.Limit})
	if err != nil {
		return nil, 0, err
	}
	return dto.NewUserResponses(users), total, nil
}

// ListAuditLogs mengambil aktivitas request terbaru untuk panel admin.
func (c *adminController) ListAuditLogs(ctx context.Context, query dto.PaginationQuery) ([]dto.AuditLogResponse, int64, error) {
	query = query.Normalize()
	logs, total, err := c.uow.AuditLog().FindAll(ctx, repository.ListParams{Page: query.Page, Limit: query.Limit})
	if err != nil {
		return nil, 0, err
	}
	return dto.NewAuditLogResponses(logs), total, nil
}
