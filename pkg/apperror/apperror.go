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

// Error mengembalikan pesan error beserta penyebab internal jika tersedia.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap mengembalikan error asli agar dapat diperiksa dengan errors.Is/As.
func (e *AppError) Unwrap() error { return e.Err }

// WithDetail mengembalikan salinan error dengan detail tambahan.
func (e *AppError) WithDetail(detail string) *AppError {
	clone := *e
	clone.Detail = detail
	return &clone
}

// New membuat AppError dengan HTTP status dan penyebab opsional.
func New(status int, message string, err error) *AppError {
	return &AppError{Status: status, Message: message, Err: err}
}

// BadRequest membuat error untuk request yang tidak valid.
func BadRequest(message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: message}
}

// Unauthorized membuat error ketika autentikasi tidak valid atau tidak ada.
func Unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Message: message}
}

// Forbidden membuat error ketika actor tidak memiliki izin.
func Forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Message: message}
}

// NotFound membuat error ketika resource tidak ditemukan.
func NotFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: message}
}

// Conflict membuat error untuk konflik state atau data.
func Conflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: message}
}

// Unprocessable membuat error untuk data yang valid secara format tetapi ditolak aturan bisnis.
func Unprocessable(message string) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Message: message}
}

// BadGateway membuat error ketika service eksternal gagal.
func BadGateway(message string, err error) *AppError {
	return &AppError{Status: http.StatusBadGateway, Message: message, Err: err}
}

// Internal membuat error server internal dengan penyebab opsional.
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

// statusOf mengambil HTTP status dari AppError yang terbungkus.
func statusOf(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return 0
}
