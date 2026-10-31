package controllers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/fieldcache"
	"Booking-Lapangan/repository"
)

type ratingTestRepository struct {
	items      map[uint]models.Rating
	nextID     uint
	createErr  error
	visibility bool
}

func newRatingTestRepository() *ratingTestRepository {
	return &ratingTestRepository{items: make(map[uint]models.Rating), nextID: 1}
}

func (r *ratingTestRepository) Create(_ context.Context, rating *models.Rating) error {
	if r.createErr != nil {
		return r.createErr
	}
	if _, err := r.FindByBookingID(context.Background(), rating.BookingID); err == nil {
		return apperror.Conflict("this booking already has a rating")
	}
	rating.ID = r.nextID
	r.nextID++
	rating.CreatedAt = time.Now()
	r.items[rating.ID] = *rating
	return nil
}

func (r *ratingTestRepository) FindByID(_ context.Context, id uint) (*models.Rating, error) {
	rating, ok := r.items[id]
	if !ok {
		return nil, apperror.NotFound("rating not found")
	}
	return &rating, nil
}

func (r *ratingTestRepository) FindByBookingID(_ context.Context, bookingID uint) (*models.Rating, error) {
	for _, rating := range r.items {
		if rating.BookingID == bookingID {
			return &rating, nil
		}
	}
	return nil, apperror.NotFound("rating not found")
}

func (r *ratingTestRepository) ListPublicByFieldID(_ context.Context, fieldID uint, params repository.ListParams) ([]models.Rating, int64, error) {
	var ratings []models.Rating
	for _, rating := range r.items {
		if rating.FieldID == fieldID && rating.IsVisible {
			ratings = append(ratings, rating)
		}
	}
	return ratings, int64(len(ratings)), nil
}

func (r *ratingTestRepository) SummaryByFieldID(_ context.Context, fieldID uint) (repository.RatingSummary, error) {
	var summary repository.RatingSummary
	for _, rating := range r.items {
		if rating.FieldID == fieldID && rating.IsVisible {
			summary.Average += float64(rating.Rating)
			summary.Total++
		}
	}
	if summary.Total > 0 {
		summary.Average /= float64(summary.Total)
	}
	return summary, nil
}

func (r *ratingTestRepository) SummariesByFieldIDs(_ context.Context, fieldIDs []uint) (map[uint]repository.RatingSummary, error) {
	summaries := make(map[uint]repository.RatingSummary, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		summary, err := r.SummaryByFieldID(context.Background(), fieldID)
		if err != nil {
			return nil, err
		}
		if summary.Total > 0 {
			summaries[fieldID] = summary
		}
	}
	return summaries, nil
}

func (r *ratingTestRepository) ListAdmin(_ context.Context, visible *bool, params repository.ListParams) ([]models.Rating, int64, error) {
	var ratings []models.Rating
	for _, rating := range r.items {
		if visible == nil || rating.IsVisible == *visible {
			ratings = append(ratings, rating)
		}
	}
	return ratings, int64(len(ratings)), nil
}

func (r *ratingTestRepository) SetVisibility(_ context.Context, id uint, visible bool) error {
	rating, ok := r.items[id]
	if !ok {
		return apperror.NotFound("rating not found")
	}
	rating.IsVisible = visible
	r.items[id] = rating
	r.visibility = visible
	return nil
}

func newRatingTestController(booking *models.Booking, ratings *ratingTestRepository, caches ...fieldcache.Store) controllers.RatingController {
	return controllers.NewRatingController(&mockUOW{
		bookingRepo: &mockBookingRepo{booking: booking},
		fieldRepo:   &mockFieldRepo{field: &models.Field{ID: 7, Type: models.FieldTypePadel}},
		ratingRepo:  ratings,
	}, caches...)
}

func TestRatingCreateRequiresCompletedOwnedBooking(t *testing.T) {
	booking := &models.Booking{ID: 25, UserID: 4, FieldID: 7, Status: models.BookingStatusCompleted}
	ratings := newRatingTestRepository()
	cache := &mockFieldCache{}
	controller := newRatingTestController(booking, ratings, cache)

	result, err := controller.Create(context.Background(), controllers.Actor{UserID: 4}, 25, dto.CreateRatingRequest{
		Rating: 5, Review: "  Lapangan bersih.  ",
	})
	if err != nil {
		t.Fatalf("create rating: %v", err)
	}
	if result.BookingID != 25 || result.FieldID != 7 || result.Rating != 5 || result.Review != "Lapangan bersih." {
		t.Fatalf("unexpected rating response: %+v", result)
	}
	if result.Author != "Pelanggan terverifikasi" {
		t.Fatalf("rating author = %q, want masked label", result.Author)
	}
	if cache.listInvalidate != 1 {
		t.Fatalf("rating list cache invalidations = %d, want 1", cache.listInvalidate)
	}

	if _, err := controller.Create(context.Background(), controllers.Actor{UserID: 4}, 25, dto.CreateRatingRequest{Rating: 4}); err == nil || !apperror.IsConflict(err) {
		t.Fatalf("second rating error = %v, want conflict", err)
	}
}

func TestRatingCreateRejectsOtherOwnerAndUnfinishedBooking(t *testing.T) {
	booking := &models.Booking{ID: 25, UserID: 4, FieldID: 7, Status: models.BookingStatusCompleted}
	controller := newRatingTestController(booking, newRatingTestRepository())
	if _, err := controller.Create(context.Background(), controllers.Actor{UserID: 5}, 25, dto.CreateRatingRequest{Rating: 5}); err == nil || !isForbidden(err) {
		t.Fatalf("other user's rating error = %v, want forbidden", err)
	}

	booking.Status = models.BookingStatusConfirmed
	if _, err := controller.Create(context.Background(), controllers.Actor{UserID: 4}, 25, dto.CreateRatingRequest{Rating: 5}); err == nil || !apperror.IsConflict(err) {
		t.Fatalf("unfinished booking rating error = %v, want conflict", err)
	}
}

func TestRatingCreateRejectsOutOfRangeValue(t *testing.T) {
	controller := newRatingTestController(
		&models.Booking{ID: 25, UserID: 4, FieldID: 7, Status: models.BookingStatusCompleted},
		newRatingTestRepository(),
	)
	if _, err := controller.Create(context.Background(), controllers.Actor{UserID: 4}, 25, dto.CreateRatingRequest{Rating: 6}); err == nil || !isUnprocessable(err) {
		t.Fatalf("rating of 6 error = %v, want unprocessable", err)
	}
}

func TestPublicRatingListMasksBookingIdentity(t *testing.T) {
	ratings := newRatingTestRepository()
	ratings.items[1] = models.Rating{ID: 1, BookingID: 25, FieldID: 7, Rating: 4, IsVisible: true}
	controller := newRatingTestController(&models.Booking{}, ratings)

	result, total, err := controller.FieldRatings(context.Background(), 7, dto.PaginationQuery{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("list field ratings: %v", err)
	}
	if total != 1 || len(result) != 1 {
		t.Fatalf("ratings = %d, total %d; want one public rating", len(result), total)
	}
	if result[0].BookingID != 0 || result[0].Author != "Pelanggan terverifikasi" {
		t.Fatalf("public rating exposed private details: %+v", result[0])
	}
}

func TestBookingRatingRejectsOtherOwner(t *testing.T) {
	controller := newRatingTestController(
		&models.Booking{ID: 25, UserID: 4, FieldID: 7},
		newRatingTestRepository(),
	)
	if _, err := controller.BookingRating(context.Background(), controllers.Actor{UserID: 5}, 25); err == nil || !isForbidden(err) {
		t.Fatalf("other user's rating read error = %v, want forbidden", err)
	}
}

func TestRatingModerationRemovesReviewFromPublicSummary(t *testing.T) {
	ratings := newRatingTestRepository()
	ratings.items[1] = models.Rating{ID: 1, BookingID: 25, FieldID: 7, Rating: 5, IsVisible: true}
	controller := newRatingTestController(&models.Booking{}, ratings)

	result, err := controller.SetVisibility(context.Background(), 1, false)
	if err != nil {
		t.Fatalf("hide rating: %v", err)
	}
	if result.IsVisible == nil || *result.IsVisible {
		t.Fatalf("moderation result visibility = %v, want false", result.IsVisible)
	}

	summary, err := controller.FieldSummary(context.Background(), 7)
	if err != nil {
		t.Fatalf("rating summary: %v", err)
	}
	if summary.TotalReviews != 0 || summary.AverageRating != 0 {
		t.Fatalf("public summary = %+v, want empty summary", summary)
	}
}

func TestRatingCreatePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	ratings := newRatingTestRepository()
	ratings.createErr = wantErr
	controller := newRatingTestController(
		&models.Booking{ID: 25, UserID: 4, FieldID: 7, Status: models.BookingStatusCompleted},
		ratings,
	)
	if _, err := controller.Create(context.Background(), controllers.Actor{UserID: 4}, 25, dto.CreateRatingRequest{Rating: 4}); !errors.Is(err, wantErr) {
		t.Fatalf("create error = %v, want database error", err)
	}
}

func isForbidden(err error) bool {
	var appErr *apperror.AppError
	return errors.As(err, &appErr) && appErr.Status == 403
}

func isUnprocessable(err error) bool {
	var appErr *apperror.AppError
	return errors.As(err, &appErr) && appErr.Status == 422
}
