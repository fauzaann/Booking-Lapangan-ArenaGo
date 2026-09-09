package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/utils"
)

// Base menyediakan helper yang dipakai seluruh handler.
type Base struct {
	Validator *utils.Validator
}

// NewBase membuat Base handler.
func NewBase(validate *utils.Validator) Base {
	return Base{Validator: validate}
}

// BindJSON membaca body JSON lalu memvalidasinya.
// Mengembalikan false jika response error sudah ditulis.
func (b Base) BindJSON(c *gin.Context, payload interface{}) bool {
	if err := c.ShouldBindJSON(payload); err != nil {
		dto.Error(c, utils.BadRequest("invalid request body: "+err.Error()))
		return false
	}
	if b.Validator != nil {
		if fields := b.Validator.Validate(payload); fields != nil {
			dto.ValidationFailed(c, fields)
			return false
		}
	}
	return true
}

// BindQuery membaca query string ke dalam struct.
func (b Base) BindQuery(c *gin.Context, payload interface{}) bool {
	if err := c.ShouldBindQuery(payload); err != nil {
		dto.Error(c, utils.BadRequest("invalid query parameter: "+err.Error()))
		return false
	}
	return true
}

// UintParam membaca path parameter numerik (misal /fields/:id).
func UintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		dto.Error(c, utils.BadRequest(name+" must be a positive number"))
		return 0, false
	}
	return uint(value), true
}
