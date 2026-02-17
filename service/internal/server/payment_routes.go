package server

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	stripeClient "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/stripe"
)

func registerPaymentRoutes(app *fiber.App, opts Options) {
	// Stripe webhook — unauthenticated, verified via signature
	app.Post("/v1/webhooks/stripe", func(c *fiber.Ctx) error {
		if opts.StripeClient == nil {
			return c.Status(503).JSON(fiber.Map{"success": false, "message": "stripe not configured"})
		}
		payload := c.Body()
		sigHeader := c.Get("Stripe-Signature")
		if sigHeader == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "missing signature"})
		}

		event, err := stripeClient.VerifyWebhookSignature(payload, sigHeader, opts.StripeWebhookSecret)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid signature"})
		}

		switch event.Type {
		case "payment_intent.succeeded":
			return handlePaymentIntentSucceeded(c, opts, event)
		case "payment_intent.payment_failed":
			return handlePaymentIntentFailed(c, opts, event)
		case "charge.refunded":
			log.Printf("stripe webhook: charge.refunded event=%s", event.ID)
			return c.JSON(fiber.Map{"received": true})
		default:
			return c.JSON(fiber.Map{"received": true})
		}
	})
}

func registerPaymentSettingsRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
	app.Get("/v1/my/shops/:slug/payment-settings", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if role != "owner" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "owner only"})
		}

		var settings []PaymentSetting
		if err := opts.DB.Select(&settings, `SELECT uuid, shop_uuid, provider, stripe_publishable_key, stripe_secret_key, stripe_webhook_secret, is_active, created_at, updated_at
		                                     FROM payment_settings WHERE shop_uuid=$1 ORDER BY provider`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for i := range settings {
			if len(settings[i].StripeSecretKey) > 8 {
				settings[i].StripeSecretKey = settings[i].StripeSecretKey[:4] + "****" + settings[i].StripeSecretKey[len(settings[i].StripeSecretKey)-4:]
			}
			if len(settings[i].StripeWebhookSecret) > 8 {
				settings[i].StripeWebhookSecret = settings[i].StripeWebhookSecret[:4] + "****"
			}
		}

		return c.JSON(fiber.Map{"success": true, "data": settings})
	})

	app.Put("/v1/my/shops/:slug/payment-settings", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if role != "owner" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "owner only"})
		}

		var body struct {
			Providers []struct {
				ID            string `json:"id"`
				Enabled       bool   `json:"enabled"`
				APIKey        string `json:"apiKey"`
				SecretKey     string `json:"secretKey"`
				WebhookSecret string `json:"webhookSecret"`
			} `json:"providers"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		for _, p := range body.Providers {
			provider := strings.TrimSpace(p.ID)
			if provider == "" {
				continue
			}
			pubKey := strings.TrimSpace(p.APIKey)
			secKey := strings.TrimSpace(p.SecretKey)
			whSecret := strings.TrimSpace(p.WebhookSecret)

			if strings.Contains(secKey, "****") {
				secKey = ""
			}
			if strings.Contains(whSecret, "****") {
				whSecret = ""
			}

			var existing uuid.UUID
			err := opts.DB.Get(&existing, `SELECT uuid FROM payment_settings WHERE shop_uuid=$1 AND provider=$2`, shop.UUID, provider)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}

			if errors.Is(err, sql.ErrNoRows) {
				if _, err := opts.DB.Exec(`INSERT INTO payment_settings(uuid, shop_uuid, provider, stripe_publishable_key, stripe_secret_key, stripe_webhook_secret, is_active)
				                           VALUES($1,$2,$3,$4,$5,$6,$7)`,
					uuid.New(), shop.UUID, provider, pubKey, secKey, whSecret, p.Enabled); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
			} else {
				sets := []string{"is_active=$3", "updated_at=now()"}
				args := []any{shop.UUID, provider, p.Enabled}
				argIdx := 4
				if pubKey != "" {
					sets = append(sets, fmt.Sprintf("stripe_publishable_key=$%d", argIdx))
					args = append(args, pubKey)
					argIdx++
				}
				if secKey != "" {
					sets = append(sets, fmt.Sprintf("stripe_secret_key=$%d", argIdx))
					args = append(args, secKey)
					argIdx++
				}
				if whSecret != "" {
					sets = append(sets, fmt.Sprintf("stripe_webhook_secret=$%d", argIdx))
					args = append(args, whSecret)
				}

				query := "UPDATE payment_settings SET " + strings.Join(sets, ", ") + " WHERE shop_uuid=$1 AND provider=$2"
				if _, err := opts.DB.Exec(query, args...); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
			}
		}

		return c.JSON(fiber.Map{"success": true, "message": "payment settings saved"})
	})
}

func handlePaymentIntentSucceeded(c *fiber.Ctx, opts Options, event *stripeClient.WebhookEvent) error {
	pi, err := stripeClient.ParsePaymentIntentFromEvent(event)
	if err != nil {
		log.Printf("stripe webhook: parse error: %v", err)
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid event data"})
	}

	var orderID uuid.UUID
	if err := opts.DB.Get(&orderID, `SELECT uuid FROM orders WHERE stripe_payment_intent_id=$1`, pi.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("stripe webhook: no order found for PI %s", pi.ID)
			return c.JSON(fiber.Map{"received": true})
		}
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}

	tx, err := opts.DB.Beginx()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE orders SET status='pending', updated_at=now() WHERE uuid=$1 AND status='awaiting_payment'`, orderID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if _, err := tx.Exec(`UPDATE payment_intents SET status='succeeded', updated_at=now() WHERE order_uuid=$1`, orderID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}

	eventMeta := map[string]any{
		"stripePaymentIntentId": pi.ID,
		"event":                 event.Type,
	}
	if _, err := recordOrderEvent(tx, orderID, "payment.succeeded", "Payment confirmed via Stripe", nil, eventMeta); err != nil {
		log.Printf("stripe webhook: record event error: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}

	return c.JSON(fiber.Map{"received": true})
}

func handlePaymentIntentFailed(c *fiber.Ctx, opts Options, event *stripeClient.WebhookEvent) error {
	pi, err := stripeClient.ParsePaymentIntentFromEvent(event)
	if err != nil {
		log.Printf("stripe webhook: parse error: %v", err)
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid event data"})
	}

	var orderID uuid.UUID
	if err := opts.DB.Get(&orderID, `SELECT uuid FROM orders WHERE stripe_payment_intent_id=$1`, pi.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(fiber.Map{"received": true})
		}
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}

	if _, err := opts.DB.Exec(`UPDATE payment_intents SET status='failed', updated_at=now() WHERE order_uuid=$1`, orderID); err != nil {
		log.Printf("stripe webhook: update PI error: %v", err)
	}

	return c.JSON(fiber.Map{"received": true})
}

func getShopStripeKey(db interface{ Get(dest interface{}, query string, args ...interface{}) error }, shopUUID uuid.UUID, globalKey string) string {
	var key string
	err := db.Get(&key, `SELECT stripe_secret_key FROM payment_settings WHERE shop_uuid=$1 AND provider='stripe' AND is_active=true AND stripe_secret_key!=''`, shopUUID)
	if err == nil && key != "" {
		return key
	}
	return globalKey
}

func createStripePaymentIntent(opts Options, orderID uuid.UUID, shopUUID uuid.UUID, amountCents int64, currency string) (string, string, error) {
	if opts.StripeClient == nil {
		return "", "", nil
	}

	secretKey := getShopStripeKey(opts.DB, shopUUID, opts.StripeClient.SecretKey)
	client := &stripeClient.Client{SecretKey: secretKey, HTTP: opts.StripeClient.HTTP}

	metadata := map[string]string{
		"order_uuid": orderID.String(),
		"shop_uuid":  shopUUID.String(),
	}

	pi, err := client.CreatePaymentIntent(amountCents, currency, orderID.String(), metadata)
	if err != nil {
		return "", "", err
	}

	if _, err := opts.DB.Exec(`INSERT INTO payment_intents(uuid, order_uuid, shop_uuid, stripe_payment_intent_id, client_secret, amount_cents, currency, status)
	                           VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		uuid.New(), orderID, shopUUID, pi.ID, pi.ClientSecret, amountCents, strings.ToLower(currency), pi.Status); err != nil {
		return "", "", err
	}

	if _, err := opts.DB.Exec(`UPDATE orders SET stripe_payment_intent_id=$2, updated_at=now() WHERE uuid=$1`, orderID, pi.ID); err != nil {
		return "", "", err
	}

	return pi.ClientSecret, pi.ID, nil
}
