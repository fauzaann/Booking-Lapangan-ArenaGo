// Package controller berisi seluruh business logic. Controller tidak
// mengetahui HTTP maupun GORM: ia hanya bergantung pada interface repository
// dan service eksternal sehingga mudah diuji.
package controllers

import (
	"time"

	"Booking-Lapangan/models"
)

// timeNow dipakai seluruh controller agar waktu dapat dipalsukan saat unit test.
var timeNow = time.Now

// Actor adalah identitas pemanggil yang sudah terautentikasi.
type Actor struct {
	UserID uint
	Email  string
	Role   models.Role
}

// IsAdmin menandakan pemanggil memiliki hak administrator.
func (a Actor) IsAdmin() bool {
	return a.Role == models.RoleAdmin
}
