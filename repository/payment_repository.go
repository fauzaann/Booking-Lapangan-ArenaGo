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

// PaymentRepository adalah kontrak akses data pembayaran.
type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	Update(ctx context.Context, payment *models.Payment) error
	FindByID(ctx context.Context, id uint) (*models.Payment, error)
	FindByExternalID(ctx context.Context, externalID string) (*models.Payment, error)
	FindByBookingID(ctx context.Context, bookingID uint) (*models.Payment, error)
	FindAll(ctx context.Context, filter PaymentFilter) ([]models.Payment, int64, error)
	SumPaidAmount(ctx context.Context) (float64, error)
	ExpireOverdue(ctx context.Context, now time.Time) ([]uint, error)
}

type paymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository membuat implementasi PaymentRepository berbasis GORM.
func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	return translate(r.db.WithContext(ctx).Create(payment).Error, "payment not found")
}

func (r *paymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	err := r.db.WithContext(ctx).Omit("Booking").Save(payment).Error
	return translate(err, "payment not found")
}

func (r *paymentRepository) FindByID(ctx context.Context, id uint) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.WithContext(ctx).First(&payment, id).Error; err != nil {
		return nil, translate(err, "payment not found")
	}
	return &payment, nil
}

func (r *paymentRepository) FindByExternalID(ctx context.Context, externalID string) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&payment).Error
	if err != nil {
		return nil, translate(err, "payment not found")
	}
	return &payment, nil
}

func (r *paymentRepository) FindByBookingID(ctx context.Context, bookingID uint) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&payment).Error
	if err != nil {
		return nil, translate(err, "payment not found")
	}
	return &payment, nil
}

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
