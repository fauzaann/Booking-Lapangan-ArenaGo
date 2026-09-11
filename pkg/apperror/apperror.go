// Package apperror menyediakan tipe error terpusat yang membawa informasi
// HTTP status, sehingga layer handler tidak perlu menebak-nebak status code.
package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError adalah error domain aplikasi.
type AppError struct {
	Status  int    // HTTP status code
	Message string // pesan yang aman untuk ditampilkan ke client
	Detail  string // detail tambahan (opsional)
	Err     error  // error asli (tidak pernah dikirim ke client)
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// WithDetail mengembalikan salinan error dengan detail tambahan.
func (e *AppError) WithDetail(detail string) *AppError {
	clone := *e
	clone.Detail = detail
	return &clone
}

func New(status int, message string, err error) *AppError {
	return &AppError{Status: status, Message: message, Err: err}
}

func BadRequest(message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: message}
}

func Unprocessable(message string) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Message: message}
}

func BadGateway(message string, err error) *AppError {
	return &AppError{Status: http.StatusBadGateway, Message: message, Err: err}
}

func Internal(message string, err error) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Message: message, Err: err}
}

// From mengubah error apa pun menjadi *AppError. Error yang tidak dikenali
// dianggap sebagai internal server error agar detailnya tidak bocor.
func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal("internal server error", err)
}

// IsNotFound membantu controller membedakan error "tidak ditemukan".
func IsNotFound(err error) bool {
	return statusOf(err) == http.StatusNotFound
}

// IsConflict membantu controller membedakan error konflik data.
func IsConflict(err error) bool {
	return statusOf(err) == http.StatusConflict
}

func statusOf(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return 0
}
