// Package apperror menyediakan tipe error terpusat yang membawa informasi
// HTTP status, sehingga layer handler tidak perlu menebak-nebak status code.
package utils

import (
	"errors"
	"fmt"
	"net/http"
)

// Error adalah error domain aplikasi.
type Error struct {
	Status  int    // HTTP status code
	Message string // pesan yang aman untuk ditampilkan ke client
	Detail  string // detail tambahan (opsional)
	Err     error  // error asli (tidak pernah dikirim ke client)
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// WithDetail mengembalikan salinan error dengan detail tambahan.
func (e *Error) WithDetail(detail string) *Error {
	clone := *e
	clone.Detail = detail
	return &clone
}

func New(status int, message string, err error) *Error {
	return &Error{Status: status, Message: message, Err: err}
}

func BadRequest(message string) *Error {
	return &Error{Status: http.StatusBadRequest, Message: message}
}

func Unauthorized(message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Status: http.StatusForbidden, Message: message}
}

func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Message: message}
}

func Conflict(message string) *Error {
	return &Error{Status: http.StatusConflict, Message: message}
}

func Unprocessable(message string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Message: message}
}

func BadGateway(message string, err error) *Error {
	return &Error{Status: http.StatusBadGateway, Message: message, Err: err}
}

func Internal(message string, err error) *Error {
	return &Error{Status: http.StatusInternalServerError, Message: message, Err: err}
}

// From mengubah error apa pun menjadi *Error. Error yang tidak dikenali
// dianggap sebagai internal server error agar detailnya tidak bocor.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
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
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return 0
}
