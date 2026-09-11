// Package xendit adalah client minimal untuk Xendit Invoice API.
// Client ini sengaja ditulis manual agar dependency tetap sedikit dan
// perilaku HTTP-nya (timeout, error handling) terlihat jelas.
package xendit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Status invoice yang dikirim Xendit.
const (
	StatusPending = "PENDING"
	StatusPaid    = "PAID"
	StatusSettled = "SETTLED"
	StatusExpired = "EXPIRED"
)

// InvoiceService adalah kontrak payment gateway yang dipakai controller.
type InvoiceService interface {
	CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*Invoice, error)
	GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error)
}

// Customer adalah data pembayar opsional pada invoice.
type Customer struct {
	GivenNames   string `json:"given_names,omitempty"`
	Email        string `json:"email,omitempty"`
	MobileNumber string `json:"mobile_number,omitempty"`
}

// Item adalah rincian barang/jasa pada invoice.
type Item struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Category string  `json:"category,omitempty"`
}

// CreateInvoiceRequest adalah payload pembuatan invoice.
type CreateInvoiceRequest struct {
	ExternalID         string    `json:"external_id"`
	Amount             float64   `json:"amount"`
	Description        string    `json:"description,omitempty"`
	InvoiceDuration    int       `json:"invoice_duration,omitempty"`
	PayerEmail         string    `json:"payer_email,omitempty"`
	Customer           *Customer `json:"customer,omitempty"`
	Items              []Item    `json:"items,omitempty"`
	Currency           string    `json:"currency,omitempty"`
	SuccessRedirectURL string    `json:"success_redirect_url,omitempty"`
	FailureRedirectURL string    `json:"failure_redirect_url,omitempty"`
}

// Invoice adalah representasi invoice yang dikembalikan Xendit.
type Invoice struct {
	ID             string     `json:"id"`
	ExternalID     string     `json:"external_id"`
	Status         string     `json:"status"`
	Amount         float64    `json:"amount"`
	PaidAmount     float64    `json:"paid_amount"`
	InvoiceURL     string     `json:"invoice_url"`
	ExpiryDate     *time.Time `json:"expiry_date"`
	PaidAt         *time.Time `json:"paid_at"`
	PaymentMethod  string     `json:"payment_method"`
	PaymentChannel string     `json:"payment_channel"`
	Description    string     `json:"description"`
}

// WebhookPayload adalah body callback invoice dari Xendit.
type WebhookPayload struct {
	ID             string     `json:"id"`
	ExternalID     string     `json:"external_id"`
	Status         string     `json:"status"`
	Amount         float64    `json:"amount"`
	PaidAmount     float64    `json:"paid_amount"`
	PaymentMethod  string     `json:"payment_method"`
	PaymentChannel string     `json:"payment_channel"`
	PaidAt         *time.Time `json:"paid_at"`
}

// APIError merepresentasikan error dari Xendit API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"error_code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("xendit api error (%d %s): %s", e.StatusCode, e.Code, e.Message)
}

// Options adalah konfigurasi client.
type Options struct {
	SecretKey  string
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// Client adalah implementasi InvoiceService berbasis HTTP.
type Client struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

// NewClient membuat client Xendit.
func NewClient(opt Options) *Client {
	baseURL := strings.TrimSuffix(opt.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.xendit.co"
	}
	httpClient := opt.HTTPClient
	if httpClient == nil {
		timeout := opt.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{secretKey: opt.SecretKey, baseURL: baseURL, httpClient: httpClient}
}

// CreateInvoice membuat invoice baru di Xendit.
func (c *Client) CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*Invoice, error) {
	if req.Currency == "" {
		req.Currency = "IDR"
	}
	return c.do(ctx, http.MethodPost, "/v2/invoices", req)
}

// GetInvoice mengambil invoice berdasarkan id (server-side verification).
func (c *Client) GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error) {
	return c.do(ctx, http.MethodGet, "/v2/invoices/"+invoiceID, nil)
}

func (c *Client) do(ctx context.Context, method, path string, payload interface{}) (*Invoice, error) {
	if c.secretKey == "" {
		return nil, fmt.Errorf("xendit secret key is not configured")
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to encode xendit request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build xendit request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	// Xendit memakai HTTP Basic Auth: secret key sebagai username, password kosong.
	request.SetBasicAuth(c.secretKey, "")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call xendit: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read xendit response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(raw, apiErr); err != nil || apiErr.Message == "" {
			apiErr.Message = strings.TrimSpace(string(raw))
		}
		return nil, apiErr
	}

	invoice := &Invoice{}
	if err := json.Unmarshal(raw, invoice); err != nil {
		return nil, fmt.Errorf("failed to decode xendit response: %w", err)
	}
	return invoice, nil
}
