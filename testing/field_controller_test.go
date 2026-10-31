package controllers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/fieldcache"
	"Booking-Lapangan/repository"
)

type mockFieldRepo struct {
	field           *models.Field
	updateErr       error
	deleteErr       error
	findAllCalls    int
	findDetailCalls int
	fields          []models.Field
}

func (m *mockFieldRepo) Create(ctx context.Context, field *models.Field) error { return nil }
func (m *mockFieldRepo) Update(ctx context.Context, field *models.Field) error { return m.updateErr }
func (m *mockFieldRepo) Delete(ctx context.Context, id uint) error             { return m.deleteErr }
func (m *mockFieldRepo) FindByID(ctx context.Context, id uint) (*models.Field, error) {
	return m.field, nil
}
func (m *mockFieldRepo) FindDetailByID(ctx context.Context, id uint) (*models.Field, error) {
	m.findDetailCalls++
	return m.field, nil
}
func (m *mockFieldRepo) FindAll(ctx context.Context, filter repository.FieldFilter) ([]models.Field, int64, error) {
	m.findAllCalls++
	return m.fields, int64(len(m.fields)), nil
}
func (m *mockFieldRepo) Count(ctx context.Context) (int64, error) { return 1, nil }

type mockScheduleRepo struct {
	schedule *models.Schedule
}

func (m *mockScheduleRepo) FindByFieldID(ctx context.Context, fieldID uint) ([]models.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) FindByFieldAndDay(ctx context.Context, fieldID uint, day models.Day) (*models.Schedule, error) {
	return m.schedule, nil
}
func (m *mockScheduleRepo) Upsert(ctx context.Context, schedules []models.Schedule) error {
	return nil
}
func (m *mockScheduleRepo) DeleteByFieldID(ctx context.Context, fieldID uint) error {
	return nil
}

type fieldTestUOW struct {
	mockUOW
	fieldRepo    repository.FieldRepository
	scheduleRepo repository.ScheduleRepository
}

func (m *fieldTestUOW) Field() repository.FieldRepository       { return m.fieldRepo }
func (m *fieldTestUOW) Schedule() repository.ScheduleRepository { return m.scheduleRepo }

type mockFieldCache struct {
	listData            fieldcache.ListData
	listKey             string
	listVersion         string
	listHit             bool
	listReadErr         error
	listInvalidateErr   error
	listSetCalls        int
	listInvalidate      int
	detailData          *dto.FieldResponse
	detailVersion       string
	detailHit           bool
	detailReadErr       error
	detailInvalidateErr error
	detailSetCalls      int
	detailInvalidate    int
}

func (m *mockFieldCache) GetList(_ context.Context, key string) (fieldcache.ListData, string, bool, error) {
	m.listKey = key
	return m.listData, m.listVersion, m.listHit, m.listReadErr
}
func (m *mockFieldCache) SetList(context.Context, string, string, fieldcache.ListData) error {
	m.listSetCalls++
	return nil
}
func (m *mockFieldCache) InvalidateLists(context.Context) error {
	m.listInvalidate++
	return m.listInvalidateErr
}
func (m *mockFieldCache) GetDetail(context.Context, uint) (*dto.FieldResponse, string, bool, error) {
	return m.detailData, m.detailVersion, m.detailHit, m.detailReadErr
}
func (m *mockFieldCache) SetDetail(context.Context, uint, string, *dto.FieldResponse) error {
	m.detailSetCalls++
	return nil
}
func (m *mockFieldCache) InvalidateDetail(context.Context, uint) error {
	m.detailInvalidate++
	return m.detailInvalidateErr
}

func TestFieldListUsesCacheHit(t *testing.T) {
	fieldRepo := &mockFieldRepo{}
	cache := &mockFieldCache{
		listData: fieldcache.ListData{
			Fields: []dto.FieldResponse{{ID: 42, Name: "Cached Field"}},
			Total:  1,
		},
		listVersion: "3",
		listHit:     true,
	}
	uow := &fieldTestUOW{fieldRepo: fieldRepo}

	fields, total, err := controllers.NewFieldController(uow, cache).List(context.Background(), dto.FieldFilterQuery{})
	if err != nil {
		t.Fatalf("list fields: %v", err)
	}
	if total != 1 || len(fields) != 1 || fields[0].ID != 42 {
		t.Fatalf("list result = (%+v, total %d), want cached field", fields, total)
	}
	if fieldRepo.findAllCalls != 0 {
		t.Fatalf("database list queried %d times on cache hit, want 0", fieldRepo.findAllCalls)
	}
}

func TestFieldListCacheFailureFallsBackToDatabase(t *testing.T) {
	fieldRepo := &mockFieldRepo{}
	cache := &mockFieldCache{listReadErr: errors.New("redis unavailable")}
	uow := &fieldTestUOW{fieldRepo: fieldRepo}

	_, _, err := controllers.NewFieldController(uow, cache).List(context.Background(), dto.FieldFilterQuery{})
	if err != nil {
		t.Fatalf("list fields with Redis unavailable: %v", err)
	}
	if fieldRepo.findAllCalls != 1 {
		t.Fatalf("database list queried %d times, want 1", fieldRepo.findAllCalls)
	}
	if cache.listSetCalls != 0 {
		t.Fatalf("cache set called %d times after a failed cache read, want 0", cache.listSetCalls)
	}
}

func TestFieldListCacheKeyIncludesFilters(t *testing.T) {
	fieldRepo := &mockFieldRepo{}
	cache := &mockFieldCache{}
	controller := controllers.NewFieldController(&fieldTestUOW{fieldRepo: fieldRepo}, cache)

	if _, _, err := controller.List(context.Background(), dto.FieldFilterQuery{Search: "alpha"}); err != nil {
		t.Fatalf("list fields with first filter: %v", err)
	}
	firstKey := cache.listKey
	if _, _, err := controller.List(context.Background(), dto.FieldFilterQuery{Search: "beta"}); err != nil {
		t.Fatalf("list fields with second filter: %v", err)
	}
	if firstKey == cache.listKey {
		t.Fatalf("cache key %q was reused for different filters", firstKey)
	}
}

func TestFieldListIncludesRatingSummaries(t *testing.T) {
	fieldRepo := &mockFieldRepo{fields: []models.Field{
		{ID: 7, Name: "Lapangan Alpha"},
		{ID: 8, Name: "Lapangan Beta"},
	}}
	ratings := newRatingTestRepository()
	ratings.items[1] = models.Rating{ID: 1, BookingID: 101, FieldID: 7, Rating: 5, IsVisible: true}
	ratings.items[2] = models.Rating{ID: 2, BookingID: 102, FieldID: 7, Rating: 3, IsVisible: true}
	ratings.items[3] = models.Rating{ID: 3, BookingID: 103, FieldID: 7, Rating: 1, IsVisible: false}
	uow := &fieldTestUOW{
		mockUOW:   mockUOW{ratingRepo: ratings},
		fieldRepo: fieldRepo,
	}

	fields, total, err := controllers.NewFieldController(uow).List(context.Background(), dto.FieldFilterQuery{})
	if err != nil {
		t.Fatalf("list fields: %v", err)
	}
	if total != 2 || len(fields) != 2 {
		t.Fatalf("field list = %d, total %d; want 2", len(fields), total)
	}
	if fields[0].AverageRating != 4 || fields[0].TotalReviews != 2 {
		t.Fatalf("first field rating summary = (%v, %d), want (4, 2)", fields[0].AverageRating, fields[0].TotalReviews)
	}
	if fields[1].AverageRating != 0 || fields[1].TotalReviews != 0 {
		t.Fatalf("field without reviews summary = (%v, %d), want (0, 0)", fields[1].AverageRating, fields[1].TotalReviews)
	}
}

func TestFieldUpdateInvalidatesCachesAfterDatabaseWrite(t *testing.T) {
	field := &models.Field{ID: 7, Name: "Lapangan Alpha", Type: models.FieldTypePadel}
	fieldRepo := &mockFieldRepo{field: field}
	cache := &mockFieldCache{}
	controller := controllers.NewFieldController(&fieldTestUOW{
		fieldRepo:    fieldRepo,
		scheduleRepo: &mockScheduleRepo{},
	}, cache)

	if _, err := controller.Update(context.Background(), field.ID, dto.UpdateFieldRequest{}); err != nil {
		t.Fatalf("update field: %v", err)
	}
	if cache.listInvalidate != 1 || cache.detailInvalidate != 1 {
		t.Fatalf("cache invalidations = (list %d, detail %d), want (1, 1)", cache.listInvalidate, cache.detailInvalidate)
	}
}

func TestFieldUpdateDoesNotInvalidateCacheWhenDatabaseWriteFails(t *testing.T) {
	field := &models.Field{ID: 7, Name: "Lapangan Alpha", Type: models.FieldTypePadel}
	fieldRepo := &mockFieldRepo{field: field, updateErr: errors.New("database unavailable")}
	cache := &mockFieldCache{}
	controller := controllers.NewFieldController(&fieldTestUOW{
		fieldRepo:    fieldRepo,
		scheduleRepo: &mockScheduleRepo{},
	}, cache)

	if _, err := controller.Update(context.Background(), field.ID, dto.UpdateFieldRequest{}); err == nil {
		t.Fatal("update succeeded despite database error")
	}
	if cache.listInvalidate != 0 || cache.detailInvalidate != 0 {
		t.Fatalf("cache invalidated after failed database write: list %d, detail %d", cache.listInvalidate, cache.detailInvalidate)
	}
}

func TestFieldUpdateReturnsDatabaseDetailWhenCacheInvalidationFails(t *testing.T) {
	field := &models.Field{ID: 7, Name: "Old Name", Type: models.FieldTypePadel}
	fieldRepo := &mockFieldRepo{field: field}
	staleCacheEntry := &dto.FieldResponse{ID: field.ID, Name: "Old Name", Type: string(models.FieldTypePadel)}
	cache := &mockFieldCache{
		listInvalidateErr: errors.New("redis unavailable"),
		detailData:        staleCacheEntry,
		detailVersion:     "0",
		detailHit:         true,
	}
	controller := controllers.NewFieldController(&fieldTestUOW{
		fieldRepo:    fieldRepo,
		scheduleRepo: &mockScheduleRepo{},
	}, cache)
	newName := "New Name"

	response, err := controller.Update(context.Background(), field.ID, dto.UpdateFieldRequest{Name: &newName})
	if err != nil {
		t.Fatalf("update field with cache unavailable: %v", err)
	}
	if response.Name != newName {
		t.Fatalf("update response name = %q, want fresh database value %q", response.Name, newName)
	}
}

func TestAvailabilityPastDateGraceful(t *testing.T) {
	mockF := &models.Field{
		ID:           1,
		Name:         "Lapangan Alpha",
		PricePerHour: 100000,
		Status:       models.FieldStatusActive,
	}
	uow := &fieldTestUOW{
		fieldRepo:    &mockFieldRepo{field: mockF},
		scheduleRepo: &mockScheduleRepo{schedule: nil},
	}

	controller := controllers.NewFieldController(uow)
	ctx := context.Background()

	// Tanggal di masa lalu (misal 2020-01-01)
	res, err := controller.Availability(ctx, 1, "2020-01-01")
	if err != nil {
		t.Fatalf("expected nil error for past date availability, got: %v", err)
	}
	if res.IsOpen {
		t.Errorf("expected IsOpen to be false for past date, got true")
	}
	if len(res.Slots) != 0 {
		t.Errorf("expected 0 slots for past date, got %d", len(res.Slots))
	}
}

func TestAvailabilityTodayPassedHours(t *testing.T) {
	baseTime := time.Now()
	if baseTime.Hour() == 0 {
		t.Skip("cannot check a passed same-day hourly slot before 01:00")
	}
	pastHour := time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day(), baseTime.Hour()-1, 0, 0, 0, baseTime.Location()).Format("15:04")
	nextHour := time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day(), baseTime.Hour()+1, 0, 0, 0, baseTime.Location()).Format("15:04")
	closeTime := "23:00"

	mockF := &models.Field{
		ID:           1,
		Name:         "Lapangan Alpha",
		PricePerHour: 100000,
		Status:       models.FieldStatusActive,
	}
	mockSched := &models.Schedule{
		FieldID:   1,
		Day:       models.DayFromWeekday(baseTime.Weekday()),
		OpenTime:  "00:00",
		CloseTime: closeTime,
		IsClosed:  false,
	}

	uow := &fieldTestUOW{
		fieldRepo:    &mockFieldRepo{field: mockF},
		scheduleRepo: &mockScheduleRepo{schedule: mockSched},
		mockUOW: mockUOW{
			bookingRepo: &mockBookingRepo{},
		},
	}

	controller := controllers.NewFieldController(uow)
	ctx := context.Background()

	todayStr := baseTime.Format("2006-01-02")
	res, err := controller.Availability(ctx, 1, todayStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.IsOpen {
		t.Fatalf("expected IsOpen to be true for open schedule")
	}

	foundPassedSlot := false
	foundFutureSlot := false
	for _, slot := range res.Slots {
		if slot.StartTime == pastHour {
			foundPassedSlot = true
			if slot.Available {
				t.Errorf("slot %s should not be available because time has passed", pastHour)
			}
			if slot.Reason != "time has passed" {
				t.Errorf("expected reason 'time has passed', got '%s'", slot.Reason)
			}
		}
		if slot.StartTime == nextHour && baseTime.Hour() < 22 {
			foundFutureSlot = true
			if !slot.Available {
				t.Errorf("slot %s should be available", nextHour)
			}
		}
	}

	if !foundPassedSlot {
		t.Errorf("did not find expected passed slot %s in availability response", pastHour)
	}
	if baseTime.Hour() < 22 && !foundFutureSlot {
		t.Errorf("did not find expected future slot %s in availability response", nextHour)
	}
}
