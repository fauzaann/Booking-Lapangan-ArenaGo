// Package handler adalah lapisan HTTP. Tugasnya hanya membaca request,
// memvalidasi payload, memanggil controller, dan menulis response.
// Tidak ada query database maupun logic pembayaran di sini.
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"Booking-Lapangan/pkg/apperror"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

// Base menyediakan helper yang dipakai seluruh handler.
type Base struct {
	validate *validator.Validator
}

// NewBase membuat Base handler.
func NewBase(validate *validator.Validator) Base {
	return Base{validate: validate}
}

// bindJSON membaca body JSON lalu memvalidasinya.
// Mengembalikan false jika response error sudah ditulis.
func (b Base) bindJSON(c *gin.Context, payload interface{}) bool {
	if err := c.ShouldBindJSON(payload); err != nil {
		response.Error(c, apperror.BadRequest("invalid request body: "+err.Error()))
		return false
	}
	if fields := b.validate.Validate(payload); fields != nil {
		response.ValidationFailed(c, fields)
		return false
	}
	return true
}

// bindQuery membaca query string ke dalam struct.
func (b Base) bindQuery(c *gin.Context, payload interface{}) bool {
	if err := c.ShouldBindQuery(payload); err != nil {
		response.Error(c, apperror.BadRequest("invalid query parameter: "+err.Error()))
		return false
	}
	return true
}

// uintParam membaca path parameter numerik (misal /fields/:id).
func uintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		response.Error(c, apperror.BadRequest(name+" must be a positive number"))
		return 0, false
	}
	return uint(value), true
}
