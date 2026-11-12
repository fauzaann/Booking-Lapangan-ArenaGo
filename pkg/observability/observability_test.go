package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMiddlewareRecordsBookingConflictsWithRouteTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	telemetry, err := New(context.Background())
	if err != nil {
		t.Fatalf("create observability: %v", err)
	}

	engine := gin.New()
	engine.Use(telemetry.Middleware())
	engine.POST("/api/v1/bookings", func(c *gin.Context) {
		c.Status(http.StatusConflict)
	})
	engine.GET("/metrics", telemetry.MetricsHandler())

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/bookings", nil))
	if response.Code != http.StatusConflict {
		t.Fatalf("booking status = %d, want %d", response.Code, http.StatusConflict)
	}

	response = httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := response.Body.String()
	for _, metric := range []string{
		`arenago_booking_conflicts_total 1`,
		`arenago_http_requests_total{method="POST",route="/api/v1/bookings",status="409"} 1`,
	} {
		if !strings.Contains(body, metric) {
			t.Errorf("metrics output missing %q:\n%s", metric, body)
		}
	}
}

func TestIsPaymentOperation(t *testing.T) {
	if !isPaymentOperation("/api/v1/checkouts/:id/sync-payment") {
		t.Fatal("expected payment sync route to be classified as payment operation")
	}
	if isPaymentOperation("/api/v1/bookings/:id") {
		t.Fatal("booking detail route must not be classified as payment operation")
	}
}
