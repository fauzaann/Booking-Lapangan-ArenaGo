package handler

import (
	"Booking-Lapangan/dto"
	"Booking-Lapangan/middleware"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	auth *AuthHandler
}

func NewUserHandler(authHandler *AuthHandler) *UserHandler {
	return &UserHandler{auth: authHandler}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	actor := middleware.MustActor(c)
	result, err := h.auth.auth.Profile(c.Request.Context(), actor.ID)
	if err != nil {
		dto.Error(c, err)
		return
	}
	dto.OK(c, "Profile retrieved successfully", result)
}

