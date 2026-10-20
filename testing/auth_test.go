package controllers_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestPrivateUserEndpointsRequireAuthentication(t *testing.T) {
	for _, path := range []string{
		"/api/v1/bookings",
		"/api/v1/notifications",
		"/api/v1/notifications/unread-count",
	} {
		name := strings.ReplaceAll(strings.Trim(path, "/"), "/", "_")
		t.Run(name, func(t *testing.T) {
			response, body := requestAPI(t, http.MethodGet, path, nil)
			requireHTTPStatus(t, response, body, http.StatusUnauthorized)
		})
	}
}
