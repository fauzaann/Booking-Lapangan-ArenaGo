package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/middleware"
	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

// BookingHandler menangani endpoint booking.
type BookingHandler struct {
	Base
	bookings controllers.BookingController
}

// NewBookingHandler membuat BookingHandler.
func NewBookingHandler(bookings controllers.BookingController, validate *validator.Validator) *BookingHandler {
	return &BookingHandler{Base: NewBase(validate), bookings: bookings}
}

// Create menangani POST /api/v1/bookings.
func (h *BookingHandler) Create(c *gin.Context) {
	var req dto.CreateBookingRequest
	if !h.bindJSON(c, &req) {
		return
	}

	actor := middleware.MustActor(c)
	result, err := h.bookings.Create(c.Request.Context(), actor, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, "Booking created", result)
}

// List menangani GET /api/v1/bookings dan GET /api/v1/admin/bookings.
func (h *BookingHandler) List(c *gin.Context) {
	var query dto.BookingFilterQuery
	if !h.bindQuery(c, &query) {
		return
	}
	query.PaginationQuery = query.PaginationQuery.Normalize()

	actor := middleware.MustActor(c)
	bookings, total, err := h.bookings.List(c.Request.Context(), actor, query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, "Bookings retrieved", bookings, response.NewMeta(query.Page, query.Limit, total))
}

// Detail menangani GET /api/v1/bookings/:id.
func (h *BookingHandler) Detail(c *gin.Context) {
	id, ok := uintParam(c, "id")
	if !ok {
		return
	}

	actor := middleware.MustActor(c)
	booking, err := h.bookings.Detail(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Booking retrieved", booking)
}

// Cancel menangani DELETE /api/v1/bookings/:id (pembatalan, bukan hard delete).
func (h *BookingHandler) Cancel(c *gin.Context) {
	id, ok := uintParam(c, "id")
	if !ok {
		return
	}

	actor := middleware.MustActor(c)
	booking, err := h.bookings.Cancel(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Booking cancelled", booking)
}

// UpdateStatus menangani PATCH /api/v1/admin/bookings/:id/status.
func (h *BookingHandler) UpdateStatus(c *gin.Context) {
	id, ok := uintParam(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateBookingStatusRequest
	if !h.bindJSON(c, &req) {
		return
	}

	booking, err := h.bookings.UpdateStatus(c.Request.Context(), id, models.BookingStatus(req.Status))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Booking status updated", booking)
}
