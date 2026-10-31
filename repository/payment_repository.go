package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
)

// PaymentFilter adalah filter pencarian pembayaran.
type PaymentFilter struct {
	ListParams
	Status string
}

type RevenueAnalytics struct {
	ByLocation []RevenueGroup
	ByField    []RevenueGroup
	ByMonth    []MonthlyRevenue
	BySlot     []BestSellingSlot
}

type RevenueGroup struct {
	ID       uint    `json:"id"`
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

// PaymentRepository adalah kontrak akses data pembayaran.
type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	Update(ctx context.Context, payment *models.Payment) error
	FindByID(ctx context.Context, id uint) (*models.Payment, error)
	FindByExternalID(ctx context.Context, externalID string) (*models.Payment, error)
	FindByBookingID(ctx context.Context, bookingID uint) (*models.Payment, error)
	FindByCheckoutID(ctx context.Context, checkoutID uint) ([]models.Payment, error)
	FindAll(ctx context.Context, filter PaymentFilter) ([]models.Payment, int64, error)
	SumPaidAmount(ctx context.Context) (float64, error)
	RevenueAnalytics(ctx context.Context, from, to time.Time) (*RevenueAnalytics, error)
	ExpireOverdue(ctx context.Context, now time.Time) ([]uint, error)
}

type paymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository membuat implementasi PaymentRepository berbasis GORM.
func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

// Create menyimpan pembayaran baru ke database.
func (r *paymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	return translate(r.db.WithContext(ctx).Create(payment).Error, "payment not found")
}

// Update menyimpan perubahan status atau metadata pembayaran.
func (r *paymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	err := r.db.WithContext(ctx).Omit("Booking").Save(payment).Error
	return translate(err, "payment not found")
}

// FindByID mengambil pembayaran berdasarkan primary key.
func (r *paymentRepository) FindByID(ctx context.Context, id uint) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.WithContext(ctx).First(&payment, id).Error; err != nil {
		return nil, translate(err, "payment not found")
	}
	return &payment, nil
}

// FindByExternalID mengambil pembayaran berdasarkan ID invoice eksternal.
func (r *paymentRepository) FindByExternalID(ctx context.Context, externalID string) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&payment).Error
	if err != nil {
		return nil, translate(err, "payment not found")
	}
	return &payment, nil
}

// FindByBookingID mengambil pembayaran yang terkait dengan booking.
func (r *paymentRepository) FindByBookingID(ctx context.Context, bookingID uint) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&payment).Error
	if err != nil {
		return nil, translate(err, "payment not found")
	}
	return &payment, nil
}

func (r *paymentRepository) FindByCheckoutID(ctx context.Context, checkoutID uint) ([]models.Payment, error) {
	var payments []models.Payment
	err := r.db.WithContext(ctx).Where("checkout_id = ?", checkoutID).Find(&payments).Error
	if err != nil {
		return nil, translate(err, "payment not found")
	}
	return payments, nil
}

// FindAll mengambil pembayaran dengan filter status dan pagination.
func (r *paymentRepository) FindAll(ctx context.Context, filter PaymentFilter) ([]models.Payment, int64, error) {
	filter.ListParams = filter.ListParams.Normalize()

	query := r.db.WithContext(ctx).Model(&models.Payment{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translate(err, "payment not found")
	}

	var payments []models.Payment
	err := query.Order("id DESC").
		Preload("Booking").
		Preload("Booking.Field").
		Preload("Booking.User").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&payments).Error
	if err != nil {
		return nil, 0, translate(err, "payment not found")
	}
	return payments, total, nil
}

// SumPaidAmount menjumlahkan seluruh pembayaran berstatus PAID (revenue).
func (r *paymentRepository) SumPaidAmount(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ?", models.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, translate(err, "payment not found")
	}
	return total, nil
}

func (r *paymentRepository) RevenueAnalytics(ctx context.Context, from, to time.Time) (*RevenueAnalytics, error) {
	db := r.db.WithContext(ctx)
	paidInPeriod := func(query *gorm.DB) *gorm.DB {
		return query.
			Joins("JOIN bookings ON bookings.id = payments.booking_id").
			Where("payments.status = ?", models.PaymentStatusPaid).
			Where("COALESCE(payments.paid_at, payments.updated_at) >= ? AND COALESCE(payments.paid_at, payments.updated_at) < ?", from, to)
	}

	result := &RevenueAnalytics{}
	if err := paidInPeriod(db.Table("payments")).
		Joins("JOIN fields ON fields.id = bookings.field_id").
		Select("fields.location AS name, COUNT(*) AS bookings, COALESCE(SUM(payments.amount), 0) AS revenue").
		Group("fields.location").
		Order("revenue DESC, fields.location ASC").
		Scan(&result.ByLocation).Error; err != nil {
		return nil, translate(err, "payment not found")
	}

	if err := paidInPeriod(db.Table("payments")).
		Joins("JOIN fields ON fields.id = bookings.field_id").
		Select("fields.id AS id, fields.name AS name, fields.location AS location, COUNT(*) AS bookings, COALESCE(SUM(payments.amount), 0) AS revenue").
		Group("fields.id, fields.name, fields.location").
		Order("revenue DESC, fields.name ASC").
		Scan(&result.ByField).Error; err != nil {
		return nil, translate(err, "payment not found")
	}

	var monthly []MonthlyRevenue
	if err := paidInPeriod(db.Table("payments")).
		Select("EXTRACT(MONTH FROM COALESCE(payments.paid_at, payments.updated_at))::int AS month, COUNT(*) AS bookings, COALESCE(SUM(payments.amount), 0) AS revenue").
		Group("month").
		Order("month ASC").
		Scan(&monthly).Error; err != nil {
		return nil, translate(err, "payment not found")
	}
	result.ByMonth = make([]MonthlyRevenue, 12)
	for month := range result.ByMonth {
		result.ByMonth[month].Month = month + 1
	}
	for _, row := range monthly {
		if row.Month >= 1 && row.Month <= len(result.ByMonth) {
			result.ByMonth[row.Month-1] = row
		}
	}

	if err := paidInPeriod(db.Table("payments")).
		Joins("JOIN booking_items ON booking_items.booking_id = bookings.id").
		Select("booking_items.start_time AS start_time, COUNT(*) AS bookings, COALESCE(SUM(booking_items.price), 0) AS revenue").
		Group("booking_items.start_time").
		Order("bookings DESC, booking_items.start_time ASC").
		Limit(5).
		Scan(&result.BySlot).Error; err != nil {
		return nil, translate(err, "payment not found")
	}

	return result, nil
}

// ExpireOverdue menandai pembayaran PENDING yang sudah melewati expired_at
// menjadi EXPIRED, lalu mengembalikan booking_id yang terpengaruh.
func (r *paymentRepository) ExpireOverdue(ctx context.Context, now time.Time) ([]uint, error) {
	var bookingIDs []uint
	err := r.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ?", models.PaymentStatusPending).
		Where("expired_at IS NOT NULL AND expired_at < ?", now).
		Pluck("booking_id", &bookingIDs).Error
	if err != nil {
		return nil, translate(err, "payment not found")
	}
	if len(bookingIDs) == 0 {
		return nil, nil
	}

	err = r.db.WithContext(ctx).Model(&models.Payment{}).
		Where("booking_id IN ?", bookingIDs).
		Where("status = ?", models.PaymentStatusPending).
		Update("status", models.PaymentStatusExpired).Error
	if err != nil {
		return nil, translate(err, "payment not found")
	}
	return bookingIDs, nil
}
