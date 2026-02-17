package email

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// Worker polls campaign_messages for scheduled emails and sends them.
type Worker struct {
	DB     *sqlx.DB
	Sender Sender
}

type campaignMsg struct {
	UUID        string     `db:"uuid"`
	ShopUUID    string     `db:"shop_uuid"`
	Subject     string     `db:"subject"`
	Body        string     `db:"body"`
	Status      string     `db:"status"`
	ScheduledAt time.Time  `db:"scheduled_at"`
	Error       *string    `db:"error"`
	RetryCount  int        `db:"retry_count"`
	Metadata    *string    `db:"metadata"`
}

const maxRetries = 3

// Start begins the background worker loop. It blocks until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	log.Println("email worker started")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("email worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	tx, err := w.DB.BeginTxx(ctx, nil)
	if err != nil {
		log.Printf("email worker: begin tx: %v", err)
		return
	}
	defer tx.Rollback()

	var messages []campaignMsg
	err = tx.SelectContext(ctx, &messages, `
		SELECT uuid, shop_uuid, subject, body, status,
		       scheduled_at, error,
		       COALESCE((metadata->>'retryCount')::int, 0) AS retry_count,
		       metadata::text AS metadata
		FROM campaign_messages
		WHERE status = 'scheduled'
		  AND scheduled_at <= now()
		ORDER BY scheduled_at
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		log.Printf("email worker: select messages: %v", err)
		return
	}

	if len(messages) == 0 {
		tx.Rollback()
		return
	}

	for _, msg := range messages {
		toEmail := w.resolveRecipient(tx, msg)
		if toEmail == "" {
			// No recipient found, mark as failed
			w.markFailed(tx, msg.UUID, "no recipient email found")
			continue
		}

		sendErr := w.Sender.Send(toEmail, msg.Subject, msg.Body)
		if sendErr != nil {
			retries := msg.RetryCount + 1
			if retries >= maxRetries {
				w.markFailed(tx, msg.UUID, sendErr.Error())
				w.logEmail(tx, msg.ShopUUID, toEmail, msg.Subject, "failed", sendErr.Error())
			} else {
				// Exponential backoff: 1m, 4m, 9m ...
				backoff := time.Duration(retries*retries) * time.Minute
				w.scheduleRetry(tx, msg.UUID, retries, backoff, sendErr.Error())
			}
			continue
		}

		w.markSent(tx, msg.UUID)
		w.logEmail(tx, msg.ShopUUID, toEmail, msg.Subject, "sent", "")
	}

	if err := tx.Commit(); err != nil {
		log.Printf("email worker: commit: %v", err)
	}
}

// resolveRecipient tries to find the email address for the message recipient.
// It checks the metadata for a userUuid and looks it up in user_profiles, then customers.
func (w *Worker) resolveRecipient(tx *sqlx.Tx, msg campaignMsg) string {
	if msg.Metadata == nil {
		return ""
	}

	// Try to extract userUuid from metadata JSON
	var userUUID string
	err := tx.Get(&userUUID, `SELECT $1::jsonb->>'userUuid'`, *msg.Metadata)
	if err != nil || userUUID == "" {
		return ""
	}

	// Try user_profiles first
	var email string
	err = tx.Get(&email, `SELECT email FROM user_profiles WHERE user_uuid=$1 AND email != ''`, userUUID)
	if err == nil && email != "" {
		return email
	}

	// Fallback to customers table
	err = tx.Get(&email, `SELECT email FROM customers WHERE user_uuid=$1 AND shop_uuid=$2 AND email != ''`, userUUID, msg.ShopUUID)
	if err == nil && email != "" {
		return email
	}

	return ""
}

func (w *Worker) markSent(tx *sqlx.Tx, msgUUID string) {
	_, err := tx.Exec(`
		UPDATE campaign_messages
		SET status = 'sent', sent_at = now(), updated_at = now()
		WHERE uuid = $1
	`, msgUUID)
	if err != nil {
		log.Printf("email worker: mark sent %s: %v", msgUUID, err)
	}
}

func (w *Worker) markFailed(tx *sqlx.Tx, msgUUID string, errMsg string) {
	_, err := tx.Exec(`
		UPDATE campaign_messages
		SET status = 'failed', error = $2, updated_at = now()
		WHERE uuid = $1
	`, msgUUID, errMsg)
	if err != nil {
		log.Printf("email worker: mark failed %s: %v", msgUUID, err)
	}
}

func (w *Worker) scheduleRetry(tx *sqlx.Tx, msgUUID string, retryCount int, backoff time.Duration, errMsg string) {
	newScheduled := time.Now().UTC().Add(backoff)
	_, err := tx.Exec(`
		UPDATE campaign_messages
		SET scheduled_at = $2,
		    error = $3,
		    metadata = jsonb_set(
		        COALESCE(metadata, '{}'::jsonb),
		        '{retryCount}',
		        to_jsonb($4::int)
		    ),
		    updated_at = now()
		WHERE uuid = $1
	`, msgUUID, newScheduled, errMsg, retryCount)
	if err != nil {
		log.Printf("email worker: schedule retry %s: %v", msgUUID, err)
	}
}

func (w *Worker) logEmail(tx *sqlx.Tx, shopUUID, toEmail, subject, status, errMsg string) {
	var sentAt *time.Time
	if status == "sent" {
		now := time.Now().UTC()
		sentAt = &now
	}
	var sqlErr *string
	if errMsg != "" {
		sqlErr = &errMsg
	}

	var shopPtr *string
	if shopUUID != "" {
		shopPtr = &shopUUID
	}

	_, err := tx.Exec(`
		INSERT INTO email_log(shop_uuid, to_email, subject, status, error, sent_at)
		VALUES($1, $2, $3, $4, $5, $6)
	`, toNullableUUID(shopPtr), toEmail, subject, status, sqlErr, sentAt)
	if err != nil {
		log.Printf("email worker: log email: %v", err)
	}
}

func toNullableUUID(s *string) interface{} {
	if s == nil || *s == "" {
		return sql.NullString{Valid: false}
	}
	return *s
}
