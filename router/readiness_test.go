package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReadinessHandlerFailsOnlyForUnavailableRequiredDependencies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/ready", readinessHandler([]ReadinessCheck{
		{Name: "postgres", Required: true, Check: func(context.Context) error { return nil }},
		{Name: "redis", Required: false, Check: func(context.Context) error { return errors.New("offline") }},
	}))

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("readiness status = %d, want %d", response.Code, http.StatusOK)
	}
	if !containsAll(response.Body.String(), `"status":"degraded"`, `"redis":{"status":"unavailable","required":false}`) {
		t.Fatalf("unexpected degraded readiness response: %s", response.Body.String())
	}

	engine = gin.New()
	engine.GET("/ready", readinessHandler([]ReadinessCheck{
		{Name: "postgres", Required: true, Check: func(context.Context) error { return errors.New("offline") }},
	}))
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
