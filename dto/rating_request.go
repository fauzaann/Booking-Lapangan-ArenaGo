package dto

import (
	"strings"
	"time"

	"Booking-Lapangan/models"
)

type CreateRatingRequest struct {
	Rating int    `json:"rating" validate:"required,gte=1,lte=5"`
	Review string `json:"review" validate:"omitempty,max=1000"`
}

type UpdateRatingVisibilityRequest struct {
	IsVisible *bool `json:"is_visible" validate:"required"`
}

type RatingListQuery struct {
	PaginationQuery
	IsVisible *bool `form:"is_visible"`
}

type RatingSummaryResponse struct {
	FieldID       uint    `json:"field_id"`
	AverageRating float64 `json:"average_rating"`
	TotalReviews  int64   `json:"total_reviews"`
}

type RatingResponse struct {
	ID        uint      `json:"id"`
	BookingID uint      `json:"booking_id,omitempty"`
	FieldID   uint      `json:"field_id"`
	FieldName string    `json:"field_name,omitempty"`
	Author    string    `json:"author"`
	Rating    int       `json:"rating"`
	Review    string    `json:"review,omitempty"`
	IsVisible *bool     `json:"is_visible,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func NewRatingResponse(rating models.Rating) RatingResponse {
	return RatingResponse{
		ID:        rating.ID,
		BookingID: rating.BookingID,
		FieldID:   rating.FieldID,
		Author:    "Pelanggan terverifikasi",
		Rating:    rating.Rating,
		Review:    strings.TrimSpace(rating.Review),
		CreatedAt: rating.CreatedAt,
	}
}

func NewRatingResponses(ratings []models.Rating) []RatingResponse {
	responses := make([]RatingResponse, 0, len(ratings))
	for _, rating := range ratings {
		response := NewRatingResponse(rating)
		response.BookingID = 0
		responses = append(responses, response)
	}
	return responses
}

func NewAdminRatingResponses(ratings []models.Rating) []RatingResponse {
	responses := make([]RatingResponse, 0, len(ratings))
	for _, rating := range ratings {
		response := NewRatingResponse(rating)
		visible := rating.IsVisible
		response.IsVisible = &visible
		if rating.Field.ID != 0 {
			response.FieldName = rating.Field.Name
		}
		responses = append(responses, response)
	}
	return responses
}

func NewRatingSummaryResponse(fieldID uint, average float64, total int64) RatingSummaryResponse {
	return RatingSummaryResponse{
		FieldID:       fieldID,
		AverageRating: average,
		TotalReviews:  total,
	}
}
