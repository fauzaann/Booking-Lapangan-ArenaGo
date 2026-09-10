package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/utils"
)

// AdminHandler menangani endpoint khusus administrator.
type AdminHandler struct {
	Base
	admin controllers.AdminController
}

// NewAdminHandler membuat AdminHandler.
func NewAdminHandler(admin controllers.AdminController, validate *utils.Validator) *AdminHandler {
	return &AdminHandler{Base: NewBase(validate), admin: admin}
}

// Dashboard menangani GET /api/v1/admin/dashboard.
func (h *AdminHandler) Dashboard(c *gin.Context) {
	data, err := h.admin.Dashboard(c.Request.Context())
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Dashboard retrieved", data)
}

// ListUsers menangani GET /api/v1/admin/users.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var query dto.PaginationQuery
	if !h.BindQuery(c, &query) {
		return
	}
	query = query.Normalize()

	users, total, err := h.admin.ListUsers(c.Request.Context(), query)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.Paginated(c, "Users retrieved", users, utils.NewMeta(query.Page, query.Limit, total))
}
