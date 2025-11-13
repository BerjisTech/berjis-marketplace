package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func registerDiscountRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
	app.Get("/v1/my/shops/:slug/discounts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var discounts []Discount
		if err := opts.DB.Select(&discounts, `SELECT uuid, shop_uuid, name, code, description, discount_type, amount_cents, percentage,
                                                     minimum_subtotal_cents, free_shipping, buy_quantity, get_quantity, get_percentage,
                                                     starts_at, ends_at,
                                                     usage_limit_total, usage_limit_per_customer, auto_apply, status, applies_to, created_at, updated_at
                                              FROM discounts
                                              WHERE shop_uuid=$1
                                              ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": discounts})
	})
	app.Get("/v1/my/shops/:slug/discounts/report", requireAuth, func(c *fiber.Ctx) error {
		return getDiscountReport(c, opts.DB)
	})

	app.Post("/v1/my/shops/:slug/discounts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			Name                  string          `json:"name"`
			Code                  string          `json:"code"`
			Description           string          `json:"description"`
			DiscountType          string          `json:"discountType"`
			AmountCents           *int64          `json:"amountCents"`
			Percentage            *float64        `json:"percentage"`
			MinimumSubtotalCents  *int64          `json:"minimumSubtotalCents"`
			FreeShipping          bool            `json:"freeShipping"`
			BuyQuantity           *int            `json:"buyQuantity"`
			GetQuantity           *int            `json:"getQuantity"`
			GetPercentage         *float64        `json:"getPercentage"`
			StartsAt              *time.Time      `json:"startsAt"`
			EndsAt                *time.Time      `json:"endsAt"`
			UsageLimitTotal       *int            `json:"usageLimitTotal"`
			UsageLimitPerCustomer *int            `json:"usageLimitPerCustomer"`
			AutoApply             bool            `json:"autoApply"`
			Status                string          `json:"status"`
			AppliesTo             json.RawMessage `json:"appliesTo"`
			ProductUUIDs          []string        `json:"productUuids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		name := strings.TrimSpace(body.Name)
		code := strings.ToUpper(strings.TrimSpace(body.Code))
		if name == "" || code == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "name and code are required"})
		}

		discountType := strings.ToLower(strings.TrimSpace(body.DiscountType))
		if discountType == "" {
			discountType = "percentage"
		}
		switch discountType {
		case "percentage", "amount", "free_shipping", "bogo":
		default:
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount type"})
		}

		amount := int64(0)
		if body.AmountCents != nil {
			amount = *body.AmountCents
		}
		percentage := 0.0
		if body.Percentage != nil {
			percentage = *body.Percentage
		}
		if discountType == "amount" && amount <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents must be greater than zero for amount discounts"})
		}
		if discountType == "percentage" && percentage <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage must be greater than zero for percentage discounts"})
		}

		minSubtotal := int64(0)
		if body.MinimumSubtotalCents != nil && *body.MinimumSubtotalCents > 0 {
			minSubtotal = *body.MinimumSubtotalCents
		}

		freeShipping := body.FreeShipping || discountType == "free_shipping"
		if discountType == "free_shipping" {
			freeShipping = true
		}

		var buyQty *int
		var getQty *int
		getPct := 100.0
		if body.GetPercentage != nil {
			getPct = *body.GetPercentage
		}
		if getPct <= 0 {
			getPct = 100
		}
		if getPct > 100 {
			getPct = 100
		}

		if discountType == "bogo" {
			if body.BuyQuantity == nil || *body.BuyQuantity <= 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "buyQuantity must be greater than zero for buy X get Y discounts"})
			}
			if body.GetQuantity == nil || *body.GetQuantity <= 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "getQuantity must be greater than zero for buy X get Y discounts"})
			}
			if len(body.ProductUUIDs) == 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "buy X get Y discounts require product targets"})
			}
			buy := *body.BuyQuantity
			get := *body.GetQuantity
			buyQty = &buy
			getQty = &get
		}

		status := normalizeDiscountStatus(body.Status)
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO discounts(uuid, shop_uuid, name, code, description, discount_type, amount_cents, percentage,
                                               minimum_subtotal_cents, free_shipping, buy_quantity, get_quantity, get_percentage,
                                               starts_at, ends_at,
                                               usage_limit_total, usage_limit_per_customer, auto_apply, status, applies_to)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
			id, shop.UUID, name, code, strings.TrimSpace(body.Description), discountType, amount, percentage,
			minSubtotal, freeShipping, buyQty, getQty, getPct,
			body.StartsAt, body.EndsAt,
			body.UsageLimitTotal, body.UsageLimitPerCustomer, body.AutoApply, status, nullIfEmptyJSON(body.AppliesTo)); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if len(body.ProductUUIDs) > 0 {
			productIDs, parseErr := parseUUIDList(body.ProductUUIDs)
			if parseErr != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": parseErr.Error()})
			}
			if err := setDiscountProducts(opts.DB, id, shop.UUID, productIDs); err != nil {
				return respondWithError(c, err)
			}
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	app.Patch("/v1/my/shops/:slug/discounts/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		discountID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var exists int
		if err := opts.DB.Get(&exists, `SELECT COUNT(1) FROM discounts WHERE uuid=$1 AND shop_uuid=$2`, discountID, shop.UUID); err != nil || exists == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 12)
		args := make([]any, 0, 12)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["name"].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				add("name", s)
			}
		}
		if v, ok := body["code"].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				add("code", strings.ToUpper(s))
			}
		}
		if v, ok := body["description"].(string); ok {
			add("description", strings.TrimSpace(v))
		}
		if v, ok := body["discountType"].(string); ok {
			t := strings.ToLower(strings.TrimSpace(v))
			if t != "" && t != "percentage" && t != "amount" && t != "free_shipping" && t != "bogo" {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount type"})
			}
			if t != "" {
				add("discount_type", t)
				if t == "free_shipping" {
					add("free_shipping", true)
				}
			}
		}
		if raw, ok := body["amountCents"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", int64(val))
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", int64(val))
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", val)
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid amountCents"})
			}
		}
		if raw, ok := body["percentage"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", val)
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", float64(val))
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", float64(val))
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid percentage"})
			}
		}
		if raw, ok := body["minimumSubtotalCents"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "minimumSubtotalCents cannot be negative"})
				}
				add("minimum_subtotal_cents", int64(val))
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "minimumSubtotalCents cannot be negative"})
				}
				add("minimum_subtotal_cents", int64(val))
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "minimumSubtotalCents cannot be negative"})
				}
				add("minimum_subtotal_cents", val)
			case nil:
				add("minimum_subtotal_cents", 0)
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid minimumSubtotalCents"})
			}
		}
		if raw, ok := body["freeShipping"]; ok {
			switch val := raw.(type) {
			case bool:
				add("free_shipping", val)
			case nil:
				add("free_shipping", false)
			}
		}
		if raw, ok := body["buyQuantity"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "buyQuantity must be positive"})
				}
				add("buy_quantity", int(val))
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "buyQuantity must be positive"})
				}
				add("buy_quantity", val)
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "buyQuantity must be positive"})
				}
				add("buy_quantity", val)
			case nil:
				add("buy_quantity", nil)
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid buyQuantity"})
			}
		}
		if raw, ok := body["getQuantity"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "getQuantity must be positive"})
				}
				add("get_quantity", int(val))
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "getQuantity must be positive"})
				}
				add("get_quantity", val)
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "getQuantity must be positive"})
				}
				add("get_quantity", val)
			case nil:
				add("get_quantity", nil)
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid getQuantity"})
			}
		}
		if raw, ok := body["getPercentage"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "getPercentage cannot be negative"})
				}
				if val > 100 {
					val = 100
				}
				add("get_percentage", val)
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "getPercentage cannot be negative"})
				}
				if val > 100 {
					val = 100
				}
				add("get_percentage", float64(val))
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "getPercentage cannot be negative"})
				}
				if val > 100 {
					val = 100
				}
				add("get_percentage", float64(val))
			case nil:
				add("get_percentage", 100.0)
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid getPercentage"})
			}
		}
		if v, ok := body["startsAt"].(string); ok {
			if v == "" {
				add("starts_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("starts_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid startsAt"})
			}
		}
		if v, ok := body["endsAt"].(string); ok {
			if v == "" {
				add("ends_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("ends_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid endsAt"})
			}
		}
		if raw, ok := body["usageLimitTotal"]; ok {
			switch val := raw.(type) {
			case float64:
				add("usage_limit_total", int(val))
			case int:
				add("usage_limit_total", val)
			case int64:
				add("usage_limit_total", val)
			case nil:
				add("usage_limit_total", nil)
			}
		}
		if raw, ok := body["usageLimitPerCustomer"]; ok {
			switch val := raw.(type) {
			case float64:
				add("usage_limit_per_customer", int(val))
			case int:
				add("usage_limit_per_customer", val)
			case int64:
				add("usage_limit_per_customer", val)
			case nil:
				add("usage_limit_per_customer", nil)
			}
		}
		if v, ok := body["autoApply"].(bool); ok {
			add("auto_apply", v)
		}
		if v, ok := body["status"].(string); ok {
			add("status", normalizeDiscountStatus(v))
		}
		if raw, ok := body["appliesTo"]; ok {
			b, err := json.Marshal(raw)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid appliesTo"})
			}
			add("applies_to", nullIfEmptyJSON(b))
		}
		var productUpdateIDs []uuid.UUID
		if raw, ok := body["productUuids"]; ok {
			switch arr := raw.(type) {
			case []any:
				strs := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						strs = append(strs, s)
					}
				}
				ids, err := parseUUIDList(strs)
				if err != nil {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
				}
				productUpdateIDs = ids
			case nil:
				productUpdateIDs = []uuid.UUID{}
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid productUuids"})
			}
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, discountID)
			query := "UPDATE discounts SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if productUpdateIDs != nil {
			if err := setDiscountProducts(opts.DB, discountID, shop.UUID, productUpdateIDs); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Delete("/v1/my/shops/:slug/discounts/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		discountID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		res, err := opts.DB.Exec(`UPDATE discounts SET status='archived', updated_at=now() WHERE uuid=$1 AND shop_uuid=$2`, discountID, shop.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if _, err := opts.DB.Exec(`DELETE FROM discount_products WHERE discount_uuid=$1`, discountID); err != nil {
			log.Printf("discount_products cleanup error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/gift-cards", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var cards []GiftCard
		if err := opts.DB.Select(&cards, `SELECT uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at, issued_at, redeemed_at, created_at, updated_at
                                         FROM gift_cards WHERE shop_uuid=$1 ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": cards})
	})

	app.Post("/v1/my/shops/:slug/gift-cards", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Code         string     `json:"code"`
			BalanceCents int64      `json:"balanceCents"`
			Currency     string     `json:"currency"`
			IssuedTo     *string    `json:"issuedToEmail"`
			Note         string     `json:"note"`
			Status       string     `json:"status"`
			ExpiresAt    *time.Time `json:"expiresAt"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		code := normalizeGiftCardCode(body.Code)
		if code == "" {
			generated, err := generateUniqueGiftCardCode(opts.DB, shop.UUID)
			if err != nil {
				log.Printf("gift card code generation failed: %v", err)
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "could not generate gift card code"})
			}
			code = generated
		} else {
			exists, err := giftCardCodeExists(opts.DB, shop.UUID, code)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if exists {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "a gift card with that code already exists"})
			}
		}
		if body.BalanceCents <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "balance must be greater than zero"})
		}
		currency := strings.ToUpper(strings.TrimSpace(body.Currency))
		if currency == "" {
			currency = "USD"
		}
		status := normalizeGiftCardStatus(body.Status)
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO gift_cards(uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			id, shop.UUID, code, body.BalanceCents, body.BalanceCents, currency, body.IssuedTo, strings.TrimSpace(body.Note), status, body.ExpiresAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := addGiftCardTransaction(opts.DB, id, body.BalanceCents, "issued"); err != nil {
			log.Printf("gift card transaction error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id, "code": code}})
	})

	app.Get("/v1/my/shops/:slug/gift-cards/report", requireAuth, func(c *fiber.Ctx) error {
		return getGiftCardReport(c, opts.DB)
	})

	app.Patch("/v1/my/shops/:slug/gift-cards/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		cardID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid gift card id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var current GiftCard
		if err := opts.DB.Get(&current, `SELECT uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at, issued_at, redeemed_at, created_at, updated_at
                                         FROM gift_cards WHERE uuid=$1 AND shop_uuid=$2`, cardID, shop.UUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 8)
		args := make([]any, 0, 8)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["note"].(string); ok {
			add("note", strings.TrimSpace(v))
		}
		if v, ok := body["expiresAt"].(string); ok {
			if v == "" {
				add("expires_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("expires_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid expiresAt"})
			}
		}
		if v, ok := body["status"].(string); ok {
			newStatus := normalizeGiftCardStatus(v)
			add("status", newStatus)
			if newStatus == "redeemed" && current.RedeemedAt == nil {
				add("redeemed_at", time.Now())
			}
		}
		if raw, ok := body["balanceAdjustmentCents"]; ok {
			var adjust int64
			switch val := raw.(type) {
			case float64:
				adjust = int64(val)
			case int:
				adjust = int64(val)
			case int64:
				adjust = val
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid balanceAdjustmentCents"})
			}
			newBalance := current.BalanceCents + adjust
			if newBalance < 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "insufficient balance"})
			}
			add("balance_cents", newBalance)
			if adjust != 0 {
				if err := addGiftCardTransaction(opts.DB, cardID, adjust, "manual_adjustment"); err != nil {
					log.Printf("gift card transaction error: %v", err)
				}
				current.BalanceCents = newBalance
			}
		}
		if email, ok := body["issuedToEmail"].(string); ok {
			if strings.TrimSpace(email) == "" {
				add("issued_to_email", nil)
			} else {
				add("issued_to_email", strings.TrimSpace(email))
			}
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, cardID)
			query := "UPDATE gift_cards SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/gift-cards/:id/transactions", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		cardID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid gift card id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var okCount int
		if err := opts.DB.Get(&okCount, `SELECT COUNT(1) FROM gift_cards WHERE uuid=$1 AND shop_uuid=$2`, cardID, shop.UUID); err != nil || okCount == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var txs []GiftCardTransaction
		if err := opts.DB.Select(&txs, `SELECT uuid, gift_card_uuid, change_cents, reason, created_at
                                        FROM gift_card_transactions
                                        WHERE gift_card_uuid=$1
                                        ORDER BY created_at DESC`, cardID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": txs})
	})

}
