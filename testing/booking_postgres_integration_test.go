package controllers_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/database"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/timeutil"
	"Booking-Lapangan/repository"
)

func TestPostgresConcurrentReschedulesCannotTakeSameSlot(t *testing.T) {
	dsn := os.Getenv("ARENAGO_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ARENAGO_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect to configured test PostgreSQL: %v", err)
	}
	adminSQL, err := adminDB.DB()
	if err != nil {
		t.Fatalf("get PostgreSQL connection pool: %v", err)
	}
	defer adminSQL.Close()

	schema := fmt.Sprintf("arenago_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}
	defer func() {
		if err := adminDB.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error; err != nil {
			t.Errorf("drop isolated test schema: %v", err)
		}
	}()

	scopedDB, err := gorm.Open(postgres.Open(postgresSchemaDSN(t, dsn, schema)), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect to isolated PostgreSQL schema: %v", err)
	}
	scopedSQL, err := scopedDB.DB()
	if err != nil {
		t.Fatalf("get isolated PostgreSQL connection pool: %v", err)
	}
	defer scopedSQL.Close()
	if err := database.Migrate(scopedDB); err != nil {
		t.Fatalf("migrate isolated PostgreSQL schema: %v", err)
	}

	ctx := context.Background()
	date := futureBookingDate(7)
	users := []models.User{
		{Name: "Booking Test One", Email: fmt.Sprintf("%d-one@example.test", time.Now().UnixNano()), Password: "unused", Role: models.RoleUser},
		{Name: "Booking Test Two", Email: fmt.Sprintf("%d-two@example.test", time.Now().UnixNano()), Password: "unused", Role: models.RoleUser},
	}
	for i := range users {
		if err := scopedDB.WithContext(ctx).Create(&users[i]).Error; err != nil {
			t.Fatalf("seed test user: %v", err)
		}
	}
	field := models.Field{
		Name: "Integration Test Field", Type: models.FieldTypePadel, Location: "Test",
		PricePerHour: 100000, Status: models.FieldStatusActive,
	}
	if err := scopedDB.WithContext(ctx).Create(&field).Error; err != nil {
		t.Fatalf("seed test field: %v", err)
	}
	schedule := models.Schedule{
		FieldID: field.ID, Day: models.DayFromWeekday(date.Weekday()),
		OpenTime: "06:00", CloseTime: "23:00",
	}
	if err := scopedDB.WithContext(ctx).Create(&schedule).Error; err != nil {
		t.Fatalf("seed field schedule: %v", err)
	}

	bookings := []models.Booking{
		{
			BookingCode: fmt.Sprintf("IT-%d-1", time.Now().UnixNano()),
			UserID:      users[0].ID, FieldID: field.ID, BookingDate: date,
			StartTime: "10:00", EndTime: "11:00", Duration: 1,
			TotalPrice: 100000, Status: models.BookingStatusPending,
		},
		{
			BookingCode: fmt.Sprintf("IT-%d-2", time.Now().UnixNano()),
			UserID:      users[1].ID, FieldID: field.ID, BookingDate: date,
			StartTime: "12:00", EndTime: "13:00", Duration: 1,
			TotalPrice: 100000, Status: models.BookingStatusPending,
		},
	}
	for i := range bookings {
		if err := scopedDB.WithContext(ctx).Create(&bookings[i]).Error; err != nil {
			t.Fatalf("seed test booking: %v", err)
		}
	}

	controller := controllers.NewBookingController(repository.NewUnitOfWork(scopedDB), nil, controllers.BookingPolicy{
		MinHours: 1, MaxHours: 8, CancelMinHours: 24,
	})
	req := dto.RescheduleBookingRequest{
		BookingDate: date.Format(timeutil.DateLayout), StartTime: "14:00", EndTime: "15:00",
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(bookings))
	for i := range bookings {
		wg.Add(1)
		go func(booking models.Booking) {
			defer wg.Done()
			_, err := controller.Reschedule(ctx, controllers.Actor{UserID: booking.UserID}, booking.ID, req)
			errs <- err
		}(bookings[i])
	}
	wg.Wait()
	close(errs)

	succeeded, conflicted := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case apperror.IsConflict(err):
			conflicted++
		default:
			t.Fatalf("unexpected reschedule error: %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("expected one reschedule success and one conflict; got %d success and %d conflict", succeeded, conflicted)
	}

	var occupying []models.Booking
	if err := scopedDB.WithContext(ctx).
		Where("field_id = ? AND booking_date = ? AND start_time = ?", field.ID, date, "14:00").
		Find(&occupying).Error; err != nil {
		t.Fatalf("verify committed slot: %v", err)
	}
	if len(occupying) != 1 {
		t.Fatalf("expected exactly one booking in the target slot, got %d", len(occupying))
	}

	if err := scopedDB.Exec(fmt.Sprintf(`
		CREATE FUNCTION "%s".reject_booking_item_insert() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'simulated booking item persistence failure';
		END;
		$$`, schema)).Error; err != nil {
		t.Fatalf("create transaction-failure test function: %v", err)
	}
	if err := scopedDB.Exec(fmt.Sprintf(`
		CREATE TRIGGER reject_booking_item_insert
		BEFORE INSERT ON "%s".booking_items
		FOR EACH ROW EXECUTE FUNCTION "%s".reject_booking_item_insert()`, schema, schema)).Error; err != nil {
		t.Fatalf("create transaction-failure test trigger: %v", err)
	}

	rollbackReq := dto.RescheduleBookingRequest{
		BookingDate: date.Format(timeutil.DateLayout), StartTime: "16:00", EndTime: "17:00",
	}
	if _, err := controller.Reschedule(ctx, controllers.Actor{UserID: occupying[0].UserID}, occupying[0].ID, rollbackReq); err == nil {
		t.Fatal("expected simulated item persistence failure")
	}
	var persisted models.Booking
	if err := scopedDB.WithContext(ctx).First(&persisted, occupying[0].ID).Error; err != nil {
		t.Fatalf("read booking after failed transaction: %v", err)
	}
	if persisted.StartTime != "14:00" || len(persisted.RescheduleHistory) != 1 {
		t.Fatalf("failed transaction partially changed booking: %+v", persisted)
	}

	if _, err := controller.Cancel(ctx, controllers.Actor{UserID: persisted.UserID}, persisted.ID); err != nil {
		t.Fatalf("cancel rescheduled booking: %v", err)
	}
	overlap, err := repository.NewBookingRepository(scopedDB).HasOverlap(ctx, field.ID, date, "14:00", "15:00")
	if err != nil {
		t.Fatalf("check released slot after cancellation: %v", err)
	}
	if overlap {
		t.Fatal("cancelled booking still blocks the slot")
	}
}

func postgresSchemaDSN(t *testing.T, dsn, schema string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	if strings.Contains(dsn, "search_path=") {
		t.Fatal("ARENAGO_TEST_DATABASE_URL must not set search_path; the test selects an isolated schema")
	}
	return strings.TrimSpace(dsn) + " search_path=" + schema
}
