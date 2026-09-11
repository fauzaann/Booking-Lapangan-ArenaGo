package config

import (
	"net/http"
	"time"

	"Booking-Lapangan/pkg/openrouter"
)

func NewOpenRouterClient(cfg *Config) *openrouter.Client {
	return openrouter.NewClient(openrouter.Options{
		APIKey:  cfg.OpenRouter.APIKey,
		BaseURL: cfg.OpenRouter.BaseURL,
		Model:   cfg.OpenRouter.Model,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	})
}
