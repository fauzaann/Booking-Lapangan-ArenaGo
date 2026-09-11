// Package config memuat seluruh konfigurasi aplikasi dari environment.
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config adalah kumpulan konfigurasi aplikasi.
type Config struct {
	App        AppConfig
	Database   DatabaseConfig
	JWT        JWTConfig
	Xendit     XenditConfig
	OpenRouter OpenRouterConfig
	Booking    BookingConfig
}

// AppConfig berisi konfigurasi umum aplikasi.
type AppConfig struct {
	Port           string
	Env            string
	FrontendURL    string
	AutoMigrate    bool
	RunSeeder      bool
	AllowedOrigins []string
}

// IsProduction menandakan aplikasi berjalan di environment produksi.
func (a AppConfig) IsProduction() bool {
	return strings.EqualFold(a.Env, "production")
}

// DatabaseConfig berisi konfigurasi koneksi PostgreSQL.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

// DSN menyusun connection string PostgreSQL.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.TimeZone,
	)
}

// BookingConfig berisi aturan bisnis booking yang dapat diubah lewat environment.
type BookingConfig struct {
	MinHours       int
	MaxHours       int
	CancelMinHours int
	ExpirerMinutes int
}

// JWTConfig berisi konfigurasi token.
type JWTConfig struct {
	Secret   string
	Duration time.Duration
}

// XenditConfig berisi kredensial dan opsi payment gateway.
type XenditConfig struct {
	SecretKey          string
	WebhookToken       string
	BaseURL            string
	InvoiceDuration    int
	SuccessRedirectURL string
	FailureRedirectURL string
}

// OpenRouterConfig berisi konfigurasi provider chatbot.
type OpenRouterConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// Load membaca file .env (jika ada) lalu environment variable.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("config: .env not loaded (%v), falling back to OS environment", err)
	}

	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")

	cfg := &Config{
		App: AppConfig{
			Port:           getEnv("APP_PORT", "8080"),
			Env:            getEnv("APP_ENV", "development"),
			FrontendURL:    frontendURL,
			AutoMigrate:    getEnvBool("APP_AUTO_MIGRATE", true),
			RunSeeder:      getEnvBool("RUN_SEEDER", false),
			AllowedOrigins: splitAndTrim(getEnv("CORS_ALLOWED_ORIGINS", "*")),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "booking_field"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			TimeZone: getEnv("DB_TIMEZONE", "Asia/Jakarta"),
		},
		JWT: JWTConfig{
			Secret:   getEnv("JWT_SECRET", ""),
			Duration: time.Duration(getEnvInt("JWT_EXPIRE_HOUR", 24)) * time.Hour,
		},
		Xendit: XenditConfig{
			SecretKey:          getEnv("XENDIT_SECRET_KEY", ""),
			WebhookToken:       getEnv("XENDIT_WEBHOOK_TOKEN", ""),
			BaseURL:            getEnv("XENDIT_BASE_URL", "https://api.xendit.co"),
			InvoiceDuration:    getEnvInt("XENDIT_INVOICE_DURATION", 3600),
			SuccessRedirectURL: strings.TrimSuffix(frontendURL, "/") + "/payment/success",
			FailureRedirectURL: strings.TrimSuffix(frontendURL, "/") + "/payment/failed",
		},
		OpenRouter: OpenRouterConfig{
			APIKey:  getEnv("OPENROUTER_API_KEY", ""),
			BaseURL: getEnv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
			Model:   getEnv("OPENROUTER_MODEL", "openai/gpt-4o-mini"),
		},
		Booking: BookingConfig{
			MinHours:       getEnvInt("BOOKING_MIN_HOUR", 1),
			MaxHours:       getEnvInt("BOOKING_MAX_HOUR", 8),
			CancelMinHours: getEnvInt("BOOKING_CANCEL_MIN_HOUR", 24),
			ExpirerMinutes: getEnvInt("BOOKING_EXPIRER_MINUTE", 1),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if len(c.JWT.Secret) < 16 {
		return fmt.Errorf("JWT_SECRET must be set and at least 16 characters long")
	}
	if c.JWT.Duration <= 0 {
		return fmt.Errorf("JWT_EXPIRE_HOUR must be greater than 0")
	}

	if c.Xendit.SecretKey == "" {
		if c.App.IsProduction() {
			return fmt.Errorf("XENDIT_SECRET_KEY is required in production")
		}
		log.Println("config: XENDIT_SECRET_KEY is empty, payment creation will fail until it is set")
	}
	if c.Booking.MinHours < 1 {
		return fmt.Errorf("BOOKING_MIN_HOUR must be at least 1")
	}
	if c.Booking.MaxHours < c.Booking.MinHours {
		return fmt.Errorf("BOOKING_MAX_HOUR must be greater than or equal to BOOKING_MIN_HOUR")
	}

	if c.Xendit.WebhookToken == "" {
		if c.App.IsProduction() {
			return fmt.Errorf("XENDIT_WEBHOOK_TOKEN is required in production")
		}
		log.Println("config: XENDIT_WEBHOOK_TOKEN is empty, webhook endpoint will reject every callback")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("config: %s is not a valid integer, using default %d", key, fallback)
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("config: %s is not a valid boolean, using default %v", key, fallback)
		return fallback
	}
	return parsed
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
