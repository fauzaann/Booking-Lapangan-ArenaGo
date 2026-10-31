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

type RevenueAnalyticsResponse struct {
	Year             int                    `json:"year"`
	TotalRevenue     float64                `json:"total_revenue"`
	PaidBookings     int64                  `json:"paid_bookings"`
	ByLocation       []RevenueGroupResponse `json:"by_location"`
	ByField          []RevenueGroupResponse `json:"by_field"`
	ByMonth          []MonthlyRevenue       `json:"by_month"`
	BestSellingSlots []BestSellingSlot      `json:"best_selling_slots"`
}

type RevenueGroupResponse struct {
	ID       uint    `json:"id,omitempty"`
	Name     string  `json:"name"`
	Location string  `json:"location,omitempty"`
	Bookings int64   `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}

type MonthlyRevenue struct {
	Month    int     `json:"month"`
	Bookings int64   `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}

type BestSellingSlot struct {
	StartTime string  `json:"start_time"`
	Bookings  int64   `json:"bookings"`
	Revenue   float64 `json:"revenue"`
}
