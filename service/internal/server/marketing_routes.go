package server

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func registerMarketingRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
	app.Get("/v1/my/shops/:slug/campaigns", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var campaigns []MarketingCampaign
		if err := opts.DB.Select(&campaigns, `SELECT uuid, shop_uuid, name, channel, status, budget_cents, spend_cents, starts_at, ends_at, metadata, created_at, updated_at
                                               FROM marketing_campaigns
                                               WHERE shop_uuid=$1
                                               ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": campaigns})
	})

	app.Post("/v1/my/shops/:slug/campaigns", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			Name        string          `json:"name"`
			Channel     string          `json:"channel"`
			BudgetCents *int64          `json:"budgetCents"`
			StartsAt    *time.Time      `json:"startsAt"`
			EndsAt      *time.Time      `json:"endsAt"`
			Metadata    json.RawMessage `json:"metadata"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		name := strings.TrimSpace(body.Name)
		channel := strings.ToLower(strings.TrimSpace(body.Channel))
		if name == "" || channel == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "name and channel required"})
		}
		budget := int64(0)
		if body.BudgetCents != nil && *body.BudgetCents > 0 {
			budget = *body.BudgetCents
		}
		if body.StartsAt != nil && body.EndsAt != nil && body.EndsAt.Before(*body.StartsAt) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "endsAt must be after startsAt"})
		}
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO marketing_campaigns(uuid, shop_uuid, name, channel, status, budget_cents, spend_cents, starts_at, ends_at, metadata)
                                    VALUES($1,$2,$3,$4,'draft',$5,0,$6,$7,$8)`,
			id, shop.UUID, name, channel, budget, body.StartsAt, body.EndsAt, nullIfEmptyJSON(body.Metadata)); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var created MarketingCampaign
		if err := opts.DB.Get(&created, `SELECT uuid, shop_uuid, name, channel, status, budget_cents, spend_cents, starts_at, ends_at, metadata, created_at, updated_at
                                         FROM marketing_campaigns WHERE uuid=$1`, id); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": created})
	})

	app.Post("/v1/my/shops/:slug/campaigns/:id/schedule-email", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		campaignID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid campaign id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var campaign MarketingCampaign
		if err := opts.DB.Get(&campaign, `SELECT uuid, shop_uuid, name, channel, status, budget_cents, spend_cents, starts_at, ends_at, metadata, created_at, updated_at
                                         FROM marketing_campaigns
                                         WHERE uuid=$1 AND shop_uuid=$2`, campaignID, shop.UUID); err != nil {
			return respondWithError(c, err)
		}
		var body struct {
			Subject     string          `json:"subject"`
			Body        string          `json:"body"`
			ScheduledAt time.Time       `json:"scheduledAt"`
			SendAfter   *time.Time      `json:"sendAfter"`
			Metadata    json.RawMessage `json:"metadata"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		subject := strings.TrimSpace(body.Subject)
		content := strings.TrimSpace(body.Body)
		if subject == "" || content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "subject and body required"})
		}
		scheduled := body.ScheduledAt
		if scheduled.IsZero() {
			scheduled = time.Now().Add(15 * time.Minute)
		}
		if body.SendAfter != nil && body.SendAfter.After(scheduled) {
			scheduled = *body.SendAfter
		}
		messageID := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO campaign_messages(uuid, campaign_uuid, shop_uuid, subject, body, status, scheduled_at, send_after, metadata)
                                    VALUES($1,$2,$3,$4,$5,'scheduled',$6,$7,$8)`,
			messageID, campaign.UUID, shop.UUID, subject, content, scheduled, body.SendAfter, nullIfEmptyJSON(body.Metadata)); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var message CampaignMessage
		if err := opts.DB.Get(&message, `SELECT uuid, campaign_uuid, shop_uuid, subject, body, status, scheduled_at, send_after, sent_at, error, metadata, created_at, updated_at
                                         FROM campaign_messages WHERE uuid=$1`, messageID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": message})
	})
}
