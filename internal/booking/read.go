package booking

import (
	"context"
	"time"
)

type ReadFilter struct {
	Page     int
	Limit    int
	UserID   *uint
	FieldID  *uint
	Status   string
	DateFrom *time.Time
	DateTo   *time.Time
}

type ReadRepository interface {
	List(ctx context.Context, filter ReadFilter) ([]ReadModel, int64, error)
	FindDetail(ctx context.Context, id uint) (*ReadModel, error)
}

type ReadModel struct {
	ID                   uint
	BookingCode          string
	UserID               uint
	FieldID              uint
	FieldName            string
	BookingDate          time.Time
	StartTime            string
	EndTime              string
	Duration             int
	OriginalPrice        float64
	DiscountAmount       float64
	DiscountPercent      float64
	PackageName          string
	TotalPrice           float64
	Status               string
	Notes                string
	CancellationReason   string
	RefundReviewRequired bool
	RescheduleHistory    []ScheduleChange
	CheckInToken         *string
	CheckedInAt          *time.Time
	Items                []ReadItem
	Payment              *ReadPayment
	CreatedAt            time.Time
}

type ScheduleChange struct {
	PreviousDate      string
	PreviousStartTime string
	PreviousEndTime   string
	NewDate           string
	NewStartTime      string
	NewEndTime        string
	ChangedAt         time.Time
}

type ReadItem struct {
	StartTime string
	EndTime   string
	Price     float64
}

type ReadPayment struct {
	ID             uint
	BookingID      uint
	ExternalID     string
	InvoiceID      string
	InvoiceURL     string
	Amount         float64
	PaymentMethod  string
	PaymentChannel string
	Status         string
	PaidAt         *time.Time
	ExpiredAt      *time.Time
	CreatedAt      time.Time
}
