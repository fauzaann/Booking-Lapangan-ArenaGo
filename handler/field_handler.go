package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/utils"
)

// FieldHandler menangani endpoint lapangan (publik dan admin).
type FieldHandler struct {
	Base
	fields controllers.FieldController
}

// NewFieldHandler membuat FieldHandler.
func NewFieldHandler(fields controllers.FieldController, validate *utils.Validator) *FieldHandler {
	return &FieldHandler{Base: NewBase(validate), fields: fields}
}

// List menangani GET /api/v1/fields dengan filter dan pagination.
func (h *FieldHandler) List(c *gin.Context) {
	var query dto.FieldFilter
	if !h.BindQuery(c, &query) {
		return
	}
	query.PaginationQuery = query.PaginationQuery.Normalize()

	fields, total, err := h.fields.List(c.Request.Context(), query)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.Paginated(c, "Fields retrieved", fields, utils.NewMeta(query.Page, query.Limit, total))
}

// Detail menangani GET /api/v1/fields/:id.
func (h *FieldHandler) Detail(c *gin.Context) {
	id, ok := UintParam(c, "id")
	if !ok {
		return
	}

	field, err := h.fields.Detail(c.Request.Context(), id)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Field retrieved", field)
}

// Availability menangani GET /api/v1/fields/:id/availability?date=YYYY-MM-DD.
func (h *FieldHandler) Availability(c *gin.Context) {
	id, ok := UintParam(c, "id")
	if !ok {
		return
	}

	date := c.Query("date")
	if date == "" {
		utils.Error(c, utils.BadRequest("date query parameter is required (YYYY-MM-DD)"))
		return
	}

	result, err := h.fields.Availability(c.Request.Context(), id, date)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Availability retrieved", result)
}

// Create menangani POST /api/v1/admin/fields.
func (h *FieldHandler) Create(c *gin.Context) {
	var req dto.CreateFieldRequest
	if !h.BindJSON(c, &req) {
		return
	}

	field, err := h.fields.Create(c.Request.Context(), req)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Field created", field)
}

// Update menangani PUT /api/v1/admin/fields/:id.
func (h *FieldHandler) Update(c *gin.Context) {
	id, ok := UintParam(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateFieldRequest
	if !h.BindJSON(c, &req) {
		return
	}

	field, err := h.fields.Update(c.Request.Context(), id, req)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Field updated", field)
}

// Delete menangani DELETE /api/v1/admin/fields/:id.
func (h *FieldHandler) Delete(c *gin.Context) {
	id, ok := UintParam(c, "id")
	if !ok {
		return
	}

	if err := h.fields.Delete(c.Request.Context(), id); err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Field deleted", nil)
}
