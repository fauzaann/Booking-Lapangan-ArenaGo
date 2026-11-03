package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Sender interface {
	SendTemplate(ctx context.Context, recipient, name, language string, parameters []string) error
}

type Client struct {
	phoneNumberID string
	accessToken   string
	baseURL       string
	httpClient    *http.Client
}

func NewClient(phoneNumberID, accessToken, baseURL string) *Client {
	return &Client{
		phoneNumberID: strings.TrimSpace(phoneNumberID),
		accessToken:   strings.TrimSpace(accessToken),
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SendTemplate(ctx context.Context, recipient, name, language string, parameters []string) error {
	if c.phoneNumberID == "" || c.accessToken == "" || c.baseURL == "" {
		return fmt.Errorf("WhatsApp Cloud API is not configured")
	}
	recipient, err := NormalizePhone(recipient)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("WhatsApp template name is required")
	}
	if language = strings.TrimSpace(language); language == "" {
		language = "id"
	}

	body := struct {
		MessagingProduct string `json:"messaging_product"`
		RecipientType    string `json:"recipient_type"`
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
			} `json:"components,omitempty"`
		} `json:"template"`
	}{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               recipient,
		Type:             "template",
	}
	body.Template.Name = name
	body.Template.Language.Code = language
	if len(parameters) > 0 {
		component := struct {
			Type       string `json:"type"`
			Parameters []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"parameters"`
		}{Type: "body"}
		for _, parameter := range parameters {
			component.Parameters = append(component.Parameters, struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{Type: "text", Text: parameter})
		}
		body.Template.Components = append(body.Template.Components, component)
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal WhatsApp template request: %w", err)
	}
	endpoint := c.baseURL + "/" + url.PathEscape(c.phoneNumberID) + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create WhatsApp template request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send WhatsApp template: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("WhatsApp Cloud API returned HTTP %d (response body unreadable): %w", response.StatusCode, readErr)
		}
		return fmt.Errorf("WhatsApp Cloud API returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return nil
}

func NormalizePhone(phone string) (string, error) {
	value := strings.TrimSpace(phone)
	if strings.HasPrefix(value, "+") {
		value = value[1:]
	}
	var digits strings.Builder
	for _, character := range value {
		if character >= '0' && character <= '9' {
			digits.WriteRune(character)
			continue
		}
		if character != ' ' && character != '-' && character != '(' && character != ')' {
			return "", fmt.Errorf("WhatsApp phone number contains an invalid character")
		}
	}
	normalized := digits.String()
	if strings.HasPrefix(normalized, "00") {
		normalized = normalized[2:]
	} else if strings.HasPrefix(normalized, "0") {
		normalized = "62" + normalized[1:]
	} else if strings.HasPrefix(normalized, "8") {
		normalized = "62" + normalized
	}
	if len(normalized) < 8 || len(normalized) > 15 {
		return "", fmt.Errorf("WhatsApp phone number must be 8 to 15 digits in international format")
	}
	return normalized, nil
}
