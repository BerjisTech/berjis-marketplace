package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.stripe.com/v1"

type Client struct {
	SecretKey string
	HTTP      *http.Client
}

type PaymentIntent struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	Status       string `json:"status"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
}

type Refund struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Amount int64  `json:"amount"`
}

type WebhookEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type webhookData struct {
	Object json.RawMessage `json:"object"`
}

func NewClient(secretKey string) *Client {
	return &Client{
		SecretKey: secretKey,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) CreatePaymentIntent(amountCents int64, currency string, idempotencyKey string, metadata map[string]string) (*PaymentIntent, error) {
	data := url.Values{}
	data.Set("amount", strconv.FormatInt(amountCents, 10))
	data.Set("currency", strings.ToLower(currency))
	data.Set("automatic_payment_methods[enabled]", "true")
	for k, v := range metadata {
		data.Set(fmt.Sprintf("metadata[%s]", k), v)
	}

	req, err := http.NewRequest("POST", apiBase+"/payment_intents", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("stripe: create request: %w", err)
	}
	req.SetBasicAuth(c.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stripe: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe: API error %d: %s", resp.StatusCode, string(body))
	}

	var pi PaymentIntent
	if err := json.Unmarshal(body, &pi); err != nil {
		return nil, fmt.Errorf("stripe: decode response: %w", err)
	}
	return &pi, nil
}

func (c *Client) CreateRefund(paymentIntentID string, amountCents int64, idempotencyKey string) (*Refund, error) {
	data := url.Values{}
	data.Set("payment_intent", paymentIntentID)
	if amountCents > 0 {
		data.Set("amount", strconv.FormatInt(amountCents, 10))
	}

	req, err := http.NewRequest("POST", apiBase+"/refunds", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("stripe: create request: %w", err)
	}
	req.SetBasicAuth(c.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stripe: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe: API error %d: %s", resp.StatusCode, string(body))
	}

	var ref Refund
	if err := json.Unmarshal(body, &ref); err != nil {
		return nil, fmt.Errorf("stripe: decode response: %w", err)
	}
	return &ref, nil
}

func VerifyWebhookSignature(payload []byte, sigHeader string, secret string) (*WebhookEvent, error) {
	parts := parseSignatureHeader(sigHeader)
	timestamp := parts["t"]
	sig := parts["v1"]
	if timestamp == "" || sig == "" {
		return nil, fmt.Errorf("stripe: missing signature components")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("stripe: invalid timestamp")
	}

	tolerance := 5 * time.Minute
	if time.Since(time.Unix(ts, 0)) > tolerance {
		return nil, fmt.Errorf("stripe: timestamp too old")
	}

	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return nil, fmt.Errorf("stripe: signature mismatch")
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("stripe: decode event: %w", err)
	}
	return &event, nil
}

func parseSignatureHeader(header string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(header, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func ParsePaymentIntentFromEvent(event *WebhookEvent) (*PaymentIntent, error) {
	var d webhookData
	if err := json.Unmarshal(event.Data, &d); err != nil {
		return nil, fmt.Errorf("stripe: decode data: %w", err)
	}
	var pi PaymentIntent
	if err := json.Unmarshal(d.Object, &pi); err != nil {
		return nil, fmt.Errorf("stripe: decode payment intent: %w", err)
	}
	return &pi, nil
}
