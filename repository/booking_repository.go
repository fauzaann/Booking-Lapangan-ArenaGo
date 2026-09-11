package repository

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"gorm.io/gorm"

	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
)

// BookingFilter adalah filter pencarian booking.
type BookingFilter struct {
	ListParams
	UserID   *uint
	FieldID  *uint
	Status   string
	DateFrom *time.Time
	DateTo   *time.Time
}

// BookingRepository adalah kontrak akses data booking.
type BookingRepository interface {
	Create(ctx context.Context, booking *models.Booking) error
	Update(ctx context.Context, booking *models.Booking) error
	FindByID(ctx context.Context, id uint) (*models.Booking, error)
	FindDetailByID(ctx context.Context, id uint) (*models.Booking, error)
	FindByCode(ctx context.Context, code string) (*models.Booking, error)
	FindAll(ctx context.Context, filter BookingFilter) ([]models.Booking, int64, error)
	FindActiveByFieldAndDate(ctx context.Context, fieldID uint, date time.Time) ([]models.Booking, error)
	HasOverlap(ctx context.Context, fieldID uint, date time.Time, startTime, endTime string) (bool, error)
	LockFieldDate(ctx context.Context, fieldID uint, date time.Time) error
	NextSequence(ctx context.Context, date time.Time) (int, error)
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context) (map[models.BookingStatus]int64, error)
	MarkExpired(ctx context.Context, bookingIDs []uint) (int64, error)
	CompleteFinished(ctx context.Context, now time.Time) (int64, error)
}

type bookingRepository struct {
	db *gorm.DB
}

// NewBookingRepository membuat implementasi BookingRepository berbasis GORM.
func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) Create(ctx context.Context, booking *models.Booking) error {
	return translate(r.db.WithContext(ctx).Create(booking).Error, "booking not found")
}

func (r *bookingRepository) Update(ctx context.Context, booking *models.Booking) error {
	// Omit asosiasi agar Save tidak ikut menulis ulang user/field/payment.
	err := r.db.WithContext(ctx).
		Omit("User", "Field", "Items", "Payment").
		Save(booking).Error
	return translate(err, "booking not found")
}

func (r *bookingRepository) FindByID(ctx context.Context, id uint) (*models.Booking, error) {
	var booking models.Booking
	if err := r.db.WithContext(ctx).First(&booking, id).Error; err != nil {
		return nil, translate(err, "booking not found")
	}
	return &booking, nil
}

func (r *bookingRepository) FindDetailByID(ctx context.Context, id uint) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).
		Preload("Field").
		Preload("User").
		Preload("Items").
		Preload("Payment").
		First(&booking, id).Error
	if err != nil {
		return nil, translate(err, "booking not found")
	}
	return &booking, nil
}

func (r *bookingRepository) FindByCode(ctx context.Context, code string) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.WithContext(ctx).
		Preload("Field").
		Preload("Payment").
		Where("booking_code = ?", code).
		First(&booking).Error
	if err != nil {
		return nil, translate(err, "booking not found")
	}
	return &booking, nil
}

func (r *bookingRepository) FindAll(ctx context.Context, filter BookingFilter) ([]models.Booking, int64, error) {
	filter.ListParams = filter.ListParams.Normalize()

	query := r.db.WithContext(ctx).Model(&models.Booking{})

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.FieldID != nil {
		query = query.Where("field_id = ?", *filter.FieldID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.DateFrom != nil {
		query = query.Where("booking_date >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("booking_date <= ?", *filter.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translate(err, "booking not found")
	}

	var bookings []models.Booking
	err := query.
		Preload("Field").
		Preload("Payment").
		Order("booking_date DESC, start_time DESC, id DESC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&bookings).Error
	if err != nil {
		return nil, 0, translate(err, "booking not found")
	}
	return bookings, total, nil
}

func (r *bookingRepository) FindActiveByFieldAndDate(ctx context.Context, fieldID uint, date time.Time) ([]models.Booking, error) {
	var bookings []models.Booking
	err := r.db.WithContext(ctx).
		Where("field_id = ? AND booking_date = ?", fieldID, date).
		Where("status IN ?", models.SlotBlockingStatuses()).
		Order("start_time ASC").
		Find(&bookings).Error
	if err != nil {
		return nil, translate(err, "booking not found")
	}
	return bookings, nil
}

// HasOverlap memeriksa tabrakan slot dengan aturan half-open [start, end).
// Kondisi overlap: existing.start < request.end AND existing.end > request.start,
// sehingga kasus seperti existing 10:00-12:00 vs request 11:00-13:00 tetap tertolak.
func (r *bookingRepository) HasOverlap(ctx context.Context, fieldID uint, date time.Time, startTime, endTime string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("field_id = ? AND booking_date = ?", fieldID, date).
		Where("status IN ?", models.SlotBlockingStatuses()).
		Where("start_time < ? AND end_time > ?", endTime, startTime).
		Count(&count).Error
	if err != nil {
		return false, translate(err, "booking not found")
	}
	return count > 0, nil
}

// LockFieldDate mengambil advisory lock transaksional PostgreSQL untuk
// kombinasi lapangan + tanggal. Lock ini mencegah dua request bersamaan
// lolos pemeriksaan availability pada slot yang sama (race condition),
// dan otomatis dilepas saat transaksi commit/rollback.
func (r *bookingRepository) LockFieldDate(ctx context.Context, fieldID uint, date time.Time) error {
	err := r.db.WithContext(ctx).
		Exec("SELECT pg_advisory_xact_lock(?)", advisoryKey(fmt.Sprintf("field:%d:%s", fieldID, date.Format("2006-01-02")))).
		Error
	if err != nil {
		return apperror.Internal("failed to acquire booking lock", err)
	}
	return nil
}

// NextSequence mengembalikan nomor urut booking berikutnya untuk tanggal tertentu.
// Dipanggil di dalam transaksi yang sudah memegang advisory lock.
func (r *bookingRepository) NextSequence(ctx context.Context, date time.Time) (int, error) {
	prefix := "BK-" + date.Format("20060102") + "-"

	if err := r.db.WithContext(ctx).
		Exec("SELECT pg_advisory_xact_lock(?)", advisoryKey("booking-code:"+date.Format("2006-01-02"))).
		Error; err != nil {
		return 0, apperror.Internal("failed to acquire booking code lock", err)
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("booking_code LIKE ?", prefix+"%").
		Count(&count).Error
	if err != nil {
		return 0, translate(err, "booking not found")
	}
	return int(count) + 1, nil
}

func (r *bookingRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&models.Booking{}).Count(&total).Error
	return total, translate(err, "booking not found")
}

func (r *bookingRepository) CountByStatus(ctx context.Context) (map[models.BookingStatus]int64, error) {
	type row struct {
		Status models.BookingStatus
		Total  int64
	}

	var rows []row
	err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Select("status, COUNT(*) AS total").
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, translate(err, "booking not found")
	}

	result := make(map[models.BookingStatus]int64, len(rows))
	for _, item := range rows {
		result[item.Status] = item.Total
	}
	return result, nil
}

func advisoryKey(value string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(value))
	return int64(hasher.Sum64())
}

// MarkExpired mengubah booking yang pembayarannya kedaluwarsa menjadi EXPIRED.
func (r *bookingRepository) MarkExpired(ctx context.Context, bookingIDs []uint) (int64, error) {
	if len(bookingIDs) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("id IN ?", bookingIDs).
		Where("status IN ?", []string{
			string(models.BookingStatusPending),
			string(models.BookingStatusWaitingPayment),
		}).
		Update("status", models.BookingStatusExpired)
	if result.Error != nil {
		return 0, translate(result.Error, "booking not found")
	}
	return result.RowsAffected, nil
}

// CompleteFinished menandai booking terkonfirmasi yang jamnya sudah lewat
// menjadi COMPLETED.
func (r *bookingRepository) CompleteFinished(ctx context.Context, now time.Time) (int64, error) {
	date := now.Format("2006-01-02")
	clock := now.Format("15:04")

	result := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("status IN ?", []string{
			string(models.BookingStatusPaid),
			string(models.BookingStatusConfirmed),
		}).
		Where("(booking_date < ?::date OR (booking_date = ?::date AND end_time <= ?))", date, date, clock).
		Update("status", models.BookingStatusCompleted)
	if result.Error != nil {
		return 0, translate(result.Error, "booking not found")
	}
	return result.RowsAffected, nil
}
