package server

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func registerCustomerRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
	app.Get("/v1/my/shops/:slug/customers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		q := strings.TrimSpace(c.Query("q"))
		tag := strings.TrimSpace(c.Query("tag"))
		query := `SELECT c.uuid, c.shop_uuid, c.user_uuid, c.email, c.first_name, c.last_name, c.phone, c.tags, c.notes, c.marketing_opt_in, c.created_at, c.updated_at,
                         COALESCE(SUM(oi.price_cents * oi.quantity),0) AS total_spent_cents,
                         COUNT(DISTINCT CASE WHEN o.uuid IS NOT NULL AND p.uuid IS NOT NULL THEN o.uuid END) AS orders_count,
                         MAX(o.created_at) AS last_order_at,
                         TRIM(BOTH ' ' FROM COALESCE(c.first_name,'') || ' ' || COALESCE(c.last_name,'')) AS customer_name
                  FROM customers c
                  LEFT JOIN orders o ON c.user_uuid IS NOT NULL AND o.user_uuid=c.user_uuid
                  LEFT JOIN order_items oi ON oi.order_uuid=o.uuid
                  LEFT JOIN products p ON p.uuid=oi.product_uuid AND p.shop_uuid=c.shop_uuid
                  WHERE c.shop_uuid=$1`
		args := []any{shop.UUID}
		if q != "" {
			like := "%" + strings.ToLower(q) + "%"
			args = append(args, like)
			idx := "$" + itoa(len(args))
			query += " AND (LOWER(c.email) LIKE " + idx + " OR LOWER(c.first_name) LIKE " + idx + " OR LOWER(c.last_name) LIKE " + idx + ")"
		}
		if tag != "" {
			args = append(args, strings.ToLower(tag))
			idx := "$" + itoa(len(args))
			query += " AND EXISTS (SELECT 1 FROM unnest(c.tags) AS tag WHERE LOWER(tag)=LOWER(" + idx + "))"
		}
		query += " GROUP BY c.uuid ORDER BY last_order_at DESC NULLS LAST, c.created_at DESC LIMIT 200"
		var customers []CustomerSummary
		if err := opts.DB.Select(&customers, query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": customers})
	})

	app.Post("/v1/my/shops/:slug/customers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Email          string   `json:"email"`
			FirstName      string   `json:"firstName"`
			LastName       string   `json:"lastName"`
			Phone          string   `json:"phone"`
			Notes          string   `json:"notes"`
			Tags           []string `json:"tags"`
			MarketingOptIn bool     `json:"marketingOptIn"`
			UserUUID       string   `json:"userUuid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		email, err := normalizeEmail(body.Email)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
		}
		first := strings.TrimSpace(body.FirstName)
		last := strings.TrimSpace(body.LastName)
		phone := strings.TrimSpace(body.Phone)
		notes := strings.TrimSpace(body.Notes)
		tags := normalizeTags(body.Tags)
		userUUID, err := parseOptionalUUID(body.UserUUID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid user uuid"})
		}
		now := time.Now()
		tempID := uuid.New()
		var userValue any
		if userUUID != nil {
			userValue = *userUUID
		}
		var created CustomerSummary
		if err := opts.DB.Get(&created, `INSERT INTO customers (uuid, shop_uuid, user_uuid, email, first_name, last_name, phone, tags, notes, marketing_opt_in, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
			ON CONFLICT (shop_uuid, email) DO UPDATE SET
				user_uuid=COALESCE(EXCLUDED.user_uuid, customers.user_uuid),
				first_name=EXCLUDED.first_name,
				last_name=EXCLUDED.last_name,
				phone=EXCLUDED.phone,
				tags=EXCLUDED.tags,
				notes=EXCLUDED.notes,
				marketing_opt_in=EXCLUDED.marketing_opt_in,
				updated_at=now()
			RETURNING uuid, shop_uuid, user_uuid, email, first_name, last_name, phone, tags, notes, marketing_opt_in, created_at, updated_at`,
			tempID, shop.UUID, userValue, email, first, last, phone, pqStringArray(tags), notes, body.MarketingOptIn, now); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		summary, err := fetchCustomerSummary(opts.DB, created.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
	})

	app.Get("/v1/customers/:id", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		summary, err := fetchCustomerSummary(opts.DB, customerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
	})

	app.Patch("/v1/customers/:id", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Email          *string   `json:"email"`
			FirstName      *string   `json:"firstName"`
			LastName       *string   `json:"lastName"`
			Phone          *string   `json:"phone"`
			Notes          *string   `json:"notes"`
			Tags           *[]string `json:"tags"`
			MarketingOptIn *bool     `json:"marketingOptIn"`
			UserUUID       *string   `json:"userUuid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		args := []any{}
		sets := []string{}
		add := func(field string, value any) {
			args = append(args, value)
			sets = append(sets, field+"=$"+itoa(len(args)))
		}
		if body.Email != nil {
			email, err := normalizeEmail(*body.Email)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
			}
			add("email", email)
		}
		if body.FirstName != nil {
			add("first_name", strings.TrimSpace(*body.FirstName))
		}
		if body.LastName != nil {
			add("last_name", strings.TrimSpace(*body.LastName))
		}
		if body.Phone != nil {
			add("phone", strings.TrimSpace(*body.Phone))
		}
		if body.Notes != nil {
			add("notes", strings.TrimSpace(*body.Notes))
		}
		if body.Tags != nil {
			add("tags", pqStringArray(normalizeTags(*body.Tags)))
		}
		if body.MarketingOptIn != nil {
			add("marketing_opt_in", *body.MarketingOptIn)
		}
		if body.UserUUID != nil {
			userUUID, err := parseOptionalUUID(*body.UserUUID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid user uuid"})
			}
			if userUUID != nil {
				args = append(args, *userUUID)
			} else {
				args = append(args, nil)
			}
			sets = append(sets, "user_uuid=$"+itoa(len(args)))
		}
		if len(sets) == 0 {
			summary, err := fetchCustomerSummary(opts.DB, customerID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			return c.JSON(fiber.Map{"success": true, "data": summary})
		}
		sets = append(sets, "updated_at=now()")
		args = append(args, customerID, meta.ShopUUID)
		query := "UPDATE customers SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args)-1) + " AND shop_uuid=$" + itoa(len(args))
		if _, err := opts.DB.Exec(query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		summary, err := fetchCustomerSummary(opts.DB, customerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
	})

	app.Get("/v1/customers/:id/orders", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		if meta.UserUUID == nil {
			return c.JSON(fiber.Map{"success": true, "data": []CustomerOrder{}})
		}
		var orders []CustomerOrder
		if err := opts.DB.Select(&orders, `SELECT o.uuid,
                                                   SUM(oi.price_cents * oi.quantity) AS total_cents,
                                                   o.currency,
                                                   o.status,
                                                   o.created_at,
                                                   o.updated_at,
                                                   o.tracking_number,
                                                   o.tracking_url,
                                                   o.shipping_carrier,
                                                   o.shipped_at,
                                                   o.delivered_at
                                            FROM orders o
                                            JOIN order_items oi ON oi.order_uuid=o.uuid
                                            JOIN products p ON p.uuid=oi.product_uuid
                                            WHERE p.shop_uuid=$1 AND o.user_uuid=$2
                                            GROUP BY o.uuid, o.currency, o.status, o.created_at, o.updated_at, o.tracking_number, o.tracking_url, o.shipping_carrier, o.shipped_at, o.delivered_at
                                            ORDER BY o.created_at DESC LIMIT 200`, meta.ShopUUID, *meta.UserUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": orders})
	})

}
