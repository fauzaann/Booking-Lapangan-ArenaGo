// Package jwt menangani pembuatan dan verifikasi access token.
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken dikembalikan untuk token yang tidak valid atau kedaluwarsa.
var ErrInvalidToken = errors.New("invalid or expired token")

const issuer = "booking-field-api"

// Claims adalah payload JWT aplikasi ini.
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwtlib.RegisteredClaims
}

// Manager adalah kontrak token service agar mudah di-mock saat testing.
type Manager interface {
	Generate(userID uint, email, role string) (string, time.Time, error)
	Verify(token string) (*Claims, error)
}

type hmacManager struct {
	secret   []byte
	duration time.Duration
}

// NewManager membuat Manager berbasis HMAC-SHA256.
func NewManager(secret string, duration time.Duration) Manager {
	return &hmacManager{secret: []byte(secret), duration: duration}
}

func (m *hmacManager) Generate(userID uint, email, role string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.duration)

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    issuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}
	return signed, expiresAt, nil
}

func (m *hmacManager) Verify(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwtlib.ParseWithClaims(token, claims, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	}, jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}), jwtlib.WithIssuer(issuer))

	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
