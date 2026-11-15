package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"Booking-Lapangan/pkg/jwt"
)

type accountStatusCheckerFunc func(context.Context, uint) (bool, error)

func (f accountStatusCheckerFunc) IsActive(ctx context.Context, userID uint) (bool, error) {
	return f(ctx, userID)
}

func TestAuthenticateChecksCurrentAccountStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwt.NewManager("test-secret-at-least-16-chars", time.Hour)
	token, _, err := manager.Generate(7, "user@example.com", "USER")
	if err != nil {
		t.Fatalf("generate test token: %v", err)
	}

	for _, testCase := range []struct {
		name       string
		checker    AccountStatusChecker
		wantStatus int
	}{
		{
			name: "active account",
			checker: accountStatusCheckerFunc(func(context.Context, uint) (bool, error) {
				return true, nil
			}),
			wantStatus: http.StatusOK,
		},
		{
			name: "inactive account",
			checker: accountStatusCheckerFunc(func(context.Context, uint) (bool, error) {
				return false, nil
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "status store unavailable",
			checker: accountStatusCheckerFunc(func(context.Context, uint) (bool, error) {
				return false, errors.New("database unavailable")
			}),
			wantStatus: http.StatusInternalServerError,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine := gin.New()
			engine.GET("/private", Authenticate(manager, testCase.checker), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			request := httptest.NewRequest(http.MethodGet, "/private", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != testCase.wantStatus {
				t.Fatalf("HTTP status = %d, want %d; body: %s", response.Code, testCase.wantStatus, response.Body)
			}
		})
	}
}
