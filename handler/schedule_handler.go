package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/utils"
)

// ScheduleHandler menangani jam operasional lapangan.
type ScheduleHandler struct {
	Base
	fields controllers.FieldController
}

// NewScheduleHandler membuat ScheduleHandler.
func NewScheduleHandler(fields controllers.FieldController, validate *utils.Validator) *ScheduleHandler {
	return &ScheduleHandler{Base: NewBase(validate), fields: fields}
}

// List menangani GET /api/v1/fields/:id/schedules.
func (h *ScheduleHandler) List(c *gin.Context) {
	fieldID, ok := UintParam(c, "id")
	if !ok {
		return
	}

	schedules, err := h.fields.ListSchedules(c.Request.Context(), fieldID)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Schedules retrieved", schedules)
}

// Upsert menangani PUT /api/v1/admin/fields/:id/schedules.
func (h *ScheduleHandler) Upsert(c *gin.Context) {
	fieldID, ok := UintParam(c, "id")
	if !ok {
		return
	}

	var req dto.UpsertScheduleRequest
	if !h.BindJSON(c, &req) {
		return
	}

	schedules, err := h.fields.UpsertSchedules(c.Request.Context(), fieldID, req)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Schedules updated", schedules)
}
