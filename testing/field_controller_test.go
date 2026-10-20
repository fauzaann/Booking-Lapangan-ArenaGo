package controllers_test

import (
	"context"
	"testing"
	"time"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
)

type mockFieldRepo struct {
	field *models.Field
}

func (m *mockFieldRepo) Create(ctx context.Context, field *models.Field) error { return nil }
func (m *mockFieldRepo) Update(ctx context.Context, field *models.Field) error { return nil }
func (m *mockFieldRepo) Delete(ctx context.Context, id uint) error             { return nil }
func (m *mockFieldRepo) FindByID(ctx context.Context, id uint) (*models.Field, error) {
	return m.field, nil
}
func (m *mockFieldRepo) FindDetailByID(ctx context.Context, id uint) (*models.Field, error) {
	return m.field, nil
}
func (m *mockFieldRepo) FindAll(ctx context.Context, filter repository.FieldFilter) ([]models.Field, int64, error) {
	return nil, 0, nil
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
