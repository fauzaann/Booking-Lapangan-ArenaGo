package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/middleware"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

type RatingHandler struct {
	Base
	ratings controllers.RatingController
}

func NewRatingHandler(ratings controllers.RatingController, validate *validator.Validator) *RatingHandler {
	return &RatingHandler{Base: NewBase(validate), ratings: ratings}
}

func (h *RatingHandler) Create(c *gin.Context) {
	bookingID, ok := uintParam(c, "id")
	if !ok {
		return
	}
	var req dto.CreateRatingRequest
	if !h.bindJSON(c, &req) {
		return
	}
	result, err := h.ratings.Create(c.Request.Context(), middleware.MustActor(c), bookingID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, "Rating created", result)
}

func (h *RatingHandler) BookingRating(c *gin.Context) {
	bookingID, ok := uintParam(c, "id")
	if !ok {
		return
	}
	result, err := h.ratings.BookingRating(c.Request.Context(), middleware.MustActor(c), bookingID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Rating retrieved", result)
}

func (h *RatingHandler) Summary(c *gin.Context) {
	fieldID, ok := uintParam(c, "id")
	if !ok {
		return
	}
	result, err := h.ratings.FieldSummary(c.Request.Context(), fieldID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Rating summary retrieved", result)
}

func (h *RatingHandler) ListField(c *gin.Context) {
	fieldID, ok := uintParam(c, "id")
	if !ok {
		return
	}
	var query dto.PaginationQuery
	if !h.bindQuery(c, &query) {
		return
	}
	query = query.Normalize()
	ratings, total, err := h.ratings.FieldRatings(c.Request.Context(), fieldID, query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, "Ratings retrieved", ratings, response.NewMeta(query.Page, query.Limit, total))
}

func (h *RatingHandler) AdminList(c *gin.Context) {
	var query dto.RatingListQuery
	if !h.bindQuery(c, &query) {
		return
	}
	query.PaginationQuery = query.PaginationQuery.Normalize()
	ratings, total, err := h.ratings.AdminRatings(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, "Ratings retrieved", ratings, response.NewMeta(query.Page, query.Limit, total))
}

func (h *RatingHandler) Moderate(c *gin.Context) {
	ratingID, ok := uintParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateRatingVisibilityRequest
	if !h.bindJSON(c, &req) {
		return
	}
	result, err := h.ratings.SetVisibility(c.Request.Context(), ratingID, *req.IsVisible)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Rating moderation updated", result)
}
