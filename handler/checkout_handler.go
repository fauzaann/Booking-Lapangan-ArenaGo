package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/middleware"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

type CheckoutHandler struct {
	Base
	checkouts controllers.CheckoutController
}

func NewCheckoutHandler(checkouts controllers.CheckoutController, validate *validator.Validator) *CheckoutHandler {
	return &CheckoutHandler{Base: NewBase(validate), checkouts: checkouts}
}

func (h *CheckoutHandler) Create(c *gin.Context) {
	var req dto.CheckoutRequest
	if !h.bindJSON(c, &req) {
		return
	}
	result, err := h.checkouts.Create(c.Request.Context(), middleware.MustActor(c), req)
	if err != nil {
		var conflict *controllers.CheckoutConflictError
		if errors.As(err, &conflict) {
			c.JSON(409, response.Body{
				Success: false,
				Message: "Lapangan ini sudah ter-booking. Periksa item yang ditandai di keranjang.",
				Error:   gin.H{"conflicts": conflict.Conflicts},
			})
			return
		}
		response.Error(c, err)
		return
	}
	response.Created(c, "Checkout created", result)
}

func (h *CheckoutHandler) Detail(c *gin.Context) {
	id, ok := uintParam(c, "id")
	if !ok {
		return
	}
	result, err := h.checkouts.Detail(c.Request.Context(), middleware.MustActor(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Checkout retrieved", result)
}
