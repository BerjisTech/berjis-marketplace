package server

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/email"
)

func registerNotificationRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {

	// GET email settings for a shop
	app.Get("/v1/my/shops/:slug/email-settings", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if role != "owner" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "owner only"})
		}

		var settings ShopEmailSettings
		err = opts.DB.Get(&settings, `SELECT uuid, shop_uuid, provider, smtp_host, smtp_port, smtp_username, smtp_password,
		                                     sendgrid_api_key, from_email, from_name, is_active, created_at, updated_at
		                              FROM shop_email_settings WHERE shop_uuid=$1`, shop.UUID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
					"provider":  "smtp",
					"smtpHost":  "",
					"smtpPort":  587,
					"fromEmail": "",
					"fromName":  "",
					"isActive":  false,
				}})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		// Mask sensitive fields
		if len(settings.SMTPPassword) > 4 {
			settings.SMTPPassword = settings.SMTPPassword[:2] + "****"
		}
		if len(settings.SendGridAPIKey) > 8 {
			settings.SendGridAPIKey = settings.SendGridAPIKey[:4] + "****"
		}

		return c.JSON(fiber.Map{"success": true, "data": settings})
	})

	// PUT email settings for a shop
	app.Put("/v1/my/shops/:slug/email-settings", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if role != "owner" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "owner only"})
		}

		var body struct {
			Provider       string `json:"provider"`
			SMTPHost       string `json:"smtpHost"`
			SMTPPort       int    `json:"smtpPort"`
			SMTPUsername   string `json:"smtpUsername"`
			SMTPPassword   string `json:"smtpPassword"`
			SendGridAPIKey string `json:"sendgridApiKey"`
			FromEmail      string `json:"fromEmail"`
			FromName       string `json:"fromName"`
			IsActive       bool   `json:"isActive"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		provider := strings.TrimSpace(body.Provider)
		if provider == "" {
			provider = "smtp"
		}
		if provider != "smtp" && provider != "sendgrid" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "provider must be smtp or sendgrid"})
		}
		if body.SMTPPort <= 0 {
			body.SMTPPort = 587
		}

		smtpPassword := strings.TrimSpace(body.SMTPPassword)
		sendgridKey := strings.TrimSpace(body.SendGridAPIKey)

		// Don't overwrite secrets if masked value was sent back
		if strings.Contains(smtpPassword, "****") {
			smtpPassword = ""
		}
		if strings.Contains(sendgridKey, "****") {
			sendgridKey = ""
		}

		var existing uuid.UUID
		err = opts.DB.Get(&existing, `SELECT uuid FROM shop_email_settings WHERE shop_uuid=$1`, shop.UUID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if errors.Is(err, sql.ErrNoRows) {
			if _, err := opts.DB.Exec(`INSERT INTO shop_email_settings(uuid, shop_uuid, provider, smtp_host, smtp_port, smtp_username, smtp_password, sendgrid_api_key, from_email, from_name, is_active)
			                           VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				uuid.New(), shop.UUID, provider,
				strings.TrimSpace(body.SMTPHost), body.SMTPPort,
				strings.TrimSpace(body.SMTPUsername), smtpPassword,
				sendgridKey,
				strings.TrimSpace(body.FromEmail), strings.TrimSpace(body.FromName),
				body.IsActive); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		} else {
			sets := []string{
				"provider=$2",
				"smtp_host=$3",
				"smtp_port=$4",
				"smtp_username=$5",
				"from_email=$6",
				"from_name=$7",
				"is_active=$8",
				"updated_at=now()",
			}
			args := []any{
				shop.UUID,
				provider,
				strings.TrimSpace(body.SMTPHost),
				body.SMTPPort,
				strings.TrimSpace(body.SMTPUsername),
				strings.TrimSpace(body.FromEmail),
				strings.TrimSpace(body.FromName),
				body.IsActive,
			}
			argIdx := 9
			if smtpPassword != "" {
				sets = append(sets, fmt.Sprintf("smtp_password=$%d", argIdx))
				args = append(args, smtpPassword)
				argIdx++
			}
			if sendgridKey != "" {
				sets = append(sets, fmt.Sprintf("sendgrid_api_key=$%d", argIdx))
				args = append(args, sendgridKey)
			}

			query := "UPDATE shop_email_settings SET " + strings.Join(sets, ", ") + " WHERE shop_uuid=$1"
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}

		return c.JSON(fiber.Map{"success": true, "message": "email settings saved"})
	})

	// GET notification preferences for a shop
	app.Get("/v1/my/shops/:slug/notification-settings", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var prefs []NotificationPreference
		if err := opts.DB.Select(&prefs, `SELECT uuid, shop_uuid, event_type, enabled, template_subject, template_body, created_at, updated_at
		                                   FROM notification_preferences WHERE shop_uuid=$1 ORDER BY event_type`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		// Return defaults for any missing event types
		existingTypes := make(map[string]bool)
		for _, p := range prefs {
			existingTypes[p.EventType] = true
		}
		defaultEvents := []string{
			"order_confirmation", "shipping_update", "refund_notification",
			"abandoned_cart_reminder", "welcome_email", "team_invitation",
		}
		for _, et := range defaultEvents {
			if !existingTypes[et] {
				prefs = append(prefs, NotificationPreference{
					EventType: et,
					Enabled:   true,
					ShopUUID:  shop.UUID,
				})
			}
		}

		return c.JSON(fiber.Map{"success": true, "data": prefs})
	})

	// PUT notification preferences for a shop
	app.Put("/v1/my/shops/:slug/notification-settings", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			Preferences []struct {
				EventType       string  `json:"eventType"`
				Enabled         bool    `json:"enabled"`
				TemplateSubject *string `json:"templateSubject"`
				TemplateBody    *string `json:"templateBody"`
			} `json:"preferences"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		for _, pref := range body.Preferences {
			eventType := strings.TrimSpace(pref.EventType)
			if eventType == "" {
				continue
			}

			if _, err := opts.DB.Exec(`
				INSERT INTO notification_preferences(uuid, shop_uuid, event_type, enabled, template_subject, template_body)
				VALUES($1, $2, $3, $4, $5, $6)
				ON CONFLICT (shop_uuid, event_type)
				DO UPDATE SET enabled=$4, template_subject=$5, template_body=$6, updated_at=now()
			`, uuid.New(), shop.UUID, eventType, pref.Enabled, pref.TemplateSubject, pref.TemplateBody); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}

		return c.JSON(fiber.Map{"success": true, "message": "notification settings saved"})
	})

	// POST send test email
	app.Post("/v1/my/shops/:slug/notifications/:type/test", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		notifType := c.Params("type")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if role != "owner" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "owner only"})
		}

		if opts.EmailSender == nil {
			return c.Status(503).JSON(fiber.Map{"success": false, "message": "email sending not configured"})
		}

		var body struct {
			ToEmail string `json:"toEmail"`
		}
		if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.ToEmail) == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "toEmail required"})
		}

		var fromName string
		_ = opts.DB.Get(&fromName, `SELECT from_name FROM shop_email_settings WHERE shop_uuid=$1`, shop.UUID)
		if fromName == "" {
			fromName = shop.Name
		}

		htmlBody, err := email.RenderEmail(notifType, email.TemplateData{
			ShopName:     fromName,
			CustomerName: "Test Customer",
			OrderNumber:  "TEST-00001",
			RefundAmount: "$10.00",
			Body:         "This is a test notification email.",
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "template error"})
		}

		subject := "Test: " + notifType
		if sendErr := opts.EmailSender.Send(strings.TrimSpace(body.ToEmail), subject, htmlBody); sendErr != nil {
			opts.DB.Exec(`INSERT INTO email_log(shop_uuid, to_email, subject, status, error) VALUES($1,$2,$3,'failed',$4)`,
				shop.UUID, body.ToEmail, subject, sendErr.Error())
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "send failed: " + sendErr.Error()})
		}

		opts.DB.Exec(`INSERT INTO email_log(shop_uuid, to_email, subject, status, sent_at) VALUES($1,$2,$3,'sent',now())`,
			shop.UUID, body.ToEmail, subject)

		return c.JSON(fiber.Map{"success": true, "message": "test email sent"})
	})
}
