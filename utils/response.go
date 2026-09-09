// Package response menstandarkan seluruh body JSON API.
package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"

)

// Meta berisi informasi pagination.
type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// Body adalah bentuk baku seluruh response.
type Body struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// NewMeta menghitung total halaman dari total item.
func NewMeta(page, limit int, total int64) *Meta {
	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return &Meta{Page: page, Limit: limit, TotalItems: total, TotalPages: totalPages}
}

// Success menulis response sukses dengan status kustom.
func Success(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, Body{Success: true, Message: message, Data: data})
}

// OK menulis response 200.
func OK(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusOK, message, data)
}

// Created menulis response 201.
func Created(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusCreated, message, data)
}

// Paginated menulis response list beserta meta pagination.
func Paginated(c *gin.Context, message string, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Body{Success: true, Message: message, Data: data, Meta: meta})
}

// Error menerjemahkan error apa pun menjadi response JSON yang konsisten.
// Error internal tetap dicatat lewat c.Error agar terlihat di log, tetapi
// detailnya tidak dikirim ke client.
func apperror.Error(c *gin.Context, err error) {
	appErr := apperror.From(err)

	if appErr.Err != nil {
		_ = c.Error(appErr.Err)
	}

	var detail interface{}
	if appErr.Detail != "" {
		detail = appErr.Detail
	}

	c.JSON(appErr.Status, Body{Success: false, Message: appErr.Message, Error: detail})
}

// AbortWithError menulis error lalu menghentikan chain middleware.
func AbortWithError(c *gin.Context, err error) {
	Error(c, err)
	c.Abort()
}

// ValidationFailed menulis response 422 berisi detail error per field.
func ValidationFailed(c *gin.Context, fields map[string]string) {
	c.JSON(http.StatusUnprocessableEntity, Body{
		Success: false,
		Message: "validation failed",
		Error:   fields,
	})
}
