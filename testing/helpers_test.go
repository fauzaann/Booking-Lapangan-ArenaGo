package controllers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const apiURLEnv = "ARENAGO_API_URL"

type apiEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func apiURL(t *testing.T, path string) string {
	t.Helper()

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(apiURLEnv)), "/")
	if baseURL == "" {
		t.Skipf("set %s to run API smoke tests against a running ArenaGO backend", apiURLEnv)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		t.Fatalf("%s must be an absolute HTTP(S) URL", apiURLEnv)
	}
	return baseURL + path
}

func requestAPI(t *testing.T, method, path string, body any) (*http.Response, []byte) {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, apiURL(t, path), requestBody)
	if err != nil {
		t.Fatalf("create API request: %v", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 8 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	t.Cleanup(func() {
		_ = response.Body.Close()
	})

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		t.Fatalf("read response for %s %s: %v", method, path, err)
	}
	return response, responseBody
}

func decodeEnvelope(t *testing.T, body []byte) apiEnvelope {
	t.Helper()

	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode API response: %v", err)
	}
	return envelope
}

func requireSuccessResponse(t *testing.T, response *http.Response, body []byte, wantStatus int) apiEnvelope {
	t.Helper()

	if response.StatusCode != wantStatus {
		t.Fatalf("HTTP status = %d, want %d; response: %s", response.StatusCode, wantStatus, body)
	}
	envelope := decodeEnvelope(t, body)
	if !envelope.Success {
		t.Fatalf("API success = false; response: %s", body)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		t.Fatalf("API response data is empty; response: %s", body)
	}
	return envelope
}

func requireHTTPStatus(t *testing.T, response *http.Response, body []byte, wantStatus int) {
	t.Helper()

	if response.StatusCode != wantStatus {
		t.Fatalf("HTTP status = %d, want %d; response: %s", response.StatusCode, wantStatus, body)
	}
}

func fieldPath(id uint, suffix string) string {
	return fmt.Sprintf("/api/v1/fields/%d%s", id, suffix)
}
