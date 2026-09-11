package config

import (
	"time"

	"Booking-Lapangan/pkg/xendit"
)

// NewXenditClient membangun client Xendit dari konfigurasi environment.
// Secret key tidak pernah ditulis di source code.
func NewXenditClient(cfg *Config) *xendit.Client {
	return xendit.NewClient(xendit.Options{
		SecretKey: cfg.Xendit.SecretKey,
		BaseURL:   cfg.Xendit.BaseURL,
		Timeout:   15 * time.Second,
	})
}
