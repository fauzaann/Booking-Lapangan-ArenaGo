package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendTemplateBuildsCloudAPIRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/12345/messages" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("unexpected authorization header %q", r.Header.Get("Authorization"))
		}

		var payload struct {
			MessagingProduct string `json:"messaging_product"`
			To               string `json:"to"`
			Type             string `json:"type"`
			Template         struct {
				Name     string `json:"name"`
				Language struct {
					Code string `json:"code"`
				} `json:"language"`
				Components []struct {
					Type       string `json:"type"`
					Parameters []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"parameters"`
				} `json:"components"`
			} `json:"template"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.MessagingProduct != "whatsapp" || payload.To != "6281234567890" || payload.Type != "template" {
			t.Fatalf("unexpected request payload: %+v", payload)
		}
		if payload.Template.Name != "booking_confirmed" || payload.Template.Language.Code != "id" {
			t.Fatalf("unexpected template: %+v", payload.Template)
		}
		if len(payload.Template.Components) != 1 || len(payload.Template.Components[0].Parameters) != 2 {
			t.Fatalf("unexpected template parameters: %+v", payload.Template.Components)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("12345", "test-token", server.URL)
	if err := client.SendTemplate(context.Background(), "+62 812-3456-7890", "booking_confirmed", "id", []string{"Rizki", "AG-01"}); err != nil {
		t.Fatalf("SendTemplate() error = %v", err)
	}
}

func TestSendTemplateReturnsCloudAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"template rejected"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient("12345", "test-token", server.URL)
	if err := client.SendTemplate(context.Background(), "081234567890", "invalid_template", "id", nil); err == nil {
		t.Fatal("SendTemplate() expected API error")
	}
}

func TestSendTemplateRejectsInvalidPhone(t *testing.T) {
	client := NewClient("12345", "test-token", "http://127.0.0.1")
	if err := client.SendTemplate(context.Background(), "not a phone", "booking_confirmed", "id", nil); err == nil {
		t.Fatal("SendTemplate() expected invalid phone error")
	}
}

func TestNormalizePhoneSupportsIndonesianLocalFormats(t *testing.T) {
	for input, expected := range map[string]string{
		"0812-3456-7890":     "6281234567890",
		"812-3456-7890":      "6281234567890",
		"0062 812 3456 7890": "6281234567890",
	} {
		got, err := NormalizePhone(input)
		if err != nil {
			t.Errorf("NormalizePhone(%q) error = %v", input, err)
			continue
		}
		if got != expected {
			t.Errorf("NormalizePhone(%q) = %q, want %q", input, got, expected)
		}
	}
}
