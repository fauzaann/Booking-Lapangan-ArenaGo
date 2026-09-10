package dto

// DashboardResponse adalah ringkasan data untuk dashboard admin.
type DashboardResponse struct {
	TotalUsers        int64   `json:"total_users"`
	TotalFields       int64   `json:"total_fields"`
	TotalBookings     int64   `json:"total_bookings"`
	TotalRevenue      float64 `json:"total_revenue"`
	PendingBookings   int64   `json:"pending_bookings"`
	ConfirmedBookings int64   `json:"confirmed_bookings"`
	CompletedBookings int64   `json:"completed_bookings"`
	CancelledBookings int64   `json:"cancelled_bookings"`
	ExpiredBookings   int64   `json:"expired_bookings"`
}
