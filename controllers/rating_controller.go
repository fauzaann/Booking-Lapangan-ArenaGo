package controllers

import (
	"context"
	"log"
	"strings"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/fieldcache"
	"Booking-Lapangan/repository"
)

type RatingController interface {
	Create(ctx context.Context, actor Actor, bookingID uint, req dto.CreateRatingRequest) (*dto.RatingResponse, error)
	BookingRating(ctx context.Context, actor Actor, bookingID uint) (*dto.RatingResponse, error)
	FieldSummary(ctx context.Context, fieldID uint) (*dto.RatingSummaryResponse, error)
	FieldRatings(ctx context.Context, fieldID uint, query dto.PaginationQuery) ([]dto.RatingResponse, int64, error)
	AdminRatings(ctx context.Context, query dto.RatingListQuery) ([]dto.RatingResponse, int64, error)
	SetVisibility(ctx context.Context, ratingID uint, visible bool) (*dto.RatingResponse, error)
}

type ratingController struct {
	uow   repository.UnitOfWork
	cache fieldcache.Store
}

func NewRatingController(uow repository.UnitOfWork, caches ...fieldcache.Store) RatingController {
	controller := &ratingController{uow: uow}
	if len(caches) > 0 {
		controller.cache = caches[0]
	}
	return controller
}

func (c *ratingController) Create(ctx context.Context, actor Actor, bookingID uint, req dto.CreateRatingRequest) (*dto.RatingResponse, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, apperror.Unprocessable("rating must be between 1 and 5")
	}

	rating := &models.Rating{}
	err := c.uow.Atomic(ctx, func(r repository.Registry) error {
		booking, err := r.Booking().FindDetailByIDForUpdate(ctx, bookingID)
		if err != nil {
			return err
		}
		if booking.UserID != actor.UserID {
			return apperror.Forbidden("booking does not belong to this user")
		}
		if booking.Status != models.BookingStatusCompleted {
			return apperror.Conflict("only completed bookings can be rated")
		}
		if _, err := r.Rating().FindByBookingID(ctx, booking.ID); err == nil {
			return apperror.Conflict("this booking already has a rating")
		} else if !apperror.IsNotFound(err) {
			return err
		}

		*rating = models.Rating{
			BookingID: booking.ID,
			UserID:    booking.UserID,
			FieldID:   booking.FieldID,
			Rating:    req.Rating,
			Review:    strings.TrimSpace(req.Review),
			IsVisible: true,
		}
		return r.Rating().Create(ctx, rating)
	})
	if err != nil {
		return nil, err
	}
	c.invalidateListCache(ctx)
	response := dto.NewRatingResponse(*rating)
	return &response, nil
}

func (c *ratingController) BookingRating(ctx context.Context, actor Actor, bookingID uint) (*dto.RatingResponse, error) {
	booking, err := c.uow.Booking().FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking.UserID != actor.UserID {
		return nil, apperror.Forbidden("booking does not belong to this user")
	}
	rating, err := c.uow.Rating().FindByBookingID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	response := dto.NewRatingResponse(*rating)
	return &response, nil
}

func (c *ratingController) FieldSummary(ctx context.Context, fieldID uint) (*dto.RatingSummaryResponse, error) {
	if _, err := c.uow.Field().FindByID(ctx, fieldID); err != nil {
		return nil, err
	}
	summary, err := c.uow.Rating().SummaryByFieldID(ctx, fieldID)
	if err != nil {
		return nil, err
	}
	response := dto.NewRatingSummaryResponse(fieldID, summary.Average, summary.Total)
	return &response, nil
}

func (c *ratingController) FieldRatings(ctx context.Context, fieldID uint, query dto.PaginationQuery) ([]dto.RatingResponse, int64, error) {
	if _, err := c.uow.Field().FindByID(ctx, fieldID); err != nil {
		return nil, 0, err
	}
	query = query.Normalize()
	ratings, total, err := c.uow.Rating().ListPublicByFieldID(ctx, fieldID, repository.ListParams{
		Page: query.Page, Limit: query.Limit,
	})
	if err != nil {
		return nil, 0, err
	}
	return dto.NewRatingResponses(ratings), total, nil
}

func (c *ratingController) AdminRatings(ctx context.Context, query dto.RatingListQuery) ([]dto.RatingResponse, int64, error) {
	query.PaginationQuery = query.PaginationQuery.Normalize()
	ratings, total, err := c.uow.Rating().ListAdmin(ctx, query.IsVisible, repository.ListParams{
		Page: query.Page, Limit: query.Limit,
	})
	if err != nil {
		return nil, 0, err
	}
	return dto.NewAdminRatingResponses(ratings), total, nil
}

func (c *ratingController) SetVisibility(ctx context.Context, ratingID uint, visible bool) (*dto.RatingResponse, error) {
	if err := c.uow.Rating().SetVisibility(ctx, ratingID, visible); err != nil {
		return nil, err
	}
	c.invalidateListCache(ctx)
	rating, err := c.uow.Rating().FindByID(ctx, ratingID)
	if err != nil {
		return nil, err
	}
	response := dto.NewAdminRatingResponses([]models.Rating{*rating})[0]
	return &response, nil
}

func (c *ratingController) invalidateListCache(ctx context.Context) {
	if c.cache == nil {
		return
	}
	if err := c.cache.InvalidateLists(ctx); err != nil {
		log.Printf("field cache: list invalidation failed after rating mutation: %v", err)
	}
}
