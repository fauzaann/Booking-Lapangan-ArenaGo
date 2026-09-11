package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/middleware"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

// UserHandler menangani profil user dan daftar user untuk admin.
type UserHandler struct {
	Base
	auth  controllers.AuthController
	admin controllers.AdminController
}

// NewUserHandler membuat UserHandler.
func NewUserHandler(auth controllers.AuthController, admin controllers.AdminController, validate *validator.Validator) *UserHandler {
	return &UserHandler{Base: NewBase(validate), auth: auth, admin: admin}
}

// Profile menangani GET /api/v1/me.
func (h *UserHandler) Profile(c *gin.Context) {
	actor := middleware.MustActor(c)

	result, err := h.auth.Profile(c.Request.Context(), actor.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Profile retrieved", result)
}

// List menangani GET /api/v1/admin/users.
func (h *UserHandler) List(c *gin.Context) {
	var query dto.PaginationQuery
	if !h.bindQuery(c, &query) {
		return
	}
	query = query.Normalize()

	users, total, err := h.admin.ListUsers(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, "Users retrieved", users, response.NewMeta(query.Page, query.Limit, total))
}
