package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

// AdminHandler menangani endpoint dashboard administrator.
type AdminHandler struct {
	Base
	admin controllers.AdminController
}

// NewAdminHandler membuat AdminHandler.
func NewAdminHandler(admin controllers.AdminController, validate *validator.Validator) *AdminHandler {
	return &AdminHandler{Base: NewBase(validate), admin: admin}
}

// Dashboard menangani GET /api/v1/admin/dashboard.
func (h *AdminHandler) Dashboard(c *gin.Context) {
	result, err := h.admin.Dashboard(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Dashboard retrieved", result)
}
