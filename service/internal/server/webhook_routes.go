package server

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	pq "github.com/lib/pq"
)

func registerWebhookRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {

	// List webhooks for a shop
	app.Get("/v1/my/shops/:slug/webhooks", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var webhooks []Webhook
		if err := opts.DB.Select(&webhooks,
			`SELECT uuid, shop_uuid, url, secret, events, is_active, created_at, updated_at
			 FROM webhooks WHERE shop_uuid=$1 ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if webhooks == nil {
			webhooks = []Webhook{}
		}
		return c.JSON(fiber.Map{"success": true, "data": webhooks})
	})

	// Create a webhook
	app.Post("/v1/my/shops/:slug/webhooks", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			URL      string   `json:"url"`
			Secret   string   `json:"secret"`
			Events   []string `json:"events"`
			IsActive *bool    `json:"isActive"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		url := strings.TrimSpace(body.URL)
		if url == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "url is required"})
		}
		if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "url must start with http:// or https://"})
		}

		events := sanitizeEvents(body.Events)
		isActive := true
		if body.IsActive != nil {
			isActive = *body.IsActive
		}

		id := uuid.New()
		_, err = opts.DB.Exec(
			`INSERT INTO webhooks (uuid, shop_uuid, url, secret, events, is_active)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			id, shop.UUID, url, strings.TrimSpace(body.Secret), pq.Array(events), isActive)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		var webhook Webhook
		if err := opts.DB.Get(&webhook,
			`SELECT uuid, shop_uuid, url, secret, events, is_active, created_at, updated_at
			 FROM webhooks WHERE uuid=$1`, id); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": webhook})
	})

	// Update a webhook
	app.Patch("/v1/webhooks/:id", requireAuth, func(c *fiber.Ctx) error {
		whID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid webhook id"})
		}

		// Look up webhook to get shop_uuid, then verify access
		var existing Webhook
		if err := opts.DB.Get(&existing,
			`SELECT uuid, shop_uuid, url, secret, events, is_active, created_at, updated_at
			 FROM webhooks WHERE uuid=$1`, whID); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "webhook not found"})
		}

		// Verify access via shop slug lookup
		var shopSlug string
		if err := opts.DB.Get(&shopSlug, `SELECT slug FROM shops WHERE uuid=$1`, existing.ShopUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		_, role, err := ensureShopAccess(c, opts.DB, shopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			URL      *string  `json:"url"`
			Secret   *string  `json:"secret"`
			Events   []string `json:"events"`
			IsActive *bool    `json:"isActive"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		if body.URL != nil {
			url := strings.TrimSpace(*body.URL)
			if url == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "url cannot be empty"})
			}
			if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "url must start with http:// or https://"})
			}
			existing.URL = url
		}
		if body.Secret != nil {
			existing.Secret = strings.TrimSpace(*body.Secret)
		}
		if body.Events != nil {
			existing.Events = pq.StringArray(sanitizeEvents(body.Events))
		}
		if body.IsActive != nil {
			existing.IsActive = *body.IsActive
		}

		_, err = opts.DB.Exec(
			`UPDATE webhooks SET url=$1, secret=$2, events=$3, is_active=$4, updated_at=now() WHERE uuid=$5`,
			existing.URL, existing.Secret, existing.Events, existing.IsActive, whID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		var webhook Webhook
		if err := opts.DB.Get(&webhook,
			`SELECT uuid, shop_uuid, url, secret, events, is_active, created_at, updated_at
			 FROM webhooks WHERE uuid=$1`, whID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": webhook})
	})

	// Delete a webhook
	app.Delete("/v1/webhooks/:id", requireAuth, func(c *fiber.Ctx) error {
		whID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid webhook id"})
		}

		var shopUUID uuid.UUID
		if err := opts.DB.Get(&shopUUID, `SELECT shop_uuid FROM webhooks WHERE uuid=$1`, whID); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "webhook not found"})
		}

		var shopSlug string
		if err := opts.DB.Get(&shopSlug, `SELECT slug FROM shops WHERE uuid=$1`, shopUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		_, role, err := ensureShopAccess(c, opts.DB, shopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		res, err := opts.DB.Exec(`DELETE FROM webhooks WHERE uuid=$1`, whID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "webhook not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// List delivery log for a webhook (paginated)
	app.Get("/v1/webhooks/:id/deliveries", requireAuth, func(c *fiber.Ctx) error {
		whID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid webhook id"})
		}

		var shopUUID uuid.UUID
		if err := opts.DB.Get(&shopUUID, `SELECT shop_uuid FROM webhooks WHERE uuid=$1`, whID); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "webhook not found"})
		}

		var shopSlug string
		if err := opts.DB.Get(&shopSlug, `SELECT slug FROM shops WHERE uuid=$1`, shopUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		_, role, err := ensureShopAccess(c, opts.DB, shopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		limit := 20
		offset := 0
		if v := c.Query("limit"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 && n <= 100 {
				limit = n
			}
		}
		if v := c.Query("offset"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n >= 0 {
				offset = n
			}
		}

		var deliveries []WebhookDelivery
		if err := opts.DB.Select(&deliveries,
			`SELECT uuid, webhook_uuid, event_type, payload, response_status, response_body,
			        attempt, status, next_retry_at, error, created_at, updated_at
			 FROM webhook_deliveries WHERE webhook_uuid=$1
			 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, whID, limit, offset); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if deliveries == nil {
			deliveries = []WebhookDelivery{}
		}

		var total int
		_ = opts.DB.Get(&total, `SELECT COUNT(1) FROM webhook_deliveries WHERE webhook_uuid=$1`, whID)

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"deliveries": deliveries,
			"total":      total,
			"limit":      limit,
			"offset":     offset,
		}})
	})
}

func sanitizeEvents(events []string) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		e = strings.TrimSpace(strings.ToLower(e))
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}
