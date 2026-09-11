package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

// ScheduleHandler menangani jam operasional lapangan.
type ScheduleHandler struct {
	Base
	fields controllers.FieldController
}

// NewScheduleHandler membuat ScheduleHandler.
func NewScheduleHandler(fields controllers.FieldController, validate *validator.Validator) *ScheduleHandler {
	return &ScheduleHandler{Base: NewBase(validate), fields: fields}
}

// List menangani GET /api/v1/fields/:id/schedules.
func (h *ScheduleHandler) List(c *gin.Context) {
	fieldID, ok := uintParam(c, "id")
	if !ok {
		return
	}

	schedules, err := h.fields.ListSchedules(c.Request.Context(), fieldID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Schedules retrieved", schedules)
}

// Upsert menangani PUT /api/v1/admin/fields/:id/schedules.
func (h *ScheduleHandler) Upsert(c *gin.Context) {
	fieldID, ok := uintParam(c, "id")
	if !ok {
		return
	}

	var req dto.UpsertScheduleRequest
	if !h.bindJSON(c, &req) {
		return
	}

	schedules, err := h.fields.UpsertSchedules(c.Request.Context(), fieldID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Schedules updated", schedules)
}
