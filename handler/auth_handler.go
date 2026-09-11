package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

// AuthHandler menangani endpoint /auth.
type AuthHandler struct {
	Base
	auth controllers.AuthController
}

// NewAuthHandler membuat AuthHandler.
func NewAuthHandler(auth controllers.AuthController, validate *validator.Validator) *AuthHandler {
	return &AuthHandler{Base: NewBase(validate), auth: auth}
}

// Register menangani POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !h.bindJSON(c, &req) {
		return
	}

	result, err := h.auth.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, "Registration successful", result)
}

// Login menangani POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !h.bindJSON(c, &req) {
		return
	}

	result, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Login successful", result)
}
