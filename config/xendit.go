package config

import (
	"time"

	"Booking-Lapangan/utils"
)

// NewXenditClient membangun client Xendit dari konfigurasi environment.
// Secret key tidak pernah ditulis di source code.
func NewXenditClient(cfg *Config) *utils.Client {
	return utils.NewClient(utils.Options{
		SecretKey: cfg.Xendit.SecretKey,
		BaseURL:   cfg.Xendit.BaseURL,
		Timeout:   15 * time.Second,
	})
}
