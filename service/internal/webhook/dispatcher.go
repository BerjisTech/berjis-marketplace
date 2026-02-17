package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	pq "github.com/lib/pq"
)

// Supported webhook event types.
var SupportedEvents = []string{
	"order.created",
	"order.paid",
	"order.cancelled",
	"order.refunded",
	"fulfillment.created",
	"fulfillment.shipped",
	"product.created",
	"product.updated",
	"product.deleted",
	"customer.created",
	"inventory.low_stock",
}

// retryDelays defines the exponential backoff schedule for delivery retries.
// Index 0 = delay before attempt 2, index 1 = delay before attempt 3, etc.
var retryDelays = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	12 * time.Hour,
}

const maxAttempts = 5

// Dispatcher handles dispatching and delivering webhook events.
type Dispatcher struct {
	DB     *sqlx.DB
	client *http.Client
}

// NewDispatcher creates a new Dispatcher with the given DB connection.
func NewDispatcher(db *sqlx.DB) *Dispatcher {
	return &Dispatcher{
		DB: db,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Dispatch creates delivery records for all active webhooks of a shop that match
// the given event type.
func (d *Dispatcher) Dispatch(shopUUID uuid.UUID, eventType string, payload map[string]any) {
	type webhookRow struct {
		UUID   uuid.UUID      `db:"uuid"`
		URL    string         `db:"url"`
		Secret string         `db:"secret"`
		Events pq.StringArray `db:"events"`
	}

	var hooks []webhookRow
	err := d.DB.Select(&hooks,
		`SELECT uuid, url, secret, events FROM webhooks WHERE shop_uuid=$1 AND is_active=true`, shopUUID)
	if err != nil {
		log.Printf("webhook dispatch: failed to query webhooks for shop %s: %v", shopUUID, err)
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook dispatch: failed to marshal payload: %v", err)
		return
	}

	for _, hook := range hooks {
		if !eventMatches(hook.Events, eventType) {
			continue
		}
		_, err := d.DB.Exec(
			`INSERT INTO webhook_deliveries (webhook_uuid, event_type, payload, status)
			 VALUES ($1, $2, $3, 'pending')`,
			hook.UUID, eventType, payloadBytes)
		if err != nil {
			log.Printf("webhook dispatch: failed to insert delivery for webhook %s: %v", hook.UUID, err)
		}
	}
}

// StartWorker starts a background goroutine that polls for pending webhook
// deliveries and attempts to deliver them. It respects the provided context
// for graceful shutdown.
func (d *Dispatcher) StartWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("webhook worker: shutting down")
				return
			case <-ticker.C:
				d.processPending(ctx)
			}
		}
	}()
}

type deliveryRow struct {
	UUID        uuid.UUID `db:"uuid"`
	WebhookUUID uuid.UUID `db:"webhook_uuid"`
	EventType   string    `db:"event_type"`
	Payload     []byte    `db:"payload"`
	Attempt     int       `db:"attempt"`
}

type webhookTarget struct {
	URL    string `db:"url"`
	Secret string `db:"secret"`
}

func (d *Dispatcher) processPending(ctx context.Context) {
	var deliveries []deliveryRow
	err := d.DB.SelectContext(ctx, &deliveries,
		`SELECT uuid, webhook_uuid, event_type, payload, attempt
		 FROM webhook_deliveries
		 WHERE status='pending' AND (next_retry_at IS NULL OR next_retry_at <= now())
		 ORDER BY created_at ASC
		 LIMIT 50`)
	if err != nil {
		log.Printf("webhook worker: query error: %v", err)
		return
	}

	for _, del := range deliveries {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var target webhookTarget
		err := d.DB.Get(&target, `SELECT url, secret FROM webhooks WHERE uuid=$1`, del.WebhookUUID)
		if err != nil {
			log.Printf("webhook worker: failed to get webhook %s: %v", del.WebhookUUID, err)
			d.markFailed(del.UUID, del.Attempt, "webhook not found")
			continue
		}

		d.deliver(del, target)
	}
}

func (d *Dispatcher) deliver(del deliveryRow, target webhookTarget) {
	body := del.Payload

	// Build the request
	req, err := http.NewRequest(http.MethodPost, target.URL, bytes.NewReader(body))
	if err != nil {
		d.handleFailure(del, fmt.Sprintf("invalid url: %v", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", del.EventType)
	req.Header.Set("X-Webhook-Delivery", del.UUID.String())

	// HMAC-SHA256 signature
	if target.Secret != "" {
		mac := hmac.New(sha256.New, []byte(target.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		d.handleFailure(del, fmt.Sprintf("request error: %v", err))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	respStatus := resp.StatusCode

	if respStatus >= 200 && respStatus < 300 {
		// Success
		_, _ = d.DB.Exec(
			`UPDATE webhook_deliveries
			 SET status='delivered', response_status=$1, response_body=$2, attempt=$3, updated_at=now(), error=NULL
			 WHERE uuid=$4`,
			respStatus, string(respBody), del.Attempt, del.UUID)
	} else {
		errMsg := fmt.Sprintf("http %d", respStatus)
		// Update response info
		_, _ = d.DB.Exec(
			`UPDATE webhook_deliveries
			 SET response_status=$1, response_body=$2, error=$3, updated_at=now()
			 WHERE uuid=$4`,
			respStatus, string(respBody), errMsg, del.UUID)
		d.handleFailure(del, errMsg)
	}
}

func (d *Dispatcher) handleFailure(del deliveryRow, errMsg string) {
	if del.Attempt >= maxAttempts {
		d.markFailed(del.UUID, del.Attempt, errMsg)
		return
	}

	// Schedule retry with exponential backoff
	delayIdx := del.Attempt - 1
	if delayIdx >= len(retryDelays) {
		delayIdx = len(retryDelays) - 1
	}
	nextRetry := time.Now().Add(retryDelays[delayIdx])

	_, _ = d.DB.Exec(
		`UPDATE webhook_deliveries
		 SET attempt=$1, next_retry_at=$2, error=$3, status='pending', updated_at=now()
		 WHERE uuid=$4`,
		del.Attempt+1, nextRetry, errMsg, del.UUID)
}

func (d *Dispatcher) markFailed(id uuid.UUID, attempt int, errMsg string) {
	_, _ = d.DB.Exec(
		`UPDATE webhook_deliveries
		 SET status='failed', error=$1, attempt=$2, updated_at=now()
		 WHERE uuid=$3`,
		errMsg, attempt, id)
}

func eventMatches(hookEvents []string, eventType string) bool {
	if len(hookEvents) == 0 {
		return false
	}
	for _, e := range hookEvents {
		if e == eventType || e == "*" {
			return true
		}
	}
	return false
}
