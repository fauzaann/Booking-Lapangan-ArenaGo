package handler

import (
    "Booking-Lapangan/controllers"
    "Booking-Lapangan/dto"
    "github.com/gin-gonic/gin"
)

type AuthHandler struct {
    auth *controllers.AuthController
}

func NewAuthHandler(authController *controllers.AuthController) *AuthHandler {
    return &AuthHandler{auth: authController}
}

func (h *AuthHandler) bindJSON(c *gin.Context, req any) bool {
    if err := c.ShouldBindJSON(req); err != nil {
        dto.Error(c, err)
        return false
    }
    return true
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req dto.RegisterRequest
    if !h.bindJSON(c, &req) {
        return
    }

    result, err := h.auth.Register(c.Request.Context(), req.Username, req.Email, req.Password)
    if err != nil {
        dto.Error(c, err)
        return
    }
    dto.Created(c, "Registration successful", result)
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req dto.LoginRequest
    if !h.bindJSON(c, &req) {
        return
    }

    result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
    if err != nil {
        dto.Error(c, err)
        return
    }
    dto.OK(c, "Login successful", result)
}