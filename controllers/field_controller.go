package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/fieldcache"
	"Booking-Lapangan/pkg/timeutil"
	"Booking-Lapangan/repository"
)

// FieldController menangani manajemen lapangan, jadwal, dan availability.
type FieldController interface {
	List(ctx context.Context, query dto.FieldFilterQuery) ([]dto.FieldResponse, int64, error)
	Detail(ctx context.Context, id uint) (*dto.FieldResponse, error)
	Create(ctx context.Context, req dto.CreateFieldRequest) (*dto.FieldResponse, error)
	Update(ctx context.Context, id uint, req dto.UpdateFieldRequest) (*dto.FieldResponse, error)
	Delete(ctx context.Context, id uint) error
	Availability(ctx context.Context, id uint, date string) (*dto.AvailabilityResponse, error)
	ListSchedules(ctx context.Context, fieldID uint) ([]dto.ScheduleResponse, error)
	UpsertSchedules(ctx context.Context, fieldID uint, req dto.UpsertScheduleRequest) ([]dto.ScheduleResponse, error)
}

type fieldController struct {
	uow   repository.UnitOfWork
	cache fieldcache.Store
}

// NewFieldController membuat FieldController.
func NewFieldController(uow repository.UnitOfWork, caches ...fieldcache.Store) FieldController {
	controller := &fieldController{uow: uow}
	if len(caches) > 0 {
		controller.cache = caches[0]
	}
	return controller
}

// List mengambil daftar lapangan padel sesuai filter dan pagination.
func (c *fieldController) List(ctx context.Context, query dto.FieldFilterQuery) ([]dto.FieldResponse, int64, error) {
	query.PaginationQuery = query.PaginationQuery.Normalize()

	if query.MinPrice != nil && query.MaxPrice != nil && *query.MinPrice > *query.MaxPrice {
		return nil, 0, apperror.BadRequest("min_price cannot be greater than max_price")
	}

	filter := repository.FieldFilter{
		ListParams: repository.ListParams{Page: query.Page, Limit: query.Limit},
		Type:       strings.ToUpper(strings.TrimSpace(query.Type)),
		Status:     strings.ToUpper(strings.TrimSpace(query.Status)),
		Location:   strings.TrimSpace(query.Location),
		Search:     strings.TrimSpace(query.Search),
		MinPrice:   query.MinPrice,
		MaxPrice:   query.MaxPrice,
	}

	var cacheKey string
	var cacheVersion string
	cacheReadSucceeded := false
	var err error
	if c.cache != nil {
		cacheKey, err = fieldListCacheKey(filter)
		if err != nil {
			return nil, 0, err
		}
		cached, version, hit, cacheErr := c.cache.GetList(ctx, cacheKey)
		if cacheErr != nil {
			log.Printf("field cache: list read failed: %v", cacheErr)
		} else {
			cacheVersion = version
			cacheReadSucceeded = true
			if hit {
				return cached.Fields, cached.Total, nil
			}
		}
	}

	fields, total, err := c.uow.Field().FindAll(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	responses := dto.NewFieldResponses(fields)
	if c.cache != nil && cacheReadSucceeded {
		if err := c.cache.SetList(ctx, cacheKey, cacheVersion, fieldcache.ListData{
			Fields: responses,
			Total:  total,
		}); err != nil {
			log.Printf("field cache: list write failed: %v", err)
		}
	}
	return responses, total, nil
}

// Detail mengambil detail lapangan beserta jadwal operasional.
func (c *fieldController) Detail(ctx context.Context, id uint) (*dto.FieldResponse, error) {
	var cacheVersion string
	cacheReadSucceeded := false
	if c.cache != nil {
		cached, version, hit, cacheErr := c.cache.GetDetail(ctx, id)
		if cacheErr != nil {
			log.Printf("field cache: detail read failed for field %d: %v", id, cacheErr)
		} else {
			cacheVersion = version
			cacheReadSucceeded = true
			if hit {
				return cached, nil
			}
		}
	}

	response, err := c.loadFieldDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.cache != nil && cacheReadSucceeded {
		if err := c.cache.SetDetail(ctx, id, cacheVersion, response); err != nil {
			log.Printf("field cache: detail write failed for field %d: %v", id, err)
		}
	}
	return response, nil
}

func (c *fieldController) loadFieldDetail(ctx context.Context, id uint) (*dto.FieldResponse, error) {
	field, err := c.uow.Field().FindDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if field.Type != models.FieldTypePadel {
		return nil, apperror.NotFound("field not found")
	}
	response := dto.NewFieldResponse(*field)
	return &response, nil
}

// Create membuat lapangan baru beserta jadwal operasionalnya.
func (c *fieldController) Create(ctx context.Context, req dto.CreateFieldRequest) (*dto.FieldResponse, error) {
	status := models.FieldStatus(strings.ToUpper(strings.TrimSpace(req.Status)))
	if status == "" {
		status = models.FieldStatusActive
	}

	field := &models.Field{
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		Type:         models.FieldType(strings.ToUpper(req.Type)),
		Location:     strings.TrimSpace(req.Location),
		PricePerHour: req.PricePerHour,
		ImageURL:     strings.TrimSpace(req.ImageURL),
		Facilities:   dto.EncodeFacilities(req.Facilities),
		Status:       status,
	}

	schedules, err := buildSchedules(req.Schedules)
	if err != nil {
		return nil, err
	}

	err = c.uow.Atomic(ctx, func(r repository.Registry) error {
		if err := r.Field().Create(ctx, field); err != nil {
			return err
		}
		if len(schedules) == 0 {
			return nil
		}
		for i := range schedules {
			schedules[i].FieldID = field.ID
		}
		return r.Schedule().Upsert(ctx, schedules)
	})
	if err != nil {
		return nil, err
	}

	if !c.invalidateFieldCache(ctx, field.ID) {
		return c.loadFieldDetail(ctx, field.ID)
	}
	return c.Detail(ctx, field.ID)
}

// Update memperbarui data lapangan yang dipilih.
func (c *fieldController) Update(ctx context.Context, id uint, req dto.UpdateFieldRequest) (*dto.FieldResponse, error) {
	field, err := c.uow.Field().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if field.Type != models.FieldTypePadel {
		return nil, apperror.NotFound("field not found")
	}

	if req.Name != nil {
		field.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		field.Description = strings.TrimSpace(*req.Description)
	}
	if req.Type != nil {
		field.Type = models.FieldType(strings.ToUpper(*req.Type))
	}
	if req.Location != nil {
		field.Location = strings.TrimSpace(*req.Location)
	}
	if req.PricePerHour != nil {
		field.PricePerHour = *req.PricePerHour
	}
	if req.ImageURL != nil {
		field.ImageURL = strings.TrimSpace(*req.ImageURL)
	}
	if req.Status != nil {
		field.Status = models.FieldStatus(strings.ToUpper(*req.Status))
	}
	if req.Facilities != nil {
		field.Facilities = dto.EncodeFacilities(*req.Facilities)
	}

	if err := c.uow.Field().Update(ctx, field); err != nil {
		return nil, err
	}
	if !c.invalidateFieldCache(ctx, field.ID) {
		return c.loadFieldDetail(ctx, field.ID)
	}
	return c.Detail(ctx, field.ID)
}

// Delete menghapus lapangan berdasarkan ID.
func (c *fieldController) Delete(ctx context.Context, id uint) error {
	if err := c.uow.Field().Delete(ctx, id); err != nil {
		return err
	}
	c.invalidateFieldCache(ctx, id)
	return nil
}

// Availability menyusun slot per jam beserta status ketersediaannya
// berdasarkan jadwal operasional dan booking aktif pada tanggal tersebut.
func (c *fieldController) Availability(ctx context.Context, id uint, date string) (*dto.AvailabilityResponse, error) {
	bookingDate, err := timeutil.ParseDate(date)
	if err != nil {
		return nil, apperror.BadRequest("date must use YYYY-MM-DD format")
	}

	field, err := c.uow.Field().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	day := models.DayFromWeekday(bookingDate.Weekday())
	response := &dto.AvailabilityResponse{
		FieldID:      field.ID,
		FieldName:    field.Name,
		Date:         bookingDate.Format(timeutil.DateLayout),
		Day:          string(day),
		PricePerHour: field.PricePerHour,
		Slots:        []dto.SlotResponse{},
	}

	// Jika tanggal berada di masa lalu, kembalikan respons non-aktif yang aman tanpa melempar error 400.
	if bookingDate.Before(timeutil.Today()) {
		response.IsOpen = false
		return response, nil
	}

	if !field.IsActive() {
		return response, nil
	}

	schedule, err := c.uow.Schedule().FindByFieldAndDay(ctx, field.ID, day)
	if err != nil {
		if apperror.IsNotFound(err) {
			return response, nil
		}
		return nil, err
	}
	if schedule.IsClosed {
		return response, nil
	}

	response.IsOpen = true
	response.OpenTime = schedule.OpenTime
	response.CloseTime = schedule.CloseTime

	bookings, err := c.uow.Booking().FindActiveByFieldAndDate(ctx, field.ID, bookingDate)
	if err != nil {
		return nil, err
	}

	now := timeutil.Today()
	isToday := bookingDate.Equal(now)

	for _, slot := range timeutil.HourlySlots(schedule.OpenTime, schedule.CloseTime) {
		item := dto.SlotResponse{
			StartTime: slot[0],
			EndTime:   slot[1],
			Price:     field.PricePerHour,
			Available: true,
		}

		for _, booking := range bookings {
			if timeutil.Overlap(slot[0], slot[1], booking.StartTime, booking.EndTime) {
				item.Available = false
				item.Reason = "already booked"
				break
			}
		}

		if item.Available && isToday {
			slotStart, convErr := timeutil.CombineDateTime(bookingDate, slot[0])
			if convErr == nil && slotStart.Before(timeNow()) {
				item.Available = false
				item.Reason = "time has passed"
			}
		}

		response.Slots = append(response.Slots, item)
	}

	return response, nil
}

// ListSchedules mengambil seluruh jadwal operasional lapangan.
func (c *fieldController) ListSchedules(ctx context.Context, fieldID uint) ([]dto.ScheduleResponse, error) {
	if _, err := c.uow.Field().FindByID(ctx, fieldID); err != nil {
		return nil, err
	}
	schedules, err := c.uow.Schedule().FindByFieldID(ctx, fieldID)
	if err != nil {
		return nil, err
	}
	return dto.NewScheduleResponses(schedules), nil
}

// UpsertSchedules menyimpan atau memperbarui jadwal operasional lapangan.
func (c *fieldController) UpsertSchedules(ctx context.Context, fieldID uint, req dto.UpsertScheduleRequest) ([]dto.ScheduleResponse, error) {
	if _, err := c.uow.Field().FindByID(ctx, fieldID); err != nil {
		return nil, err
	}

	schedules, err := buildSchedules(req.Schedules)
	if err != nil {
		return nil, err
	}
	for i := range schedules {
		schedules[i].FieldID = fieldID
	}

	if err := c.uow.Schedule().Upsert(ctx, schedules); err != nil {
		return nil, err
	}
	c.invalidateFieldCache(ctx, fieldID)
	return c.ListSchedules(ctx, fieldID)
}

func (c *fieldController) invalidateFieldCache(ctx context.Context, fieldID uint) bool {
	if c.cache == nil {
		return true
	}
	succeeded := true
	if err := c.cache.InvalidateLists(ctx); err != nil {
		log.Printf("field cache: list invalidation failed after field %d mutation: %v", fieldID, err)
		succeeded = false
	}
	if err := c.cache.InvalidateDetail(ctx, fieldID); err != nil {
		log.Printf("field cache: detail invalidation failed for field %d: %v", fieldID, err)
		succeeded = false
	}
	return succeeded
}

func fieldListCacheKey(filter repository.FieldFilter) (string, error) {
	encoded, err := json.Marshal(filter)
	if err != nil {
		return "", apperror.Internal("failed to create field cache key", err)
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:]), nil
}

// buildSchedules memvalidasi request jadwal dan mengubahnya menjadi model.
func buildSchedules(requests []dto.ScheduleRequest) ([]models.Schedule, error) {
	schedules := make([]models.Schedule, 0, len(requests))
	seen := make(map[models.Day]bool, len(requests))

	for _, item := range requests {
		day := models.Day(strings.ToUpper(strings.TrimSpace(item.Day)))
		if !day.Valid() {
			return nil, apperror.Unprocessable("invalid schedule day: " + item.Day)
		}
		if seen[day] {
			return nil, apperror.Unprocessable("duplicated schedule for day " + string(day))
		}
		seen[day] = true

		if !item.IsClosed {
			if _, err := timeutil.DurationHours(item.OpenTime, item.CloseTime); err != nil {
				return nil, apperror.Unprocessable("invalid operating hours for " + string(day) + ": " + err.Error())
			}
		}

		schedules = append(schedules, models.Schedule{
			Day:       day,
			OpenTime:  item.OpenTime,
			CloseTime: item.CloseTime,
			IsClosed:  item.IsClosed,
		})
	}
	return schedules, nil
}
